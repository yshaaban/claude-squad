package web

import (
	"bytes"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web/types"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Ensure TerminalMonitor implements TerminalMonitorInterface
var _ types.TerminalMonitorInterface = (*TerminalMonitor)(nil)

// TerminalMonitor watches for changes in terminal output.
type TerminalMonitor struct {
	storage            *session.Storage
	contentMap         map[string]string
	hashMap            map[string][]byte
	monitoredInstances []*session.Instance // Cached list of instances
	subscribers        map[string][]chan types.TerminalUpdate
	taskCache          map[string][]types.TaskItem
	taskCacheTimestamp map[string]time.Time
	mutex              sync.RWMutex
	ticker             *time.Ticker
	done               chan struct{}
	
	// Rate-limited loggers to prevent excessive logging
	inactiveLogger     *log.Every  // Logger for "no active instances" messages
	contentLogger      *log.Every  // Logger for content change messages
	nottyLogger        *log.Every  // Logger for terminal issues
}

// Set this to true to enable detailed debug logging
const debugLogging = false

// Patterns to extract task items from Claude's output
// Primary pattern for explicitly marked tasks like "1. [TODO] Task description"
var taskRegexp = regexp.MustCompile(`(?m)^(\d+)\.\s+\[([\w\s]+)\]\s+(.+)$`)

// Additional patterns for other task formats
var todoRegexp = regexp.MustCompile(`(?m)^(\d+)\.\s+(?:TODO|To-do|To do):\s+(.+)$`)        // For "1. TODO: Task description"
var doneRegexp = regexp.MustCompile(`(?m)^(\d+)\.\s+(?:DONE|Completed|✓):\s+(.+)$`)       // For "1. DONE: Task description" or "1. ✓: Task description"
var progressRegexp = regexp.MustCompile(`(?m)^(\d+)\.\s+(?:IN PROGRESS|WIP|Doing):\s+(.+)$`) // For "1. IN PROGRESS: Task description"

// NewTerminalMonitor creates a new terminal monitor.
func NewTerminalMonitor(storage *session.Storage) *TerminalMonitor {
	return &TerminalMonitor{
		storage:            storage,
		contentMap:         make(map[string]string),
		hashMap:            make(map[string][]byte),
		subscribers:        make(map[string][]chan types.TerminalUpdate),
		taskCache:          make(map[string][]types.TaskItem),
		taskCacheTimestamp: make(map[string]time.Time),
		done:               make(chan struct{}),
	}
}

// Start begins monitoring terminal output.
func (tm *TerminalMonitor) Start() {
	tm.ticker = time.NewTicker(500 * time.Millisecond) // Polling for UI updates
	go func() {
		tm.refreshMonitoredInstances() // Initial load
		
		// Create ticker for refreshing instance list (much less frequent)
		instanceRefreshTicker := time.NewTicker(10 * time.Second)
		defer instanceRefreshTicker.Stop()
		
		for {
			select {
			case <-tm.ticker.C:
				tm.checkForUpdates()
			case <-instanceRefreshTicker.C:
				tm.refreshMonitoredInstances() // Refresh list occasionally
			case <-tm.done:
				return
			}
		}
	}()
}

// refreshMonitoredInstances updates the local cache of instances.
// This is called periodically at a slow rate to detect new instances
// or instances that have been removed.
func (tm *TerminalMonitor) refreshMonitoredInstances() {
	LogWebDebug("MONITOR: Refreshing monitored instances list")
	instances, err := tm.storage.LoadInstances()
	if err != nil {
		log.FileOnlyErrorLog.Printf("MONITOR: Error loading instances for monitoring: %v", err)
		return
	}
	tm.mutex.Lock()
	tm.monitoredInstances = instances
	tm.mutex.Unlock()
	LogWebDebug("MONITOR: Refreshed, now monitoring %d instances", len(instances))
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
	updates := make(chan types.TerminalUpdate, 20) // Increased buffer size
	
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	tm.subscribers[instanceTitle] = append(tm.subscribers[instanceTitle], updates)
	
	// Send initial content if available
	content, exists := tm.contentMap[instanceTitle]
	if exists {
		// Get instance for status
		instances, err := tm.storage.LoadInstances()
		var status string = "current"
		var hasPrompt bool = false
		
		if err == nil {
			for _, instance := range instances {
				if instance.Title == instanceTitle {
					status = string(instance.Status)
					_, hasPrompt = instance.HasUpdated()
					break
				}
			}
		}
		
		select {
		case updates <- types.TerminalUpdate{
			InstanceTitle: instanceTitle,
			Content:       content,
			Timestamp:     time.Now(),
			Status:        status,
			HasPrompt:     hasPrompt,
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
	// Only log detailed debug info if needed, and only to file to avoid UI disruption
	if debugLogging {
		log.FileOnlyInfoLog.Printf("GetContent called for instance %s", instanceTitle)
	}
	
	// First check our cache
	tm.mutex.RLock()
	content, exists := tm.contentMap[instanceTitle]
	contentLen := len(content)
	tm.mutex.RUnlock()
	
	// Special case: Force retry for web mode instances (they might not be in cache yet)
	if !exists && strings.HasPrefix(instanceTitle, "web-") {
		log.FileOnlyInfoLog.Printf("Special handling for web instance %s - forcing content fetch", instanceTitle)
		tm.checkForUpdates() // Force an update check
		
		// Check cache again after update
		tm.mutex.RLock()
		content, exists = tm.contentMap[instanceTitle] 
		contentLen = len(content)
		tm.mutex.RUnlock()
	}
	
	if debugLogging {
		log.FileOnlyInfoLog.Printf("Cache check for %s: exists=%v, content length=%d", 
			instanceTitle, exists, contentLen)
	}
	
	// If we don't have content in our cache or it's empty, try to get it from the instance
	if !exists || content == "" {
		if debugLogging {
			log.FileOnlyInfoLog.Printf("No cached content for %s, fetching from instance", instanceTitle)
		}
		
		// Load all instances
		instances, err := tm.storage.LoadInstances()
		if err != nil {
			log.ErrorLog.Printf("Error loading instances: %v", err)
			return "", false
		}
		
		instanceFound := false
		// Find the instance with matching title
		for _, instance := range instances {
			if instance.Title == instanceTitle {
				instanceFound = true
				if debugLogging {
					log.FileOnlyInfoLog.Printf("Found instance %s, getting preview", instanceTitle)
				}
				
				// Get preview content (with retry for robustness)
				var preview string
				var previewErr error
				
				for retries := 0; retries < 3; retries++ {
					preview, previewErr = instance.Preview()
					if previewErr == nil && preview != "" {
						break
					}
					// Only log retries for actual errors, not empty preview (which is common)
					if previewErr != nil {
						log.WarningLog.Printf("Retry %d: Error getting preview for %s: %v", 
							retries, instanceTitle, previewErr)
					}
					time.Sleep(100 * time.Millisecond)
				}
				
				if previewErr != nil {
					log.ErrorLog.Printf("All retries failed: Error getting preview for %s: %v", 
						instanceTitle, previewErr)
					return "", false
				}
				
				if preview == "" {
					// This is a common case, only log at warning level in debug mode
					if debugLogging {
						log.WarningLog.Printf("Got empty preview for instance %s despite successful call", 
							instanceTitle)
					}
					// Return empty but valid to allow placeholder to be shown
					
					// Update empty cache anyway
					tm.mutex.Lock()
					tm.contentMap[instanceTitle] = preview
					tm.mutex.Unlock()
					
					return "", true
				}
				
				if debugLogging {
					log.FileOnlyInfoLog.Printf("Got preview for %s, length: %d", instanceTitle, len(preview))
				}
				
				// Update our cache
				tm.mutex.Lock()
				tm.contentMap[instanceTitle] = preview
				tm.mutex.Unlock()
				
				return preview, true
			}
		}
		
		// This is a legitimate warning, keep it
		if !instanceFound {
			log.WarningLog.Printf("Instance %s not found in storage", instanceTitle)
		}
		
		return "", false
	}
	
	if debugLogging {
		log.FileOnlyInfoLog.Printf("Returning cached content for %s, length: %d", instanceTitle, len(content))
	}
	return content, exists
}

// SendInput sends input to the terminal for an instance.
func (tm *TerminalMonitor) SendInput(instanceTitle string, input string) error {
	instances, err := tm.storage.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}
	
	for _, instance := range instances {
		if instance.Title == instanceTitle {
			if !instance.Started() || instance.Paused() {
				return fmt.Errorf("instance has no active tmux session")
			}
			
			err := instance.SendPrompt(input)
			if err != nil {
				return fmt.Errorf("failed to send keys to tmux: %w", err)
			}
			return nil
		}
	}
	
	return fmt.Errorf("instance not found: %s", instanceTitle)
}

// ResizeTerminal resizes the terminal for an instance.
func (tm *TerminalMonitor) ResizeTerminal(instanceTitle string, cols, rows int) error {
	instances, err := tm.storage.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}
	
	for _, instance := range instances {
		if instance.Title == instanceTitle {
			if !instance.Started() || instance.Paused() {
				return fmt.Errorf("instance has no active tmux session")
			}
			
			err := instance.SetPreviewSize(cols, rows)
			if err != nil {
				return fmt.Errorf("failed to resize terminal: %w", err)
			}
			return nil
		}
	}
	
	return fmt.Errorf("instance not found: %s", instanceTitle)
}

// GetTasks extracts and returns tasks from Claude's terminal output.
func (tm *TerminalMonitor) GetTasks(instanceTitle string) ([]types.TaskItem, error) {
	// Check if we have a recent cache (less than 5 seconds old)
	tm.mutex.RLock()
	lastUpdate, hasTimestamp := tm.taskCacheTimestamp[instanceTitle]
	if hasTimestamp && time.Since(lastUpdate) < 5*time.Second {
		tasks := tm.taskCache[instanceTitle]
		tm.mutex.RUnlock()
		return tasks, nil
	}
	tm.mutex.RUnlock()
	
	// Get terminal content
	content, exists := tm.GetContent(instanceTitle)
	if !exists {
		return nil, fmt.Errorf("no content found for instance: %s", instanceTitle)
	}
	
	// Extract tasks using multiple regex patterns
	var tasks []types.TaskItem
	
	// 1. Primary pattern: "1. [STATUS] Task description"
	matches := taskRegexp.FindAllStringSubmatch(content, -1)
	for i, match := range matches {
		if len(match) >= 4 {
			status := "pending"
			// Parse status from match[2] (e.g., "TODO", "DONE", "IN PROGRESS")
			switch match[2] {
			case "TODO", "TO DO", "PENDING", "NOT STARTED":
				status = "pending"
			case "DONE", "COMPLETED", "FINISHED", "FIXED", "RESOLVED", "✓":
				status = "completed"
			case "IN PROGRESS", "WIP", "STARTED", "WORKING", "ONGOING":
				status = "in_progress"
			case "CANCELLED", "SKIPPED", "DEPRECATED":
				status = "cancelled"
			}
			
			// Determine priority based on position
			priority := "medium"
			if i < 3 {
				priority = "high"
			} else if i > 10 {
				priority = "low"
			}
			
			task := types.TaskItem{
				ID:       match[1], // Use the number as ID
				Content:  match[3],
				Status:   status,
				Priority: priority,
			}
			tasks = append(tasks, task)
		}
	}
	
	// 2. To-do pattern: "1. TODO: Task description"
	todoMatches := todoRegexp.FindAllStringSubmatch(content, -1)
	for i, match := range todoMatches {
		if len(match) >= 3 {
			// Check if this ID already exists
			isDuplicate := false
			for _, task := range tasks {
				if task.ID == match[1] {
					isDuplicate = true
					break
				}
			}
			
			if !isDuplicate {
				// Determine priority based on position
				priority := "medium"
				if i < 3 {
					priority = "high"
				} else if i > 10 {
					priority = "low"
				}
				
				task := types.TaskItem{
					ID:       match[1], // Use the number as ID
					Content:  match[2],
					Status:   "pending",
					Priority: priority,
				}
				tasks = append(tasks, task)
			}
		}
	}
	
	// 3. Done pattern: "1. DONE: Task description"
	doneMatches := doneRegexp.FindAllStringSubmatch(content, -1)
	for i, match := range doneMatches {
		if len(match) >= 3 {
			// Check if this ID already exists
			isDuplicate := false
			for _, task := range tasks {
				if task.ID == match[1] {
					isDuplicate = true
					break
				}
			}
			
			if !isDuplicate {
				// Determine priority based on position
				priority := "medium"
				if i < 3 {
					priority = "high"
				} else if i > 10 {
					priority = "low"
				}
				
				task := types.TaskItem{
					ID:       match[1], // Use the number as ID
					Content:  match[2],
					Status:   "completed",
					Priority: priority,
				}
				tasks = append(tasks, task)
			}
		}
	}
	
	// 4. In Progress pattern: "1. IN PROGRESS: Task description"
	progressMatches := progressRegexp.FindAllStringSubmatch(content, -1)
	for i, match := range progressMatches {
		if len(match) >= 3 {
			// Check if this ID already exists
			isDuplicate := false
			for _, task := range tasks {
				if task.ID == match[1] {
					isDuplicate = true
					break
				}
			}
			
			if !isDuplicate {
				// Determine priority based on position
				priority := "medium"
				if i < 3 {
					priority = "high"
				} else if i > 10 {
					priority = "low"
				}
				
				task := types.TaskItem{
					ID:       match[1], // Use the number as ID
					Content:  match[2],
					Status:   "in_progress",
					Priority: priority,
				}
				tasks = append(tasks, task)
			}
		}
	}
	
	// Sort tasks by ID
	// (We don't need to sort them since they'll be in order by how they appear in the text)
	
	// Log the found tasks
	if debugLogging {
		log.FileOnlyInfoLog.Printf("Found %d tasks for instance %s", len(tasks), instanceTitle)
		for i, task := range tasks {
			log.FileOnlyInfoLog.Printf("Task %d: ID=%s, Status=%s, Priority=%s, Content=%s", 
				i, task.ID, task.Status, task.Priority, task.Content)
		}
	}
	
	// Cache the tasks
	tm.mutex.Lock()
	tm.taskCache[instanceTitle] = tasks
	tm.taskCacheTimestamp[instanceTitle] = time.Now()
	tm.mutex.Unlock()
	
	return tasks, nil
}

// Done returns a channel that is closed when the monitor stops.
func (tm *TerminalMonitor) Done() <-chan struct{} {
	return tm.done
}

// checkForUpdates polls for changes in terminal output.
func (tm *TerminalMonitor) checkForUpdates() {
	//LogWebDebug("MONITOR: Starting update check") // Too verbose
	
	tm.mutex.RLock()
	instancesToCheck := make([]*session.Instance, len(tm.monitoredInstances))
	copy(instancesToCheck, tm.monitoredInstances)
	tm.mutex.RUnlock()
	
	if len(instancesToCheck) == 0 {
		if tm.inactiveLogger == nil {
			tm.inactiveLogger = log.NewEvery(30 * time.Second)
		}
		if tm.inactiveLogger.ShouldLog() {
			log.FileOnlyInfoLog.Printf("TerminalMonitor: No instances currently monitored")
		}
		return
	}
	
	activeInstances := 0
	if debugLogging {
		log.FileOnlyInfoLog.Printf("Found %d instances total to monitor", len(instancesToCheck))
	}
	
	for _, currentInstance := range instancesToCheck {
		// Add debug logging to help diagnose active instance issues
		if debugLogging {
			log.FileOnlyInfoLog.Printf("Instance %s: Started=%v, Paused=%v", 
				currentInstance.Title, currentInstance.Started(), currentInstance.Paused())
		}
		
		// Add enhanced debug logging for every instance
		LogWebDebug("MONITOR: Checking instance %s: Started=%v, Paused=%v, Status=%v", 
			currentInstance.Title, currentInstance.Started(), currentInstance.Paused(), currentInstance.Status)
		
		// Initialize logger for terminal monitoring if needed
		if tm.nottyLogger == nil {
			tm.nottyLogger = log.NewEvery(30 * time.Second)
		}
		
		if !currentInstance.Started() || currentInstance.Paused() {
			// LogWebDebug("MONITOR: Skipping inactive instance: %s", currentInstance.Title) // Too verbose
			if debugLogging {
				log.FileOnlyInfoLog.Printf("Skipping inactive instance: %s", currentInstance.Title)
			}
			continue
		}
		
		// Log that we found an active instance
		// LogWebDebug("MONITOR: Found ACTIVE instance: %s", currentInstance.Title) // Too verbose
		
		activeInstances++
		if debugLogging {
			log.FileOnlyInfoLog.Printf("Found active instance: %s", currentInstance.Title)
		}
		
		// Get updated content
		content, err := currentInstance.Preview()
		if err != nil {
			log.ErrorLog.Printf("Error capturing content for %s: %v", currentInstance.Title, err)
			continue
		}
		
		// Skip empty content - only log in debug mode to avoid console spam
		if content == "" {
			if debugLogging {
				log.WarningLog.Printf("Empty content received for active instance %s", currentInstance.Title)
			}
			continue
		}
		
		// Calculate hash for change detection
		hasher := sha256.New()
		hasher.Write([]byte(content))
		newHash := hasher.Sum(nil)
		
		tm.mutex.Lock()
		oldHash, exists := tm.hashMap[currentInstance.Title]
		hashChanged := !exists || !bytes.Equal(oldHash, newHash)
		
		// Only log content checks in debug mode
		if debugLogging {
			if exists {
				log.FileOnlyInfoLog.Printf("Content check for %s: hashChanged=%v, contentLength=%d", 
					currentInstance.Title, hashChanged, len(content))
			} else {
				log.FileOnlyInfoLog.Printf("First content for %s: contentLength=%d", 
					currentInstance.Title, len(content))
			}
		}
		
		if hashChanged {
			// Initialize content logger if not already done
			if tm.contentLogger == nil {
				tm.contentLogger = log.NewEvery(15 * time.Second) // Log less frequently
			}
			
			// Rate-limit content change logs to avoid console spam
			if tm.contentLogger.ShouldLog() {
				log.FileOnlyInfoLog.Printf("Content changed for instance %s", currentInstance.Title)
			}
			
			// Update our content map and hash
			tm.contentMap[currentInstance.Title] = content
			tm.hashMap[currentInstance.Title] = newHash
			
			// Get prompt status
			// Pass content to HasUpdated to use cached version
			updatedStatus, hasPrompt := currentInstance.HasUpdated(content)
			
			// Only log prompt state changes in debug mode
			if updatedStatus && debugLogging { // updatedStatus implies a change that might include prompt
				log.FileOnlyInfoLog.Printf("State/prompt change for %s: hasPrompt=%v",
					currentInstance.Title, hasPrompt)
			}
			
			// Create update
			update := types.TerminalUpdate{
				InstanceTitle: currentInstance.Title,
				Content:       content,
				Timestamp:     time.Now(),
				Status:        string(currentInstance.Status),
				HasPrompt:     hasPrompt,
			}
			
			// Get subscribers
			subscribers := tm.subscribers[currentInstance.Title]
			numSubscribers := len(subscribers)
			
			// Only log broadcast details in debug mode
			if debugLogging && numSubscribers > 0 {
				log.FileOnlyInfoLog.Printf("Broadcasting update to %d subscribers for %s", 
					numSubscribers, currentInstance.Title)
			}
			
			tm.mutex.Unlock()
			
			// Notify subscribers
			sentCount := 0
			for _, sub := range subscribers {
				select {
				case sub <- update:
					sentCount++
				default:
					// This is a genuine warning - keep it
					log.WarningLog.Printf("Channel full, skipped update for a subscriber of %s", 
						currentInstance.Title)
				}
			}
			
			// Only log detailed results in debug mode
			if debugLogging && numSubscribers > 0 {
				log.FileOnlyInfoLog.Printf("Sent updates to %d/%d subscribers for %s", 
					sentCount, numSubscribers, currentInstance.Title)
			}
			
			// When content changes, invalidate task cache
			tm.mutex.Lock()
			delete(tm.taskCacheTimestamp, currentInstance.Title)
			tm.mutex.Unlock()
		} else {
			tm.mutex.Unlock()
		}
	}
	
	// Never show "no active instances" message in console output
	// In web mode, we still want to log this to the file but NEVER to console
	// Rate limit this message to avoid filling the log file unnecessarily
	if tm.inactiveLogger == nil {
		tm.inactiveLogger = log.NewEvery(30 * time.Second)
	}
	
	if tm.inactiveLogger.ShouldLog() {
		// Use file-only logger to prevent console pollution in web mode
		// This will only log to file, never to stdout/stderr
		if activeInstances == 0 {
			log.FileOnlyInfoLog.Printf("TerminalMonitor: No active instances to monitor.")
		}
	}
}
