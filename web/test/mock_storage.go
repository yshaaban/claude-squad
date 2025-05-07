package test

import (
	"claude-squad/session"
	"fmt"
)

// MockStorage is a mock implementation of session.Storage.
type MockStorage struct {
	instances []*MockInstance
}

// NewMockStorage creates a new mock storage with sample instances.
func NewMockStorage() *MockStorage {
	storage := &MockStorage{}
	
	// Create 3 mock instances
	for i := 1; i <= 3; i++ {
		instance := NewMockInstance(
			fmt.Sprintf("instance%d", i),
			fmt.Sprintf("/path/to/test/instance%d", i),
		)
		storage.instances = append(storage.instances, instance)
	}
	
	return storage
}

// LoadInstances returns mock instances as session.Instance objects.
func (s *MockStorage) LoadInstances() ([]*session.Instance, error) {
	// This is not ideal but will work for testing
	// We convert our mock instances to session.Instance objects
	result := make([]*session.Instance, len(s.instances))
	
	for i, instance := range s.instances {
		// Create a minimal instance with the properties we need
		mockInstance, _ := session.NewInstance(session.InstanceOptions{
			Title:   instance.Title,
			Path:    instance.Path,
			Program: "claude",
		})
		
		// Set properties that matter for testing
		mockInstance.SetStatus(instance.Status)
		
		// Add to result
		result[i] = mockInstance
	}
	
	return result, nil
}

// SaveInstances is a no-op for testing.
func (s *MockStorage) SaveInstances([]*session.Instance) error {
	return nil
}

// DeleteInstance is a no-op for testing.
func (s *MockStorage) DeleteInstance(instanceTitle string) error {
	for i, instance := range s.instances {
		if instance.Title == instanceTitle {
			s.instances = append(s.instances[:i], s.instances[i+1:]...)
			break
		}
	}
	return nil
}

// GetMockInstances returns the actual mock instances for testing.
func (s *MockStorage) GetMockInstances() []*MockInstance {
	return s.instances
}