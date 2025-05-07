# Terminal Logging Fix

## Overview

We've implemented a balanced approach to terminal logging that addresses two 
competing concerns:

1. Preventing logs from corrupting the terminal UI during normal operations
2. Ensuring critical/fatal errors remain visible to the user

## Implemented Strategy

### Non-Critical Logs Moved to File-Only

The following types of logs have been redirected to file-only output:

1. **Debug/Informational Messages**
   - "nuked first stdin" during attach
   - Window resizing status updates
   - Normal operational logs

2. **Common Errors**
   - Failed window size updates
   - Terminal input capture errors
   - Non-fatal recovery messages

### Critical Errors Remain Visible

The following types of errors still appear in the terminal:

1. **Session Existence Issues**
   - "Tmux session doesn't exist during restore"
   - Critical for users to understand why a session failed

2. **Session Recovery Problems**
   - Errors during tmux session restoration
   - Failed pipe creation for recovery operations

3. **Other Fatal Issues**
   - Any errors that would prevent the application from functioning normally

## Benefits of This Approach

- Terminal UI stays clean during normal operations
- Critical errors remain immediately visible to help users diagnose problems
- All logs are still captured to file for detailed troubleshooting
- Better user experience during attach/detach operations

## Implementation Details

Changed non-critical logs from:
```go
log.InfoLog.Printf("...")
log.ErrorLog.Printf("...")
```

To:
```go
log.FileOnlyInfoLog.Printf("...")
log.FileOnlyErrorLog.Printf("...")
```

While keeping critical error logs as:
```go
log.ErrorLog.Printf("...")
```