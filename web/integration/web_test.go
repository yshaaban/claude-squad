package integration_test

import (
	"claude-squad/config"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/web"
	"claude-squad/web/mock"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// MockStorage is a simplified storage implementation for testing
type MockStorage struct {
	instances map[string]*session.Instance
	mutex     sync.RWMutex
}

// NewMockStorage creates a new mock storage
func NewMockStorage() *MockStorage {
	return &MockStorage{
		instances: make(map[string]*session.Instance),
	}
}

// LoadInstances returns all instances
func (s *MockStorage) LoadInstances() ([]*session.Instance, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	instances := make([]*session.Instance, 0, len(s.instances))
	for _, instance := range s.instances {
		instances = append(instances, instance)
	}
	
	return instances, nil
}

// SaveInstances saves instances
func (s *MockStorage) SaveInstances(instances []*session.Instance) error {
	return nil
}

// DeleteInstance deletes an instance
func (s *MockStorage) DeleteInstance(title string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	delete(s.instances, title)
	return nil
}

// DeleteAllInstances deletes all instances
func (s *MockStorage) DeleteAllInstances() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.instances = make(map[string]*session.Instance)
	return nil
}

// AddInstance adds an instance
func (s *MockStorage) AddInstance(instance *session.Instance) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.instances[instance.Title] = instance
	return nil
}

// TestWebServer tests the basic functionality of the web server
func TestWebServer(t *testing.T) {
	// Enable logging
	log.Initialize(false)
	defer log.Close()
	
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "claude-squad-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create mock storage
	storage := NewMockStorage()
	
	// Create test instance
	instance, err := session.NewInstance(session.InstanceOptions{
		Title:   "test-instance",
		Path:    tempDir,
		Program: "claude",
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	
	// Set instance fields for testing
	instance.Status = session.Running
	instance.CreatedAt = time.Now().Add(-1 * time.Hour)
	instance.UpdatedAt = time.Now()
	
	// Add to storage
	storage.AddInstance(instance)
	
	// Create config with web server enabled
	cfg := config.DefaultConfig()
	cfg.WebServerEnabled = true
	cfg.WebServerPort = 0 // Use a random port
	cfg.WebServerHost = "localhost"
	cfg.WebServerAllowLocalhost = true
	
	// Create and start server
	server, err := web.NewServer(storage, cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	// Create test HTTP server
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	
	// Access instances endpoint
	url := fmt.Sprintf("%s/api/instances", ts.URL)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to get instances: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Decode response
	var result struct {
		Instances []map[string]interface{} `json:"instances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Check if we have our instance
	if len(result.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(result.Instances))
	} else if result.Instances[0]["title"] != "test-instance" {
		t.Errorf("Expected instance title 'test-instance', got '%v'", result.Instances[0]["title"])
	} else {
		t.Logf("Successfully verified web server functionality")
	}
}