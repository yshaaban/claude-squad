package handlers

import (
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web/types"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// Note: The following helper functions are defined in instances.go:
// - findInstanceByTitle
// - convertAnsiToHtml
// - stripAnsi

// WebSocketHandler handles terminal output streaming via WebSocket with bidirectional communication.
func WebSocketHandler(storage *session.Storage, monitor types.TerminalMonitorInterface) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// TODO: Implement proper origin checking
			return true
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Add detailed connection logging
		log.InfoLog.Printf("WebSocket: New connection request from %s", r.RemoteAddr)
		
		instanceTitle := chi.URLParam(r, "name")
		if instanceTitle == "" {
			log.ErrorLog.Printf("WebSocket: Missing instance name parameter")
			http.Error(w, "Instance name required", http.StatusBadRequest)
			return
		}
		log.InfoLog.Printf("WebSocket: Connection request for instance: %s", instanceTitle)

		// Verify instance exists
		instance, err := findInstanceByTitle(storage, instanceTitle)
		if err != nil {
			log.ErrorLog.Printf("WebSocket: Instance %s not found: %v", instanceTitle, err)
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		log.InfoLog.Printf("WebSocket: Found instance %s with status=%s, started=%v", 
			instanceTitle, string(instance.Status), instance.Started())

		// Get privileges parameter (read-only vs read-write)
		privileges := r.URL.Query().Get("privileges")
		if privileges == "" {
			privileges = "read-only" // Default to read-only for safety
		}

		// Ensure privileges is valid
		if privileges != "read-only" && privileges != "read-write" {
			log.ErrorLog.Printf("WebSocket: Invalid privileges parameter: %s", privileges)
			http.Error(w, "Invalid privileges parameter", http.StatusBadRequest)
			return
		}
		log.InfoLog.Printf("WebSocket: Using privileges=%s for instance %s", privileges, instanceTitle)

		// Upgrade HTTP connection to WebSocket
		log.InfoLog.Printf("WebSocket: Upgrading connection for instance %s", instanceTitle)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.ErrorLog.Printf("WebSocket upgrade failed: %v", err)
			return
		}
		log.InfoLog.Printf("WebSocket: Connection successfully upgraded for %s", instanceTitle)
		defer conn.Close()

		// Get requested format
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "ansi"
		}

		// Verify format is valid
		if format != "ansi" && format != "html" && format != "text" {
			log.ErrorLog.Printf("WebSocket: Invalid format parameter: %s", format)
			conn.WriteJSON(map[string]string{"error": "Invalid format parameter"})
			conn.Close()
			return
		}
		log.InfoLog.Printf("WebSocket: Using format=%s for instance %s", format, instanceTitle)

		// Create update channel
		log.InfoLog.Printf("WebSocket: Subscribing to updates for instance %s", instanceTitle)
		updates := monitor.Subscribe(instanceTitle)
		defer monitor.Unsubscribe(instanceTitle, updates)

		// Send initial content if available
		initialContent, exists := monitor.GetContent(instanceTitle)
		if exists {
			log.InfoLog.Printf("WebSocket: Initial content available for %s (len: %d)", 
				instanceTitle, len(initialContent))
			
			// Apply format conversion if needed
			formattedContent := initialContent
			if format == "html" {
				formattedContent = convertAnsiToHtml(initialContent)
				log.InfoLog.Printf("WebSocket: Converted initial content to HTML format")
			} else if format == "text" {
				formattedContent = stripAnsi(initialContent)
				log.InfoLog.Printf("WebSocket: Converted initial content to plain text format")
			}

			// Make sure we actually have content to send
			if len(formattedContent) == 0 {
				log.ErrorLog.Printf("WebSocket: Empty initial content for instance %s after formatting", instanceTitle)
				formattedContent = "[Terminal content is empty. The instance may still be initializing.]"
			}

			// Use HasUpdated method to check for prompt status
			updated, hasPrompt := instance.HasUpdated()
			log.InfoLog.Printf("WebSocket: Instance %s has updated=%v, has prompt=%v", 
				instanceTitle, updated, hasPrompt)

			// Send initial update
			initialUpdate := types.TerminalUpdate{
				InstanceTitle: instanceTitle,
				Content:       formattedContent,
				Timestamp:     time.Now(),
				Status:        string(instance.Status),
				HasPrompt:     hasPrompt,
			}

			log.InfoLog.Printf("WebSocket: Sending initial update for %s, content length: %d, status: %s", 
				instanceTitle, len(formattedContent), string(instance.Status))
			
			// Send with timeout protection
			writeErrorChan := make(chan error, 1)
			go func() {
				writeErrorChan <- conn.WriteJSON(initialUpdate)
			}()
			
			select {
			case err := <-writeErrorChan:
				if err != nil {
					log.ErrorLog.Printf("Error sending initial update: %v", err)
					return
				}
				log.InfoLog.Printf("WebSocket: Successfully sent initial update for %s", instanceTitle)
			case <-time.After(5 * time.Second):
				log.ErrorLog.Printf("Timeout sending initial WebSocket update for %s", instanceTitle)
				return
			}
		} else {
			log.InfoLog.Printf("WebSocket: No initial content available for instance %s", instanceTitle)
			
			// Send an empty update with a message
			emptyUpdate := types.TerminalUpdate{
				InstanceTitle: instanceTitle,
				Content:       "[No terminal content available yet. Please wait...]",
				Timestamp:     time.Now(),
				Status:        string(instance.Status),
				HasPrompt:     false,
			}
			
			log.InfoLog.Printf("WebSocket: Sending empty placeholder update for %s", instanceTitle)
			if err := conn.WriteJSON(emptyUpdate); err != nil {
				log.ErrorLog.Printf("Error sending empty initial update: %v", err)
				return
			}
			log.InfoLog.Printf("WebSocket: Successfully sent empty placeholder for %s", instanceTitle)
		}

		// Send terminal configuration
		config := map[string]interface{}{
			"type":       "config",
			"privileges": privileges,
			"theme":      "dark", // Default theme
			"fontFamily": "Menlo, Monaco, 'Courier New', monospace",
			"fontSize":   14,
		}
		log.InfoLog.Printf("WebSocket: Sending terminal configuration for %s", instanceTitle)
		if err := conn.WriteJSON(config); err != nil {
			log.ErrorLog.Printf("Error sending config: %v", err)
			return
		}
		log.InfoLog.Printf("WebSocket: Successfully sent terminal configuration for %s", instanceTitle)

		// Mutex for websocket writes
		var writeMu sync.Mutex

		// Handle incoming messages from client (bidirectional communication)
		if privileges == "read-write" {
			log.InfoLog.Printf("WebSocket: Starting read-write handler for %s", instanceTitle)
			go func() {
				for {
					// Read message from client
					_, message, err := conn.ReadMessage()
					if err != nil {
						if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
							log.ErrorLog.Printf("WebSocket read error: %v", err)
						}
						return
					}
					log.InfoLog.Printf("WebSocket: Received message from client for %s, length: %d", 
						instanceTitle, len(message))

					// Parse message
					var input types.TerminalInput
					if err := json.Unmarshal(message, &input); err != nil {
						log.ErrorLog.Printf("Error parsing WebSocket input: %v", err)
						
						writeMu.Lock()
						conn.WriteJSON(map[string]string{"error": "Invalid message format"})
						writeMu.Unlock()
						continue
					}

					// Handle different types of input
					if input.IsCommand {
						// Handle special commands
						cmd := input.Content
						log.InfoLog.Printf("WebSocket: Received command: %s for %s", cmd, instanceTitle)
						var response map[string]interface{}

						switch {
						case cmd == "get_tasks":
							// Get tasks for terminal
							log.InfoLog.Printf("WebSocket: Processing get_tasks command for %s", instanceTitle)
							tasks, err := monitor.GetTasks(instanceTitle)
							if err != nil {
								log.ErrorLog.Printf("Error getting tasks: %v", err)
								response = map[string]interface{}{
									"type":  "command_response",
									"command": "get_tasks",
									"success": false,
									"error": err.Error(),
								}
							} else {
								log.InfoLog.Printf("WebSocket: Found %d tasks for %s", len(tasks), instanceTitle)
								response = map[string]interface{}{
									"type":  "command_response",
									"command": "get_tasks",
									"success": true,
									"tasks": tasks,
								}
							}

						case cmd == "clear_terminal":
							// Clear terminal not supported directly, just acknowledge
							log.InfoLog.Printf("WebSocket: Clear terminal command not supported for %s", instanceTitle)
							response = map[string]interface{}{
								"type":    "command_response",
								"command": "clear_terminal",
								"success": false,
								"error":   "Clear terminal not supported directly",
							}

						default:
							// Unknown command
							log.InfoLog.Printf("WebSocket: Unknown command: %s for %s", cmd, instanceTitle)
							response = map[string]interface{}{
								"type":    "command_response",
								"command": cmd,
								"success": false,
								"error":   "Unknown command",
							}
						}

						writeMu.Lock()
						log.InfoLog.Printf("WebSocket: Sending command response for %s", instanceTitle)
						conn.WriteJSON(response)
						writeMu.Unlock()
					} else {
						// Regular terminal input - send to terminal
						log.InfoLog.Printf("WebSocket: Received terminal input for %s: %s", 
							instanceTitle, input.Content)
						err := monitor.SendInput(instanceTitle, input.Content)
						if err != nil {
							log.ErrorLog.Printf("Error sending input to terminal: %v", err)
							
							writeMu.Lock()
							conn.WriteJSON(map[string]string{
								"type":  "input_response",
								"error": fmt.Sprintf("Failed to send input: %v", err),
							})
							writeMu.Unlock()
						} else {
							log.InfoLog.Printf("WebSocket: Successfully sent input to terminal for %s", 
								instanceTitle)
						}
					}
				}
			}()
		} else {
			log.InfoLog.Printf("WebSocket: Read-only mode active for %s, input disabled", instanceTitle)
		}

		// Handle ping messages to keep connection alive
		log.InfoLog.Printf("WebSocket: Starting ping handler for %s", instanceTitle)
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					writeMu.Lock()
					log.InfoLog.Printf("WebSocket: Sending ping to %s", instanceTitle)
					if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
						log.ErrorLog.Printf("WebSocket: Ping failed for %s: %v", instanceTitle, err)
						writeMu.Unlock()
						return
					}
					writeMu.Unlock()
				case <-monitor.Done():
					log.InfoLog.Printf("WebSocket: Monitor done, closing ping handler for %s", instanceTitle)
					return
				}
			}
		}()

		// Listen for updates and send to client
		log.InfoLog.Printf("WebSocket: Starting update listener for %s", instanceTitle)
		updateCounter := 0
		for update := range updates {
			updateCounter++
			log.InfoLog.Printf("WebSocket: Received update #%d for %s, content length: %d", 
				updateCounter, update.InstanceTitle, len(update.Content))
			
			// Skip empty updates
			if len(update.Content) == 0 {
				log.WarningLog.Printf("WebSocket: Skipping empty update #%d for %s", 
					updateCounter, instanceTitle)
				continue
			}
			
			// Apply format conversion if needed
			if format == "html" {
				update.Content = convertAnsiToHtml(update.Content)
				log.InfoLog.Printf("WebSocket: Converted update to HTML format for %s", instanceTitle)
			} else if format == "text" {
				update.Content = stripAnsi(update.Content)
				log.InfoLog.Printf("WebSocket: Converted update to plain text format for %s", instanceTitle)
			}
			
			// Make sure we still have content after conversion
			if len(update.Content) == 0 {
				log.WarningLog.Printf("WebSocket: Empty content after format conversion for %s, adding placeholder", 
					instanceTitle)
				update.Content = "[Terminal content unavailable]"
			}

			writeMu.Lock()
			log.InfoLog.Printf("WebSocket: Sending update #%d to client for %s, content length: %d", 
				updateCounter, instanceTitle, len(update.Content))
			
			if err := conn.WriteJSON(update); err != nil {
				log.ErrorLog.Printf("Error sending WebSocket update for %s: %v", instanceTitle, err)
				writeMu.Unlock()
				break
			} else {
				log.InfoLog.Printf("WebSocket: Successfully sent update #%d for %s", 
					updateCounter, instanceTitle)
			}
			writeMu.Unlock()
		}
		
		log.InfoLog.Printf("WebSocket: Connection handler completed for %s", instanceTitle)
	}
}