# Simple Mode Implementation Plan

Based on my analysis of the Claude Squad codebase, here's a detailed implementation plan for adding the simple mode feature (`-s` flag).

## Current Status

I've found that some of the groundwork for the simple mode feature already exists in the codebase:

1. The `-s` flag is already defined in `main.go` (lines 148-149)
2. The `simpleModeFlag` boolean is declared in `main.go` (line 24)
3. The flag is passed to the `app.Run()` function (line 74)
4. The `home` struct in `app.go` already has a `simpleMode` field (line 50)
5. The `InstanceOptions` struct in `instance.go` already has an `InPlace` field (line 156)
6. The `Instance` struct already has an `InPlace` field (line 55)
7. The `Start()` method in `instance.go` already has special handling for simple mode (lines 214-221)
8. The `UpdateDiffStats()` method already handles simple mode differently (lines 497-501)
9. The `Pause()` method already prevents pausing simple mode instances (lines 391-393)

## Changes Needed

Despite these existing implementations, there are still a few components that need adjustment:

### 1. App Initialization Logic

The `newHome()` function in `app.go` (lines 116-148) already contains code to create and start a simple mode instance, but needs refinement:

- The instance name should be improved (currently just uses the directory name)
- Error handling could be enhanced
- A clear indicator of simple mode in the UI is missing

### 2. UI Components

The UI needs to clearly indicate when an instance is running in simple mode:

- Add a visual indicator in the list view
- Disable inappropriate menu items (pause, create PR, etc.)
- Add messaging in the diff view to explain limitations
- Update help screens to include simple mode information

### 3. Instance Management

Some instance management functions need adjustments:

- `handleKeyPress()` in `app.go` should disable certain operations for simple mode
- Better error messages for operations that aren't supported in simple mode

### 4. Documentation

Update documentation to reflect the new simple mode:

- Update README.md
- Update help text
- Update CLAUDE.md

## Implementation Steps

### Step 1: Enhance UI Indication

1. Modify `ui/list.go` to show a "simple" badge next to simple mode instances
2. Update `ui/menu.go` to disable inappropriate options for simple mode

### Step 2: Improve Instance Creation in Simple Mode

Improve the simple mode instance creation in `newHome()`:

```go
// Create a default instance name based on timestamp
instanceName := fmt.Sprintf("simple-%s", time.Now().Format("20060102-150405"))

// Create a new instance that runs in-place (no worktree)
instance, err := session.NewInstance(session.InstanceOptions{
    Title:     instanceName,
    Path:      currentDir,
    Program:   program,
    AutoYes:   true,
    InPlace:   true,
})
```

### Step 3: Update Diff View for Simple Mode

Modify `ui/diff.go` to display a message when viewing a simple mode instance:

```go
if instance != nil && instance.InPlace {
    return "Simple mode active: Changes are made directly to your working directory.\n" +
           "Git diff tracking is disabled in simple mode."
}
```

### Step 4: Update Help Screens

Add simple mode instructions to help screens in `app/help.go`

### Step 5: Documentation Updates

Update the README and CLAUDE.md to include information about simple mode.

## Edge Case Handling

### Multiple Simple Mode Instances

Problem: Users might try to create multiple simple mode instances on the same repository.

Solution: Add a warning when attempting to create a second simple mode instance:

```go
// Check if a simple mode instance already exists for this repository
for _, existingInstance := range instances {
    if existingInstance.InPlace && filepath.Clean(existingInstance.Path) == filepath.Clean(currentDir) {
        // Show warning and offer to reuse existing instance
    }
}
```

### Switching Branches with Simple Mode Active

Problem: User might switch git branches while a simple mode instance is running.

Solution: Add a warning in the UI about branch changes not being handled by Claude Squad in simple mode.

### Git Operations

Problem: Simple mode doesn't track diffs or provide branch isolation.

Solution:
- Clear messaging in the UI about limitations
- Disable git-related functionality in the menu for simple mode instances

## Testing Plan

1. Test launching with `-s` flag
2. Verify program is running in current directory
3. Test auto-yes functionality
4. Verify UI clearly indicates simple mode
5. Test UI features with simple mode
6. Test interaction between simple and regular mode instances
7. Verify simple mode instance is properly cleaned up when terminated