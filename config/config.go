// Package config re-exports from internal/config for backward compatibility.
package config

import (
	internalconfig "github.com/vstratful/openrouter-cli/internal/config"
)

// Re-export types from internal/config for backward compatibility.
type (
	Config    = internalconfig.Config
	AppConfig = internalconfig.AppConfig
)

// Re-export constants
const (
	DefaultModel         = internalconfig.DefaultModel
	DefaultStreamTimeout = internalconfig.DefaultStreamTimeout
	DefaultTerminalWidth = internalconfig.DefaultTerminalWidth
)

// Re-export functions from internal/config for backward compatibility.
var (
	NewAppConfig    = internalconfig.NewAppConfig
	GetConfigDir    = internalconfig.GetConfigDir
	GetConfigPath   = internalconfig.GetConfigPath
	Load            = internalconfig.Load
	Save            = internalconfig.Save
	PromptForAPIKey = internalconfig.PromptForAPIKey
)
