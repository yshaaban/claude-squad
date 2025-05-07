#!/bin/bash
set -e

echo "Applying terminal connection and heartbeat fixes..."

# 1. Update TerminalInput struct in types.go
echo "Adding Cols and Rows fields to TerminalInput struct..."
# Check if fields already exist to prevent duplicate additions
if ! grep -q "Cols.*interface{}" /Users/ysh/src/claude-squad/web/types/types.go; then
  sed -i '' 's/type TerminalInput struct {/type TerminalInput struct {/' /Users/ysh/src/claude-squad/web/types/types.go
  sed -i '' 's/IsCommand     bool   `json:"is_command"` \/\/ True if this is a command like resize/IsCommand     bool        `json:"is_command"` \/\/ True if this is a command like resize\n\tCols          interface{} `json:"cols,omitempty"`\n\tRows          interface{} `json:"rows,omitempty"`/' /Users/ysh/src/claude-squad/web/types/types.go
  echo "Added Cols and Rows fields to TerminalInput struct"
else
  echo "Fields already exist in TerminalInput struct, skipping"
fi

# 2. Modify Terminal.tsx to remove custom heartbeat
echo "Removing custom heartbeat mechanism from Terminal.tsx..."

# Remove PING/PONG constants
sed -i '' 's/const PING_MESSAGE = '\''p'\''.charCodeAt(0)/\/\/ Removed ping\/pong constants/' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/const PONG_MESSAGE = '\''P'\''.charCodeAt(0)//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Remove ping interval refs
sed -i '' 's/const pingIntervalRef = useRef<number | null>(null)/\/\/ Removed ping interval ref/' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/const missedHeartbeatsRef = useRef<number>(0)/\/\/ Removed missed heartbeats ref/' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Remove sendPing function
sed -i '' -e '/\/\/ Send ping to keep connection alive/,/}, \[socket, sendMessage, log\])/ c\
  \/\/ Removed custom ping function - using standard WebSocket protocol
' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Remove startPingInterval function
sed -i '' -e '/\/\/ Start ping interval/,/}, \[socket, sendPing, log\])/ c\
  \/\/ Removed ping interval function - using standard WebSocket protocol
' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Remove pingInterval call from websocket onopen
sed -i '' 's/sendResize()\n          startPingInterval()/sendResize()/' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Remove pingInterval cleanup
sed -i '' 's/\/\/ Clear ping interval.*$/\/\/ Ping interval removed - using standard WebSocket protocol/' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/if (pingIntervalRef.current) {.*$//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/clearInterval(pingIntervalRef.current).*$//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/pingIntervalRef.current = null.*$//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/}.*$//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Remove pong message handler
sed -i '' 's/case PONG_MESSAGE:.*$/\/\/ Removed PONG_MESSAGE handling/' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/missedHeartbeatsRef.current = 0.*$//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/log('\''info'\'', '\''Received pong'\'').*$//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx
sed -i '' 's/break.*$//' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Add cleanup for close message
sed -i '' 's/const CLOSE_MESSAGE = '\''c'\''.charCodeAt(0)/\/\/ Removed close message constant/' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# 3. Fix the JSON message handling by removing binary fallbacks
echo "Removing binary fallbacks from Terminal.tsx message methods..."

# Update sendInput to remove binary fallback
sed -i '' -e '/\/\/ Send input to terminal/,/}, \[socket, sendMessage, log\])/ c\
  \/\/ Send input to terminal\
  const sendInput = useCallback((text: string) => {\
    if (!socket || socket.readyState !== WebSocket.OPEN || !text) {\
      log('\''warn'\'', '\''Cannot send input - not connected or empty text'\'')\
      return\
    }\
    \
    try {\
      \/\/ Send using JSON protocol\
      const message = {\
        content: text,\
        isCommand: false\
      }\
      \
      socket.send(JSON.stringify(message))\
      log('\''info'\'', `Sent input: ${text}`)\
    } catch (error) {\
      \/\/ Log error - don'\''t fallback to binary protocol\
      log('\''error'\'', `Failed to send input: ${error}. Connection may be unstable.`)\
    }\
  }, [socket, log])
' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Update sendResize to remove binary fallback
sed -i '' -e '/\/\/ Send terminal resize/,/}, \[socket, sendMessage, terminal, log\])/ c\
  \/\/ Send terminal resize\
  const sendResize = useCallback(() => {\
    if (!socket || socket.readyState !== WebSocket.OPEN || !terminalRef.current) {\
      return\
    }\
    \
    \/\/ Only proceed if we have valid dimensions\
    if (!terminal || !fitAddonRef.current) return\
    \
    try {\
      \/\/ Get terminal dimensions from xterm directly\
      const dimensions = terminal.options\
      \
      if (!dimensions.cols || !dimensions.rows) {\
        log('\''warn'\'', '\''Invalid terminal dimensions'\'')\
        return\
      }\
      \
      const cols = dimensions.cols\
      const rows = dimensions.rows\
      \
      \/\/ Send JSON format\
      const message = {\
        cols: cols,\
        rows: rows,\
        isCommand: true,\
        content: '\''resize'\''\
      }\
      \
      socket.send(JSON.stringify(message))\
      log('\''info'\'', `Sent resize: ${cols}x${rows}`)\
    } catch (error) {\
      log('\''error'\'', `Failed to send resize command: ${error}`)\
    }\
  }, [socket, terminal, log])
' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# Update clearTerminal to remove binary fallback
sed -i '' -e '/\/\/ Clear terminal/,/}, \[terminal, socket, sendMessage, log\])/ c\
  \/\/ Clear terminal\
  const clearTerminal = useCallback(() => {\
    if (terminal) {\
      terminal.clear()\
      log('\''info'\'', '\''Terminal cleared'\'')\
      \
      \/\/ Also try to tell server to clear (if supported)\
      if (socket && socket.readyState === WebSocket.OPEN) {\
        try {\
          \/\/ Send using JSON protocol\
          const message = {\
            isCommand: true,\
            content: '\''clear_terminal'\''\
          }\
          socket.send(JSON.stringify(message))\
        } catch (error) {\
          log('\''error'\'', `Failed to send clear command: ${error}`)\
        }\
      }\
    }\
  }, [terminal, socket, log])
' /Users/ysh/src/claude-squad/frontend/src/components/terminal/Terminal.tsx

# 4. Create a patch file for websocket.go to add resize command handling
echo "Creating patch for websocket.go to add resize command handling..."

cat > /Users/ysh/src/claude-squad/websocket_resize_patch.diff << 'EOF'
--- websocket.go.old	2025-05-07 17:37:58
+++ websocket.go	2025-05-07 17:37:58
@@ -310,6 +310,37 @@
 									"error":   "Clear terminal not supported directly",
 								}
 
+							case cmd == "resize":
+								// Handle resize command
+								cols, colsOk := input.Cols.(float64)
+								rows, rowsOk := input.Rows.(float64)
+								
+								if colsOk && rowsOk && cols > 0 && rows > 0 {
+									log.FileOnlyInfoLog.Printf("WebSocket: Received resize command for '%s': %dx%d", 
+										instanceTitle, int(cols), int(rows))
+									
+									// Try to resize terminal if applicable
+									if err := monitor.ResizeTerminal(instanceTitle, int(cols), int(rows)); err != nil {
+										log.FileOnlyErrorLog.Printf("WebSocket: Error resizing terminal for '%s': %v", instanceTitle, err)
+										response = map[string]interface{}{
+											"type":    "command_response",
+											"command": "resize",
+											"success": false,
+											"error":   fmt.Sprintf("Failed to resize terminal: %v", err),
+										}
+									} else {
+										log.FileOnlyInfoLog.Printf("WebSocket: Successfully resized terminal for '%s'", instanceTitle)
+										response = map[string]interface{}{
+											"type":    "command_response",
+											"command": "resize",
+											"success": true,
+										}
+									}
+								} else {
+									log.FileOnlyWarningLog.Printf("WebSocket: Invalid resize dimensions for '%s': cols=%v, rows=%v", 
+										instanceTitle, input.Cols, input.Rows)
+									response = map[string]interface{}{
+EOF

# Continue the patch file
cat >> /Users/ysh/src/claude-squad/websocket_resize_patch.diff << 'EOF'
+										"type":    "command_response",
+										"command": "resize",
+										"success": false,
+										"error":   "Invalid dimensions",
+									}
+								}
+
 							default:
 								// Unknown command
 								log.FileOnlyInfoLog.Printf("WebSocket: Unknown command: %s for '%s'", cmd, instanceTitle)
EOF

# 5. Create a patch file for websocket.go to add standard pong handler
echo "Creating patch for websocket.go to add standard pong handler..."

cat > /Users/ysh/src/claude-squad/websocket_pong_patch.diff << 'EOF'
--- websocket.go.old	2025-05-07 17:37:58
+++ websocket.go	2025-05-07 17:37:58
@@ -115,6 +115,15 @@
 			log.FileOnlyInfoLog.Printf("WebSocket: Connection successfully upgraded for '%s' from %s", 
 				instanceTitle, r.RemoteAddr)
 			defer conn.Close()
+			
+			// Set ping handler to keep connection alive using standard WebSocket protocol
+			conn.SetPongHandler(func(string) error {
+				log.FileOnlyInfoLog.Printf("WebSocket: Received standard pong from client for '%s'", instanceTitle)
+				// Extend read deadline on successful pong
+				conn.SetReadDeadline(time.Now().Add(70 * time.Second))
+				return nil
+			})
+			
+			// Set initial read deadline
+			conn.SetReadDeadline(time.Now().Add(70 * time.Second))
 
 			// Get requested format
EOF

echo "Apply these patches manually by running:"
echo "  patch /Users/ysh/src/claude-squad/web/handlers/websocket.go /Users/ysh/src/claude-squad/websocket_resize_patch.diff"
echo "  patch /Users/ysh/src/claude-squad/web/handlers/websocket.go /Users/ysh/src/claude-squad/websocket_pong_patch.diff"

echo "Terminal connection fixes script completed. Please run the patch commands manually as shown above."