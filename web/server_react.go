package web

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"claude-squad/log"
	"claude-squad/web/handlers"
	webmiddleware "claude-squad/web/middleware"
	"claude-squad/web/static"
)

// setupReactServer configures the router to serve the React SPA
func (s *Server) setupReactServer() {
	// Create router with middleware
	router := chi.NewRouter()
	
	// Add core middleware - skip Logger to prevent terminal UI corruption
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.StripSlashes)
	
	// Authentication Middleware - disabled for local connections
	// For development and local usage, skip authentication entirely
	log.FileOnlyInfoLog.Printf("Authentication disabled for all connections in React mode")
	
	// Add rate limiting - exempt WebSocket connections from rate limiting
	// Increase to 500/minute to handle SPA route changes and asset requests
	router.Use(webmiddleware.RateLimitMiddleware(500, time.Minute, true)) // 500 requests per minute, WebSockets exempt
	
	// Set up CORS - allow all origins for testing
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins for testing
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	
	// API routes
	router.Route("/api", func(r chi.Router) {
		r.Get("/instances", s.handleInstances)
		r.Route("/instances/{name}", func(r chi.Router) {
			r.Get("/", s.handleInstanceDetail)
			r.Get("/output", s.handleInstanceOutput)
			r.Get("/diff", s.handleInstanceDiff)
		})
		r.Get("/status", s.handleServerStatus)
	})
	
	// WebSocket route for terminal streaming
	webSocketHandler := handlers.WebSocketHandler(s.storage, s.terminalMonitor)
	
	// Primary route pattern for new clients
	router.Get("/ws/{name}", webSocketHandler)
	
	// Backward compatibility route for existing clients that use /ws/terminal/{name}
	router.Get("/ws/terminal/{name}", webSocketHandler)
	
	// Compatibility route for clients that use query params: /ws?instance=...
	router.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		if instanceName := r.URL.Query().Get("instance"); instanceName != "" {
			// Create chi context with URL params to pass to the handler
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("name", instanceName)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
			webSocketHandler(w, r)
			return
		}
		
		// If no instance name provided, return an error
		log.FileOnlyWarningLog.Printf("WebSocket: /ws called without instance parameter from %s", r.RemoteAddr)
		http.Error(w, "Instance name required via /ws/{name}, /ws/terminal/{name}, or /ws?instance=name", http.StatusBadRequest)
	})

	// For backward compatibility, maintain these explicitly defined routes
	router.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		static.ReactFileServer().ServeHTTP(w, r)
	}))
	
	router.Get("/index.html", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		static.ReactFileServer().ServeHTTP(w, r)
	}))
	
	// Serve static files and SPA routes
	router.Handle("/*", static.ReactFileServer())
	
	s.router = router
}

// UseReactServer configures the server to use the React SPA
func (s *Server) UseReactServer() {
	// Set up the React server
	s.setupReactServer()
	
	// Update HTTP server handler
	s.srv.Handler = s.router
}