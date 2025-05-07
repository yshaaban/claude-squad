package terminal

import (
	"claude-squad/log"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"
	
	"github.com/creack/pty"
)

// TmuxAttachment represents a connection to an existing tmux session
type TmuxAttachment struct {
	sessionName string    // Name of the tmux session to attach to
	cmd         *exec.Cmd // Command for the tmux attach process
	pty         *os.File  // PTY file for the tmux process
	ptyClosed   chan struct{}
	mutex       sync.Mutex
	windowSize  struct {
		rows    int
		columns int
	}
}

// NewTmuxAttachment creates a new attachment to an existing tmux session
func NewTmuxAttachment(sessionName string) (*TmuxAttachment, error) {
	log.FileOnlyInfoLog.Printf("Creating new tmux attachment for session: %s", sessionName)
	
	// Verify the session exists
	if !doesSessionExist(sessionName) {
		return nil, fmt.Errorf("tmux session '%s' does not exist", sessionName)
	}

	// Create and return the attachment (don't connect yet)
	attachment := &TmuxAttachment{
		sessionName: sessionName,
		ptyClosed:   make(chan struct{}),
	}
	
	return attachment, nil
}

// Connect establishes the connection to the tmux session
func (t *TmuxAttachment) Connect() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	// If already connected, do nothing
	if t.pty != nil {
		return nil
	}
	
	log.FileOnlyInfoLog.Printf("Connecting to tmux session: %s", t.sessionName)
	
	// Create the command for attaching to tmux session
	cmd := exec.Command("tmux", "attach-session", "-t", t.sessionName)
	
	// Start the command with a PTY
	pty, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start tmux attach command: %w", err)
	}
	
	// Store the command and PTY
	t.cmd = cmd
	t.pty = pty
	
	// Start a goroutine to wait for the command to exit
	go func() {
		defer func() {
			t.mutex.Lock()
			t.pty.Close()
			t.pty = nil
			t.cmd = nil
			close(t.ptyClosed)
			t.ptyClosed = make(chan struct{})
			t.mutex.Unlock()
			
			log.FileOnlyInfoLog.Printf("Tmux attachment closed for session: %s", t.sessionName)
		}()
		
		// Wait for the command to exit
		t.cmd.Wait()
	}()
	
	// Apply window size if set
	if t.windowSize.rows > 0 && t.windowSize.columns > 0 {
		t.ResizeTerminal(t.windowSize.columns, t.windowSize.rows)
	}
	
	return nil
}

// Read reads from the tmux session
func (t *TmuxAttachment) Read(p []byte) (n int, err error) {
	t.mutex.Lock()
	pty := t.pty
	t.mutex.Unlock()
	
	if pty == nil {
		return 0, fmt.Errorf("not connected to tmux session")
	}
	
	return pty.Read(p)
}

// Write writes to the tmux session
func (t *TmuxAttachment) Write(p []byte) (n int, err error) {
	t.mutex.Lock()
	pty := t.pty
	t.mutex.Unlock()
	
	if pty == nil {
		return 0, fmt.Errorf("not connected to tmux session")
	}
	
	return pty.Write(p)
}

// Close closes the tmux attachment
func (t *TmuxAttachment) Close() error {
	t.mutex.Lock()
	cmd := t.cmd
	pty := t.pty
	ptyClosed := t.ptyClosed
	t.mutex.Unlock()
	
	if cmd == nil || pty == nil {
		return nil
	}
	
	log.FileOnlyInfoLog.Printf("Closing tmux attachment for session: %s", t.sessionName)
	
	// Try to detach from tmux session cleanly
	_, err := pty.Write([]byte{0x01, 'd'}) // Ctrl+A d to detach
	if err != nil {
		log.FileOnlyWarningLog.Printf("Error sending detach command: %v, will force close", err)
	}
	
	// Wait for the process to exit with timeout
	select {
	case <-ptyClosed:
		return nil
	case <-time.After(500 * time.Millisecond):
		// Force kill if it doesn't exit cleanly
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
		}
		
		// Wait again with timeout
		select {
		case <-ptyClosed:
			return nil
		case <-time.After(500 * time.Millisecond):
			// Force kill
			if cmd.Process != nil {
				log.FileOnlyWarningLog.Printf("Force killing attachment process")
				cmd.Process.Kill()
			}
		}
	}
	
	return nil
}

// ResizeTerminal resizes the terminal window
func (t *TmuxAttachment) ResizeTerminal(columns int, rows int) error {
	// Save the window size for reconnection
	t.windowSize.rows = rows
	t.windowSize.columns = columns
	
	t.mutex.Lock()
	pty := t.pty
	t.mutex.Unlock()
	
	if pty == nil {
		// Size will be applied when connected
		return nil
	}
	
	// Set the terminal window size
	window := struct {
		row uint16
		col uint16
		x   uint16
		y   uint16
	}{
		uint16(rows),
		uint16(columns),
		0,
		0,
	}
	
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		pty.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&window)),
	)
	
	if errno != 0 {
		return errno
	}
	
	return nil
}

// CaptureOutput captures the current content of the tmux pane
func (t *TmuxAttachment) CaptureOutput() (string, error) {
	// Use tmux capture-pane to get the current content
	cmd := exec.Command("tmux", "capture-pane", "-p", "-t", t.sessionName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to capture tmux pane content: %w", err)
	}
	
	return string(output), nil
}

// doesSessionExist checks if a tmux session exists
func doesSessionExist(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()
	return err == nil
}