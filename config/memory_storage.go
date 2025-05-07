package config

import (
	"encoding/json"
	"sync"
)

// MemoryStorage implements InstanceStorage in memory for testing
type MemoryStorage struct {
	mu           sync.Mutex
	instancesData json.RawMessage
	helpScreensSeen uint32
}

// SaveInstances saves the raw instance data
func (m *MemoryStorage) SaveInstances(instancesJSON json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.instancesData = make(json.RawMessage, len(instancesJSON))
	copy(m.instancesData, instancesJSON)
	return nil
}

// GetInstances returns the raw instance data
func (m *MemoryStorage) GetInstances() json.RawMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.instancesData == nil {
		return json.RawMessage("[]")
	}
	return m.instancesData
}

// DeleteAllInstances removes all stored instances
func (m *MemoryStorage) DeleteAllInstances() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.instancesData = json.RawMessage("[]")
	return nil
}

// GetHelpScreensSeen returns the bitmask of seen help screens
func (m *MemoryStorage) GetHelpScreensSeen() uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.helpScreensSeen
}

// SetHelpScreensSeen updates the bitmask of seen help screens
func (m *MemoryStorage) SetHelpScreensSeen(seen uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.helpScreensSeen = seen
	return nil
}