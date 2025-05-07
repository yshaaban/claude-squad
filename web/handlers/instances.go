package handlers

import (
	"claude-squad/log"
	"claude-squad/session"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// InstanceSummary represents condensed instance information for APIs.
type InstanceSummary struct {
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	Path       string    `json:"path"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Program    string    `json:"program"`
	InPlace    bool      `json:"in_place"`
	DiffStats  DiffStats `json:"diff_stats,omitempty"`
}

// InstanceDetail represents detailed instance information.
type InstanceDetail struct {
	InstanceSummary
	HasPrompt     bool   `json:"has_prompt"`
	TMuxSession   string `json:"tmux_session,omitempty"`
}

// DiffStats represents git diff statistics.
type DiffStats struct {
	Added     int `json:"added"`
	Removed   int `json:"removed"`
}

// InstanceOutput represents terminal output information.
type InstanceOutput struct {
	Content    string    `json:"content"`
	Format     string    `json:"format"`
	Timestamp  time.Time `json:"timestamp"`
	HasPrompt  bool      `json:"has_prompt"`
}

// InstancesHandler handles listing all instances.
func InstancesHandler(storage *session.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.FileOnlyInfoLog.Printf("API: InstancesHandler called from %s", r.RemoteAddr)
		
		// Load all instances
		instances, err := storage.LoadInstances()
		if err != nil {
			// Don't fail the whole request if there's just an issue with an existing tmux session
			if strings.Contains(err.Error(), "failed to start new session: tmux session already exists") {
				// This is an expected case for web mode with existing sessions
				log.FileOnlyWarningLog.Printf("API: Non-fatal error loading instances: %v", err)
				// Continue with empty instances list
				instances = []*session.Instance{}
			} else {
				// For other errors, still log and return error
				log.FileOnlyErrorLog.Printf("API: Error loading instances: %v", err)
				http.Error(w, "Error loading instances", http.StatusInternalServerError)
				return
			}
		}
		
		// Log all instances
		log.FileOnlyInfoLog.Printf("API: Loaded %d instances for InstancesHandler", len(instances))
		for i, instance := range instances {
			log.FileOnlyInfoLog.Printf("API: Instance %d: Title=%s, Status=%v", 
				i, instance.Title, instance.Status)
		}
		
		// Filter by status if requested
		filter := r.URL.Query().Get("filter")
		
		// Convert to summary objects
		summaries := make([]InstanceSummary, 0, len(instances))
		for _, instance := range instances {
			// Apply filter if needed
			if filter != "" && filter != "all" {
				if (filter == "running" && !instance.Started()) || 
				   (filter == "paused" && !instance.Paused()) {
					continue
				}
			}
			
			summary := instanceToSummary(instance)
			summaries = append(summaries, summary)
		}
		
		// Return as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"instances": summaries,
		}); err != nil {
			log.FileOnlyErrorLog.Printf("API: Error encoding instances: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// InstanceDetailHandler handles getting details for a specific instance.
func InstanceDetailHandler(storage *session.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name required", http.StatusBadRequest)
			return
		}
		
		// Find the instance
		instance, err := findInstanceByTitle(storage, name)
		if err != nil {
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		
		// Create detailed response
		detail := InstanceDetail{
			InstanceSummary: instanceToSummary(instance),
			HasPrompt:       false, // Determine prompt status from output if needed
		}
		
		// Include tmux session info if running
		if instance.Started() && !instance.Paused() {
			// Use instance title to derive tmux session name
			detail.TMuxSession = "claudesquad_" + instance.Title
		}
		
		// Return as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(detail); err != nil {
			log.FileOnlyErrorLog.Printf("API: Error encoding instance detail: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// InstanceOutputHandler handles getting terminal output for a specific instance.
func InstanceOutputHandler(storage *session.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name required", http.StatusBadRequest)
			return
		}
		
		// Find the instance
		instance, err := findInstanceByTitle(storage, name)
		if err != nil {
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		
		// Get format parameter (ansi, html, text)
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "ansi"
		}
		
		// Verify format is valid
		if format != "ansi" && format != "html" && format != "text" {
			http.Error(w, "Invalid format parameter", http.StatusBadRequest)
			return
		}
		
		// Only provide output for running instances
		if !instance.Started() || instance.Paused() {
			http.Error(w, "Instance is not running", http.StatusBadRequest)
			return
		}
		
		// Get terminal output
		content, err := instance.Preview()
		if err != nil {
			log.FileOnlyErrorLog.Printf("API: Error getting terminal output for '%s': %v", name, err)
			http.Error(w, "Error getting terminal output", http.StatusInternalServerError)
			return
		}
		
		// Convert format if needed
		if format == "html" {
			content = convertAnsiToHtml(content)
		} else if format == "text" {
			content = stripAnsi(content)
		}
		
		// Apply line limit if specified
		limit := r.URL.Query().Get("limit")
		if limit != "" {
			// Parse limit and apply (implementation left as TODO)
			// This would truncate content to the specified number of lines
		}
		
		// Determine prompt status
		_, hasPrompt := instance.HasUpdated(content)
		
		// Create response
		output := InstanceOutput{
			Content:    content,
			Format:     format,
			Timestamp:  time.Now(),
			HasPrompt:  hasPrompt,
		}
		
		// Return as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(output); err != nil {
			log.ErrorLog.Printf("Error encoding output: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// ServerStatusHandler handles getting server status information.
func ServerStatusHandler(version string, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"version": version,
			"uptime":  time.Since(startTime).String(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(status); err != nil {
			log.FileOnlyErrorLog.Printf("API: Error encoding server status: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// Helper functions

// findInstanceByTitle finds an instance by its title.
func findInstanceByTitle(storage *session.Storage, title string) (*session.Instance, error) {
	instances, err := storage.LoadInstances()
	if err != nil {
		return nil, err
	}
	
	for _, instance := range instances {
		if instance.Title == title {
			return instance, nil
		}
	}
	
	return nil, fmt.Errorf("instance not found: %s", title)
}

// instanceToSummary converts an Instance to an InstanceSummary.
func instanceToSummary(instance *session.Instance) InstanceSummary {
	diffStats := DiffStats{}
	if instance.Started() && !instance.Paused() {
		// Try to get diff stats if available
		stats := instance.GetDiffStats()
		if stats != nil {
			diffStats.Added = stats.Added
			diffStats.Removed = stats.Removed
		}
	}
	
	// Convert Status enum to proper string representation
	var statusStr string
	switch instance.Status {
	case session.Running:
		statusStr = "running"
	case session.Ready:
		statusStr = "ready"
	case session.Loading:
		statusStr = "loading"
	case session.Paused:
		statusStr = "paused"
	default:
		statusStr = "unknown"
	}
	
	return InstanceSummary{
		Title:     instance.Title,
		Status:    statusStr, // Use proper string representation
		Path:      instance.Path,
		CreatedAt: instance.CreatedAt,
		UpdatedAt: instance.UpdatedAt,
		Program:   instance.Program,
		InPlace:   instance.InPlace,
		DiffStats: diffStats,
	}
}

// ANSI conversion function
func convertAnsiToHtml(content string) string {
	// Replace special HTML characters
	content = strings.ReplaceAll(content, "&", "&amp;")
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	
	// Replace newlines with <br>
	content = strings.ReplaceAll(content, "\r\n", "<br>")
	content = strings.ReplaceAll(content, "\n", "<br>")
	
	// Replace tabs with spaces
	content = strings.ReplaceAll(content, "\t", "    ")
	
	// Add basic styling
	return "<pre style=\"white-space: pre-wrap; font-family: monospace;\">" + content + "</pre>"
}

func stripAnsi(content string) string {
	// ANSI escape code pattern
	re := regexp.MustCompile(`\x1B\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(content, "")
}