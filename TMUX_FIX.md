# Terminal Attach/Detach Fix

## Issue Fixed

The issue with terminal corruption during attach/detach operations has been fixed. Specifically:

1. During the tmux attach process, the code was correctly handling incoming control sequences from the terminal by "nuking" them (discarding them before passing to tmux).

2. However, these operations were being logged directly to `log.InfoLog`, which sends output to stdout. This caused text to appear in the terminal UI during attach/detach operations, corrupting the display.

## Fix Applied

The fix was simple but effective:

1. Changed the logging from `log.InfoLog` to `log.FileOnlyInfoLog` for the "nuked first stdin" message:

```go
// Before:
log.InfoLog.Printf("nuked first stdin: %s", buf[:nr])

// After:
log.FileOnlyInfoLog.Printf("nuked first stdin: %s", buf[:nr])
```

This change ensures that the log message is only written to the log file and not displayed on the console, preventing terminal corruption.

## Benefits

With this fix:

1. Terminal attach/detach operations remain clean with no visible log messages
2. The debugging information is still available in the log file if needed
3. The terminal UI maintains its intended appearance throughout operations

This complements our earlier fixes that removed the NoTTY mode, helping ensure a consistent and clean terminal experience.