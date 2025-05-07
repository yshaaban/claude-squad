package handlers

import (
	"claude-squad/log"
	"claude-squad/session"
	"crypto/sha256"
	"encoding/hex"
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
	ClearMessage   = 'C' // Clear terminal
)

// ResizeData represents a terminal resize request
type ResizeData struct {
	Columns int `json:"cols"`
	Rows    int `json:"rows"`
}

// ContentHash stores checksums of sent content to avoid duplicates
type ContentHash struct {
	hash      string
	timestamp time.Time
}

// TerminalHandler handles websocket connections for terminals
type TerminalHandler struct {
	instances        *session.Storage
	upgrader         websocket.Upgrader
	activeInstances  map[string]*activeInstance
	mutex            sync.Mutex
	sentHashes       map[string][]ContentHash // Map of instance ID to content hashes
	hashMutex        sync.RWMutex
	hashCleanupTimer *time.Ticker
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
	handler := &TerminalHandler{
		instances: instances,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		activeInstances: make(map[string]*activeInstance),
		sentHashes:      make(map[string][]ContentHash),
	}

	// Start a cleanup timer for the content hashes
	handler.hashCleanupTimer = time.NewTicker(30 * time.Minute)
	go func() {
		for range handler.hashCleanupTimer.C {
			handler.cleanupOldHashes()
		}
	}()

	return handler
}

// cleanupOldHashes removes old content hashes to prevent memory leaks
func (h *TerminalHandler) cleanupOldHashes() {
	h.hashMutex.Lock()
	defer h.hashMutex.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	for instanceID, hashes := range h.sentHashes {
		var newHashes []ContentHash
		for _, hash := range hashes {
			if hash.timestamp.After(cutoff) {
				newHashes = append(newHashes, hash)
			}
		}

		if len(newHashes) == 0 {
			delete(h.sentHashes, instanceID)
		} else if len(newHashes) < len(hashes) {
			h.sentHashes[instanceID] = newHashes
		}
	}
}

