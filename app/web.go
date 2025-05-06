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
	if h.list.NumInstances() == 0 {
		go func() {
			// Small delay to ensure server is fully started
			time.Sleep(500 * time.Millisecond)
			
			// Create a new instance for web mode (similar to pressing 'n' in the app)
			err := h.createWebInstance()
			if err != nil {
				log.FileOnlyErrorLog.Printf("Failed to create web instance: %v", err)
			}
		}()
	} else {
		// Add any existing instances to the monitor
		log.FileOnlyInfoLog.Printf("Web server started - %d existing instances will be monitored", h.list.NumInstances())
	}
	
	return nil
}

// createWebInstance creates a standard instance for web mode, similar to pressing 'n' in the app
func (h *home) createWebInstance() error {
	// Create instance name based on timestamp
	instanceName := fmt.Sprintf("web-%s", time.Now().Format("20060102-150405"))
	
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Create a new instance
	instance, err := session.NewInstance(session.InstanceOptions{
		Title:   instanceName,
		Path:    currentDir,
		Program: h.program,
		AutoYes: true, // Auto-confirm any prompts
		InPlace: true,  // Run in current directory
	})
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}
	
	// Start the instance
	if err := instance.Start(true); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	
	// Add to list and select it
	h.list.AddInstance(instance)()
	
	// Save instances to storage
	if err := h.storage.SaveInstances(h.list.GetInstances()); err != nil {
		log.FileOnlyWarningLog.Printf("Failed to save instances: %v", err)
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