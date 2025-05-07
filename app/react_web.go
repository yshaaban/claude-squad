package app

import (
	"claude-squad/log"
	"claude-squad/web"
	"fmt"
	"time"
)

// StartReactWebServer initializes and starts the web monitoring server with React frontend.
func (h *home) StartReactWebServer() error {
	// Skip if web server is not enabled
	if !h.appConfig.WebServerEnabled {
		return nil
	}

	// Create and start web server
	server := web.NewServer(h.storage, h.appConfig)

	// Configure to use React frontend
	server.UseReactServer()

	// Store server reference for cleanup
	h.webServer = server

	// Start the server
	if err := server.Start(); err != nil {
		return err
	}

	log.FileOnlyInfoLog.Printf("Web monitoring server with React UI started on http://%s:%d", 
		h.appConfig.WebServerHost, h.appConfig.WebServerPort)
		
	// Also log to standard error for visibility
	fmt.Printf("\nWeb monitoring server with React UI started: http://%s:%d\n", 
		h.appConfig.WebServerHost, h.appConfig.WebServerPort)
		
	// Update menu with web server info
	h.menu.SetWebServerInfo(true, h.appConfig.WebServerHost, h.appConfig.WebServerPort)
	
	// Create a standard session for web server if no instances exist
	log.FileOnlyInfoLog.Printf("DEBUG: app/react_web.go: NumInstances() returned %d instances", h.list.NumInstances())
	
	// Log instance titles for debugging
	instances := h.list.GetInstances()
	for i, instance := range instances {
		log.FileOnlyInfoLog.Printf("DEBUG: app/react_web.go: Instance %d: Title=%s Started=%v Paused=%v Status=%v", 
			i, instance.Title, instance.Started(), instance.Paused(), instance.Status)
	}
	
	if h.list.NumInstances() == 0 {
		log.FileOnlyInfoLog.Printf("DEBUG: app/react_web.go: No instances found, creating one")
		go func() {
			// Small delay to ensure server is fully started
			time.Sleep(500 * time.Millisecond)
			
			// Create a new instance for web mode (similar to pressing 'n' in the app)
			err := h.createWebInstance()
			if err != nil {
				log.FileOnlyErrorLog.Printf("Failed to create web instance: %v", err)
			} else {
				log.FileOnlyInfoLog.Printf("DEBUG: app/react_web.go: Successfully created web instance")
				
				// Force save the newly created instance to ensure it's available to web server
				if err := h.storage.SaveInstances(h.list.GetInstances()); err != nil {
					log.FileOnlyErrorLog.Printf("Failed to save new instance: %v", err)
				}
			}
		}()
	} else {
		// Add any existing instances to the monitor
		log.FileOnlyInfoLog.Printf("React web server started - %d existing instances will be monitored", h.list.NumInstances())
		
		// Save instances to storage to ensure they're available to the web server
		if err := h.storage.SaveInstances(h.list.GetInstances()); err != nil {
			log.FileOnlyErrorLog.Printf("Failed to save instances: %v", err)
		} else {
			log.FileOnlyInfoLog.Printf("DEBUG: app/react_web.go: Successfully saved %d instances to storage", len(instances))
		}
	}
	
	return nil
}