# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Squad is a terminal application that manages multiple AI coding assistant instances (Claude Code, OpenAI Codex, Aider, etc.) in separate workspaces. It provides a terminal-based UI for creating, managing, and switching between multiple AI assistant sessions while keeping their work isolated.

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