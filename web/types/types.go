// Package types provides shared data structures for web monitoring components.
package types

import (
	"time"
)

// TerminalUpdate contains information about terminal content updates.
type TerminalUpdate struct {
	InstanceTitle string    `json:"instance_title"`
	Content       string    `json:"content"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
	HasPrompt     bool      `json:"has_prompt"`
}

// TerminalInput represents input sent to a terminal from a client.
type TerminalInput struct {
	InstanceTitle string      `json:"instance_title"`
	Content       string      `json:"content"`
	IsCommand     bool        `json:"is_command"` // True if this is a command like resize
	Cols          interface{} `json:"cols,omitempty"`
	Rows          interface{} `json:"rows,omitempty"`
}

// TaskItem represents a single task item from Claude's todo list.
type TaskItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"` // "pending", "in_progress", "completed", "cancelled"
	Priority string `json:"priority"` // "high", "medium", "low"
}

// TerminalMonitorInterface defines the interface for terminal monitoring components.
type TerminalMonitorInterface interface {
	// Subscribe returns a channel for receiving terminal updates for an instance.
	Subscribe(instanceTitle string) chan TerminalUpdate
	
	// Unsubscribe removes a channel from receiving updates.
	Unsubscribe(instanceTitle string, ch chan TerminalUpdate)
	
	// GetContent returns the current content for an instance.
	GetContent(instanceTitle string) (string, bool)
	
	// SendInput sends input to the terminal for an instance.
	SendInput(instanceTitle string, input string) error
	
	// GetTasks returns the tasks associated with an instance.
	GetTasks(instanceTitle string) ([]TaskItem, error)
	
	// ResizeTerminal resizes the terminal for an instance.
	ResizeTerminal(instanceTitle string, cols, rows int) error
	
	// Done returns a channel that is closed when the monitor stops.
	Done() <-chan struct{}
}