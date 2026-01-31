package picker

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/vstratful/openrouter-cli/internal/config"
)

// SessionItem wraps a SessionSummary for display in a picker.
type SessionItem struct {
	Summary config.SessionSummary
}

func (i SessionItem) Title() string {
	return i.Summary.UpdatedAt.Format("Jan 2, 15:04")
}

func (i SessionItem) Description() string {
	if i.Summary.Model != "" {
		return fmt.Sprintf("[%s] \"%s\" (%d messages)", i.Summary.Model, i.Summary.Preview, i.Summary.MessageCount)
	}
	return fmt.Sprintf("\"%s\" (%d messages)", i.Summary.Preview, i.Summary.MessageCount)
}

func (i SessionItem) FilterValue() string {
	return i.Summary.Preview
}

// NewSessionPicker creates a new picker for sessions.
func NewSessionPicker(summaries []config.SessionSummary, width, height int) Model {
	items := make([]list.Item, len(summaries))
	for i, s := range summaries {
		items[i] = SessionItem{Summary: s}
	}

	return New(Config{
		Title:  "Resume a previous session",
		Items:  items,
		Width:  width,
		Height: height,
	})
}

// GetSessionSummary extracts the SessionSummary from a selected item.
func GetSessionSummary(item list.Item) *config.SessionSummary {
	if si, ok := item.(SessionItem); ok {
		return &si.Summary
	}
	return nil
}
