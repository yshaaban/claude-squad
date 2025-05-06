//go:build windows

package web

import (
	"claude-squad/log"
	"os"
	"os/signal"
	"syscall"
)

// setupPlatformSignals sets up platform-specific signal handling.
func (s *Server) setupPlatformSignals() {
	signalChan := make(chan os.Signal, 1)
	// Windows only supports a limited set of signals
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		for sig := range signalChan {
			log.InfoLog.Printf("Received signal: %v", sig)
			
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown
				log.InfoLog.Printf("Shutting down web server due to signal: %v", sig)
				s.Stop()
			}
		}
	}()
}