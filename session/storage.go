package session

import (
	"claude-squad/config"
	"claude-squad/log"
	"encoding/json"
	"fmt"
	"time"
)

// InstanceData represents the serializable data of an Instance
type InstanceData struct {
	Title     string    `json:"title"`
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	Status    Status    `json:"status"`
	Height    int       `json:"height"`
	Width     int       `json:"width"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	AutoYes   bool      `json:"auto_yes"`
	NoTTY     bool      `json:"no_tty"`
	InPlace   bool      `json:"in_place"`

	Program   string          `json:"program"`
	Worktree  GitWorktreeData `json:"worktree"`
	DiffStats DiffStatsData   `json:"diff_stats"`
}

// GitWorktreeData represents the serializable data of a GitWorktree
type GitWorktreeData struct {
	RepoPath      string `json:"repo_path"`
	WorktreePath  string `json:"worktree_path"`
	SessionName   string `json:"session_name"`
	BranchName    string `json:"branch_name"`
	BaseCommitSHA string `json:"base_commit_sha"`
}

// DiffStatsData represents the serializable data of a DiffStats
type DiffStatsData struct {
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Content string `json:"content"`
}

// Storage handles saving and loading instances using the state interface
type Storage struct {
	state config.InstanceStorage
}

// NewStorage creates a new storage instance
func NewStorage(state config.InstanceStorage) (*Storage, error) {
	return &Storage{
		state: state,
	}, nil
}

// SaveInstances saves the list of instances to disk
func (s *Storage) SaveInstances(instances []*Instance) error {
	// Convert instances to InstanceData
	data := make([]InstanceData, 0)
	for _, instance := range instances {
		if instance.Started() {
			data = append(data, instance.ToInstanceData())
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal instances: %w", err)
	}

	return s.state.SaveInstances(jsonData)
}

// LoadInstances loads the list of instances from disk
func (s *Storage) LoadInstances() ([]*Instance, error) {
	jsonData := s.state.GetInstances()

	// Print detailed debug info when there's an issue
	log.FileOnlyInfoLog.Printf("LoadInstances: got %d bytes of JSON data", len(jsonData))
	
	var instancesData []InstanceData
	if err := json.Unmarshal(jsonData, &instancesData); err != nil {
		log.FileOnlyInfoLog.Printf("LoadInstances: JSON unmarshal error: %v", err)
		return nil, fmt.Errorf("failed to unmarshal instances: %w", err)
	}

	log.FileOnlyInfoLog.Printf("LoadInstances: Unmarshaled %d instances", len(instancesData))
	
	instances := make([]*Instance, len(instancesData))
	for i, data := range instancesData {
		log.FileOnlyInfoLog.Printf("LoadInstances: Loading instance %d: Title=%s Status=%v", 
			i, data.Title, data.Status)
		
		instance, err := FromInstanceData(data)
		if err != nil {
			log.FileOnlyInfoLog.Printf("LoadInstances: Failed to create instance %s: %v", 
				data.Title, err)
			return nil, fmt.Errorf("failed to create instance %s: %w", data.Title, err)
		}
		
		log.FileOnlyInfoLog.Printf("LoadInstances: Successfully loaded instance %s", data.Title)
		instances[i] = instance
	}

	return instances, nil
}

// PreloadSimpleMode ensures that an empty instance list can be loaded even if storage is corrupt
func (s *Storage) PreloadSimpleMode() {
	// Check if we can load instances
	_, err := s.LoadInstances()
	if err != nil {
		// If we can't load instances, save an empty list to reset the storage
		log.FileOnlyInfoLog.Printf("Error loading instances, resetting storage: %v", err)
		s.SaveInstances([]*Instance{})
	}
}

// DeleteInstance removes an instance from storage
func (s *Storage) DeleteInstance(title string) error {
	// Try to grab raw JSON first to see if we can at least get that (for debugging)
	jsonData := s.state.GetInstances()
	log.FileOnlyInfoLog.Printf("DeleteInstance: Raw storage has %d bytes for instance '%s'", 
		len(jsonData), title)
	
	instances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	found := false
	newInstances := make([]*Instance, 0)
	for _, instance := range instances {
		data := instance.ToInstanceData()
		if data.Title != title {
			newInstances = append(newInstances, instance)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("instance not found: %s", title)
	}

	return s.SaveInstances(newInstances)
}

// UpdateInstance updates an existing instance in storage
func (s *Storage) UpdateInstance(instance *Instance) error {
	instances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	data := instance.ToInstanceData()
	found := false
	for i, existing := range instances {
		existingData := existing.ToInstanceData()
		if existingData.Title == data.Title {
			instances[i] = instance
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("instance not found: %s", data.Title)
	}

	return s.SaveInstances(instances)
}

// DeleteAllInstances removes all stored instances
func (s *Storage) DeleteAllInstances() error {
	return s.state.DeleteAllInstances()
}
