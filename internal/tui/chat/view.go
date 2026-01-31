package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vstratful/openrouter-cli/internal/api"
	"github.com/vstratful/openrouter-cli/internal/tui"
)

// View renders the chat model.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Header - minimal, just show resumed status if applicable
	var header string
	if m.isResumed {
		header = tui.HelpStyle.Render("(Resumed session)")
	}

	// Footer - show model name and status
	var footer string
	modelInfo := tui.DimHelpStyle.Render(m.modelName)
	sep := tui.DimHelpStyle.Render(" • ")

	// Session warning (if session save failed)
	var sessionWarning string
	if m.sessionErr != nil {
		sessionWarning = sep + tui.SessionWarningStyle.Render("⚠ Session save failed")
	}

	switch m.state {
	case StateStreaming:
		escHint := sep + tui.KeyHintStyle.Render("Esc") + tui.DimHelpStyle.Render(": cancel")
		if m.currentContent == "" {
			footer = modelInfo + sep + m.spinner.View() + " Thinking..." + escHint
		} else {
			footer = modelInfo + sep + m.spinner.View() + " Streaming..." + escHint
		}
	case StateEscPending:
		// Warning state
		escAction := "clear input"
		if m.escState.action == EscActionExit {
			escAction = "exit"
		}
		footer = modelInfo + sep + tui.EscWarningStyle.Render("Press ⎋ again to "+escAction)
	case StateIdle:
		if m.history.IsBrowsing() {
			// History browsing mode
			historyPos := fmt.Sprintf("browsing history (%d/%d)",
				m.history.HistoryLen()-m.history.Index(), m.history.HistoryLen())
			footer = modelInfo + sep + tui.HistoryModeStyle.Render(historyPos) +
				sep + tui.DimHelpStyle.Render("↑↓: navigate • Enter: use • ⎋: cancel")
		} else {
			// Normal state with styled hints
			hints := []string{
				tui.KeyHintStyle.Render("Enter") + tui.DimHelpStyle.Render(": send"),
				tui.KeyHintStyle.Render("↑↓") + tui.DimHelpStyle.Render(": history"),
				tui.KeyHintStyle.Render("/") + tui.DimHelpStyle.Render(": commands"),
			}
			footer = modelInfo + sep + strings.Join(hints, sep)
		}
	}
	footer += sessionWarning

	// Render autocomplete if showing
	var autocompleteView string
	if m.autocomplete.Visible() && len(m.autocomplete.Filtered()) > 0 {
		autocompleteView = m.renderAutocomplete()
	}

	// Style the input box - change border color based on state
	currentInputStyle := tui.InputBoxStyle
	switch m.state {
	case StateEscPending:
		currentInputStyle = tui.EscWarningBoxStyle
	case StateIdle:
		if m.history.IsBrowsing() {
			currentInputStyle = tui.HistoryBorderStyle
		}
	}

	// Render input box - show summary for very long text
	var inputBox string
	if m.showingSummary {
		visualLines := m.calculateVisualLines()
		summaryText := tui.DimHelpStyle.Render(fmt.Sprintf("[text input: %d lines] ", visualLines)) + "Enter: send | Backspace: clear"
		inputBox = currentInputStyle.Width(m.width - 4).Render(summaryText)
	} else {
		inputBox = currentInputStyle.Width(m.width - 4).Render(m.textarea.View())
	}

	if autocompleteView != "" {
		return fmt.Sprintf(
			"%s\n%s\n%s\n%s\n%s",
			header,
			m.viewport.View(),
			autocompleteView,
			inputBox,
			footer,
		)
	}

	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		header,
		m.viewport.View(),
		inputBox,
		footer,
	)
}

// renderAutocomplete renders the autocomplete dropdown.
func (m *Model) renderAutocomplete() string {
	var items []string
	for i, cmd := range m.autocomplete.Filtered() {
		var line string
		if i == m.autocomplete.Index() {
			line = tui.AutocompleteSelectedStyle.Render("> " + cmd.Name)
		} else {
			line = tui.AutocompleteItemStyle.Render(cmd.Name)
		}
		line += " " + tui.AutocompleteDescStyle.Render(cmd.Description)
		items = append(items, line)
	}
	content := strings.Join(items, "\n")
	return tui.AutocompleteBoxStyle.Render(content)
}

// wrapText wraps text to the specified width.
func (m *Model) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	return strings.TrimRight(wrapped, "\n")
}

// renderMarkdown renders content as markdown, falling back to plain text on error.
func (m *Model) renderMarkdown(content string, width int) string {
	if m.mdRenderer == nil {
		return m.wrapText(content, width)
	}

	rendered, err := m.mdRenderer.Render(content)
	if err != nil {
		return m.wrapText(content, width)
	}

	// Trim trailing newlines that glamour adds
	return strings.TrimRight(rendered, "\n")
}

// contentWidth returns the available width for content rendering.
func (m *Model) contentWidth() int {
	width := m.width - 2
	if width < 10 {
		width = 80
	}
	return width
}

// renderSingleMessage renders a single message and returns the rendered string.
func (m *Model) renderSingleMessage(msg api.Message) string {
	var sb strings.Builder
	if msg.Role == "user" {
		sb.WriteString(tui.UserStyle.Render("You: "))
		sb.WriteString(m.wrapText(msg.Content, m.contentWidth()-5))
	} else {
		sb.WriteString(tui.AssistantStyle.Render("Assistant: "))
		sb.WriteString(m.renderMarkdown(msg.Content, m.contentWidth()-11))
	}
	sb.WriteString("\n\n")
	return sb.String()
}

// rebuildRenderedHistory re-renders all completed messages from scratch.
// Called on resize or session load.
func (m *Model) rebuildRenderedHistory() {
	var sb strings.Builder
	for _, msg := range m.messages {
		sb.WriteString(m.renderSingleMessage(msg))
	}
	m.renderedHistory = sb.String()
	m.renderedWidth = m.width
}

// appendRenderedMessage renders and appends a single message to the cache.
func (m *Model) appendRenderedMessage(msg api.Message) {
	m.renderedHistory += m.renderSingleMessage(msg)
}

// updateViewportContent updates the viewport with current messages.
func (m *Model) updateViewportContent() {
	// Check if width changed (need full re-render)
	if m.renderedWidth != m.width {
		m.rebuildRenderedHistory()
	}

	var sb strings.Builder
	sb.WriteString(m.renderedHistory) // Use cached content for completed messages

	if m.state == StateStreaming {
		sb.WriteString(tui.AssistantStyle.Render("Assistant: "))
		if m.currentContent != "" {
			sb.WriteString(m.renderMarkdown(m.currentContent, m.contentWidth()-11))
		}
		sb.WriteString("▋")
	}

	if m.err != nil {
		sb.WriteString(tui.ErrorStyle.Render("Error: "+m.err.Error()) + "\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}
