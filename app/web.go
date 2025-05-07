package app

import (
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web"
	"fmt"
	"os"
	"time"
)

// StartOptions contains options for starting Claude Squad.
type StartOptions struct {
	Program          string
	AutoYes          bool
	SimpleMode       bool
	WebServerEnabled bool
	WebServerPort    int
}

// StartWebServer initializes and starts the web monitoring server.
func (h *home) StartWebServer() error {
	// Skip if web server is not enabled
	if !h.appConfig.WebServerEnabled {
		return nil
	}

	// Create and start web server
	server := web.NewServer(h.storage, h.appConfig)

	// Store server reference for cleanup
	h.webServer = server

	// Start the server
	if err := server.Start(); err != nil {
		return err
	}

	log.FileOnlyInfoLog.Printf("Web monitoring server started on http://%s:%d", 
		h.appConfig.WebServerHost, h.appConfig.WebServerPort)
		
	// Also log to standard error for visibility
	fmt.Printf("\nWeb monitoring server started: http://%s:%d\n", 
		h.appConfig.WebServerHost, h.appConfig.WebServerPort)
		
	// Update menu with web server info
	h.menu.SetWebServerInfo(true, h.appConfig.WebServerHost, h.appConfig.WebServerPort)
	
	// Create a standard session for web server if no instances exist
	log.FileOnlyInfoLog.Printf("DEBUG: app/web.go: NumInstances() returned %d instances", h.list.NumInstances())
	
	// Log instance titles for debugging
	instances := h.list.GetInstances()
	for i, instance := range instances {
		log.FileOnlyInfoLog.Printf("DEBUG: app/web.go: Instance %d: Title=%s Started=%v Paused=%v Status=%v", 
			i, instance.Title, instance.Started(), instance.Paused(), instance.Status)
	}
	
	if h.list.NumInstances() == 0 {
		log.FileOnlyInfoLog.Printf("DEBUG: app/web.go: No instances found, creating one")
		go func() {
			// Small delay to ensure server is fully started
			time.Sleep(500 * time.Millisecond)
			
			// Create a new instance for web mode (similar to pressing 'n' in the app)
			err := h.createWebInstance()
			if err != nil {
				log.FileOnlyErrorLog.Printf("Failed to create web instance: %v", err)
			} else {
				log.FileOnlyInfoLog.Printf("DEBUG: app/web.go: Successfully created web instance")
				
				// Force save the newly created instance to ensure it's available to web server
				if err := h.storage.SaveInstances(h.list.GetInstances()); err != nil {
					log.FileOnlyErrorLog.Printf("Failed to save new instance: %v", err)
				}
			}
		}()
	} else {
		// Add any existing instances to the monitor
		log.FileOnlyInfoLog.Printf("Web server started - %d existing instances will be monitored", h.list.NumInstances())
		
		// Save instances to storage to ensure they're available to the web server
		if err := h.storage.SaveInstances(h.list.GetInstances()); err != nil {
			log.FileOnlyErrorLog.Printf("Failed to save instances: %v", err)
		} else {
			log.FileOnlyInfoLog.Printf("DEBUG: app/web.go: Successfully saved %d instances to storage", len(instances))
		}
	}
	
	return nil
}

// createWebInstance creates a standard instance for web mode, similar to pressing 'n' in the app
func (h *home) createWebInstance() error {
	// Create instance name based on timestamp
	instanceName := fmt.Sprintf("web-%s", time.Now().Format("20060102-150405"))
	log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Creating instance %s", instanceName)
	
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.FileOnlyErrorLog.Printf("DEBUG: createWebInstance: Failed to get current directory: %v", err)
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Using directory %s", currentDir)
	
	// Create a new instance
	instance, err := session.NewInstance(session.InstanceOptions{
		Title:   instanceName,
		Path:    currentDir,
		Program: h.program,
		AutoYes: true, // Auto-confirm any prompts
		InPlace: true,  // Run in current directory
	})
	if err != nil {
		log.FileOnlyErrorLog.Printf("DEBUG: createWebInstance: Failed to create instance: %v", err)
		return fmt.Errorf("failed to create instance: %w", err)
	}
	log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Instance created successfully")
	
	// Start the instance
	log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Starting instance %s", instanceName)
	if err := instance.Start(true); err != nil {
		log.FileOnlyErrorLog.Printf("DEBUG: createWebInstance: Failed to start instance: %v", err)
		return fmt.Errorf("failed to start instance: %w", err)
	}
	log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Instance started successfully")
	
	// Add to list and select it
	h.list.AddInstance(instance)()
	log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Instance added to list, new count: %d", h.list.NumInstances())
	
	// Save instances to storage
	log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Saving instances to storage")
	if err := h.storage.SaveInstances(h.list.GetInstances()); err != nil {
		log.FileOnlyWarningLog.Printf("DEBUG: createWebInstance: Failed to save instances: %v", err)
	} else {
		log.FileOnlyInfoLog.Printf("DEBUG: createWebInstance: Instances saved successfully")
	}
	
	log.FileOnlyInfoLog.Printf("Created new web instance: %s", instanceName)
	return nil
}

// StopWebServer gracefully stops the web server.
func (h *home) StopWebServer() {
	if h.webServer != nil {
		log.FileOnlyInfoLog.Printf("Shutting down web server...")
		if err := h.webServer.Stop(); err != nil {
			log.FileOnlyErrorLog.Printf("Error stopping web server: %v", err)
		}
		
		// Clear web server info from menu
		h.menu.SetWebServerInfo(false, "", 0)
	}
}