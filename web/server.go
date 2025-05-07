// Package web provides a web server for monitoring Claude Squad instances.
package web

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"claude-squad/config"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web/handlers"
	webmiddleware "claude-squad/web/middleware" // Our custom middleware
	"claude-squad/web/static" // Static file handler
)

// Server manages the HTTP server for monitoring Claude Squad.
type Server struct {
	storage         *session.Storage
	config          *config.Config
	router          chi.Router
	srv             *http.Server
	terminalMonitor *TerminalMonitor
	done            chan struct{}
	startTime       time.Time
}

// Handler returns the http.Handler for testing.
func (s *Server) Handler() http.Handler {
	return s.router
}

// NewServer creates a new monitoring server.
func NewServer(storage *session.Storage, config *config.Config) *Server {
	// Initialize special empty lists
	storage.PreloadSimpleMode()

	server := &Server{
		storage:   storage,
		config:    config,
		done:      make(chan struct{}),
		startTime: time.Now(),
	}

	// Create terminal monitor
	server.terminalMonitor = NewTerminalMonitor(storage)

	// Create router with middleware
	router := chi.NewRouter()
	
	// Add core middleware - skip Logger to prevent terminal UI corruption
	router.Use(chimiddleware.RealIP)
	// Logger middleware disabled to prevent terminal UI corruption - use file logging instead
	// router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.StripSlashes)
	
	// Authentication Middleware
	// Forcing auth to be disabled for all connections by using 'true ||'
	// This is a development convenience that bypasses auth completely
	if true || config.WebServerAllowLocalhost {
		log.FileOnlyInfoLog.Printf("Authentication completely disabled for all connections")
	} else {
		// This branch is currently unreachable due to the 'true ||' above
		// It's kept for future use if auth is needed
		router.Use(webmiddleware.AuthMiddleware(config))
	}
	
	// Add rate limiting
	router.Use(webmiddleware.RateLimitMiddleware(100, time.Minute)) // 100 requests per minute
	
	// Set up CORS - allow all origins for testing
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins for testing
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	
	// Set up minimal logging for server - only log important events to avoid UI corruption
	// Info logs about every request would be too noisy and risk terminal UI issues
	
	// API routes
	router.Route("/api", func(r chi.Router) {
		r.Get("/instances", server.handleInstances)
		r.Route("/instances/{name}", func(r chi.Router) {
			r.Get("/", server.handleInstanceDetail)
			r.Get("/output", server.handleInstanceOutput)
			r.Get("/diff", server.handleInstanceDiff)
		})
		r.Get("/status", server.handleServerStatus)
	})
	
	// WebSocket route for terminal streaming.
	// Use the TerminalMonitor-based handler for all WebSocket connections
	webSocketHandler := handlers.WebSocketHandler(server.storage, server.terminalMonitor)
	
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

	// Explicit redirect from the root to easy-terminal.html
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/easy-terminal.html", http.StatusFound)
	})
	
	// Explicit redirect from /index.html to easy-terminal.html
	router.Get("/index.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/easy-terminal.html", http.StatusFound)
	})
	
	// Static files for web UI
	router.Handle("/*", static.FileServer())
	
	server.router = router
	
	// Configure HTTP server with timeouts
	server.srv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.WebServerHost, config.WebServerPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	// Add TLS if enabled
	if config.WebServerUseTLS {
		server.srv.TLSConfig = configureTLS(config)
	}
	
	return server
}

