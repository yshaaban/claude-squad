package test

import (
	"claude-squad/session"
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

// DoesSessionExist returns true if the session exists.
func (m *MockTmuxSession) DoesSessionExist() bool {
	return m.isAlive
}

// Start starts the tmux session for testing.
func (m *MockTmuxSession) Start(program string, path string) error {
	return nil
}

// Restore restores the tmux session for testing.
func (m *MockTmuxSession) Restore() error {
	return nil
}

// Close ends the mock session.
func (m *MockTmuxSession) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.isAlive = false
	return nil
}

// Detach detaches from the session.
func (m *MockTmuxSession) Detach() error {
	return nil
}

// Attach attaches to the session.
func (m *MockTmuxSession) Attach() (chan struct{}, error) {
	done := make(chan struct{})
	return done, nil
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

// SetDetachedSize sets the terminal size for a detached session.
func (m *MockTmuxSession) SetDetachedSize(cols int, rows int) error {
	return nil
}

// TriggerUpdates simulates terminal activity.
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

// MockInstance is a session.Instance with a mock tmux session.
type MockInstance struct {
	Title     string
	Path      string
	Status    session.Status
	CreatedAt time.Time
	UpdatedAt time.Time
	mockTmux  *MockTmuxSession
}

// NewMockInstance creates a new mock instance.
func NewMockInstance(title, path string) *MockInstance {
	initialContent := fmt.Sprintf("Claude %s Session\n===================\n\nReady to assist you!\n", title)
	mockTmux := NewMockTmuxSession("claudesquad_"+title, initialContent)
	
	return &MockInstance{
		Title:     title,
		Path:      path,
		Status:    session.Running,
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now(),
		mockTmux:  mockTmux,
	}
}

// Core Instance interface methods for testing
func (m *MockInstance) Preview() (string, error) {
	return m.mockTmux.CapturePaneContent()
}

func (m *MockInstance) HasUpdated() (bool, bool) {
	return m.mockTmux.HasUpdated()
}

func (m *MockInstance) SendPrompt(input string) error {
	return m.mockTmux.SendKeys(input)
}

func (m *MockInstance) Started() bool {
	return true
}

func (m *MockInstance) Paused() bool {
	return false
}

func (m *MockInstance) SetPreviewSize(width, height int) error {
	return m.mockTmux.SetDetachedSize(width, height)
}

// SimulateActivity triggers simulated activity on the instance.
func (m *MockInstance) SimulateActivity(duration time.Duration) {
	m.mockTmux.SimulateActivity(duration)
}