// isDuplicateContent checks if we've recently sent this content
func (h *TerminalHandler) isDuplicateContent(instanceID, content string) bool {
	if content == "" {
		return true // Empty content is always a duplicate
	}

	// Calculate hash of content
	hasher := sha256.New()
	hasher.Write([]byte(content))
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Always use write lock to avoid race conditions
	h.hashMutex.Lock()
	defer h.hashMutex.Unlock()

	hashes, exists := h.sentHashes[instanceID]
	
	if !exists {
		// First content for this instance
		h.sentHashes[instanceID] = []ContentHash{{
			hash:      contentHash,
			timestamp: time.Now(),
		}}
		return false
	}

	// Check for duplicate - use slice for quick access to recent hashes
	for i := len(hashes) - 1; i >= 0 && i >= len(hashes)-10; i-- {
		if hashes[i].hash == contentHash {
			// Update timestamp of existing hash to prevent cleanup
			hashes[i].timestamp = time.Now()
			h.sentHashes[instanceID] = hashes
			return true // Found a duplicate
		}
	}

	// For older hashes, only check if we haven't found it in the recent ones
	for i := len(hashes) - 11; i >= 0; i-- {
		if hashes[i].hash == contentHash {
			// Update timestamp of existing hash to prevent cleanup
			hashes[i].timestamp = time.Now()
			h.sentHashes[instanceID] = hashes
			return true // Found a duplicate
		}
	}

	// Not a duplicate, add to hashes
	// Limit number of stored hashes to prevent unbounded growth
	if len(hashes) > 100 {
		// Keep only the most recent 50 hashes
		hashes = hashes[len(hashes)-50:]
	}
	
	h.sentHashes[instanceID] = append(hashes, ContentHash{
		hash:      contentHash,
		timestamp: time.Now(),
	})
	
	return false
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
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.FileOnlyErrorLog.Printf("Websocket error: %v", err)
				}
				return
			}

			// Handle JSON message format first (new protocol)
			if messageType == websocket.TextMessage {
				// Try to parse as JSON
				var jsonMsg map[string]interface{}
				if err := json.Unmarshal(message, &jsonMsg); err != nil {
					// If not valid JSON and starts with 'c', might be a close message
					if len(message) > 0 && message[0] == 'c' {
						log.FileOnlyInfoLog.Printf("Received close command, closing connection for instance: %s", instance.Title)
						return
					}
					log.FileOnlyErrorLog.Printf("Error parsing JSON message: %v", err)
					continue
				}
				
				// Process JSON message
				isCommand, ok := jsonMsg["isCommand"].(bool)
				if ok && isCommand {
					content, ok := jsonMsg["content"].(string)
					if ok {
						switch content {
						case "resize":
							// Handle resize
							cols, _ := jsonMsg["cols"].(float64)
							rows, _ := jsonMsg["rows"].(float64)
							if cols > 0 && rows > 0 {
								if err := instance.SetPreviewSize(int(cols), int(rows)); err != nil {
									log.FileOnlyErrorLog.Printf("Error resizing terminal: %v", err)
								} else {
									log.FileOnlyInfoLog.Printf("Resized terminal to %dx%d", int(cols), int(rows))
								}
							}
						case "clear_terminal":
							// Just acknowledge - clearing happens on client
							log.FileOnlyInfoLog.Printf("Received clear terminal request for instance: %s", instance.Title)
						case "close":
							// Client requested close
							log.FileOnlyInfoLog.Printf("Received close command via JSON for instance: %s", instance.Title)
							return
						}
					}
				} else {
					// Handle non-command JSON message (input)
					content, ok := jsonMsg["content"].(string)
					if ok && content != "" {
						if err := instance.SendPrompt(content); err != nil {
							log.FileOnlyErrorLog.Printf("Error sending input to instance: %v", err)
						}
					}
				}
				continue
			}
			
			// Process binary message based on type
			if messageType == websocket.BinaryMessage && len(message) > 0 {
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
					
				case ClearMessage:
					// Client requested terminal clear
					// Just acknowledge - actual clearing happens on client side
					log.FileOnlyInfoLog.Printf("Received clear terminal request for instance: %s", instance.Title)
				
				case CloseMessage:
					// Client requested close
					log.FileOnlyInfoLog.Printf("Received close command via binary for instance: %s", instance.Title)
					return
				}
			}
		}
	}()

	// Set up periodic content updates
	ticker := time.NewTicker(500 * time.Millisecond) // Further reduced rate to 500ms to reduce connection issues
	defer ticker.Stop()

	// Maintain connection state to avoid sending after closed connection
	connectionActive := true

	// Set up ping ticker to keep connection alive
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	// Send initial content immediately
	content, err := instance.Preview()
	if err == nil && content != "" && connectionActive {
		// Add detailed debug logging
		log.FileOnlyInfoLog.Printf("Sending initial terminal content (length: %d) to websocket for instance %s", 
			len(content), instance.Title)
		
		// Create the binary message with message type prefix
		message := append([]byte{OutputMessage}, []byte(content)...)
		
		// Send content update with binary protocol
		if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
			log.FileOnlyErrorLog.Printf("Error sending initial content update: %v", err)
			// If this is a serious error, mark connection as inactive
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) ||
			   strings.Contains(err.Error(), "broken pipe") ||
			   strings.Contains(err.Error(), "close sent") {
				log.FileOnlyErrorLog.Printf("Fatal error on initial content, marking connection inactive: %v", err)
				connectionActive = false
			}
		} else {
			// Record that we've sent this content
			h.isDuplicateContent(instance.Title, content)
		}
	}

	var lastContent string
	for {
		select {
		case <-ticker.C:
			// Skip if connection is no longer active
			if !connectionActive {
				continue
			}
			
			// Get current content
			content, err := instance.Preview()
			if err != nil {
				log.FileOnlyErrorLog.Printf("Error getting preview: %v", err)
				continue
			}

			// Only send if content has changed, is not empty, and is not a duplicate
			if content != lastContent && content != "" && !h.isDuplicateContent(instance.Title, content) {
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
						connectionActive = false
						return
					}
					// Non-fatal error, just log and continue
					log.FileOnlyErrorLog.Printf("Non-fatal error sending content, will retry on next tick: %v", err)
				}
			}
			
		case <-pingTicker.C:
			// Skip if connection is no longer active
			if !connectionActive {
				continue
			}
			
			// Send ping to keep the connection alive
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.FileOnlyErrorLog.Printf("Error sending ping: %v", err)
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) ||
				   strings.Contains(err.Error(), "close sent") ||
				   strings.Contains(err.Error(), "broken pipe") {
					log.FileOnlyErrorLog.Printf("Fatal error on ping, closing connection: %v", err)
					connectionActive = false
					return
				}
			}

		case <-doneCh:
			// Instance was detached
			log.FileOnlyInfoLog.Printf("Instance detached, closing websocket")
			connectionActive = false
			return
		}
	}
}