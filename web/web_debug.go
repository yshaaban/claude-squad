package web

import (
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/session/tmux"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DebugEnabled = true
	DebugLogFile = "web_debug.log"
)

// DebugLog provides enhanced logging for web mode debugging
type DebugLog struct {
	file *os.File
}

var debugLog *DebugLog

// InitDebugLog initializes detailed web debug logging
func InitDebugLog() {
	if !DebugEnabled {
		return
	}

	logPath := filepath.Join(os.TempDir(), DebugLogFile)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.ErrorLog.Printf("Failed to create web debug log: %v", err)
		return
	}

	debugLog = &DebugLog{
		file: file,
	}

	debugLog.LogMessage("Web debug logging initialized")
}

// CloseDebugLog closes the debug log file
func CloseDebugLog() {
	if debugLog != nil && debugLog.file != nil {
		debugLog.LogMessage("Web debug logging closed")
		debugLog.file.Close()
	}
}

// LogMessage logs a message with timestamp
func (d *DebugLog) LogMessage(format string, args ...interface{}) {
	if d == nil || d.file == nil {
		return
	}

	timestamp := time.Now().Format("2006/01/02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(d.file, "[%s] %s\n", timestamp, message)
}

// LogJSON logs a JSON representation of any value
func (d *DebugLog) LogJSON(prefix string, v interface{}) {
	if d == nil || d.file == nil {
		return
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		d.LogMessage("%s ERROR marshaling JSON: %v", prefix, err)
		return
	}

	d.LogMessage("%s %s", prefix, string(data))
}

// LogInstances logs detailed instance information
func (d *DebugLog) LogInstances(prefix string, instances []*session.Instance) {
	if d == nil || d.file == nil {
		return
	}

	d.LogMessage("%s: Found %d instances", prefix, len(instances))
	for i, instance := range instances {
		d.LogMessage("  Instance %d: Title=%s Started=%v Paused=%v Status=%v", 
			i, instance.Title, instance.Started(), instance.Paused(), instance.Status)
		
		// Log tmux session info if available
		if instance.Started() && !instance.Paused() {
			tmuxName := instance.GetTmuxSessionName()
			if tmuxName != "" {
				exists := "NO"
				if tmux.DoesSessionExist(tmuxName) {
					exists = "YES"
				}
				d.LogMessage("    Tmux session: %s (exists: %s)", tmuxName, exists)
			} else {
				d.LogMessage("    No tmux session name available")
			}
		}
	}
}

// Global helper functions for debug logging

// LogWebDebug logs a message to the web debug log
func LogWebDebug(format string, args ...interface{}) {
	if debugLog != nil {
		debugLog.LogMessage(format, args...)
	}
}

// LogWebJSON logs a JSON representation to the web debug log
func LogWebJSON(prefix string, v interface{}) {
	if debugLog != nil {
		debugLog.LogJSON(prefix, v)
	}
}

// LogWebInstances logs instance information to the web debug log
func LogWebInstances(prefix string, instances []*session.Instance) {
	if debugLog != nil {
		debugLog.LogInstances(prefix, instances)
	}
}