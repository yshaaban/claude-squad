package static

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed dist
var ReactApp embed.FS

// spaFileServer is a custom file server that serves a Single Page Application (SPA)
type spaFileServer struct {
	fs http.FileSystem
}

func (f *spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set standard security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	
	// Clean path to prevent directory traversal
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	upath = path.Clean(upath)

	// Print debug information about the request path
	fmt.Printf("DEBUG: React file server request for path: %s\n", upath)

	// Check for special cases like assets/ directory
	isAssetRequest := strings.HasPrefix(upath, "/assets/")
	
	// Determine file existence and handle path remapping if needed
	if isAssetRequest {
		// Convert absolute paths (/assets/) to relative paths (assets/)
		trimmedPath := strings.TrimPrefix(upath, "/")
		
		// Try to locate the asset with different path formats
		filePaths := []string{
			path.Join("dist", upath),          // /assets/file.js
			path.Join("dist", trimmedPath),    // assets/file.js
			path.Join("dist/assets", path.Base(upath)), // Just the filename in assets dir
		}
		
		for _, filePath := range filePaths {
			if _, err := fs.Stat(DistFS, filePath); err == nil {
				fmt.Printf("DEBUG: Found asset at %s\n", filePath)
				// Set the corrected path
				r.URL.Path = "/" + strings.TrimPrefix(filePath, "dist/")
				break
			}
		}
	} else if _, err := fs.Stat(DistFS, path.Join("dist", upath)); err != nil {
		// For non-asset requests, handle SPA routing
		if !strings.HasPrefix(upath, "/api") && !strings.HasPrefix(upath, "/ws") {
			// For React SPA, serve index.html for all routes that don't match a file
			fmt.Printf("DEBUG: Redirecting to index.html for path %s\n", upath)
			r.URL.Path = "/index.html"
		}
	} else {
		fmt.Printf("DEBUG: Serving existing file at dist%s\n", upath)
	}

	// Serve the file or index.html for SPA routes
	http.StripPrefix("", http.FileServer(f.fs)).ServeHTTP(w, r)
}

// Create a sub-filesystem for the dist directory
var DistFS, _ = fs.Sub(ReactApp, "dist")

// createDirectServeHandler creates a direct file server that handles asset paths
func createDirectServeHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		
		fmt.Printf("DEBUG direct serve: %s (Referer: %s)\n", r.URL.Path, r.Header.Get("Referer"))
		
		// Clean the path
		upath := path.Clean(r.URL.Path)
		
		// Handle asset requests more explicitly
		if strings.HasPrefix(upath, "/assets/") {
			// Try different variations of the asset path
			assetName := path.Base(upath)
			assetPaths := []string{
				filepath.Join(dir, upath),                       // /assets/file.js -> dir/assets/file.js
				filepath.Join(dir, "assets", assetName),         // /assets/file.js -> dir/assets/file.js (without leading slash)
				filepath.Join(dir, strings.TrimPrefix(upath, "/")), // /assets/file.js -> dir/assets/file.js (as-is without leading slash)
			}
			
			// Try each path and log attempts
			fmt.Printf("DEBUG: Asset request %s - trying multiple paths\n", upath)
			for _, assetPath := range assetPaths {
				fmt.Printf("DEBUG: Checking for asset at %s\n", assetPath)
				if _, err := os.Stat(assetPath); err == nil {
					fmt.Printf("DEBUG: Found asset at %s\n", assetPath)
					http.ServeFile(w, r, assetPath)
					return
				} else {
					fmt.Printf("DEBUG: Asset not found at %s: %v\n", assetPath, err)
				}
			}
			
			// If we get here, no asset was found - try direct serve as fallback
			fmt.Printf("DEBUG: Falling back to standard file server for %s\n", upath)
		} else if strings.HasSuffix(upath, ".js") || 
		          strings.HasSuffix(upath, ".css") || 
		          strings.HasSuffix(upath, ".png") || 
		          strings.HasSuffix(upath, ".svg") {
			// For known asset types, try common variations
			assetPaths := []string{
				filepath.Join(dir, strings.TrimPrefix(upath, "/")),
				filepath.Join(dir, "assets", path.Base(upath)),
			}
			
			// Try each path and log attempts
			fmt.Printf("DEBUG: Static file request %s - trying multiple paths\n", upath)
			for _, assetPath := range assetPaths {
				fmt.Printf("DEBUG: Checking for static file at %s\n", assetPath)
				if _, err := os.Stat(assetPath); err == nil {
					fmt.Printf("DEBUG: Found static file at %s\n", assetPath)
					http.ServeFile(w, r, assetPath)
					return
				}
			}
		}
		
		// Handle SPA routes
		filePath := filepath.Join(dir, strings.TrimPrefix(upath, "/"))
		if _, err := os.Stat(filePath); err != nil && 
		   !strings.HasPrefix(upath, "/api") && 
		   !strings.HasPrefix(upath, "/ws") {
			// Check if this is a route or asset request
			if !strings.Contains(upath, ".") {
				// SPA route
				fmt.Printf("DEBUG: Serving index.html for SPA route: %s\n", upath)
				http.ServeFile(w, r, filepath.Join(dir, "index.html"))
				return
			} else {
				// Missing asset - log it clearly
				fmt.Printf("DEBUG: Asset not found: %s (file: %s)\n", upath, filePath)
			}
		} else {
			fmt.Printf("DEBUG: Serving existing file at %s\n", filePath)
		}
		
		// Use standard file server for all other paths
		http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
	})
}

// ReactFileServer returns a handler that serves HTTP requests with the contents of the embedded React app.
// It implements SPA behavior, returning index.html for all routes that don't match a static file.
func ReactFileServer() http.Handler {
	// Check various possible directories for the React build
	dirs := []string{
		"web/static/dist",   // Standard path after build
		"frontend/dist",     // Dev path within frontend
		"frontend/build",    // Alternate dev build path
		"static/dist",       // Relative path depending on working dir
	}
	
	// Try each directory and use first one that exists
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			fmt.Printf("DEBUG: Serving React app from file system: %s\n", dir)
			return createDirectServeHandler(dir)
		}
	}
	
	// If no directories found, use embedded files
	fmt.Printf("DEBUG: Serving React app from embedded files\n")
	return &spaFileServer{
		fs: http.FS(DistFS),
	}
}