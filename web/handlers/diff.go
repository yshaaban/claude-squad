package handlers

import (
	"claude-squad/log"
	"claude-squad/session"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

// FileDiff represents diff information for a single file.
type FileDiff struct {
	Path     string `json:"path"`
	Added    int    `json:"added"`
	Removed  int    `json:"removed"`
	IsNew    bool   `json:"is_new"`
	IsDelete bool   `json:"is_delete"`
	IsBinary bool   `json:"is_binary"`
	Hunks    []Hunk `json:"hunks"`
}

// Hunk represents a group of changes in a diff.
type Hunk struct {
	Header  string     `json:"header"`
	Changes []DiffLine `json:"changes"`
}

// DiffLine represents a single line in a diff.
type DiffLine struct {
	Type      string `json:"type"` // "add", "remove", "context"
	Content   string `json:"content"`
	Number    *int   `json:"number,omitempty"`
	OldNumber *int   `json:"old_number,omitempty"`
}

// WebDiffStats is the enhanced diff statistics for web visualization.
type WebDiffStats struct {
	Added   int        `json:"added"`
	Removed int        `json:"removed"`
	Files   []FileDiff `json:"files"`
}

// DiffHandler handles getting git diff information for a specific instance.
func DiffHandler(storage *session.Storage) http.HandlerFunc {
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
		
		// Only provide diff for running instances
		if !instance.Started() || instance.Paused() {
			http.Error(w, "Instance is not running", http.StatusBadRequest)
			return
		}
		
		// Get diff stats
		diffStats := instance.GetDiffStats()
		if diffStats == nil {
			log.ErrorLog.Printf("Error getting diff stats: %v", err)
			http.Error(w, "Error getting diff stats", http.StatusInternalServerError)
			return
		}
		
		if diffStats == nil {
			// No diff available
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"added":   0,
				"removed": 0,
				"files":   []interface{}{},
			})
			return
		}
		
		// Get format parameter (raw, parsed, stats)
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "parsed"
		}
		
		switch format {
		case "raw":
			// Return raw diff content
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(diffStats.Content))
			
		case "stats":
			// Return just the statistics
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"added":   diffStats.Added,
				"removed": diffStats.Removed,
			})
			
		case "parsed":
			// Parse and structure the diff
			webDiff, err := parseDiffOutput(diffStats.Content, diffStats.Added, diffStats.Removed)
			if err != nil {
				log.ErrorLog.Printf("Error parsing diff: %v", err)
				http.Error(w, "Error parsing diff", http.StatusInternalServerError)
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(webDiff)
			
		default:
			http.Error(w, "Invalid format parameter", http.StatusBadRequest)
		}
	}
}

// DiffHistoryHandler handles getting historical snapshots of diffs.
func DiffHistoryHandler(storage *session.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement diff history tracking
		http.Error(w, "Diff history not implemented", http.StatusNotImplemented)
	}
}

// parseDiffOutput parses git diff output into a structured format.
func parseDiffOutput(diffContent string, totalAdded, totalRemoved int) (*WebDiffStats, error) {
	result := &WebDiffStats{
		Added:   totalAdded,
		Removed: totalRemoved,
		Files:   make([]FileDiff, 0),
	}
	
	if diffContent == "" {
		return result, nil
	}
	
	// Parse diff content
	lines := strings.Split(diffContent, "\n")
	var currentFile *FileDiff
	var currentHunk *Hunk
	
	fileHeaderRegex := regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+),(\d+) \+(\d+),(\d+) @@(.*)$`)
	
	var oldLineNum, newLineNum int
	
	for _, line := range lines {
		// Detect file headers
		if strings.HasPrefix(line, "diff --git ") {
			// Add previous file if any
			if currentFile != nil {
				result.Files = append(result.Files, *currentFile)
			}
			
			// Start new file
			currentFile = &FileDiff{
				Hunks: make([]Hunk, 0),
			}
			
			// Extract file path
			matches := fileHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				currentFile.Path = matches[2] // Use the b/ path
			}
			currentHunk = nil
			continue
		}
		
		// Detect binary files
		if strings.Contains(line, "Binary files") {
			if currentFile != nil {
				currentFile.IsBinary = true
			}
			continue
		}
		
		// Detect new/deleted files
		if strings.HasPrefix(line, "new file") && currentFile != nil {
			currentFile.IsNew = true
			continue
		}
		if strings.HasPrefix(line, "deleted file") && currentFile != nil {
			currentFile.IsDelete = true
			continue
		}
		
		// Detect hunks
		if strings.HasPrefix(line, "@@") {
			matches := hunkHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 5 && currentFile != nil {
				// Reset line counters
				oldLineNum = parseIntSafe(matches[1])
				newLineNum = parseIntSafe(matches[3])
				
				currentHunk = &Hunk{
					Header:  line,
					Changes: make([]DiffLine, 0),
				}
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}
			continue
		}
		
		// Handle changed lines
		if currentFile != nil && len(currentFile.Hunks) > 0 {
			var diffLine DiffLine
			
			if len(line) > 0 {
				switch line[0] {
				case '+':
					diffLine = DiffLine{
						Type:    "add",
						Content: line[1:],
						Number:  &newLineNum,
					}
					newLineNum++
					currentFile.Added++
					
				case '-':
					diffLine = DiffLine{
						Type:      "remove",
						Content:   line[1:],
						OldNumber: &oldLineNum,
					}
					oldLineNum++
					currentFile.Removed++
					
				default:
					diffLine = DiffLine{
						Type:      "context",
						Content:   line,
						Number:    &newLineNum,
						OldNumber: &oldLineNum,
					}
					newLineNum++
					oldLineNum++
				}
				
				// Add to current hunk
				hunkIndex := len(currentFile.Hunks) - 1
				currentFile.Hunks[hunkIndex].Changes = append(
					currentFile.Hunks[hunkIndex].Changes, 
					diffLine,
				)
			}
		}
	}
	
	// Add the last file if any
	if currentFile != nil {
		result.Files = append(result.Files, *currentFile)
	}
	
	return result, nil
}

// parseIntSafe parses a string to an integer with a default of 0 on error.
func parseIntSafe(s string) int {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return 0
	}
	return i
}