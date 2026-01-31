// Package config re-exports session types from internal/config for backward compatibility.
package config

import (
	internalconfig "github.com/vstratful/openrouter-cli/internal/config"
)

// Re-export session types from internal/config for backward compatibility.
type (
	Session        = internalconfig.Session
	SessionMessage = internalconfig.SessionMessage
	SessionSummary = internalconfig.SessionSummary
)

// Re-export session functions from internal/config for backward compatibility.
var (
	NewSession       = internalconfig.NewSession
	GetSessionDir    = internalconfig.GetSessionDir
	LoadSession      = internalconfig.LoadSession
	ListSessions     = internalconfig.ListSessions
	GetLatestSession = internalconfig.GetLatestSession
)
