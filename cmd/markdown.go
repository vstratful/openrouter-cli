package cmd

import (
	"github.com/vstratful/openrouter-cli/internal/tui"
)

// Re-export types from internal/tui for backward compatibility.
type MarkdownRenderer = tui.MarkdownRenderer

// Re-export functions from internal/tui for backward compatibility.
var NewMarkdownRenderer = tui.NewMarkdownRenderer
