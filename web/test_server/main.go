package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Create a simple file server for the dist directory
	fs := http.FileServer(http.Dir("../static/dist"))
	
	http.Handle("/", fs)
	
	// Start the server
	port := 8099
	fmt.Printf("Starting test server on port %d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
		os.Exit(1)
	}
}