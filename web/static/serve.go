package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed *.html
var StaticFiles embed.FS

// FileServer returns a handler that serves HTTP requests with the contents of the embedded static files.
func FileServer() http.Handler {
	content, err := fs.Sub(StaticFiles, ".")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(content))
}