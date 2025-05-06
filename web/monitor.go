package web

import (
	"bytes"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web/types"
	"crypto/sha256"
	"sync"
	"time"
)

// Ensure TerminalMonitor implements TerminalMonitorInterface
var _ types.TerminalMonitorInterface = (*TerminalMonitor)(nil)

// TerminalMonitor watches for changes in terminal output.
type TerminalMonitor struct {
	storage      *session.Storage
	contentMap   map[string]string
	hashMap      map[string][]byte
	subscribers  map[string][]chan types.TerminalUpdate
	mutex        sync.RWMutex
	ticker       *time.Ticker
	done         chan struct{}
}

// NewTerminalMonitor creates a new terminal monitor.
func NewTerminalMonitor(storage *session.Storage) *TerminalMonitor {
	return &TerminalMonitor{
		storage:     storage,
		contentMap:  make(map[string]string),
		hashMap:     make(map[string][]byte),
		subscribers: make(map[string][]chan types.TerminalUpdate),
		done:        make(chan struct{}),
	}
}

// Start begins monitoring terminal output.
func (tm *TerminalMonitor) Start() {
	tm.ticker = time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-tm.ticker.C:
				tm.checkForUpdates()
			case <-tm.done:
				return
			}
		}
	}()
}

// Stop ends the monitoring.
func (tm *TerminalMonitor) Stop() {
	if tm.ticker != nil {
		tm.ticker.Stop()
	}
	close(tm.done)
	
	// Close all subscriber channels
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	for _, subscribers := range tm.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
	}
	tm.subscribers = make(map[string][]chan types.TerminalUpdate)
}

// Subscribe registers a channel to receive updates for an instance.
func (tm *TerminalMonitor) Subscribe(instanceTitle string) chan types.TerminalUpdate {
	updates := make(chan types.TerminalUpdate, 10)
	
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	tm.subscribers[instanceTitle] = append(tm.subscribers[instanceTitle], updates)
	
	// Send initial content if available
	content, exists := tm.contentMap[instanceTitle]
	if exists {
		select {
		case updates <- types.TerminalUpdate{
			InstanceTitle: instanceTitle,
			Content:       content,
			Timestamp:     time.Now(),
			Status:        "current",
		}:
		default:
		}
	}
	
	return updates
}

// Unsubscribe removes a channel from receiving updates.
func (tm *TerminalMonitor) Unsubscribe(instanceTitle string, ch chan types.TerminalUpdate) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	subs, exists := tm.subscribers[instanceTitle]
	if !exists {
		return
	}
	
	for i, sub := range subs {
		if sub == ch {
			// Remove this subscriber
			tm.subscribers[instanceTitle] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

// GetContent returns the current content for an instance.
func (tm *TerminalMonitor) GetContent(instanceTitle string) (string, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	content, exists := tm.contentMap[instanceTitle]
	return content, exists
}

// Done returns a channel that is closed when the monitor stops.
func (tm *TerminalMonitor) Done() <-chan struct{} {
	return tm.done
}

// checkForUpdates polls for changes in terminal output.
func (tm *TerminalMonitor) checkForUpdates() {
	instances, err := tm.storage.LoadInstances()
	if err != nil {
		log.ErrorLog.Printf("Error loading instances: %v", err)
		return
	}
	
	for _, instance := range instances {
		if !instance.Started() || instance.Paused() {
			continue
		}
		
		content, err := instance.Preview()
		if err != nil {
			log.ErrorLog.Printf("Error capturing content for %s: %v", instance.Title, err)
			continue
		}
		
		// Calculate hash
		hasher := sha256.New()
		hasher.Write([]byte(content))
		newHash := hasher.Sum(nil)
		
		tm.mutex.Lock()
		oldHash, exists := tm.hashMap[instance.Title]
		hashChanged := !exists || !bytes.Equal(oldHash, newHash)
		
		if hashChanged {
			tm.contentMap[instance.Title] = content
			tm.hashMap[instance.Title] = newHash
			
			// Create update
			update := types.TerminalUpdate{
				InstanceTitle: instance.Title,
				Content:       content,
				Timestamp:     time.Now(),
				Status:        string(instance.Status),
				HasPrompt:     false, // Determine from content if needed
			}
			
			// Get subscribers
			subscribers := tm.subscribers[instance.Title]
			tm.mutex.Unlock()
			
			// Notify subscribers
			for _, sub := range subscribers {
				select {
				case sub <- update:
					// Successfully sent
				default:
					// Channel is full, skip this update
				}
			}
		} else {
			tm.mutex.Unlock()
		}
	}
}