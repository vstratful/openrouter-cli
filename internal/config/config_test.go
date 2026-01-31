package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "openrouter")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create test config dir: %v", err)
	}

	// Override GetConfigDir for testing
	originalGetConfigDir := GetConfigDir
	GetConfigDir = func() (string, error) {
		return configDir, nil
	}
	defer func() { GetConfigDir = originalGetConfigDir }()

	t.Run("returns empty config when file does not exist", func(t *testing.T) {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if cfg.APIKey != "" {
			t.Errorf("cfg.APIKey = %q, want empty string", cfg.APIKey)
		}
	})

	t.Run("loads config from file", func(t *testing.T) {
		// Write test config
		testConfig := Config{APIKey: "test-api-key"}
		data, _ := json.MarshalIndent(testConfig, "", "  ")
		configPath := filepath.Join(configDir, "config.json")
		if err := os.WriteFile(configPath, data, 0600); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if cfg.APIKey != "test-api-key" {
			t.Errorf("cfg.APIKey = %q, want %q", cfg.APIKey, "test-api-key")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		configPath := filepath.Join(configDir, "config.json")
		if err := os.WriteFile(configPath, []byte("not valid json"), 0600); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		_, err := Load()
		if err == nil {
			t.Error("Load() error = nil, want parse error")
		}
	})
}

func TestSave(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "openrouter")

	// Override GetConfigDir for testing
	originalGetConfigDir := GetConfigDir
	GetConfigDir = func() (string, error) {
		return configDir, nil
	}
	defer func() { GetConfigDir = originalGetConfigDir }()

	t.Run("creates config directory and file", func(t *testing.T) {
		cfg := &Config{APIKey: "test-api-key"}
		if err := Save(cfg); err != nil {
			t.Fatalf("Save() error = %v, want nil", err)
		}

		// Verify file was created
		configPath := filepath.Join(configDir, "config.json")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config file: %v", err)
		}

		var loaded Config
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("failed to parse config file: %v", err)
		}

		if loaded.APIKey != "test-api-key" {
			t.Errorf("loaded.APIKey = %q, want %q", loaded.APIKey, "test-api-key")
		}
	})

	t.Run("file has secure permissions", func(t *testing.T) {
		cfg := &Config{APIKey: "test-api-key"}
		if err := Save(cfg); err != nil {
			t.Fatalf("Save() error = %v, want nil", err)
		}

		configPath := filepath.Join(configDir, "config.json")
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("failed to stat config file: %v", err)
		}

		// Check permissions (0600 = owner read/write only)
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("file permissions = %o, want %o", perm, 0600)
		}
	})
}

func TestGetConfigDir(t *testing.T) {
	// Save original and restore after test
	originalGetConfigDir := GetConfigDir
	GetConfigDir = func() (string, error) {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(configDir, "openrouter"), nil
	}
	defer func() { GetConfigDir = originalGetConfigDir }()

	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() error = %v, want nil", err)
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("GetConfigDir() = %q, want absolute path", dir)
	}

	if filepath.Base(dir) != "openrouter" {
		t.Errorf("GetConfigDir() = %q, want path ending in 'openrouter'", dir)
	}
}

func TestNewAppConfig(t *testing.T) {
	cfg := NewAppConfig()

	if cfg.DefaultModel != DefaultModel {
		t.Errorf("DefaultModel = %q, want %q", cfg.DefaultModel, DefaultModel)
	}

	if cfg.StreamTimeout != DefaultStreamTimeout {
		t.Errorf("StreamTimeout = %v, want %v", cfg.StreamTimeout, DefaultStreamTimeout)
	}

	if cfg.TerminalWidth != 0 {
		t.Errorf("TerminalWidth = %d, want 0 (auto-detect)", cfg.TerminalWidth)
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have sensible values
	if EscDoublePressTimeout <= 0 {
		t.Errorf("EscDoublePressTimeout = %v, want positive duration", EscDoublePressTimeout)
	}

	if PreviewTruncateLength <= 0 {
		t.Errorf("PreviewTruncateLength = %d, want positive value", PreviewTruncateLength)
	}

	if StreamChannelBuffer <= 0 {
		t.Errorf("StreamChannelBuffer = %d, want positive value", StreamChannelBuffer)
	}

	if DefaultTerminalWidth <= 0 {
		t.Errorf("DefaultTerminalWidth = %d, want positive value", DefaultTerminalWidth)
	}
}
