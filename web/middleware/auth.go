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
func RateLimitMiddleware(requests int, duration time.Duration) func(http.Handler) http.Handler {
	type client struct {
		count     int
		lastReset time.Time
	}
	
	clients := make(map[string]*client)
	var mu sync.Mutex
	
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
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			
			mu.Lock()
			
			// Get or create client record
			c, exists := clients[ip]
			if !exists {
				c = &client{0, time.Now()}
				clients[ip] = c
			}
			
			// Reset count if time window expired
			if time.Since(c.lastReset) > duration {
				c.count = 0
				c.lastReset = time.Now()
			}
			
			// Check if rate exceeded
			if c.count >= requests {
				mu.Unlock()
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(duration.Seconds())))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				log.WarningLog.Printf("Rate limit exceeded for %s", ip)
				return
			}
			
			// Increment count and continue
			c.count++
			mu.Unlock()
			
			next.ServeHTTP(w, r)
		})
	}
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