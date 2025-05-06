# Claude Squad - Architecture

This document provides a detailed overview of the Claude Squad architecture, components, and workflows.

## Project Overview

Claude Squad is a terminal application that manages multiple AI coding assistant instances (Claude Code, OpenAI Codex, Aider, etc.) in separate workspaces. It provides a terminal-based UI for creating, managing, and switching between multiple AI assistant sessions while keeping their work isolated using git worktrees.

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                         Main Package                         │
│                 (CLI commands & initialization)              │
└───────────────────────────────┬─────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                         App Package                          │
│                (Core logic & UI event handling)              │
└───────┬───────────────────────┬──────────────────────┬──────┘
        │                       │                      │
        ▼                       ▼                      ▼
┌──────────────┐       ┌────────────────┐     ┌────────────────┐
│     UI       │       │    Session     │     │    Config      │
│   Package    │◄─────►│    Package     │     │    Package     │
│(Bubble Tea UI)│       │(Instance Mgmt) │     │(App Settings)  │
└──────┬───────┘       └────────┬───────┘     └────────────────┘
       │                        │
       │                        │
┌──────▼───────┐       ┌───────▼────────┐     ┌────────────────┐
│   Overlay    │       │      Git       │     │    Daemon      │
│  Subpackage  │       │   Subpackage   │     │    Package     │
│ (Modal UIs)  │       │ (Worktree Ops) │     │ (Bkg Processes)│
└──────────────┘       └────────────────┘     └────────────────┘
                                │
                                ▼
                       ┌────────────────┐
                       │     Tmux       │
                       │  Subpackage    │
                       │(Terminal Mgmt) │
                       └────────────────┘
```

### 1. Main Package (`main.go`)

- Entry point with Cobra CLI commands
- Handles command line flags and initialization
- Sets up logging and validates git repository
- Passes control to the App package

### 2. App Package (`app/`)

- Core application logic and UI rendering
- Uses Bubble Tea framework for terminal UI
- Manages application state via the `home` struct
- Handles user input events and state transitions
- Implements the main event loop and UI update cycle

### 3. Session Package (`session/`)

Manages AI assistant instances and their lifecycle:

- `Instance` struct represents a running AI assistant
- Status management (Running, Ready, Loading, Paused)
- Lifecycle methods (Start, Pause, Resume, Kill)
- `Storage` component for persistence of instance data

#### 3.1 Git Subpackage (`session/git/`)

- Manages git worktrees for isolated workspace environments
- `GitWorktree` struct handles git operations for a session
- Creates/removes worktrees while preserving branches
- Handles commits, diffs, and branch management

#### 3.2 Tmux Subpackage (`session/tmux/`)

- Manages terminal sessions using tmux
- Creates and attaches to tmux sessions
- Runs the AI assistant program in the session
- Captures session output for UI preview

### 4. UI Package (`ui/`)

- Implements terminal UI components using Bubble Tea/Lip Gloss
- `List` component displays available instances
- `TabbedWindow` container with Preview and Diff tabs
- `Menu` shows available commands and keybindings
- `ErrBox` displays error messages

#### 4.1 Overlay Subpackage (`ui/overlay/`)

- Implements modal overlays for text input and information
- Text input for naming instances and entering commands
- Text display for help information

### 5. Config Package (`config/`)

- Manages application configuration and state
- Defines settings like DefaultProgram and AutoYes mode
- Handles loading/saving configuration
- Maintains persistent application state

### 6. Daemon Package (`daemon/`)

- Handles background processes
- Platform-specific implementations (Unix/Windows)
- Manages daemon lifecycle (start, stop)

## Workflows

### Instance Lifecycle

#### Standard Mode

1. **Creation**:
   - User creates a new instance with a title
   - New git branch created (prefix: "session/")
   - Worktree set up from current HEAD
   - Tmux session started in worktree
   - AI program launched

#### Simple Mode

1. **Creation**:
   - User runs with the `-s` flag
   - Instance created in the current directory (no worktree)
   - Tmux session started in current directory
   - AI program launched with auto-yes enabled
   - Prompt dialog opened immediately

2. **Interaction**:
   - User can view instance status in list view
   - Preview pane shows session content
   - User can attach to session for direct control
   - Diff view shows changes in worktree

3. **Pausing**:
   - Changes committed to branch
   - Tmux session closed
   - Worktree removed (branch preserved)

4. **Resuming**:
   - Worktree recreated from branch
   - New tmux session started
   - Program relaunched

5. **Termination (Standard Mode)**:
   - Tmux session closed
   - Worktree and branch removed
   - Instance remains in storage for future use

6. **Termination (Simple Mode)**:
   - Tmux session closed and Claude process terminated
   - Instance removed from storage
   - No branch/worktree cleanup needed

## Design Patterns

- **MVC**: Separation of model (instances), view (UI), and controller (event handlers)
- **Event-Driven**: Bubble Tea framework with message-based updates
- **Factory Pattern**: Constructor functions for creating objects
- **Command Pattern**: Cobra CLI commands and Tea commands
- **Observer**: Status monitoring for tmux output changes

## Cross-Platform Support

- Platform-specific implementations for Unix/Windows
- Daemon management tailored to each platform
- Path handling with filepath package for compatibility

## Key Interfaces

### Session Management

```go
// Creates a new AI assistant instance
NewInstance(opts InstanceOptions) 

// Lifecycle methods
Instance.Start(firstTimeSetup bool)
Instance.Pause()
Instance.Resume()
Instance.Kill()
```

### Git Operations

```go
// Creates a new git worktree
NewGitWorktree(path, sessionName string)

// Key operations
GitWorktree.Setup()
GitWorktree.PushChanges(commitMsg string, push bool)
GitWorktree.IsDirty()
```

### UI Components

```go
// Main UI model
home struct in app.go

// Primary UI components
List, TabbedWindow, Menu, ErrBox
```

## Error Handling Principles

1. **Graceful Degradation**:
   - Functions return errors rather than panicking
   - Resources are properly cleaned up on error
   - Fallback mechanisms exist for critical components
   
2. **Defensive Programming**:
   - Nil checks before accessing objects
   - Error checking for all external operations
   - Recovery mechanisms for unstable operations (PTY, tmux)
   
3. **User Feedback**:
   - Errors displayed in UI via error box
   - Informational messages for important operations
   - Detailed logging with optional file output

## Logging Architecture

1. **Multi-level Logging**:
   - InfoLog for general information
   - WarningLog for potential issues
   - ErrorLog for critical problems
   
2. **Configurable Destinations**:
   - Console output by default (stdout/stderr)
   - Optional file logging with --log-to-file flag
   - Multi-writer approach when both enabled

## Conclusion

Claude Squad uses a modular architecture with clear separation of concerns to manage multiple AI coding assistant instances effectively. The combination of git worktrees for isolation and tmux for terminal session management creates a powerful environment for working with multiple AI assistants simultaneously, while the Bubble Tea terminal UI provides an intuitive interface for users.

The Simple Mode feature enhances usability by allowing quick operation in the current directory. Robust error handling and graceful failure recovery improve reliability, especially when running from different directories or encountering terminal state issues.