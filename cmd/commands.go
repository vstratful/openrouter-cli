package cmd

import (
	"github.com/vstratful/openrouter-cli/internal/tui/chat"
)

// Re-export types from internal/tui/chat for backward compatibility.
type Command = chat.Command

// Re-export functions from internal/tui/chat for backward compatibility.
var (
	AvailableCommands = chat.AvailableCommands
	FilterCommands    = chat.FilterCommands
)
