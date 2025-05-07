package terminal

import (
	"claude-squad/log"
	"claude-squad/session"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types
const (
	OutputMessage  = 'o' // Output from terminal to client
	InputMessage   = 'i' // Input from client to terminal
	ResizeMessage  = 'r' // Resize terminal window
	PingMessage    = 'p' // Ping from client
	PongMessage    = 'P' // Pong response
	CloseMessage   = 'c' // Close connection
)

// ResizeMessage represents a terminal resize request
type ResizeData struct {
	Columns int `json:"cols"`
	Rows    int `json:"rows"`
}

// Manager handles terminal websocket connections
type Manager struct {
	instances        *session.Storage
	upgrader         websocket.Upgrader
	activeAttachments map[string]*TmuxAttachment
	mutex            sync.Mutex
}

// NewManager creates a new terminal websocket manager
func NewManager(instances *session.Storage) *Manager {
	return &Manager{
		instances:        instances,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		activeAttachments: make(map[string]*TmuxAttachment),
	}
}

// HandleWebSocket handles a websocket connection for a terminal
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get instance name from URL path or query parameter
	instanceName := r.URL.Query().Get("instance")
	if instanceName == "" {
		http.Error(w, "Instance name required", http.StatusBadRequest)
		return
	}

	// Load instances
	instances, err := m.instances.LoadInstances()
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

	// Check if instance has a tmux session
	if !targetInstance.Started() || targetInstance.Paused() {
		http.Error(w, "Instance has no active tmux session", http.StatusBadRequest)
		return
	}

	log.FileOnlyInfoLog.Printf("New terminal websocket connection for instance: %s", instanceName)

	// Upgrade HTTP connection to WebSocket
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.ErrorLog.Printf("Failed to upgrade websocket: %v", err)
		return
	}
	defer conn.Close()

	// Get or create tmux attachment
	attachment, err := m.getOrCreateAttachment(instanceName, targetInstance)
	if err != nil {
		log.ErrorLog.Printf("Failed to create tmux attachment: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to connect to terminal: "+err.Error()))
		return
	}
	defer m.closeAttachment(instanceName)

	// Handle the terminal session
	m.handleTerminalSession(conn, attachment)
}

// getOrCreateAttachment gets an existing attachment or creates a new one
func (m *Manager) getOrCreateAttachment(instanceName string, instance *session.Instance) (*TmuxAttachment, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if we already have an attachment
	attachment, exists := m.activeAttachments[instanceName]
	if exists {
		return attachment, nil
	}

	// Create a new attachment
	attachment, err := NewTmuxAttachment(instance.GetTmuxSessionName())
	if err != nil {
		return nil, err
	}

	// Connect to the tmux session
	err = attachment.Connect()
	if err != nil {
		return nil, err
	}

	// Store the attachment
	m.activeAttachments[instanceName] = attachment
	return attachment, nil
}

// closeAttachment closes and removes an attachment
func (m *Manager) closeAttachment(instanceName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	attachment, exists := m.activeAttachments[instanceName]
	if !exists {
		return
	}

	// Close the attachment
	attachment.Close()
	delete(m.activeAttachments, instanceName)
}

