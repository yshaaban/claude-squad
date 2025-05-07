# WebSocket Connection Fix

This document outlines the necessary changes to fix WebSocket connection issues between the React frontend and the server. The problem is a fundamental mismatch between client and server WebSocket protocols, causing disconnections and "broken pipe" errors.

## Root Cause Analysis

The core issues are:

1. **Custom Heartbeat Mechanism Mismatch**:
   - Client (`Terminal.tsx`) uses a custom binary ping/pong protocol (sending 'p' byte, expecting 'P' byte back)
   - Server uses standard WebSocket ping/pong frames
   - After 3 missed custom heartbeats (~45s), the client forcefully closes the connection
   - When the server tries to send data to a closed connection, it encounters "broken pipe" errors

2. **Binary Protocol Fallbacks**:
   - Client tries to send JSON messages but falls back to binary formats (prefixed with type bytes) when JSON fails
   - Server primarily expects JSON messages and can't interpret client's binary format

## Required Changes

### 1. In frontend/src/components/terminal/Terminal.tsx

#### A. Remove Custom Ping/Pong Constants & Protocol
```javascript
// REMOVE these constants
const PING_MESSAGE = 'p'.charCodeAt(0)
const PONG_MESSAGE = 'P'.charCodeAt(0)

// REMOVE these references
const pingIntervalRef = useRef<number | null>(null)
const missedHeartbeatsRef = useRef<number>(0)

// REMOVE sendPing function completely
const sendPing = useCallback(() => {
  if (!socket || socket.readyState !== WebSocket.OPEN) return
  
  sendMessage(PING_MESSAGE, new Uint8Array(0))
  missedHeartbeatsRef.current++
  log('info', `Sending keep-alive ping (missed: ${missedHeartbeatsRef.current})`)
}, [socket, sendMessage, log])

// REMOVE startPingInterval function completely
const startPingInterval = useCallback(() => {
  // Clear existing interval if any
  if (pingIntervalRef.current) {
    clearInterval(pingIntervalRef.current)
  }
  
  // Reset heartbeat counter
  missedHeartbeatsRef.current = 0
  
  // Start new interval
  const MAX_MISSED_HEARTBEATS = 3
  
  pingIntervalRef.current = window.setInterval(() => {
    // ...interval logic...
  }, 15000) // Every 15 seconds
}, [socket, sendPing, log])

// REMOVE startPingInterval call in WebSocket onopen handler
ws.onopen = () => {
  // ...
  setTimeout(() => {
    sendResize()
    startPingInterval() // REMOVE THIS LINE
  }, 300)
}

// REMOVE pingInterval cleanup in various places
if (pingIntervalRef.current) {
  clearInterval(pingIntervalRef.current)
  pingIntervalRef.current = null
}

// REMOVE PONG_MESSAGE case in message handler
case PONG_MESSAGE:
  missedHeartbeatsRef.current = 0
  log('info', 'Received pong')
  break
```

#### B. Modify sendInput to Remove Binary Fallback
```javascript
// REPLACE the function with this version
const sendInput = useCallback((text: string) => {
  if (!socket || socket.readyState !== WebSocket.OPEN || !text) {
    log('warn', 'Cannot send input - not connected or empty text')
    return
  }
  
  try {
    // Send using JSON protocol only
    const message = {
      content: text,
      isCommand: false
    }
    
    socket.send(JSON.stringify(message))
    log('info', `Sent input: ${text}`)
  } catch (error) {
    // Just log error - don't fallback to binary protocol
    log('error', `Failed to send input: ${error}. Connection may be unstable.`)
  }
}, [socket, log])
```

#### C. Modify sendResize to Remove Binary Fallback
```javascript
// REPLACE the function with this version
const sendResize = useCallback(() => {
  if (!socket || socket.readyState !== WebSocket.OPEN || !terminalRef.current) {
    return
  }
  
  // Only proceed if we have valid dimensions
  if (!terminal || !fitAddonRef.current) return
  
  try {
    // Get terminal dimensions from xterm directly
    const dimensions = terminal.options
    
    if (!dimensions.cols || !dimensions.rows) {
      log('warn', 'Invalid terminal dimensions')
      return
    }
    
    const cols = dimensions.cols
    const rows = dimensions.rows
    
    // Send using JSON format only
    const message = {
      cols: cols,
      rows: rows,
      isCommand: true,
      content: 'resize'
    }
    
    socket.send(JSON.stringify(message))
    log('info', `Sent resize: ${cols}x${rows}`)
  } catch (error) {
    log('error', `Failed to send resize command: ${error}`)
  }
}, [socket, terminal, log])
```

