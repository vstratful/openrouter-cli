package tui

import (
	"github.com/charmbracelet/glamour"
	"github.com/vstratful/openrouter-cli/internal/config"
)

// MarkdownRenderer wraps glamour for rendering markdown to styled terminal output.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
}

// NewMarkdownRenderer creates a new markdown renderer with the specified width.
func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	if width <= 0 {
		width = config.DefaultTerminalWidth
	}

	// Use DarkStyle explicitly to avoid terminal detection which can
	// interfere with Bubble Tea's terminal handling
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &MarkdownRenderer{
		renderer: renderer,
		width:    width,
	}, nil
}

// SetWidth updates the word wrap width by creating a new renderer.
func (m *MarkdownRenderer) SetWidth(width int) error {
	if width <= 0 {
		width = config.DefaultTerminalWidth
	}

	// Only recreate if width changed
	if width == m.width {
		return nil
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}

	m.renderer = renderer
	m.width = width
	return nil
}

// Render renders markdown content to styled terminal output.
func (m *MarkdownRenderer) Render(content string) (string, error) {
	return m.renderer.Render(content)
}
