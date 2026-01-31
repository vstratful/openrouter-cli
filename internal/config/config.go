// Package config provides configuration management for the OpenRouter CLI.
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Default configuration values.
const (
	// DefaultModel is the default model to use when not configured.
	DefaultModel = "moonshotai/kimi-k2.5"

	// DefaultImageModel is the default model for image generation.
	DefaultImageModel = "google/gemini-2.5-flash-image"

	// DefaultStreamTimeout is the default timeout for streaming requests.
	DefaultStreamTimeout = 5 * time.Minute

	// StreamChunkTimeout is the timeout for waiting for a single chunk.
	// If no data is received within this time, the stream is considered hung.
	StreamChunkTimeout = 30 * time.Second

	// DefaultTerminalWidth is the default terminal width when auto-detection fails.
	DefaultTerminalWidth = 80

	// EscDoublePressTimeout is the timeout for double-press ESC actions.
	EscDoublePressTimeout = 2 * time.Second

	// PreviewTruncateLength is the max length for session preview text.
	PreviewTruncateLength = 50

	// StreamChannelBuffer is the buffer size for stream chunk channels.
	StreamChannelBuffer = 100
)

// Config holds the application configuration that is persisted to disk.
type Config struct {
	APIKey            string `json:"api_key"`
	DefaultModel      string `json:"default_model,omitempty"`
	DefaultImageModel string `json:"default_image_model,omitempty"`
}

// AppConfig holds all runtime configuration.
type AppConfig struct {
	// APIKey is the OpenRouter API key.
	APIKey string

	// DefaultModel is the default model to use.
	DefaultModel string

	// StreamTimeout is the timeout for streaming requests.
	StreamTimeout time.Duration

	// TerminalWidth is the terminal width (0 = auto-detect).
	TerminalWidth int
}

// NewAppConfig creates a new AppConfig with default values.
func NewAppConfig() *AppConfig {
	return &AppConfig{
		DefaultModel:  DefaultModel,
		StreamTimeout: DefaultStreamTimeout,
		TerminalWidth: 0, // Auto-detect
	}
}

// GetConfigDir returns the platform-specific config directory for openrouter.
// This is a variable to allow mocking in tests.
var GetConfigDir = func() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "openrouter"), nil
}

// GetConfigPath returns the full path to the config file.
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// Load reads the config file and returns the Config struct.
// Returns an empty Config if the file doesn't exist.
// Applies default values for any missing model fields.
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing model fields (handles existing configs)
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultModel
	}
	if cfg.DefaultImageModel == "" {
		cfg.DefaultImageModel = DefaultImageModel
	}

	return &cfg, nil
}

// Save writes the config to disk with secure permissions.
func Save(cfg *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory with user-only permissions
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file with user-only read/write permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// PromptForAPIKey interactively prompts the user for their API key.
func PromptForAPIKey() (string, error) {
	fmt.Println("No OpenRouter API key found.")
	fmt.Println("You can get an API key from: https://openrouter.ai/keys")
	fmt.Print("\nEnter your OpenRouter API key: ")

	reader := bufio.NewReader(os.Stdin)
	key, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return key, nil
}
