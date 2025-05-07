package main

import (
	"claude-squad/config"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	// Parse command line flags
	webPort := flag.Int("port", 8086, "Web server port")
	logToFile := flag.Bool("log-to-file", false, "Enable logging to file")
	flag.Parse()
	
	// Initialize logging
	if *logToFile {
		tempLog := filepath.Join(os.TempDir(), "claudesquad.log")
		log.SetupLogging(tempLog)
		fmt.Printf("Logging to: %s\n", tempLog)
	}
	
	// Create config
	cfg := &config.Config{
		WebServerEnabled:       true,
		WebServerPort:          *webPort,
		WebServerHost:          "",
		WebServerAllowLocalhost: true,
		WebServerAuthToken:     "test_token",
		WebServerUseTLS:        false,
	}
	
	// Create config storage - implements config.StateManager interface
	configStorage := &config.MemoryStorage{}
	
	// Create storage for instances
	storage, err := session.NewStorage(configStorage)
	if err != nil {
		fmt.Printf("Error creating storage: %v\n", err)
		os.Exit(1)
	}
	
	// Create web server
	server := web.NewServer(storage, cfg)
	
	// Configure to use React server
	server.UseReactServer()
	
	// Start the server
	if err := server.Start(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("\nReact test server running at http://localhost:%d\n", *webPort)
	fmt.Printf("Press Ctrl+C to stop\n\n")
	fmt.Printf("Access these endpoints:\n")
	fmt.Printf("- Main React app: http://localhost:%d/\n", *webPort)
	fmt.Printf("- Test page: http://localhost:%d/test.html\n", *webPort)
	fmt.Printf("- Asset test page: http://localhost:%d/asset-test.html\n", *webPort)
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	// Shut down gracefully
	fmt.Println("\nShutting down server...")
	server.Stop()
	time.Sleep(500 * time.Millisecond) // Give a moment for logs to flush
	fmt.Println("Server stopped")
}