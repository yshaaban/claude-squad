//go:build windows

package tmux

import (
	"claude-squad/log"
	"os"
	"time"

	"golang.org/x/term"
)

// monitorWindowSize monitors and handles window resize events while attached.
func (t *TmuxSession) monitorWindowSize() {
	// In noTTY mode, use default size and don't monitor window size
	if t.noTTY {
		// Use default size of 80x24 in noTTY mode
		if err := t.updateWindowSize(80, 24); err != nil {
			log.ErrorLog.Printf("failed to set default window size in noTTY mode: %v", err)
		}
		return
	}

	// Use the current terminal height and width.
	doUpdate := func() {
		cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			log.ErrorLog.Printf("failed to update window size: %v", err)
		} else {
			if err := t.updateWindowSize(cols, rows); err != nil {
				log.ErrorLog.Printf("failed to update window size: %v", err)
			}
		}
	}

	// Do one at the start to set the initial size
	doUpdate()

	// On Windows, we'll just periodically check for window size changes
	// since SIGWINCH is not available
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var lastCols, lastRows int
	lastCols, lastRows, _ = term.GetSize(int(os.Stdin.Fd()))

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		for {
			select {
			case <-t.ctx.Done():
				return
			case <-ticker.C:
				cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
				if err != nil {
					continue
				}
				if cols != lastCols || rows != lastRows {
					lastCols, lastRows = cols, rows
					doUpdate()
				}
			}
		}
	}()
}
