# Terminal Resource and WebSocket Fixes

This document explains fixes implemented to address two critical issues:

1. "Insufficient resources" WebSocket errors
2. xterm.js dimensions errors (`Cannot read properties of undefined (reading 'dimensions')`)

## Problem Description

After the initial WebSocket fixes, there were still issues with connections:

1. **WebSocket Resource Exhaustion**:
   - Browser errors: `WebSocket connection failed: Insufficient resources`
   - Multiple parallel connection attempts overwhelming the browser
   - Connections being closed and reopened too rapidly

2. **xterm.js Dimension Errors**:
   - Console errors: `Cannot read properties of undefined (reading 'dimensions')`
   - Terminal not rendering correctly because dimensions weren't properly initialized
   - Race conditions between terminal initialization and WebSocket connections

## Root Causes

### WebSocket Resource Exhaustion
- The browser has a limit on the number of WebSocket connections to the same host
- Previous fixes didn't properly clean up inactive connections
- Rapid reconnection attempts during errors created too many parallel connections
- No connection throttling or rate limiting

### xterm.js Dimension Errors
- The terminal component was trying to use dimensions before they were properly initialized
- Race conditions between DOM updates and terminal rendering
- Insufficient safety checks around terminal dimensions
- No fallback dimension values when calculations failed

## Implemented Fixes

### 1. WebSocket Connection Management
- Added connection throttling with minimum wait time between connection attempts
- Improved cleanup of existing connections before creating new ones
- Enhanced connection locking with proper release in all code paths
- Added timeout handling for connection attempts

```typescript
// Connection throttling implementation
const CONNECTION_WAIT_TIME = 1000; // 1 second
const lastConnectionAttemptRef = useRef<number>(0);

// Inside connectWebSocket:
const now = Date.now();
const timeSinceLastAttempt = now - lastConnectionAttemptRef.current;

if (timeSinceLastAttempt < CONNECTION_WAIT_TIME) {
  const waitTime = CONNECTION_WAIT_TIME - timeSinceLastAttempt;
  log('info', `Throttling connection attempt, waiting ${waitTime}ms before trying again`);
  await new Promise(resolve => setTimeout(resolve, waitTime));
}

// Record connection attempt time
lastConnectionAttemptRef.current = Date.now();
```

### 2. Improved WebSocket Cleanup
- Added robust cleanup of WebSocket resources on component unmount
- Implemented better connection error handling with proper state reset
- Enhanced connection abort mechanism for timed-out connections
- Added proper tracking of connection state

```typescript
// Cleanup existing connections
const cleanupExistingConnections = useCallback(() => {
  if (socket) {
    try {
      log('info', `Closing existing WebSocket for instance ${instanceName}`);
      socket.close(1000, "Cleanup for new connection");
      setSocket(null); // Remove the reference immediately
    } catch (err) {
      log('error', `Error closing existing WebSocket: ${err}`);
    }
  }
}, [socket, instanceName, log]);
```

### 3. Fixed Terminal Dimensions
- Added explicit default dimensions (80x24) as safe fallbacks
- Enhanced dimension initialization with proper null checks
- Added verification of dimensions before using them
- Implemented defensive coding to prevent undefined property access

```typescript
// Fixed default dimensions as safe fallbacks
const DEFAULT_COLS = 80;
const DEFAULT_ROWS = 24;

// Before accessing dimensions, ensure they exist
if (!term.cols || term.cols < 2) term.resize(DEFAULT_COLS, term.rows || DEFAULT_ROWS);
if (!term.rows || term.rows < 2) term.resize(term.cols || DEFAULT_COLS, DEFAULT_ROWS);
```

### 4. Improved Error Handling
- Added comprehensive error catching at all levels
- Enhanced logging to track connection and dimension issues
- Implemented proper async/await error handling
- Added cleanup on errors to prevent resource leaks

## Benefits

1. **Reliability**: Connections are now more reliable with proper state management
2. **Performance**: Reduced browser resource usage by preventing parallel connections
3. **Stability**: Terminal displays correctly even when dimensions cannot be calculated
4. **Resilience**: Component properly handles errors and recovers from failure states

## Usage

No changes are needed to use these improvements. The fixes are integrated into the existing codebase and provide better reliability and stability for the terminal component.

## Future Improvements

1. **Server-Side Connection Limits**: Consider adding server-side connection limits per client
2. **Connection Pooling**: Implement WebSocket connection pooling to reuse connections
3. **Better Dimension Detection**: Improve the terminal dimension detection algorithm
4. **Progressive Enhancement**: Add fallback rendering when WebSocket is unavailable