package app

import (
	"claude-squad/log"
	"claude-squad/web"
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

	log.InfoLog.Printf("Web monitoring server started on %s:%d", 
		h.appConfig.WebServerHost, h.appConfig.WebServerPort)
	return nil
}

// StopWebServer gracefully stops the web server.
func (h *home) StopWebServer() {
	if h.webServer != nil {
		log.InfoLog.Printf("Shutting down web server...")
		if err := h.webServer.Stop(); err != nil {
			log.ErrorLog.Printf("Error stopping web server: %v", err)
		}
	}
}