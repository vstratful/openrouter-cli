package chat

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vstratful/openrouter-cli/internal/api"
)

// Update handles messages for the chat model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Skip updates if showing picker (handled by parent)
	if m.ShowingPicker || m.ShowingModelPicker {
		return m, nil
	}

	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle autocomplete navigation when visible
		if m.autocomplete.Visible() {
			return m.updateAutocomplete(msg)
		}

		// Handle backspace in summary mode - clear the input
		if m.showingSummary && (msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete) {
			m.textarea.Reset()
			m.updateTextareaState()
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			isEmpty := strings.TrimSpace(m.textarea.Value()) == ""
			now := time.Now()

			if m.escTimeoutActive && now.Sub(m.escPressedAt) < 2*time.Second {
				// Second ESC within 2s
				if m.escActionIsExit {
					// Exit the application
					return m, tea.Quit
				}
				// Clear input
				m.textarea.Reset()
				m.updateTextareaState()
				m.escTimeoutActive = false
				m.history.Reset()
				return m, nil
			}

			// First ESC - show prompt, start timer
			m.escPressedAt = now
			m.escTimeoutActive = true
			m.escActionIsExit = isEmpty
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return EscTimeoutMsg{}
			})
		case tea.KeyCtrlU:
			// Unix standard: clear line
			m.textarea.Reset()
			m.updateTextareaState()
			m.history.Reset()
			m.escTimeoutActive = false
			return m, nil
		case tea.KeyPgUp:
			m.viewport.ViewUp()
			return m, nil
		case tea.KeyPgDown:
			m.viewport.ViewDown()
			return m, nil
		case tea.KeyUp:
			if !m.streaming {
				if strings.TrimSpace(m.textarea.Value()) == "" || m.history.IsBrowsing() {
					// Navigate history when empty or already browsing history
					if entry := m.history.Up(m.textarea.Value()); entry != "" {
						m.textarea.SetValue(entry)
					}
				} else {
					// Pass to textarea for cursor navigation in multi-line text
					m.textarea.KeyMap.LinePrevious.SetEnabled(true)
					m.textarea, _ = m.textarea.Update(msg)
					m.textarea.KeyMap.LinePrevious.SetEnabled(false)
				}
				m.updateTextareaState()
			}
			return m, nil
		case tea.KeyDown:
			if !m.streaming {
				if strings.TrimSpace(m.textarea.Value()) == "" || m.history.IsBrowsing() {
					// Navigate history when empty or already browsing history
					if entry := m.history.Down(); entry != "" || m.history.Index() == -1 {
						m.textarea.SetValue(entry)
					}
				} else {
					// Pass to textarea for cursor navigation in multi-line text
					m.textarea.KeyMap.LineNext.SetEnabled(true)
					m.textarea, _ = m.textarea.Update(msg)
					m.textarea.KeyMap.LineNext.SetEnabled(false)
				}
				m.updateTextareaState()
			}
			return m, nil
		case tea.KeyEnter:
			if m.streaming {
				return m, nil
			}
			userInput := strings.TrimSpace(m.textarea.Value())
			if userInput == "" {
				return m, nil
			}

			// Handle /resume command - signal to parent
			if userInput == "/resume" {
				m.textarea.Reset()
				m.updateTextareaState()
				m.ShowingPicker = true
				return m, nil
			}

			// Handle /models command - signal to parent
			if userInput == "/models" {
				m.textarea.Reset()
				m.updateTextareaState()
				m.ShowingModelPicker = true
				return m, nil
			}

			// Handle /quit and /exit commands
			if userInput == "/quit" || userInput == "/exit" {
				return m, tea.Quit
			}

			// Save to history
			m.history.Add(userInput)
			m.session.AppendHistory(userInput)
			m.history.Reset()

			// Save user message to session for resume
			m.session.AppendMessage("user", userInput)

			m.messages = append(m.messages, api.Message{Role: "user", Content: userInput})
			m.textarea.Reset()
			m.updateTextareaState()
			m.streaming = true
			m.currentContent = ""
			m.err = nil

			m.updateViewportContent()

			return m, tea.Batch(m.StartStream(), m.spinner.Tick)
		}

	case tea.MouseMsg:
		// Handle mouse wheel scrolling for viewport (3 lines at a time)
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.viewport.SetYOffset(m.viewport.YOffset - 3)
			return m, nil
		case tea.MouseButtonWheelDown:
			m.viewport.SetYOffset(m.viewport.YOffset + 3)
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Account for border and padding in textarea width
		m.textarea.SetWidth(msg.Width - 8)

		// Update markdown renderer width for proper word wrapping
		contentWidth := msg.Width - 4
		if contentWidth < 10 {
			contentWidth = 80
		}
		if m.mdRenderer != nil {
			m.mdRenderer.SetWidth(contentWidth)
		}

		// Calculate dynamic textarea height
		m.updateTextareaState()
		textareaHeight := m.textarea.Height()

		headerHeight := 1
		inputBoxHeight := textareaHeight + 2 // textarea + border
		footerHeight := 1
		verticalMargins := headerHeight + inputBoxHeight + footerHeight + 1

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargins)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargins
		}

		m.updateViewportContent()

	case StreamChunkMsg:
		m.currentContent += string(msg)
		m.updateViewportContent()
		return m, m.WaitForChunk()

	case StreamDoneMsg:
		if m.currentContent != "" {
			m.messages = append(m.messages, api.Message{Role: "assistant", Content: m.currentContent})
			// Save assistant message to session for resume
			m.session.AppendMessage("assistant", m.currentContent)
		}
		m.streaming = false
		m.currentContent = ""
		m.updateViewportContent()
		return m, nil

	case StreamErrMsg:
		m.err = msg.Err
		m.streaming = false
		m.currentContent = ""
		m.updateViewportContent()
		return m, nil

	case EscTimeoutMsg:
		m.escTimeoutActive = false
		return m, nil

	case spinner.TickMsg:
		if m.streaming {
			m.spinner, spCmd = m.spinner.Update(msg)
			return m, spCmd
		}
	}

	if !m.streaming {
		m.textarea, tiCmd = m.textarea.Update(msg)
		m.autocomplete.Update(m.textarea.Value())
		m.updateTextareaState()
	}
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

