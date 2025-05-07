# Complementary Web UI Strategy

## Core Concept
The web UI should serve as a complementary view into an active terminal session, not replace it. The terminal remains the primary interaction point.

## Key Requirements

1. **Terminal Primacy**: 
   - Terminal must continue to function normally
   - Terminal interaction remains the primary input method
   - No degradation of terminal rendering or experience

2. **Web as Observer**:
   - Web UI provides read-only view of terminal state
   - Web UI can show additional metadata not visible in terminal
   - No critical functions depend on web UI

## Implementation Fixes

### Terminal Rendering Issues

1. **PTY Handling**: 
   - Keep proper PTY setup in all cases
   - Never replace PTY with pipes
   - Ensure terminal state is properly managed

2. **Logging Separation**:
   - Never disable console logging when terminal is active
   - Create separate log channels for web server
   - Ensure logs don't corrupt terminal UI

3. **Session Management**:
   - Terminal session must be created first
   - Web UI attaches to existing sessions
   - No auto-sending of empty prompts

### Web Integration

1. **Safe Terminal Monitoring**:
   - Use file-based monitoring rather than intercepting terminal output
   - Web server reads terminal content from tmux, doesn't inject into it
   - Web server runs in background thread, not main UI thread

2. **Clean API Layer**:
   - Create clear API boundary between terminal and web
   - Web reads state through storage and safe APIs
   - Web doesn't directly manipulate terminal state

## Specific Technical Recommendations

1. Keep original terminal rendering code intact
2. Move web server to separate goroutine that doesn't affect terminal
3. Use tmux capture-pane for reading terminal state instead of intercepting
4. Create proper API in storage layer for web UI to read session data
5. Maintain separate logging channels for terminal and web

By implementing the web UI as a true complement to the terminal, we preserve the core terminal experience while adding web visibility.