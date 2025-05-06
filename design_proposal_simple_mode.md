# Design Proposal: Simple Mode for Claude Squad

## Overview

Add a new "Simple Mode" (`-s` flag) to Claude Squad that:
1. Launches Claude directly in the current repository directory (no worktree creation)
2. Automatically enables "auto yes" mode
3. Provides a streamlined workflow for quick AI assistant usage

## Motivation

While Claude Squad's worktree-based approach is powerful for managing multiple isolated AI sessions, many users have simpler needs:
- Quickly launch Claude in the current directory
- Work directly on the main branch
- Auto-approve actions to reduce friction

The proposed Simple Mode provides this streamlined experience while maintaining access to the UI and session management capabilities of Claude Squad.

## Implementation Details

### 1. Command Line Flag

Add a new flag in `main.go`:

```go
var simpleMode bool
rootCmd.PersistentFlags().BoolVarP(&simpleMode, "simple", "s", false, "Simple mode: run in current directory with auto-yes")
```

### 2. Instance Model Changes

Modify the `InstanceOptions` struct in `session/instance.go`:

```go
type InstanceOptions struct {
    // Existing fields
    Name        string
    Program     string
    // New field
    InPlace     bool   // Indicates simple mode (no worktree)
}
```

Add corresponding field to the `Instance` struct and update constructors.

### 3. Modified Workflow for Simple Mode

#### A. Instance Creation

When Simple Mode is active:
1. Create a special instance with `InPlace: true`
2. Use the repository's root directory as the working directory
3. Skip branch creation and worktree setup
4. Default name to "simple-session" or timestamp-based name

#### B. Git Integration Bypass

For simple mode instances:
1. Bypass the `GitWorktree.Setup()` method
2. Disable diff tracking functionality
3. Prevent usage of features that depend on worktrees (pause/resume)

#### C. UI Adjustments

1. Detect simple mode instances in the UI
2. Disable or hide irrelevant operations (pause, create PR, etc.)
3. Replace diff view with a message explaining simple mode limitations
4. Show clear visual indicator of simple mode status

### 4. Tmux Session Management

Simple mode still uses tmux for terminal management, but:
1. Session starts in the current directory
2. Auto-yes is enabled by default
3. No git operations are performed automatically

## Edge Cases and Considerations

### A. Multiple Instances

**Issue:** Simple mode could conflict with other Claude Squad instances working on the same directory.

**Solution:**
1. Check if any existing sessions are using the repository
2. Warn user about potential conflicts and offer options:
   - Continue anyway
   - Switch to full mode instead
   - Attach to existing session

### B. Git Changes Management

**Issue:** Simple mode doesn't track changes via separate branches.

**Solution:**
1. Simple mode instances will not show diffs in the UI
2. Changes are made directly to the working directory
3. User is responsible for managing commits/branches manually
4. Add a warning when starting simple mode about manual git management

### C. Persistence

**Issue:** Simple mode instances don't have a dedicated branch for persistence.

**Solution:**
1. Simple mode instances are considered ephemeral
2. They can be killed but not paused/resumed
3. Store minimal metadata for UI display purposes
4. Display clear UI indicators for ephemeral status

### D. Command Conflicts

**Issue:** `-s` might conflict with other flags.

**Solution:**
1. Ensure `-s` works well with other flags like `-p` (program selection)
2. Make auto-yes implied by simple mode, but allow explicit override
3. Add validation to prevent incompatible flag combinations

## User Experience

### Example Workflows

**Quick Session:**
```bash
# Launch Claude in current directory with auto-yes
cs -s
```

**Quick Session with Specific Program:**
```bash
# Launch specific AI assistant in simple mode
cs -s -p aider
```

### UI Indicators

1. List view shows "simple" badge next to instance
2. Disabled operations are grayed out
3. Info message explains limitations of simple mode
4. Clear visual distinction between simple and full mode instances

## Implementation Plan

1. Add the command line flag
2. Modify session and instance models
3. Update instance creation workflow
4. Add simple mode detection in UI components
5. Add user warnings for edge cases
6. Update help documentation

## Testing Strategy

1. Test simple mode launch with various programs
2. Verify behavior with existing repositories
3. Test interaction between simple and full mode instances
4. Ensure UI properly indicates simple mode status
5. Verify that git operations behave as expected

## Documentation Updates

1. Add simple mode to README
2. Update help text in application
3. Add examples to CONTRIBUTING.md
4. Update CLAUDE.md with simple mode details