// updateAutocomplete handles key events when autocomplete is visible.
func (m Model) updateAutocomplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Hide autocomplete but don't quit
		m.autocomplete.Hide()
		return m, nil

	case tea.KeyUp:
		m.autocomplete.Up()
		return m, nil

	case tea.KeyDown:
		m.autocomplete.Down()
		return m, nil

	case tea.KeyEnter:
		// Fill selected command into textarea
		if selected := m.autocomplete.Select(); selected != "" {
			m.textarea.SetValue(selected)
			m.updateTextareaState()
		}
		m.autocomplete.Hide()
		return m, nil

	default:
		// Pass to textarea then update state
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.autocomplete.Update(m.textarea.Value())
		m.updateTextareaState()
		return m, cmd
	}
}

// calculateVisualLines calculates how many visual rows the content takes.
func (m *Model) calculateVisualLines() int {
	content := m.textarea.Value()
	if content == "" {
		return 1
	}

	// Use a slightly smaller width than the textarea to be conservative
	textWidth := m.width - 10
	if textWidth <= 0 {
		return 1
	}

	totalLines := 0
	for _, line := range strings.Split(content, "\n") {
		if len(line) == 0 {
			totalLines++
			continue
		}
		// Count runes (not bytes) for proper unicode handling
		runeCount := 0
		for range line {
			runeCount++
		}
		// Calculate how many visual rows this line needs
		rows := (runeCount + textWidth - 1) / textWidth
		if rows == 0 {
			rows = 1
		}
		totalLines += rows
	}
	return totalLines
}

// updateTextareaState updates textarea height and summary state.
func (m *Model) updateTextareaState() {
	visualLines := m.calculateVisualLines()

	// Show summary for very long text (more than 2x max height)
	if visualLines > maxTextareaHeight*2 {
		m.showingSummary = true
		return
	}
	m.showingSummary = false

	// Set textarea height to match content (capped at max)
	// Add 1 line buffer when there's wrapped content to prevent scroll issues
	newHeight := visualLines
	if visualLines > 1 {
		newHeight++ // Buffer for potential calculation mismatch
	}
	if newHeight > maxTextareaHeight {
		newHeight = maxTextareaHeight
	}
	if newHeight < 1 {
		newHeight = 1
	}
	m.textarea.SetHeight(newHeight)

	// Update viewport to account for new input height
	if m.ready && m.height > 0 {
		headerHeight := 1
		inputBoxHeight := newHeight + 2 // textarea + border
		footerHeight := 1
		verticalMargins := headerHeight + inputBoxHeight + footerHeight + 1
		m.viewport.Height = m.height - verticalMargins
	}
}