#### D. Modify clearTerminal to Remove Binary Fallback
```javascript
// REPLACE the function with this version
const clearTerminal = useCallback(() => {
  if (terminal) {
    terminal.clear()
    log('info', 'Terminal cleared')
    
    // Also try to tell server to clear (if supported)
    if (socket && socket.readyState === WebSocket.OPEN) {
      try {
        // Send using JSON protocol only
        const message = {
          isCommand: true,
          content: 'clear_terminal'
        }
        socket.send(JSON.stringify(message))
      } catch (error) {
        log('error', `Failed to send clear command: ${error}`)
      }
    }
  }
}, [terminal, socket, log])
```

#### E. Remove Binary Close Message
```javascript
// REPLACE this code
if (socket.readyState === WebSocket.OPEN) {
  try {
    // Try to send a close message to the server
    sendMessage(CLOSE_MESSAGE, new Uint8Array(0))
    
    // Give it a moment to send before actually closing
    setTimeout(() => {
      socket.close(1000, "Terminal component unmounting")
    }, 100)
  } catch (err) {
    // Just close directly if sending fails
    socket.close()
  }
}

// WITH this code
if (socket.readyState === WebSocket.OPEN) {
  try {
    // Standard WebSocket close with reason
    socket.close(1000, "Terminal component unmounting")
  } catch (err) {
    // If closing with reason fails, just close it
    socket.close()
  }
}
```

### 2. In web/types/types.go

#### Add Cols and Rows Fields to TerminalInput Struct
```go
// CHANGE from
type TerminalInput struct {
  InstanceTitle string `json:"instance_title"`
  Content       string `json:"content"`
  IsCommand     bool   `json:"is_command"` // True if this is a command like resize
}

// TO
type TerminalInput struct {
  InstanceTitle string      `json:"instance_title"`
  Content       string      `json:"content"`
  IsCommand     bool        `json:"is_command"` // True if this is a command like resize
  Cols          interface{} `json:"cols,omitempty"`
  Rows          interface{} `json:"rows,omitempty"`
}
```

### 3. In web/handlers/websocket.go

#### A. Add Resize Command Handler
```go
// ADD THIS CASE in the switch statement handling commands
case cmd == "resize":
  // Handle resize command
  cols, colsOk := input.Cols.(float64)
  rows, rowsOk := input.Rows.(float64)
  
  if colsOk && rowsOk && cols > 0 && rows > 0 {
    log.FileOnlyInfoLog.Printf("WebSocket: Received resize command for '%s': %dx%d", 
      instanceTitle, int(cols), int(rows))
    
    // Try to resize terminal if applicable
    if err := monitor.ResizeTerminal(instanceTitle, int(cols), int(rows)); err != nil {
      log.FileOnlyErrorLog.Printf("WebSocket: Error resizing terminal for '%s': %v", instanceTitle, err)
      response = map[string]interface{}{
        "type":    "command_response",
        "command": "resize",
        "success": false,
        "error":   fmt.Sprintf("Failed to resize terminal: %v", err),
      }
    } else {
      log.FileOnlyInfoLog.Printf("WebSocket: Successfully resized terminal for '%s'", instanceTitle)
      response = map[string]interface{}{
        "type":    "command_response",
        "command": "resize",
        "success": true,
      }
    }
  } else {
    log.FileOnlyWarningLog.Printf("WebSocket: Invalid resize dimensions for '%s': cols=%v, rows=%v", 
      instanceTitle, input.Cols, input.Rows)
    response = map[string]interface{}{
      "type":    "command_response",
      "command": "resize",
      "success": false,
      "error":   "Invalid dimensions",
    }
  }
```

#### B. Add Standard Ping/Pong Handler
Add this right after the successful WebSocket upgrade:

```go
// After upgrading the connection
log.FileOnlyInfoLog.Printf("WebSocket: Connection successfully upgraded for '%s' from %s", 
  instanceTitle, r.RemoteAddr)
defer conn.Close()

// ADD THIS CODE
// Set ping handler to keep connection alive using standard WebSocket protocol
conn.SetPongHandler(func(string) error {
  log.FileOnlyInfoLog.Printf("WebSocket: Received standard pong from client for '%s'", instanceTitle)
  // Extend read deadline on successful pong
  conn.SetReadDeadline(time.Now().Add(70 * time.Second))
  return nil
})

// Set initial read deadline
conn.SetReadDeadline(time.Now().Add(70 * time.Second))
```

## Explanation

These changes address the root causes by:

1. Removing the client's custom binary ping/pong mechanism and relying on standard WebSocket ping/pong frames that the browser handles automatically.

2. Adding a proper server-side pong handler with read deadlines to detect and close dead connections.

3. Eliminating all binary fallbacks in favor of consistent JSON messaging, which the server already expects.

4. Adding proper resize command handling on the server side.

Together, these changes ensure that:
- The client and server use the same protocol for heartbeats/keepalives
- Messages use a consistent JSON format that both sides understand
- The terminal resize functionality works correctly
- WebSocket connections remain stable without unexpected disconnections

This should resolve the "broken pipe" errors and repeated connection attempts that are currently observed.