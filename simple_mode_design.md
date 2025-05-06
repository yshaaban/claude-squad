# Simple Mode Design Document

## Overview

This document outlines the design and implementation of a new "Simple Mode" feature for Claude Squad, activated via the `-s` flag. Simple Mode provides a streamlined workflow for users who want to quickly start a Claude session in their current repository without the overhead of creating separate git worktrees.

## Goals

1. Provide a simplified workflow for quick Claude usage
2. Reduce UI complexity for common use cases
3. Enable immediate prompting with auto-approval
4. Maintain all core functionality while simplifying the experience

## User Experience

### Current Workflow

The current Claude Squad workflow requires several steps:
1. Launch Claude Squad
2. Create a new instance
3. Enter a name for the instance
4. Wait for git worktree setup
5. Enter a prompt (optional)
6. Interact with Claude

### Simple Mode Workflow

The Simple Mode workflow is streamlined:
1. Launch Claude Squad with `-s` flag
2. Enter a prompt immediately
3. Interact with Claude with auto-approval enabled

### Command Line Interface

```bash
# Launch Claude Squad in Simple Mode
cs -s

# Launch with specific program
cs -s -p "aider"
```

## Technical Design

### 1. Command Line Flag

The `-s` flag is already implemented in `main.go`:

```go
rootCmd.Flags().BoolVarP(&simpleModeFlag, "simple", "s", false,
    "Run Claude in the current repository directory (no worktree) with auto-yes enabled")
```

### 2. Instance Model

The `Instance` and `InstanceOptions` structs already have an `InPlace` field to support Simple Mode:

```go
type InstanceOptions struct {
    // Existing fields
    // ...
    InPlace bool // Indicates simple mode (no worktree)
}

type Instance struct {
    // Existing fields
    // ...
    InPlace bool // Indicates simple mode instance
}
```

### 3. Application Initialization

In `app.go`, the `newHome` function needs enhanced Simple Mode behavior:

```go
func newHome(ctx context.Context, program string, autoYes bool, simpleMode bool) *home {
    // ... existing initialization ...
    
    if simpleMode {
        // Create a simple mode instance with timestamp-based name
        instanceName := fmt.Sprintf("simple-%s", time.Now().Format("20060102-150405"))
        
        // Create instance in current directory
        currentDir, err := os.Getwd()
        if err != nil {
            fmt.Printf("Failed to get current directory: %v\n", err)
            os.Exit(1)
        }
        
        // Create and start the instance
        instance, err := session.NewInstance(session.InstanceOptions{
            Title:     instanceName,
            Path:      currentDir,
            Program:   program,
            AutoYes:   true,
            InPlace:   true,
        })
        if err != nil {
            fmt.Printf("Failed to create instance: %v\n", err)
            os.Exit(1)
        }
        
        // Start the instance
        if err := instance.Start(true); err != nil {
            fmt.Printf("Failed to start instance: %v\n", err)
            os.Exit(1)
        }
        
        // Add instance to list and select it
        h.list.AddInstance(instance)()
        h.list.SetSelectedInstance(0)
        
        // Immediately open prompt dialog
        h.state = statePrompt
        h.menu.SetState(ui.StatePrompt)
        h.textInputOverlay = overlay.NewTextInputOverlay("Enter prompt", "")
    } else {
        // ... existing standard mode code ...
    }
    
    return h
}
```

### 4. UI Adjustments

The UI should be modified to minimize the list view in Simple Mode and maximize screen space for Claude output:

