package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrSessionNotFound is returned when a session cannot be found.
var ErrSessionNotFound = errors.New("session not found")

// SessionMessage represents a message in the conversation.
type SessionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Session represents a CLI session with its history.
type Session struct {
	ID        string           `json:"id"`
	Model     string           `json:"model,omitempty"` // Model used for this session
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	History   []string         `json:"history"`  // User input history for arrow key navigation
	Messages  []SessionMessage `json:"messages"` // Full conversation for resume
}

// SessionSummary represents a session for list display.
type SessionSummary struct {
	ID           string
	Model        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
	Preview      string // First user message, truncated to ~50 chars
}

// NewSession creates a new session with a generated UUID.
func NewSession() *Session {
	now := time.Now()
	return &Session{
		ID:        uuid.New().String(),
		CreatedAt: now,
		UpdatedAt: now,
		History:   []string{},
	}
}

// GetSessionDir returns the directory where sessions are stored.
// This is a variable to allow mocking in tests.
var GetSessionDir = func() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sessions"), nil
}

// Save writes the session to disk.
func (s *Session) Save() error {
	sessionDir, err := GetSessionDir()
	if err != nil {
		return err
	}

	// Create sessions directory with user-only permissions
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Update the timestamp on each save
	s.UpdatedAt = time.Now()

	sessionPath := filepath.Join(sessionDir, s.ID+".json")

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// AppendHistory adds an entry to the history and saves.
func (s *Session) AppendHistory(entry string) error {
	s.History = append(s.History, entry)
	return s.Save()
}

// AppendMessage adds a message to the conversation and saves.
func (s *Session) AppendMessage(role, content string) error {
	s.Messages = append(s.Messages, SessionMessage{Role: role, Content: content})
	return s.Save()
}

// LoadSession loads an existing session by ID.
func LoadSession(id string) (*Session, error) {
	sessionDir, err := GetSessionDir()
	if err != nil {
		return nil, err
	}

	sessionPath := filepath.Join(sessionDir, id+".json")

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &session, nil
}

// ListSessions returns summaries of all sessions sorted by UpdatedAt descending.
func ListSessions() ([]SessionSummary, error) {
	sessionDir, err := GetSessionDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionSummary{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var summaries []SessionSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		session, err := LoadSession(id)
		if err != nil {
			// Skip corrupted files
			continue
		}

		// Filter out empty sessions
		if len(session.Messages) == 0 {
			continue
		}

		// Get preview from first user message
		preview := ""
		for _, msg := range session.Messages {
			if msg.Role == "user" {
				preview = msg.Content
				break
			}
		}
		if len(preview) > PreviewTruncateLength {
			preview = preview[:PreviewTruncateLength-3] + "..."
		}

		summaries = append(summaries, SessionSummary{
			ID:           session.ID,
			Model:        session.Model,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
			MessageCount: len(session.Messages),
			Preview:      preview,
		})
	}

	// Sort by UpdatedAt descending (most recent first)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})

	return summaries, nil
}

// GetLatestSession returns the most recently updated session.
func GetLatestSession() (*Session, error) {
	summaries, err := ListSessions()
	if err != nil {
		return nil, err
	}

	if len(summaries) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	return LoadSession(summaries[0].ID)
}
