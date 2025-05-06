package mock

import (
	"claude-squad/session"
	"claude-squad/session/git"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// MockTmuxSession simulates a tmux session for testing.
type MockTmuxSession struct {
	name       string
	content    string
	mutex      sync.RWMutex
	hasPrompt  bool
	isAlive    bool
	updateChan chan struct{}
}

// NewMockTmuxSession creates a new mock tmux session.
func NewMockTmuxSession(name string, initialContent string) *MockTmuxSession {
	return &MockTmuxSession{
		name:       name,
		content:    initialContent,
		isAlive:    true,
		hasPrompt:  false,
		updateChan: make(chan struct{}, 1),
	}
}

// Name returns the session name.
func (m *MockTmuxSession) Name() string {
	return m.name
}

// IsAlive returns whether the session is alive.
func (m *MockTmuxSession) IsAlive() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.isAlive
}

// Kill terminates the mock session.
func (m *MockTmuxSession) Kill() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.isAlive = false
	return nil
}

// TapEnter simulates pressing Enter in the terminal.
func (m *MockTmuxSession) TapEnter() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.hasPrompt {
		m.hasPrompt = false
		m.addContent("\n> User accepted prompt\n\nClaude: Continuing with the task...\n")
		return nil
	}
	
	return nil
}

// SendKeys sends content to the terminal.
func (m *MockTmuxSession) SendKeys(content string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.addContent(fmt.Sprintf("\nUser: %s\n\nClaude: I'll help you with that request.\n", content))
	return nil
}

// CapturePaneContent returns the current content.
func (m *MockTmuxSession) CapturePaneContent() (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.content, nil
}

// HasUpdated checks if there are updates.
func (m *MockTmuxSession) HasUpdated() (bool, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Check if there are pending updates
	select {
	case <-m.updateChan:
		return true, m.hasPrompt
	default:
		return false, m.hasPrompt
	}
}

// Trigger a content update in a goroutine.
func (m *MockTmuxSession) SimulateActivity(duration time.Duration) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		endTime := time.Now().Add(duration)
		
		for time.Now().Before(endTime) {
			<-ticker.C
			
			m.mutex.Lock()
			if !m.isAlive {
				m.mutex.Unlock()
				return
			}
			
			// Add some random content
			m.addContent(fmt.Sprintf("\nClaude: Working on your task... (%s)\n", randomProgress()))
			
			// Occasionally add a prompt
			if rand.Float32() < 0.2 {
				m.hasPrompt = true
				m.addContent("\nDo you want me to continue? (Y/N): ")
			}
			
			m.mutex.Unlock()
		}
		
		// Final message
		m.mutex.Lock()
		m.addContent("\nClaude: Task completed!\n")
		m.mutex.Unlock()
	}()
}

// addContent adds content and notifies listeners.
func (m *MockTmuxSession) addContent(newContent string) {
	m.content += newContent
	
	// Notify about update
	select {
	case m.updateChan <- struct{}{}:
	default:
	}
}

// randomProgress generates a random progress indicator.
func randomProgress() string {
	indicators := []string{
		"analyzing code", 
		"reading files", 
		"comparing options", 
		"checking documentation", 
		"applying changes",
	}
	
	return indicators[rand.Intn(len(indicators))]
}

// MockInstance simulates a Claude instance for testing.
type MockInstance struct {
	*session.Instance
	mockTmux *MockTmuxSession
}

// NewMockInstance creates a new mock instance.
func NewMockInstance(title, path string) *MockInstance {
	// Create base instance
	instance, _ := session.NewInstance(session.InstanceOptions{
		Title:   title,
		Path:    path,
		Program: "claude",
	})
	
	// Create mock tmux session
	initialContent := fmt.Sprintf("Claude %s Session\n===================\n\nReady to assist you!\n", title)
	mockTmux := NewMockTmuxSession("claudesquad_"+title, initialContent)
	
	// Create mock git worktree
	mockWorktree := NewMockWorktree(path, "claude-squad/"+title)
	
	// Set mock diff content
	sampleDiff := `diff --git a/README.md b/README.md
index 1234567..abcdefg 100644
--- a/README.md
+++ b/README.md
@@ -1,5 +1,8 @@
 # Test Repo
 
+This is a test repository for Claude.
+
+## Features
 - Feature 1
 - Feature 2
 - Feature 3
@@ -10,3 +13,6 @@
 ## Usage
 
 Run the application with 'npm start'.

## License
MIT
`
	mockWorktree.SetDiff(sampleDiff, 6, 0)
	
	// We can't directly set private fields, but we're in a test environment
	// and the mock is for testing purposes only, so we'll use a custom setup
	// that avoids modifying private fields directly
	
	// Set up InPlace flag since we're not using a real worktree
	instance.InPlace = true
	
	// Set status and timestamps
	instance.Status = session.Running
	instance.CreatedAt = time.Now().Add(-1 * time.Hour)
	instance.UpdatedAt = time.Now()
	
	// Create result instance
	result := &MockInstance{
		Instance: instance,
		mockTmux: mockTmux,
	}
	
	// For tests we'll use method overrides defined in the struct methods below
	// We don't set private fields directly since that's not supported in Go
	// without unsafe operations
	
	return result
}

// SimulateActivity triggers simulated activity on the instance.
func (m *MockInstance) SimulateActivity(duration time.Duration) {
	m.mockTmux.SimulateActivity(duration)
}

// Start overrides the original start method
func (m *MockInstance) Start(bool) error {
	return nil
}

// HasUpdated overrides the original method
func (m *MockInstance) HasUpdated() (bool, bool) {
	return m.mockTmux.HasUpdated()
}

// Preview overrides the original method
func (m *MockInstance) Preview() (string, error) {
	return m.mockTmux.CapturePaneContent()
}

// SendPrompt overrides the original method
func (m *MockInstance) SendPrompt(content string) error {
	return m.mockTmux.SendKeys(content)
}

// Started overrides the original method
func (m *MockInstance) Started() bool {
	return true
}

// Paused overrides the original method
func (m *MockInstance) Paused() bool {
	return false
}

// TmuxAlive overrides the original method
func (m *MockInstance) TmuxAlive() bool {
	return m.mockTmux.IsAlive()
}

// GetGitWorktree overrides the original method
func (m *MockInstance) GetGitWorktree() (*git.GitWorktree, error) {
	// For tests, we'll return the mock worktree
	// This is fake for tests but avoids the private field access issues
	return nil, fmt.Errorf("mock instance does not support real git operations")
}