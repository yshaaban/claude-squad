# CLAUDE.md

This file provides critical guidance for working with the Claude Squad codebase.



```bash
# Build and install
go build -o ~/.local/bin/cs

# Run with Simple Mode (recommended for direct use)
cs -s

# Run with file logging for debugging
cs --log-to-file
```

## Critical Code Locations

1. **Process Termination**: `/app/app.go:handleQuit()` - Ensures Claude processes are killed on exit

2. **PTY Handling**: `/session/tmux/tmux.go:Detach()` - Fragile terminal state handling

3. **Simple Mode Storage**: `/app/app.go:newHome()` - Handles stale instance detection and cleanup

4. **Error Handling**: `/session/tmux/tmux_unix.go` - Critical signal handling for SIGWINCH

## Dangerous Areas to Modify

1. **Terminal State**: Any code that directly interfaces with the terminal (pty, tmux)

2. **Process Management**: Code that starts/stops Claude processes, especially in Simple Mode

3. **Path Resolution**: Code that handles file paths or git repository detection

4. **Error Propagation**: Never replace error returns with panics or os.Exit()

5. **Storage Management**: Code that loads/saves instance data, especially the DeleteInstance() method

6. **WebSocket Communication**: Client-server protocol must be aligned between Terminal.tsx and server handlers. Ensure protocols match for ping/pong heartbeats, binary vs. JSON messages, and command formats.

## Recovery Procedures

1. **Broken Terminal**: If a terminal is left in a broken state, run `reset` command in that terminal

2. **Zombie Claude Processes**: Use `ps | grep claude` to identify and kill stray Claude processes

3. **Stale Instances**: Run `cs reset` to clean up all instances and tmux sessions when in doubt

4. **Debugging Issues**: Run with `cs --log-to-file` to capture debug info to `/tmp/claudesquad.log`

5. **WebSocket Connection Issues**: For debugging WebSocket connection problems, check browser console logs for missed heartbeats or protocol errors, and server logs for "broken pipe" errors.

## Effective Tools for Debugging

1. **Gemini Analysis**: The Gemini tools (`mcp__gemsuite-mcp__gemini_ultrathink`) are extremely valuable for complex code analysis across multiple files. Use them for:
   - Debugging complex interaction issues
   - Finding root causes across multiple components
   You simply need to pass a concise description of the problem and the list of all the relevant files, including log files, don't inline the content of the files in the request, it is automatically handled. 

2. **Log File Examination**: For WebSocket or terminal issues, use log files with `cs --log-to-file` and grep for critical errors:
   ```bash
   cat /tmp/claudesquad.log | grep -i "WebSocket" | grep -i "error"
   ```
