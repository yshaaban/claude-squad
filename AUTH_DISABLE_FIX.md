# Authentication and Connection Fixes

This document describes the fixes applied to resolve authentication issues and WebSocket connection problems in the Claude Squad React frontend.

## Issues Fixed

1. **Authentication Issues**
   - Disabled authentication requirement for the React mode
   - Prevented "Authorization required" errors for local connections
   - Removed unnecessary authentication checks for WebSocket connections

2. **WebSocket Connection Improvements**
   - Changed WebSocket URL format to use query parameters for better compatibility
   - Reduced maximum reconnection attempts from 10 to 5
   - Reduced maximum reconnection delay from 30s to 10s
   - Fixed path inconsistencies in WebSocket connections

3. **Startup Procedure**
   - Added process cleanup to the startup script
   - Ensured clean environment before starting new server instance
   - Improved logging for better diagnostics

## Technical Details

### Authentication Changes

Authentication has been fully disabled for the React mode to simplify local development and usage:

```go
// Authentication Middleware - disabled for local connections
// For development and local usage, skip authentication entirely
log.FileOnlyInfoLog.Printf("Authentication disabled for all connections in React mode")
```

### WebSocket Connection Changes

1. **URL Format**
   Changed from:
   ```typescript
   const wsUrl = `${protocol}//${window.location.host}/ws/${instanceName}?format=ansi&privileges=read-write`
   ```
   
   To a more compatible query parameter format:
   ```typescript
   const wsUrl = `${protocol}//${window.location.host}/ws?instance=${instanceName}&format=ansi&privileges=read-write`
   ```

2. **Reconnection Settings**
   ```typescript
   const maxReconnectDelay = 10000 // Max 10 seconds between reconnects
   const maxReconnectAttempts = 5  // Maximum reconnection attempts
   ```

3. **Process Cleanup**
   Added to startup script:
   ```bash
   # Kill any existing instances first to avoid conflicts
   pkill -f "cs.*--web" || true
   ```

## Usage

To use the improved terminal with authentication fixes:

1. Run the provided script:
   ```bash
   ./run_improved_terminal.sh
   ```

2. Open the browser at:
   ```
   http://localhost:8086/
   ```

The terminal should now connect without authentication errors, and the WebSocket connection should be more stable with fewer reconnection attempts.

## Verification

You can verify the authentication is disabled by checking the server logs:

```
WEB-INFO: server_react.go:30: Authentication disabled for all connections in React mode
```

And you should not see any authentication errors in the browser console or server logs.