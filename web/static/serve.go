package static

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
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

	// Use the redirect.html as the index page
	if upath == "/" || upath == "/index.html" {
		upath = "/" + f.indexAs
	}

	// Serve the file
	http.StripPrefix("", http.FileServer(f.fs)).ServeHTTP(w, r)
}

// FileServer returns a handler that serves HTTP requests with the contents of the embedded static files.
// It serves redirect.html as the index page.
func FileServer() http.Handler {
	content, err := fs.Sub(StaticFiles, ".")
	if err != nil {
		panic(err)
	}
	
	return &redirectingFileServer{
		fs:      http.FS(content),
		indexAs: "easy-terminal.html",
	}
}