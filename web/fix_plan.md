# Web Server Integration Fix Plan

## Identified Issues

1. **Console Logging in Web Mode**:
   - Main app is a full-screen terminal app, but logs are being written to the console in web mode
   - This disrupts the terminal UI

2. **Web Server Integration Issues**:
   - Web server starts when `-s -web` flags are used but introduces regressions to the main app
   - Multiple log messages interfere with the terminal UI

3. **Logging Configuration Issues**:
   - The logging system allows for file-only logging, but this isn't properly enabled in web mode
   - `FileOnlyInfoLog` and other file-only loggers are properly set up but not consistently used

4. **Terminal Status Check Issues**:
   - `web/monitor.go` uses `HasPrompt` instead of `HasUpdated`
   - Direct access to unexported field `tmuxSession` instead of using proper accessor methods

5. **Startup Flow Issues**:
   - In web mode, a different startup flow is causing disruptions to the terminal UI
   - Automatic empty prompt is being sent which might be causing unexpected behavior

## Required Changes

### 1. Fix Logging Issues

- Modify `main.go`:
  - Enable file-based logging and disable console logging when using web mode

```go
// In main.go - when web monitoring is enabled
if webMonitoringFlag {
    log.EnableFileLogging()
    log.SetConsoleLoggingDisabled(true)
}
```

- Update `app/app.go` and `app/web.go` to use file-only loggers:
  - Replace `log.InfoLog` with `log.FileOnlyInfoLog` for debug messages
  - Only use console logging for critical errors or at end of execution

### 2. Fix Integration Points in Web Server

- Fix `web/monitor.go`:
  - Replace all `instance.HasPrompt()` calls with `instance.HasUpdated()`
  - Use proper accessor methods instead of direct `tmuxSession` access
  - Ensure all debugging and status messages use `FileOnlyInfoLog` or `FileOnlyErrorLog`

- Update `web/server.go`:
  - Remove unnecessary console logging
  - Use file-only loggers for debugging information

### 3. Improve Simple Mode Detection with Web Server

- Update `app/app.go`:
  - Modify the Simple Mode + Web Server flow to avoid disrupting the terminal UI
  - Fix automatic empty prompt behavior to avoid unexpected results

### 4. Add Fallback and Recovery Options

- Add better fallback options when errors occur
- Ensure terminal state is properly managed 

### 5. Update Error Handling

- Add better error handling to avoid disrupting the terminal UI
- Use file-only logging for non-critical errors

## Implementation Plan

1. First fix the logging configuration in `main.go`
2. Update all web server components to use file-only loggers
3. Fix all direct tmuxSession access and HasPrompt references
4. Fix startup flow for simple mode with web server
5. Test changes to ensure web server works without disrupting the main app

By implementing these changes, `cs -s` and `cs -s -web` should behave identically except for the web server starting in the latter case.