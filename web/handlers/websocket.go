package handlers

import (
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web/types"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles terminal output streaming via WebSocket.
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
		instanceTitle := chi.URLParam(r, "name")
		if instanceTitle == "" {
			http.Error(w, "Instance name required", http.StatusBadRequest)
			return
		}

		// Verify instance exists
		instance, err := findInstanceByTitle(storage, instanceTitle)
		if err != nil {
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.ErrorLog.Printf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Get requested format
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "ansi"
		}

		// Verify format is valid
		if format != "ansi" && format != "html" && format != "text" {
			conn.WriteJSON(map[string]string{"error": "Invalid format parameter"})
			conn.Close()
			return
		}

		// Create update channel
		updates := monitor.Subscribe(instanceTitle)
		defer monitor.Unsubscribe(instanceTitle, updates)

		// Send initial content if available
		initialContent, exists := monitor.GetContent(instanceTitle)
		if exists {
			// Apply format conversion if needed
			formattedContent := initialContent
			if format == "html" {
				formattedContent = convertAnsiToHtml(initialContent)
			} else if format == "text" {
				formattedContent = stripAnsi(initialContent)
			}

			// Send initial update
			initialUpdate := types.TerminalUpdate{
				InstanceTitle: instanceTitle,
				Content:       formattedContent,
				Timestamp:     time.Now(),
				Status:        string(instance.Status),
				HasPrompt:     false, // Determine from content if needed
			}

			if err := conn.WriteJSON(initialUpdate); err != nil {
				log.ErrorLog.Printf("Error sending initial update: %v", err)
				return
			}
		}

		// Handle ping messages to keep connection alive
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
						return
					}
				case <-monitor.Done():
					return
				}
			}
		}()

		// Listen for updates and send to client
		for update := range updates {
			// Apply format conversion if needed
			if format == "html" {
				update.Content = convertAnsiToHtml(update.Content)
			} else if format == "text" {
				update.Content = stripAnsi(update.Content)
			}

			if err := conn.WriteJSON(update); err != nil {
				log.ErrorLog.Printf("Error sending WebSocket update: %v", err)
				break
			}
		}
	}
}