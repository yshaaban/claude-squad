package handlers

import (
	"claude-squad/log"
	"claude-squad/session"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// Message types for websocket communication
const (
	OutputMessage  = 'o' // Output from terminal to client
	InputMessage   = 'i' // Input from client to terminal
	ResizeMessage  = 'r' // Resize terminal window
	PingMessage    = 'p' // Ping from client
	PongMessage    = 'P' // Pong response
	CloseMessage   = 'c' // Close connection
)

// ResizeData represents a terminal resize request
type ResizeData struct {
	Columns int `json:"cols"`
	Rows    int `json:"rows"`
}

// TerminalHandler handles websocket connections for terminals
type TerminalHandler struct {
	instances        *session.Storage
	upgrader         websocket.Upgrader
	activeInstances  map[string]*activeInstance
	mutex            sync.Mutex
}

// activeInstance tracks an active websocket connection to an instance
type activeInstance struct {
	instance    *session.Instance
	attachChan  chan struct{}
	connections int
	lastActive  time.Time
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(instances *session.Storage) *TerminalHandler {
	return &TerminalHandler{
		instances: instances,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		activeInstances: make(map[string]*activeInstance),
	}
}

// HandleWebSocket handles a websocket connection for terminal access
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get instance name from URL parameter
	instanceName := chi.URLParam(r, "name")
	
	// For backward compatibility with the existing WebSocket code
	if instanceName == "" {
		instanceName = r.URL.Query().Get("instance")
	}
	
	if instanceName == "" {
		http.Error(w, "Instance name is required", http.StatusBadRequest)
		return
	}

	// Load all instances
	instances, err := h.instances.LoadInstances()
	if err != nil {
		log.ErrorLog.Printf("Failed to load instances: %v", err)
		http.Error(w, "Failed to load instances", http.StatusInternalServerError)
		return
	}

	// Find the requested instance
	var targetInstance *session.Instance
	for _, instance := range instances {
		if instance.Title == instanceName {
			targetInstance = instance
			break
		}
	}

	if targetInstance == nil {
		http.Error(w, fmt.Sprintf("Instance '%s' not found", instanceName), http.StatusNotFound)
		return
	}

	// Check if instance is running
	if !targetInstance.Started() || targetInstance.Paused() {
		http.Error(w, "Instance is not running", http.StatusBadRequest)
		return
	}

	// Upgrade connection to websocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.ErrorLog.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	log.FileOnlyInfoLog.Printf("New websocket connection for instance: %s", instanceName)

	// Get or create active instance tracking
	activeInst, err := h.getOrCreateActiveInstance(instanceName, targetInstance)
	if err != nil {
		log.ErrorLog.Printf("Failed to activate instance: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to connect to terminal: "+err.Error()))
		return
	}
	defer h.releaseActiveInstance(instanceName)

	// Handle the connection
	h.handleConnection(conn, targetInstance, activeInst.attachChan)
}

// getOrCreateActiveInstance gets or creates an active instance tracking
func (h *TerminalHandler) getOrCreateActiveInstance(name string, instance *session.Instance) (*activeInstance, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Check if we already have this instance active
	active, exists := h.activeInstances[name]
	if exists {
		active.connections++
		active.lastActive = time.Now()
		return active, nil
	}

	// Create a new active instance
	attachChan, err := instance.Attach()
	if err != nil {
		return nil, fmt.Errorf("failed to attach to instance: %w", err)
	}

	active = &activeInstance{
		instance:    instance,
		attachChan:  attachChan,
		connections: 1,
		lastActive:  time.Now(),
	}

	h.activeInstances[name] = active
	return active, nil
}

// releaseActiveInstance decrements the connection count for an instance
func (h *TerminalHandler) releaseActiveInstance(name string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	active, exists := h.activeInstances[name]
	if !exists {
		return
	}

	active.connections--
	active.lastActive = time.Now()

	// If no more connections, detach after a delay to allow for reconnection
	if active.connections <= 0 {
		go func() {
			// Wait a bit before detaching in case of reconnections
			time.Sleep(5 * time.Second)

			h.mutex.Lock()
			defer h.mutex.Unlock()

			// Check if still no connections and it's been inactive for at least 5 seconds
			active, exists := h.activeInstances[name]
			if exists && active.connections <= 0 && 
			   time.Since(active.lastActive) >= 5*time.Second {
				   
				// Verify instance is valid before detaching
				if active.instance != nil && active.instance.Started() {
					log.FileOnlyInfoLog.Printf("Detaching from instance after inactivity: %s", name)
					active.instance.Detach()
				}
				
				// Remove from active instances
				delete(h.activeInstances, name)
				log.FileOnlyInfoLog.Printf("Removed inactive instance from tracking: %s", name)
			}
		}()
	}
}

