package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testSessionDir is used to override the session directory for testing
var testSessionDir string

func init() {
	// Override GetSessionDir for testing
	originalGetSessionDir := GetSessionDir
	GetSessionDir = func() (string, error) {
		if testSessionDir != "" {
			return testSessionDir, nil
		}
		return originalGetSessionDir()
	}
}

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "openrouter-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	testSessionDir = filepath.Join(tempDir, "sessions")
	return tempDir, func() {
		testSessionDir = ""
		os.RemoveAll(tempDir)
	}
}

func TestNewSession(t *testing.T) {
	s := NewSession()

	if s.ID == "" {
		t.Error("ID should not be empty")
	}

	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if s.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	if len(s.History) != 0 {
		t.Errorf("History should be empty, got %d items", len(s.History))
	}

	if len(s.Messages) != 0 {
		t.Errorf("Messages should be empty, got %d items", len(s.Messages))
	}
}

func TestSessionSaveAndLoad(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Create and save a session
	s := NewSession()
	s.Model = "test-model"
	s.History = []string{"hello", "world"}
	s.Messages = []SessionMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load the session
	loaded, err := LoadSession(s.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}

	// Verify loaded data
	if loaded.ID != s.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, s.ID)
	}
	if loaded.Model != s.Model {
		t.Errorf("Model = %q, want %q", loaded.Model, s.Model)
	}
	if len(loaded.History) != len(s.History) {
		t.Errorf("History length = %d, want %d", len(loaded.History), len(s.History))
	}
	if len(loaded.Messages) != len(s.Messages) {
		t.Errorf("Messages length = %d, want %d", len(loaded.Messages), len(s.Messages))
	}
}

func TestSessionAppendHistory(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	s := NewSession()

	// Append history entries
	if err := s.AppendHistory("first"); err != nil {
		t.Fatalf("AppendHistory() error = %v", err)
	}
	if err := s.AppendHistory("second"); err != nil {
		t.Fatalf("AppendHistory() error = %v", err)
	}

	if len(s.History) != 2 {
		t.Errorf("History length = %d, want 2", len(s.History))
	}
	if s.History[0] != "first" {
		t.Errorf("History[0] = %q, want %q", s.History[0], "first")
	}
	if s.History[1] != "second" {
		t.Errorf("History[1] = %q, want %q", s.History[1], "second")
	}
}

func TestSessionAppendMessage(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	s := NewSession()

	// Append messages
	if err := s.AppendMessage("user", "Hello"); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}
	if err := s.AppendMessage("assistant", "Hi!"); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	if len(s.Messages) != 2 {
		t.Errorf("Messages length = %d, want 2", len(s.Messages))
	}
	if s.Messages[0].Role != "user" {
		t.Errorf("Messages[0].Role = %q, want %q", s.Messages[0].Role, "user")
	}
	if s.Messages[0].Content != "Hello" {
		t.Errorf("Messages[0].Content = %q, want %q", s.Messages[0].Content, "Hello")
	}
}

func TestLoadSessionNotFound(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	_, err := LoadSession("nonexistent-id")
	if err == nil {
		t.Error("LoadSession() should return error for nonexistent session")
	}
}

func TestListSessions(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Create some sessions
	s1 := NewSession()
	s1.Model = "model1"
	s1.AppendMessage("user", "First message")
	s1.Save()

	// Small delay to ensure different UpdatedAt
	time.Sleep(10 * time.Millisecond)

	s2 := NewSession()
	s2.Model = "model2"
	s2.AppendMessage("user", "Second message")
	s2.Save()

	// Empty session should be filtered out
	s3 := NewSession()
	s3.Save()
	_ = s3 // Unused, but that's intentional

	// List sessions
	summaries, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	// Should have 2 sessions (empty one filtered out)
	if len(summaries) != 2 {
		t.Errorf("ListSessions() returned %d sessions, want 2", len(summaries))
	}

	// Most recent should be first (s2)
	if len(summaries) > 0 && summaries[0].ID != s2.ID {
		t.Errorf("First session ID = %q, want %q", summaries[0].ID, s2.ID)
	}
}

func TestSessionSummaryPreview(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Create a session with a long message
	s := NewSession()
	longMessage := "This is a very long message that should be truncated when displayed as a preview in the session list"
	s.AppendMessage("user", longMessage)
	s.Save()

	summaries, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(summaries))
	}

	// Preview should be truncated to 50 chars
	if len(summaries[0].Preview) > 50 {
		t.Errorf("Preview length = %d, should be <= 50", len(summaries[0].Preview))
	}
}

func TestGetLatestSession(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// No sessions should return error
	_, err := GetLatestSession()
	if err == nil {
		t.Error("GetLatestSession() should return error when no sessions exist")
	}

	// Create sessions
	s1 := NewSession()
	s1.AppendMessage("user", "First")
	s1.Save()

	time.Sleep(10 * time.Millisecond)

	s2 := NewSession()
	s2.AppendMessage("user", "Second")
	s2.Save()

	// Get latest should return s2
	latest, err := GetLatestSession()
	if err != nil {
		t.Fatalf("GetLatestSession() error = %v", err)
	}
	if latest.ID != s2.ID {
		t.Errorf("GetLatestSession().ID = %q, want %q", latest.ID, s2.ID)
	}
}
