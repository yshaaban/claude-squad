# CLAUDE.md

This file provides critical guidance for working with the Claude Squad codebase.

## CRITICAL GUIDELINES

### Terminal Integration (HIGH RISK)

1. **NEVER USE PANICS**: Panics leave the terminal in a corrupted state. Always use error returns instead.

2. **NO OS.EXIT() CALLS**: Never directly call os.Exit() in library functions. Return errors to the caller.

3. **PTY HANDLING**: The pty library integration is fragile. Always handle errors gracefully for terminal operations.

4. **TMUX SESSION MANAGEMENT**: Always clean up tmux sessions properly. Detached tmux sessions consume resources.

5. **TERMINAL STATE CHANGES**: Be extremely careful with term.SetRaw() or any code that changes terminal state.

### Process Management

1. **CLEANUP ON EXIT**: In Simple Mode, kill Claude processes when the app exits. Call proper cleanup in handleQuit().

2. **STORAGE PURGING**: Remove Simple Mode instances from storage when terminated or app will show stale instances.

3. **PROPER SIGNAL HANDLING**: Always check returns from syscalls, especially SIGWINCH in tmux code.

### Cross-Directory Compatibility

1. **ABSOLUTE PATHS**: Always use absolute path resolution. Application can be run from any directory.

2. **GIT REPO DETECTION**: Git repo detection must traverse up directories to find .git. Never assume current dir.

### Logging Guidelines

1. **NO DEFAULT FILE LOGGING**: Logging to files is disabled by default. Use --log-to-file only for debugging.

## Key Commands

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

## Recovery Procedures

1. **Broken Terminal**: If a terminal is left in a broken state, run `reset` command in that terminal

2. **Zombie Claude Processes**: Use `ps | grep claude` to identify and kill stray Claude processes

3. **Stale Instances**: Run `cs reset` to clean up all instances and tmux sessions when in doubt

4. **Debugging Issues**: Run with `cs --log-to-file` to capture debug info to `/tmp/claudesquad.log`