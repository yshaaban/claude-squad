package mock

import (
	"claude-squad/session/git"
	"fmt"
	"time"
)

// MockWorktree is a mock implementation of GitWorktree for testing.
type MockWorktree struct {
	branchName   string
	path         string
	isDirty      bool
	diffContent  string
	added        int
	removed      int
	branchExists bool
}

// NewMockWorktree creates a new mock git worktree.
func NewMockWorktree(path, branch string) *MockWorktree {
	return &MockWorktree{
		branchName:   branch,
		path:         path,
		isDirty:      false,
		diffContent:  "",
		added:        0,
		removed:      0,
		branchExists: true,
	}
}

// SetDiff sets the mock diff content.
func (m *MockWorktree) SetDiff(content string, added, removed int) {
	m.diffContent = content
	m.added = added
	m.removed = removed
	m.isDirty = true
}

// Path returns the path to the worktree.
func (m *MockWorktree) Path() string {
	return m.path
}

// BranchName returns the branch name.
func (m *MockWorktree) BranchName() string {
	return m.branchName
}

// IsBranchCheckedOut returns whether the branch is checked out.
func (m *MockWorktree) IsBranchCheckedOut() (bool, error) {
	return false, nil
}

// IsDirty returns whether the worktree is dirty.
func (m *MockWorktree) IsDirty() (bool, error) {
	return m.isDirty, nil
}

// Diff returns the diff stats.
func (m *MockWorktree) Diff() (*git.DiffStats, error) {
	if !m.isDirty {
		return &git.DiffStats{
			Content: "",
			Added:   0,
			Removed: 0,
		}, nil
	}
	
	return &git.DiffStats{
		Content: m.diffContent,
		Added:   m.added,
		Removed: m.removed,
	}, nil
}

// PushChanges simulates pushing changes.
func (m *MockWorktree) PushChanges(commitMsg string, withPush bool) error {
	if !m.isDirty {
		return fmt.Errorf("no changes to push")
	}
	
	// Simulate successful push
	m.isDirty = false
	return nil
}

// Create creates a new worktree.
func (m *MockWorktree) Create() error {
	return nil
}

// Remove removes the worktree.
func (m *MockWorktree) Remove() error {
	return nil
}

// SetAutoPush sets the auto push flag.
func (m *MockWorktree) SetAutoPush(autoPush bool) {
	// No-op for mock
}

// GetAutoPush gets the auto push flag.
func (m *MockWorktree) GetAutoPush() bool {
	return false
}

// LastCommitTime returns the time of the last commit.
func (m *MockWorktree) LastCommitTime() (time.Time, error) {
	return time.Now(), nil
}

// DoesBranchExist checks if a branch exists.
func (m *MockWorktree) DoesBranchExist() (bool, error) {
	return m.branchExists, nil
}

// HasRemote checks if the worktree has a remote.
func (m *MockWorktree) HasRemote() bool {
	return true
}