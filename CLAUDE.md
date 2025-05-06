# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Squad is a terminal application that manages multiple AI coding assistant instances (Claude Code, OpenAI Codex, Aider, etc.) in separate workspaces. It provides a terminal-based UI for creating, managing, and switching between multiple AI assistant sessions while keeping their work isolated.

## Important Development Guidelines

### Error Handling

1. **Avoid Using Panics**: Never use panics in production code. Instead, return errors and handle them appropriately. Panics can leave the terminal in an inconsistent state.

2. **Don't Use os.Exit()**: Avoid direct calls to os.Exit() in libraries or application code. Return errors instead and let the main function decide how to handle application termination.

3. **Graceful Recovery**: Add recovery mechanisms for operations that might fail, especially when dealing with external systems like tmux.

4. **Signal Handling**: Always check return values from syscall operations and handle potential errors gracefully.

5. **Resource Cleanup**: Always ensure proper resource cleanup (file handles, processes) even when errors occur.

### Cross-Directory Compatibility

1. **Absolute Paths**: Always use absolute paths when working with files and directories to ensure compatibility when running from different directories.

2. **Git Repository Detection**: Use robust git repository detection that handles edge cases when running from subdirectories.

### Simple Mode Considerations

1. **Process Termination**: Ensure all processes (like Claude) are properly terminated when the application exits, especially in Simple Mode.

2. **Storage Cleanup**: Remove Simple Mode instances from storage when they're terminated to prevent stale entries.

3. **Stale Instance Detection**: Check for and clean up stale Simple Mode instances before creating new ones.

### Logging Best Practices

1. **Avoid File Logging by Default**: Don't write logs to files by default; use stdout/stderr instead to avoid cluttering the filesystem.

2. **Optional File Logging**: Provide an opt-in mechanism for file logging when needed for debugging.

3. **Multi-writer Logging**: When writing logs to a file, also send them to stdout/stderr to maintain visibility.

## Build, Test, and Run Commands

### Building

```bash
# Download dependencies
go mod download

# Build the project
go build
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./session/git
```

### Running

```bash
# Run the application
go run main.go

# Run with auto-accept mode
go run main.go -y

# Run with a specific program
go run main.go -p "aider"

# Run in simple mode (current directory, auto-yes, immediate prompt)
go run main.go -s

# Enable file logging (for debugging)
go run main.go --log-to-file
```

### Installing

```bash
# Install from source
go build -o ~/.local/bin/cs

# Install using the installation script
./install.sh
```

## Architecture

### Core Components

1. **Main Package**
   - Entry point with Cobra CLI commands
   - Handles command line flags and initialization

2. **App Package**
   - Core application logic
   - UI rendering and event handling
   - State management

3. **Session Package**
   - Manages AI assistant instances
   - `Instance` struct represents a running AI assistant
   - Handles instance lifecycle (create, start, pause, resume, kill)
   - Sub-packages:
     - `git`: Manages isolated git worktrees
     - `tmux`: Manages terminal sessions

4. **UI Package**
   - Bubble Tea/Lip Gloss UI components
   - List view for instances
   - Tabbed window with preview and diff panes
   - Menu bar and error box

5. **Config Package**
   - Application configuration and state management
   - Default config includes program choice and auto-yes mode

### Workflow

1. When a new instance is created:
   - A new git branch is created and a worktree is set up
   - A tmux session is started in that worktree
   - The specified program (e.g., Claude Code) is launched

2. When an instance is paused:
   - Changes are committed
   - The tmux session is closed
   - The worktree is removed (branch is preserved)

3. When an instance is resumed:
   - The worktree is recreated from the branch
   - A new tmux session is started
   - The program is relaunched

## Key Interfaces

### Session Management

- `NewInstance(opts InstanceOptions)`: Creates a new AI assistant instance
- `Instance.Start(firstTimeSetup bool)`: Initializes and starts an instance
- `Instance.Pause()`: Pauses an instance, preserving its branch
- `Instance.Resume()`: Resumes a paused instance
- `Instance.Kill()`: Terminates an instance and cleans up resources

### Git Operations

- `NewGitWorktree(path, sessionName string)`: Creates a new git worktree
- `GitWorktree.Setup()`: Sets up the worktree
- `GitWorktree.PushChanges(commitMsg string, push bool)`: Commits and optionally pushes changes
- `GitWorktree.IsDirty()`: Checks if there are uncommitted changes

### UI Components

- `home` struct in app.go is the main UI model
- `List`, `TabbedWindow`, `Menu`, `ErrBox` are the primary UI components
- Uses the Bubble Tea event loop for handling user input and rendering

## Cross-Platform Support

- Platform-specific implementations for Unix/Windows
- Daemon package handles background processes
- Uses appropriate path separators and system-specific commands
- Simple mode works across all platforms

## Common Issues and Solutions

### Process Termination

If Claude processes aren't properly terminated when quitting:
- Ensure `handleQuit()` in app.go properly kills instances
- Target only the specific Simple Mode instance to avoid killing unrelated Claude processes
- Call `instance.Kill()` which will properly terminate the tmux session

### Running from Different Directories

If the application crashes when run from a different directory:
- Check all path handling to ensure absolute paths are used
- Add proper error handling instead of panics or `os.Exit()`
- Add debugging statements to identify where failures occur
- Ensure robust git repository detection

### Terminal State Issues

If the terminal is left in a bad state:
- Replace panics with proper error handling, especially in tmux handling
- Add recovery mechanisms for PTY operations
- Ensure proper signal handling for SIGWINCH and other signals
- Add graceful fallbacks when operations fail