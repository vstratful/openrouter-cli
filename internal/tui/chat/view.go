package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	if m.streaming {
		if m.currentContent == "" {
			footer = modelInfo + sep + m.spinner.View() + " Thinking..."
		} else {
			footer = modelInfo + sep + m.spinner.View() + " Streaming..."
		}
	} else if m.escTimeoutActive {
		// Warning state
		escAction := "clear input"
		if m.escActionIsExit {
			escAction = "exit"
		}
		footer = modelInfo + sep + tui.EscWarningStyle.Render("Press ⎋ again to "+escAction)
	} else if m.history.IsBrowsing() {
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

	// Render autocomplete if showing
	var autocompleteView string
	if m.autocomplete.Visible() && len(m.autocomplete.Filtered()) > 0 {
		autocompleteView = m.renderAutocomplete()
	}

	// Style the input box - change border color based on state
	currentInputStyle := tui.InputBoxStyle
	if m.escTimeoutActive {
		currentInputStyle = tui.EscWarningBoxStyle
	} else if m.history.IsBrowsing() {
		currentInputStyle = tui.HistoryBorderStyle
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

// updateViewportContent updates the viewport with current messages.
func (m *Model) updateViewportContent() {
	var sb strings.Builder
	contentWidth := m.width - 2
	if contentWidth < 10 {
		contentWidth = 80
	}

	for _, msg := range m.messages {
		if msg.Role == "user" {
			sb.WriteString(tui.UserStyle.Render("You: "))
			sb.WriteString(m.wrapText(msg.Content, contentWidth-5))
		} else {
			sb.WriteString(tui.AssistantStyle.Render("Assistant: "))
			sb.WriteString(m.renderMarkdown(msg.Content, contentWidth-11))
		}
		sb.WriteString("\n\n")
	}

	if m.streaming {
		sb.WriteString(tui.AssistantStyle.Render("Assistant: "))
		if m.currentContent != "" {
			sb.WriteString(m.renderMarkdown(m.currentContent, contentWidth-11))
		}
		sb.WriteString("▋")
	}

	if m.err != nil {
		sb.WriteString(tui.ErrorStyle.Render("Error: "+m.err.Error()) + "\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}
