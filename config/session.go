package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Session represents a CLI session with its history
type Session struct {
	ID      string   `json:"id"`
	History []string `json:"history"`
}

// NewSession creates a new session with a generated UUID
func NewSession() *Session {
	return &Session{
		ID:      uuid.New().String(),
		History: []string{},
	}
}

// GetSessionDir returns the directory where sessions are stored
func GetSessionDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sessions"), nil
}

// Save writes the session to disk
func (s *Session) Save() error {
	sessionDir, err := GetSessionDir()
	if err != nil {
		return err
	}

	// Create sessions directory with user-only permissions
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

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

// AppendHistory adds an entry to the history and saves
func (s *Session) AppendHistory(entry string) error {
	s.History = append(s.History, entry)
	return s.Save()
}

// LoadSession loads an existing session by ID
func LoadSession(id string) (*Session, error) {
	sessionDir, err := GetSessionDir()
	if err != nil {
		return nil, err
	}

	sessionPath := filepath.Join(sessionDir, id+".json")

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &session, nil
}
