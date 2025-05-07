package middleware

import (
	"claude-squad/config"
	"claude-squad/log"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AuthMiddleware creates middleware for API authentication.
func AuthMiddleware(config *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for localhost when configured
			if config.WebServerAllowLocalhost {
				host, _, err := net.SplitHostPort(r.RemoteAddr)
				if err == nil && (host == "127.0.0.1" || host == "::1" || host == "localhost") {
					next.ServeHTTP(w, r)
					return
				}
			}
			
			// Get auth token from header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization required", http.StatusUnauthorized)
				log.WarningLog.Printf("Auth attempt with no token from %s", r.RemoteAddr)
				return
			}
			
			// Validate token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				log.WarningLog.Printf("Auth attempt with invalid format from %s", r.RemoteAddr)
				return
			}
			
			token := parts[1]
			
			// Validate token
			if token != config.WebServerAuthToken {
				http.Error(w, "Invalid authorization token", http.StatusUnauthorized)
				log.WarningLog.Printf("Auth attempt with invalid token from %s", r.RemoteAddr)
				return
			}
			
			// Token valid, continue
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware creates middleware for rate limiting.
func RateLimitMiddleware(requests int, duration time.Duration, exemptWebSockets ...bool) func(http.Handler) http.Handler {
	// Different rate limits for different endpoints
	const (
		ApiRequestsLimit = 1000 // Higher limit for API requests
	)
	
	type client struct {
		count      int       // Regular endpoint count
		apiCount   int       // API endpoint count
		lastReset  time.Time // Last reset time
	}
	
	clients := make(map[string]*client)
	var mu sync.Mutex
	
	// Check if WebSockets should be exempt from rate limiting
	exemptWS := false
	if len(exemptWebSockets) > 0 && exemptWebSockets[0] {
		exemptWS = true
	}
	
	// Start cleanup goroutine to prevent memory leaks
	go func() {
		for range time.Tick(duration) {
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastReset) > duration*2 {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Don't rate limit WebSocket connections if exemption is enabled
			if exemptWS && isWebSocketRequest(r) {
				next.ServeHTTP(w, r)
				return
			}
			
			// Check if it's an API request (has higher limits)
			isApi := isApiRequest(r)
			
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			
			mu.Lock()
			
			// Get or create client record
			c, exists := clients[ip]
			if !exists {
				c = &client{0, 0, time.Now()}
				clients[ip] = c
			}
			
			// Reset count if time window expired
			if time.Since(c.lastReset) > duration {
				c.count = 0
				c.apiCount = 0
				c.lastReset = time.Now()
			}
			
			// Determine which rate limit to use
			limitExceeded := false
			if isApi {
				// Use API rate limit
				if c.apiCount >= ApiRequestsLimit {
					limitExceeded = true
				} else {
					c.apiCount++
				}
			} else {
				// Use regular rate limit
				if c.count >= requests {
					limitExceeded = true
				} else {
					c.count++
				}
			}
			
			// Check if rate exceeded
			if limitExceeded {
				mu.Unlock()
				// Set retry-after header (in seconds)
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(duration.Seconds())))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				log.WarningLog.Printf("Rate limit exceeded for %s (API: %v)", ip, isApi)
				return
			}
			
			mu.Unlock()
			
			next.ServeHTTP(w, r)
		})
	}
}

// isWebSocketRequest checks if the request is a WebSocket upgrade request
func isWebSocketRequest(r *http.Request) bool {
	// Check both standard WebSocket upgrade headers
	isWebSocket := strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
		
	// Also check for WebSocket paths - these should also be exempt from rate limiting
	isWebSocketPath := strings.HasPrefix(r.URL.Path, "/ws") || 
		strings.Contains(r.URL.Path, "/terminal/") ||
		r.URL.Query().Get("instance") != ""
		
	return isWebSocket || isWebSocketPath
}

// isApiRequest checks if the request is for an API endpoint
func isApiRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/")
}

// CORSMiddleware creates middleware for handling CORS.
func CORSMiddleware(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}