// handleTerminalSession handles the terminal websocket session
func (m *Manager) handleTerminalSession(conn *websocket.Conn, attachment *TmuxAttachment) {
	// Set up channels for communication
	doneCh := make(chan struct{})
	defer close(doneCh)

	// Start reading from websocket (client to terminal)
	go func() {
		defer func() {
			doneCh <- struct{}{}
		}()

		for {
			// Read message from websocket
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.FileOnlyErrorLog.Printf("Websocket error: %v", err)
				}
				return
			}

			// Process the message
			if len(message) > 0 {
				switch message[0] {
				case InputMessage:
					// Write input to terminal
					if len(message) > 1 {
						_, err := attachment.Write(message[1:])
						if err != nil {
							log.FileOnlyErrorLog.Printf("Failed to write to terminal: %v", err)
							return
						}
					}

				case ResizeMessage:
					// Resize terminal
					if len(message) > 1 {
						var resizeData ResizeData
						err := json.Unmarshal(message[1:], &resizeData)
						if err != nil {
							log.FileOnlyErrorLog.Printf("Failed to parse resize message: %v", err)
							continue
						}

						err = attachment.ResizeTerminal(resizeData.Columns, resizeData.Rows)
						if err != nil {
							log.FileOnlyErrorLog.Printf("Failed to resize terminal: %v", err)
						}
					}

				case PingMessage:
					// Respond with pong
					err := conn.WriteMessage(websocket.TextMessage, []byte{PongMessage})
					if err != nil {
						log.FileOnlyErrorLog.Printf("Failed to send pong: %v", err)
						return
					}

				case CloseMessage:
					// Client requested close
					return
				}
			}
		}
	}()

	// Helper function to ensure ANSI escape sequences are not split
	// Returns the safe content to send and any remaining incomplete sequence
	ensureCompleteAnsiSequences := func(data []byte) ([]byte, []byte) {
		// If no data or very small, return as is (might be incomplete)
		if len(data) < 3 {
			return data, nil
		}
		
		// Check if the buffer ends with an incomplete ANSI escape sequence
		// Look backwards from the end to find the last ESC character
		for i := len(data) - 1; i >= 2; i-- {
			// Found ESC character
			if data[i] == 0x1b && data[i+1] == '[' {
				// We found an ESC[ near the end, check if it's a complete sequence
				// ANSI sequences end with a letter (ASCII range 64-126)
				isComplete := false
				for j := i + 2; j < len(data); j++ {
					// If we find a letter, the sequence is complete
					if data[j] >= 64 && data[j] <= 126 {
						isComplete = true
						break
					}
				}
				
				if !isComplete {
					// This is an incomplete sequence at the end
					return data[:i], data[i:]
				}
				
				// If we found a complete sequence, no need to check earlier positions
				break
			}
		}
		
		return data, nil
	}
	
	// Set up larger buffer for reading from terminal to avoid splitting ANSI sequences
	// and a buffer to hold incomplete ANSI sequences between reads
	buffer := make([]byte, 32768)  // Increased size to 32KB
	var ansiBuffer []byte
	
	// Read from terminal and send to websocket (terminal to client)
	go func() {
		defer func() {
			doneCh <- struct{}{}
		}()

		for {
			n, err := attachment.Read(buffer)
			if err != nil {
				log.FileOnlyErrorLog.Printf("Failed to read from terminal: %v", err)
				return
			}

			if n > 0 {
				// Process the buffer to handle partial ANSI sequences
				data := buffer[:n]
				
				// If we have incomplete ANSI sequence from last read, prepend it
				if len(ansiBuffer) > 0 {
					data = append(ansiBuffer, data...)
					ansiBuffer = nil // Clear the buffer
				}
				
				// Check for incomplete ANSI sequence at the end
				content, remaining := ensureCompleteAnsiSequences(data)
				
				// Store any incomplete sequence for next read
				if len(remaining) > 0 {
					ansiBuffer = remaining
					log.FileOnlyInfoLog.Printf("Saved incomplete ANSI sequence (%d bytes) for next read", len(remaining))
				}
				
				// Create message with output type prefix
				message := make([]byte, len(content)+1)
				message[0] = OutputMessage
				copy(message[1:], content)

				// Send to websocket
				err = conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.FileOnlyErrorLog.Printf("Failed to write to websocket: %v", err)
					return
				}
			}
		}
	}()

	// Start pinging to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				// Send ping
				err := conn.WriteMessage(websocket.TextMessage, []byte{PingMessage})
				if err != nil {
					log.FileOnlyErrorLog.Printf("Failed to send ping: %v", err)
					return
				}
			case <-doneCh:
				return
			}
		}
	}()

	// Wait for either goroutine to finish
	<-doneCh
}