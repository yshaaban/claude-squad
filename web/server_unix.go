//go:build !windows

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
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	
	go func() {
		for sig := range signalChan {
			log.InfoLog.Printf("Received signal: %v", sig)
			
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown
				log.InfoLog.Printf("Shutting down web server due to signal: %v", sig)
				s.Stop()
			case syscall.SIGHUP:
				// Reload configuration (not implemented yet)
				log.InfoLog.Printf("Reload configuration (not implemented)")
			}
		}
	}()
}