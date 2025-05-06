package web

import (
	"claude-squad/config"
	"claude-squad/log"
	"claude-squad/session"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// TestAPIEndpoints tests the API endpoints directly.
func TestAPIEndpoints(t *testing.T) {
	// Skip for now - we'll use the e2e script instead
	t.Skip("API tests are run by the e2e-test.sh script")
	
	// Enable logging
	log.Initialize(false)
	defer log.Close()
	
	// Create mock storage
	storage := &testStorage{
		instances: make(map[string]*session.Instance),
	}
	
	// Create test instance
	tempDir, err := os.MkdirTemp("", "claude-squad-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test instance
	instance, err := session.NewInstance(session.InstanceOptions{
		Title:   "test-instance",
		Path:    tempDir,
		Program: "claude",
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	
	// Set instance fields
	instance.Status = session.Running
	
	// Add to storage
	storage.AddInstance(instance)
	
	// Create server
	cfg := config.DefaultConfig()
	server, err := NewServer(storage, cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	// Create test HTTP server
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	
	// Test instances endpoint
	t.Run("Instances", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/instances")
		if err != nil {
			t.Fatalf("Failed to get instances: %v", err)
		}
		defer resp.Body.Close()
		
		// Verify response status
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		// Parse response
		var result struct {
			Instances []map[string]interface{} `json:"instances"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		// Verify instances
		if len(result.Instances) != 1 {
			t.Errorf("Expected 1 instance, got %d", len(result.Instances))
		} else if result.Instances[0]["title"] != "test-instance" {
			t.Errorf("Wrong instance title: %v", result.Instances[0]["title"])
		}
	})
}

// testStorage is a simple implementation of the Storage interface for testing
type testStorage struct {
	instances map[string]*session.Instance
	mutex     sync.RWMutex
}

func (s *testStorage) LoadInstances() ([]*session.Instance, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	result := make([]*session.Instance, 0, len(s.instances))
	for _, inst := range s.instances {
		result = append(result, inst)
	}
	
	return result, nil
}

func (s *testStorage) SaveInstances(instances []*session.Instance) error {
	return nil
}

func (s *testStorage) DeleteInstance(title string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	delete(s.instances, title)
	return nil
}

func (s *testStorage) DeleteAllInstances() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.instances = make(map[string]*session.Instance)
	return nil
}

func (s *testStorage) AddInstance(instance *session.Instance) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.instances[instance.Title] = instance
	return nil
}