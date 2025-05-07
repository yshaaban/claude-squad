package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := flag.Int("port", 8085, "Port to listen on")
	flag.Parse()

	// Check if the React app is available
	reactAppPath := "web/static/dist/index.html"
	if _, err := os.Stat(reactAppPath); err == nil {
		log.Printf("Found React app at %s", reactAppPath)
	} else {
		log.Printf("React app not found at %s: %v", reactAppPath, err)
	}

	// Serve static files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, reactAppPath)
			return
		}
		
		// Otherwise, try to serve from the static directory
		filePath := "web/static/dist" + r.URL.Path
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
		} else {
			http.NotFound(w, r)
		}
	})

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting test server on %s", addr)
	log.Printf("Open http://localhost:%d in your browser", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}