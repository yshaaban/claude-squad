# TTY and WebSocket Connection Fixes

This document explains the fixes for two related issues:
1. Terminal corruption when running the application
2. WebSocket connection failures

## Problem Description

When running `./build.sh && ./cs -s --web --web-port 8086 --react`, two issues occur:

1. **Terminal Corruption**: The source terminal becomes "messed up" with errors like:
   ```
   could not open a new TTY: open /dev/tty: device not configured
   Error: could not open a new TTY: open /dev/tty: device not configured
   ```

2. **WebSocket Connection Failures**: The React web UI can't establish WebSocket connections to terminal instances.

## Root Causes

### Terminal Corruption
The primary issue is that the application is trying to open a TTY device but can't access it properly. This happens because:

1. The application modifies terminal settings using the `term.SetRaw()` function but doesn't properly reset them when errors occur.
2. In certain environments (like running through Claude or in CI), there's no proper TTY available.
3. When terminal initialization fails, it doesn't gracefully handle the error, leaving the parent terminal in a corrupted state.

### WebSocket Connection Failures
The WebSocket connections fail because:

1. The client is trying to connect to instances that don't exist or weren't properly created due to the TTY issues.
2. Our previous fixes improved client-side handling but didn't address the server-side terminal initialization issue.
3. When the terminal can't be initialized, there's no valid WebSocket endpoint to connect to.

## Implemented Fixes

### 1. Client-Side WebSocket Improvements
- Added instance existence verification before attempting connection
- Enhanced error handling for missing or invalid instances
- Implemented connection locking to prevent multiple parallel connection attempts
- Improved reconnection backoff logic with jitter

### 2. Terminal State Protection
Created `run_improved_terminal.sh` script that:
- Runs the application detached from the parent terminal using `nohup`
- Redirects output to a log file to avoid terminal corruption
- Sets appropriate environment variables to ensure proper terminal behavior
- Provides a way to safely terminate the application
- Prevents TTY-related errors from corrupting the parent terminal

## Using the Fixed Script

Instead of running `./build.sh && ./cs -s --web --web-port 8086 --react` directly, use:

```bash
./run_improved_terminal.sh
```

This script:
1. Builds the application
2. Runs it detached from the current terminal
3. Provides the URL to access the web UI
4. Logs output to `/tmp/claudesquad_run.log`
5. Saves the process ID for easy termination

## Future Improvements

1. **Server-Side TTY Detection**: Enhance the application to detect when it's running in a non-TTY environment and adjust behavior accordingly.

2. **Graceful Fallback**: Implement a graceful fallback mechanism when a proper TTY isn't available.

3. **Enhanced Error Handling**: Improve error handling in the terminal and tmux code to ensure proper cleanup of terminal state even when errors occur.

4. **Broadcast Instance Status**: Implement a server-side mechanism to broadcast instance deletions or errors to connected clients.

5. **Better Resource Management**: Ensure resources like terminal sessions and tmux panes are properly cleaned up when instances are removed.