```go
func (m *home) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
    var listWidth, tabsWidth int
    
    if m.simpleMode {
        // Minimal list width in simple mode
        listWidth = int(float32(msg.Width) * 0.1) // 10% width for list
    } else {
        // Standard list width
        listWidth = int(float32(msg.Width) * 0.3) // 30% width for list
    }
    
    tabsWidth = msg.Width - listWidth
    
    // ... rest of sizing logic ...
}

func (m *home) View() string {
    // Option to completely hide list in simple mode
    if m.simpleMode && m.state != statePrompt && m.state != stateHelp {
        // Render view without list
        mainView := lipgloss.JoinVertical(
            lipgloss.Center,
            lipgloss.NewStyle().PaddingTop(1).Render(m.tabbedWindow.String()),
            m.menu.String(),
            m.errBox.String(),
        )
        return mainView
    }
    
    // Standard view with list
    // ... existing view code ...
}
```

### 5. Handling Operations in Simple Mode

Operations that aren't applicable to Simple Mode should be disabled or handled appropriately:

```go
func (m *home) handleKeyPress(msg tea.KeyMsg) (mod tea.Model, cmd tea.Cmd) {
    // ... existing code ...
    
    switch name {
    case keys.KeyPause:
        selected := m.list.GetSelectedInstance()
        if selected == nil {
            return m, nil
        }
        
        // Prevent pausing Simple Mode instances
        if selected.InPlace {
            return m, m.handleError(fmt.Errorf("cannot pause in-place instances (simple mode)"))
        }
        
        // ... existing pause code ...
    
    // ... other operations ...
    }
    
    // ... rest of function ...
}
```

### 6. Git Diff Handling

The `UpdateDiffStats` method already handles Simple Mode correctly:

```go
func (i *Instance) UpdateDiffStats() error {
    // ... existing code ...
    
    if i.InPlace {
        // Simple mode doesn't use worktrees, so no diff stats
        i.diffStats = nil
        return nil
    }
    
    // ... existing diff code ...
}
```

## Implementation Plan

### Phase 1: Core Functionality

1. Update `newHome` to implement improved Simple Mode initialization
2. Add immediate prompt dialog opening in Simple Mode
3. Ensure auto-yes behavior works correctly in Simple Mode

### Phase 2: UI Enhancements

1. Implement minimized list view in Simple Mode
2. Add clear visual indicators for Simple Mode
3. Ensure all menus and commands are correctly enabled/disabled

### Phase 3: Documentation and Testing

1. Update documentation to reflect Simple Mode functionality
2. Add examples to README and help text
3. Comprehensive testing of Simple Mode features

## Edge Cases and Considerations

### Multiple Simple Mode Instances

Simple Mode is designed primarily for single-instance usage. Starting multiple Simple Mode instances on the same repository could lead to conflicts since they all operate directly on the working directory.

Solution: Focus on optimizing for the single-instance case, but ensure the application behaves predictably if multiple instances are created.

### Session Management

Simple Mode instances are ephemeral and not intended for long-term use. They cannot be paused/resumed like regular instances.

Solution: Clearly indicate the ephemeral nature in the UI and provide appropriate error messages for unsupported operations.

### Git Integration

Simple Mode bypasses the worktree isolation system, so all changes are made directly to the current working directory.

Solution: Clear messaging in the UI about how Simple Mode affects git operations, and disable diff views that depend on worktree isolation.

## User Documentation

### Command-Line Help

```
USAGE:
  cs [flags]

FLAGS:
  -p, --program string   Program to run (default "claude-code")
  -y, --autoyes          Automatically accept prompts
  -s, --simple           Simple mode: run in current directory with auto-yes
```

### README Instructions

```markdown
## Simple Mode

For quick Claude sessions in your current directory, use the `-s` flag:

```bash
cs -s
```

This will:
- Launch Claude directly in your current repository
- Enable auto-yes mode
- Skip worktree creation
- Start with a prompt dialog

Simple Mode is ideal for quick questions or tasks where you don't need branch isolation.
```

## Future Enhancements

1. Option to customize Simple Mode behavior (e.g., `-s` with additional flags)
2. Ability to convert a Simple Mode session to a full session with branch isolation
3. Better handling of multiple Simple Mode instances
4. Improved UI options for maximizing Claude output display