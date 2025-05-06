package mock

import (
	"claude-squad/session"
	"fmt"
	"sync"
	"time"
)

// MockStorage simulates the storage interface for testing.
type MockStorage struct {
	instances map[string]*session.Instance
	mutex     sync.RWMutex
}

// NewMockStorage creates a new mock storage with simulated instances.
func NewMockStorage() *MockStorage {
	storage := &MockStorage{
		instances: make(map[string]*session.Instance),
	}
	
	// Create some sample instances
	storage.CreateSampleInstances()
	
	return storage
}

// CreateSampleInstances creates sample instances for testing.
func (s *MockStorage) CreateSampleInstances() {
	// Create instances with different statuses
	instance1 := NewMockInstance("instance1", "/path/to/repo1")
	instance2 := NewMockInstance("instance2", "/path/to/repo2")
	instance3 := NewMockInstance("instance3", "/path/to/repo3")
	
	// Set different states
	instance2.Status = session.Paused
	instance2.UpdatedAt = time.Now().Add(-24 * time.Hour)
	
	// Add to storage
	s.instances[instance1.Title] = instance1.Instance
	s.instances[instance2.Title] = instance2.Instance
	s.instances[instance3.Title] = instance3.Instance
	
	// Simulate activity on running instances
	instance1.SimulateActivity(20 * time.Minute)
	instance3.SimulateActivity(10 * time.Minute)
}

// LoadInstances returns all instances.
func (s *MockStorage) LoadInstances() ([]*session.Instance, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	instances := make([]*session.Instance, 0, len(s.instances))
	for _, instance := range s.instances {
		instances = append(instances, instance)
	}
	
	return instances, nil
}

// SaveInstances simulates saving instances.
func (s *MockStorage) SaveInstances(instances []*session.Instance) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Clear existing instances
	s.instances = make(map[string]*session.Instance)
	
	// Add the new instances
	for _, instance := range instances {
		s.instances[instance.Title] = instance
	}
	
	return nil
}

// DeleteInstance removes an instance.
func (s *MockStorage) DeleteInstance(title string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if _, exists := s.instances[title]; !exists {
		return fmt.Errorf("instance not found: %s", title)
	}
	
	delete(s.instances, title)
	return nil
}

// DeleteAllInstances removes all instances.
func (s *MockStorage) DeleteAllInstances() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.instances = make(map[string]*session.Instance)
	return nil
}

// AddInstance adds a new instance.
func (s *MockStorage) AddInstance(instance *session.Instance) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.instances[instance.Title] = instance
	return nil
}

// GetInstanceByTitle gets an instance by title.
func (s *MockStorage) GetInstanceByTitle(title string) (*session.Instance, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	instance, exists := s.instances[title]
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", title)
	}
	
	return instance, nil
}