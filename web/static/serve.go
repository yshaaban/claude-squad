package static

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed *.html *.js *.css
var StaticFiles embed.FS

// redirectingFileServer is a custom file server that serves redirect.html as the index
type redirectingFileServer struct {
	fs      http.FileSystem
	indexAs string
}

func (f *redirectingFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean path to prevent directory traversal
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	upath = path.Clean(upath)

	// Redirect to easy-terminal.html for index page
	if upath == "/" || upath == "/index.html" {
		upath = "/easy-terminal.html"
	}

	// Serve the file
	http.StripPrefix("", http.FileServer(f.fs)).ServeHTTP(w, r)
}

// reactFileSystemServer is a special file server for serving React assets
type reactFileSystemServer struct {
	root http.Dir
}

func (f *reactFileSystemServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add trailing slash to make relative paths work correctly
	path := strings.TrimPrefix(r.URL.Path, "/")
	
	// If the path is empty or requests index page, serve index.html
	if path == "" || path == "index.html" {
		path = "index.html"
	}
	
	// Check if file exists
	filePath := filepath.Join(string(f.root), path)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// For SPA routes, serve index.html
		if !strings.Contains(path, ".") && !strings.HasPrefix(path, "api/") && !strings.HasPrefix(path, "ws/") {
			http.ServeFile(w, r, filepath.Join(string(f.root), "index.html"))
			return
		}
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	// Serve the file
	http.ServeFile(w, r, filePath)
}

// FileServer returns a handler that serves HTTP requests with the contents of the embedded static files.
func FileServer() http.Handler {
	// Static file handler that prioritizes the React app
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Setup basic security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		
		// Check if React app is available
		reactDirExists := false
		if _, err := os.Stat("web/static/dist"); err == nil {
			reactDirExists = true
		}

		// If React is available, use the special file server
		if reactDirExists {
			fs := &reactFileSystemServer{root: http.Dir("web/static/dist")}
			fs.ServeHTTP(w, r)
			return
		}
		
		// If React isn't available, fall back to embedded files
		content, err := fs.Sub(StaticFiles, ".")
		if err != nil {
			http.Error(w, "Static files unavailable", http.StatusInternalServerError)
			return
		}
		
		// For root or index when React is not available, redirect to easy-terminal.html
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.Redirect(w, r, "/easy-terminal.html", http.StatusFound)
			return
		}
		
		// Use the standard file server for embedded assets
		fileServer := http.FileServer(http.FS(content))
		fileServer.ServeHTTP(w, r)
	})
}