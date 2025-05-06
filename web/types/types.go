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

// TerminalMonitorInterface defines the interface for terminal monitoring components.
type TerminalMonitorInterface interface {
	// Subscribe returns a channel for receiving terminal updates for an instance.
	Subscribe(instanceTitle string) chan TerminalUpdate
	
	// Unsubscribe removes a channel from receiving updates.
	Unsubscribe(instanceTitle string, ch chan TerminalUpdate)
	
	// GetContent returns the current content for an instance.
	GetContent(instanceTitle string) (string, bool)
	
	// Done returns a channel that is closed when the monitor stops.
	Done() <-chan struct{}
}