// Start begins the web server and background polling.
func (s *Server) Start() error {
	// Initialize detailed debug logging
	InitDebugLog()
	LogWebDebug("==== STARTING WEB SERVER ====")
	
	// Log instances at startup
	instances, err := s.storage.LoadInstances()
	if err != nil {
		LogWebDebug("ERROR loading instances: %v", err)
	} else {
		LogWebInstances("STARTUP_INSTANCES", instances)
	}
	
	// Start terminal monitor
	s.terminalMonitor.Start()
	
	// Set up platform-specific signal handling
	s.setupPlatformSignals()
	
	// Start HTTP server
	go func() {
		var err error
		if s.config.WebServerUseTLS {
			log.FileOnlyInfoLog.Printf("Starting HTTPS server on %s:%d",
				s.config.WebServerHost, s.config.WebServerPort)
			err = s.srv.ListenAndServeTLS("", "")  // Uses TLSConfig
		} else {
			log.FileOnlyInfoLog.Printf("Starting HTTP server on %s:%d",
				s.config.WebServerHost, s.config.WebServerPort)
			err = s.srv.ListenAndServe()
		}
		
		if err != nil && err != http.ErrServerClosed {
			log.ErrorLog.Printf("HTTP server error: %v", err)
		}
	}()
	
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	LogWebDebug("==== STOPPING WEB SERVER ====")
	close(s.done)
	
	// Log instance state before shutdown
	instances, err := s.storage.LoadInstances()
	if err != nil {
		LogWebDebug("ERROR loading instances during shutdown: %v", err)
	} else {
		LogWebInstances("SHUTDOWN_INSTANCES", instances)
	}
	
	// Stop terminal monitor
	LogWebDebug("Stopping terminal monitor")
	s.terminalMonitor.Stop()
	
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Gracefully shutdown HTTP server
	LogWebDebug("Shutting down HTTP server")
	err = s.srv.Shutdown(ctx)
	
	// Close debug logging
	CloseDebugLog()
	
	return err
}

// getInstanceByTitle retrieves an instance by title.
func (s *Server) getInstanceByTitle(title string) (*session.Instance, error) {
	instances, err := s.storage.LoadInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to load instances: %w", err)
	}
	
	for _, instance := range instances {
		if instance.Title == title {
			return instance, nil
		}
	}
	
	return nil, fmt.Errorf("instance not found: %s", title)
}

// configureTLS creates the TLS configuration for the server.
func configureTLS(config *config.Config) *tls.Config {
	// Check for custom certificates
	var cert tls.Certificate
	var err error
	
	if config.WebServerTLSCert != "" && config.WebServerTLSKey != "" {
		// Use provided certificates
		cert, err = tls.LoadX509KeyPair(config.WebServerTLSCert, config.WebServerTLSKey)
		if err != nil {
			log.ErrorLog.Printf("Error loading TLS certificates: %v", err)
			// Fall back to self-signed
		}
	}
	
	// Generate self-signed if needed
	if cert.Certificate == nil {
		cert, err = generateSelfSignedCert()
		if err != nil {
			log.ErrorLog.Printf("Error generating self-signed cert: %v", err)
		}
	}
	
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
}

// Generate self-signed certificate.
func generateSelfSignedCert() (tls.Certificate, error) {
	// Generate private key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	
	// Set up certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1 year
	
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Claude Squad Self-Signed"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
	}
	
	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	
	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	
	// Load certificate
	return tls.X509KeyPair(certPEM, keyPEM)
}

// Handler methods - these delegate to the appropriate implementation
func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	handlers.InstancesHandler(s.storage)(w, r)
}

func (s *Server) handleInstanceDetail(w http.ResponseWriter, r *http.Request) {
	handlers.InstanceDetailHandler(s.storage)(w, r)
}

func (s *Server) handleInstanceOutput(w http.ResponseWriter, r *http.Request) {
	handlers.InstanceOutputHandler(s.storage)(w, r)
}

func (s *Server) handleInstanceDiff(w http.ResponseWriter, r *http.Request) {
	handlers.DiffHandler(s.storage)(w, r)
}

func (s *Server) handleServerStatus(w http.ResponseWriter, r *http.Request) {
	version := "1.0.0" // TODO: Get from app
	handlers.ServerStatusHandler(version, s.startTime)(w, r)
}

func (s *Server) handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	// This function is effectively replaced by the logic in router.Get("/ws", ...)
}