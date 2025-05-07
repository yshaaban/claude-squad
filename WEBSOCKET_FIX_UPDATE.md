# WebSocket Connection Fix Update

This document provides additional fixes to the WebSocket connection issues identified in WEBSOCKET_FIX.md. These updates focus on improving client-side terminal initialization and error handling.

## Client-Side Fixes (Terminal.tsx)

### 1. Fixed XTerm.js Initialization

The client-side terminal initialization has been improved to prevent "Cannot read properties of undefined (reading 'dimensions')" errors:

```typescript
// Create a shared function for fitting and resizing
const attemptFitAndResize = useCallback(() => {
  if (!terminalRef.current || !terminal || !fitAddonRef.current) {
    log('warn', 'Fit attempt skipped: terminal or addons not ready.');
    return false;
  }
  if (terminalRef.current.offsetWidth === 0 || terminalRef.current.offsetHeight === 0) {
    log('warn', 'Fit attempt skipped: terminal container has no dimensions.');
    return false;
  }
  try {
    fitAddonRef.current.fit();
    log('info', `Terminal sized to ${terminalRef.current.offsetWidth}x${terminalRef.current.offsetHeight}`);
    // Send resize after successful fit
    setTimeout(sendResize, 50);
    return true;
  } catch (err) {
    log('error', `Failed to fit terminal: ${err}`);
    return false;
  }
}, [terminal, log, sendResize]);

// Initialization sequence
// 1. Create terminal
const term = new XTerm({/* options */});

// 2. Load addons
const fitAddon = new FitAddon();
fitAddonRef.current = fitAddon;
term.loadAddon(fitAddon);

// 3. Important: Open terminal BEFORE storing in state
term.open(terminalRef.current);

// 4. THEN store terminal in state
setTerminal(term);

// 5. Try to fit after DOM is ready
setTimeout(() => attemptFitAndResize(), 150);
```

### 2. Fixed Terminal Resize

The resize function now properly gets current terminal dimensions:

```typescript
const sendResize = useCallback(() => {
  const currentSocket = socketRef.current;
  if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN || !terminalRef.current) {
    return;
  }
  
  // Only proceed if we have valid dimensions
  if (!terminal || !fitAddonRef.current) return;
  
  try {
    // Get terminal dimensions directly from the terminal instance
    // instead of from options (which are just initial values)
    const cols = terminal.cols;
    const rows = terminal.rows;
    
    if (!cols || !rows) {
      log('warn', 'Invalid terminal dimensions');
      return;
    }
    
    // Send JSON format
    const message = {
      cols: cols,
      rows: rows,
      isCommand: true,
      content: 'resize'
    };
    
    currentSocket.send(JSON.stringify(message));
    log('info', `Sent resize: ${cols}x${rows}`);
  } catch (error) {
    log('error', `Failed to send resize command: ${error}`);
  }
}, [terminal, log]);
```

### 3. Added Binary Message Protocol Support

Added support for binary messages for better compatibility:

```typescript
// In the WebSocket message handler
if (event.data instanceof ArrayBuffer) {
  // Handle binary protocol messages (for backward compatibility)
  const buffer = new Uint8Array(event.data);
  if (buffer.length > 0) {
    // Only handle OUTPUT_MESSAGE type for binary protocol
    const messageType = buffer[0];
    if (messageType === 'o'.charCodeAt(0) && buffer.length > 1) {
      const content = new TextDecoder().decode(buffer.slice(1));
      if (terminal) {
        const contentHash = hashContent(content);
        if (!processedContentHashRef.current.has(contentHash)) {
          terminal.write(content);
          processedContentHashRef.current.add(contentHash);
        }
      }
    } else {
      log('warn', `Received unsupported binary message type: ${String.fromCharCode(messageType)}`);
    }
  }
}
```

## Server-Side Recommendations

For the server, we recommend adding panic recovery to all goroutines:

```go
go func() {
  // Add panic recovery to prevent crashes
  defer func() {
    if r := recover(); r != nil {
      log.FileOnlyErrorLog.Printf("WebSocket: PANIC in goroutine: %v\n%s", 
        r, debug.Stack())
      // Signal other goroutines to terminate
      cancel()
    }
  }()
  
  // Original goroutine code...
}()
```

These changes work together with the previous fixes in WEBSOCKET_FIX.md to provide a more robust WebSocket connection between the client and server.

## Implementation

To implement these fixes:
1. Apply the changes to Terminal.tsx as outlined in this document
2. Use mutex when writing to WebSocket connections to ensure thread safety
3. Add proper error handling and recovery to all goroutines that handle WebSocket connections