// handleConnection manages a websocket connection to a terminal
func (h *TerminalHandler) handleConnection(conn *websocket.Conn, instance *session.Instance, doneCh chan struct{}) {
	// Check if doneCh is nil to avoid a panic
	if doneCh == nil {
		log.FileOnlyErrorLog.Printf("nil done channel provided to handleConnection, creating a replacement")
		// Create a dummy channel that will never close unless the function ends
		doneCh = make(chan struct{})
		defer close(doneCh)
	}

	// Set up reader for websocket messages
	go func() {
		defer conn.Close()
		
		for {
			// Read message from websocket
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.FileOnlyErrorLog.Printf("Websocket error: %v", err)
				}
				return
			}

			// Process message based on type
			if len(message) > 0 {
				switch message[0] {
				case InputMessage:
					// Send input to instance
					if len(message) > 1 {
						if err := instance.SendPrompt(string(message[1:])); err != nil {
							log.FileOnlyErrorLog.Printf("Error sending input to instance: %v", err)
						}
					}
				
				case ResizeMessage:
					// Resize terminal
					if len(message) > 1 {
						var resize ResizeData
						if err := json.Unmarshal(message[1:], &resize); err != nil {
							log.FileOnlyErrorLog.Printf("Error parsing resize message: %v", err)
							continue
						}
						
						if err := instance.SetPreviewSize(resize.Columns, resize.Rows); err != nil {
							log.FileOnlyErrorLog.Printf("Error resizing terminal: %v", err)
						}
					}
				
				case PingMessage:
					// Send pong response - with more forgiving error handling
					pongMsg := []byte{PongMessage}
					if err := conn.WriteMessage(websocket.BinaryMessage, pongMsg); err != nil {
						log.FileOnlyErrorLog.Printf("Error sending pong: %v", err)
						// Only return/disconnect if it's a critical error
						if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) ||
						   strings.Contains(err.Error(), "close sent") ||
						   strings.Contains(err.Error(), "broken pipe") {
							log.FileOnlyErrorLog.Printf("Fatal WebSocket error while sending pong, closing connection: %v", err)
							return
						}
					} else {
						log.FileOnlyInfoLog.Printf("Pong sent successfully")
					}
				
				case CloseMessage:
					// Client requested close
					return
				}
			}
		}
	}()

	// Set up periodic content updates
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastContent string
	for {
		select {
		case <-ticker.C:
			// Get current content
			content, err := instance.Preview()
			if err != nil {
				log.FileOnlyErrorLog.Printf("Error getting preview: %v", err)
				continue
			}

			// Only send if content has changed and is not empty
			if content != lastContent && content != "" {
				lastContent = content
				
				// Add detailed debug logging
				log.FileOnlyInfoLog.Printf("Sending terminal content (length: %d) to websocket for instance %s", 
					len(content), instance.Title)
				
				// Create the binary message with message type prefix
				message := append([]byte{OutputMessage}, []byte(content)...)
				
				// Send content update with binary protocol
				if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
					log.FileOnlyErrorLog.Printf("Error sending content update: %v", err)
					// Check if this is a fatal error that requires termination
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) || 
					   strings.Contains(err.Error(), "close sent") ||
					   strings.Contains(err.Error(), "broken pipe") {
						log.FileOnlyErrorLog.Printf("Fatal websocket error, closing connection: %v", err)
						return
					}
					// Non-fatal error, just log and continue
					log.FileOnlyErrorLog.Printf("Non-fatal error sending content, will retry on next tick: %v", err)
				} else {
					log.FileOnlyInfoLog.Printf("Successfully sent terminal content to websocket for instance %s", instance.Title)
				}
			}

		case <-doneCh:
			// Instance was detached
			log.FileOnlyInfoLog.Printf("Instance detached, closing websocket")
			return
		}
	}
}