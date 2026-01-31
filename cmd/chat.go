package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vstratful/openrouter-cli/config"
)

var (
	userStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	assistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	inputBoxStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	// Autocomplete styles
	autocompleteBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)
	autocompleteItemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	autocompleteSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	autocompleteDescStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// State-aware styles
	escWarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")). // Coral red - urgent but not alarming
			Bold(true)

	escWarningBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF6B6B")). // Match warning color
				Padding(0, 1)

	historyModeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA")). // Soft purple - "you're in the past"
				Italic(true)

	keyHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6EE7B7")). // Mint green - actionable
			Bold(true)

	dimHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")) // Slightly brighter than current 241

	historyBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#A78BFA")).
				Padding(0, 1)
)

type streamChunkMsg string
type streamDoneMsg string
type streamErrMsg struct{ err error }
type escTimeoutMsg struct{}

type chatModel struct {
	viewport       viewport.Model
	textarea       textarea.Model
	messages       []Message
	streaming      bool
	currentContent string
	spinner        spinner.Model
	apiKey         string
	modelName      string
	err            error
	ready          bool
	width          int
	height         int
	session        *config.Session // Current session (has ID and History)
	historyIndex   int             // -1 = not browsing, otherwise index into history
	currentDraft   string          // Preserve current input when navigating
	isResumed      bool            // Whether this is a resumed session
	showingPicker  bool            // Whether session picker is showing
	pickerModel    *sessionPickerModel
	showingModelPicker bool            // Whether model picker is showing
	modelPickerModel   *modelPickerModel

	// Markdown renderer for assistant messages
	mdRenderer *MarkdownRenderer

	// Command autocomplete state
	showingAutocomplete bool
	autocompleteIndex   int
	filteredCommands    []Command

	// Input summary mode (for very long text)
	showingSummary bool

	// ESC double-press state
	escPressedAt     time.Time // Time of first ESC press
	escTimeoutActive bool      // Whether we're waiting for second ESC
}

const maxTextareaHeight = 5

func newChatModel(apiKey, modelName string, existingSession *config.Session) chatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = ""
	ta.CharLimit = 0 // No limit
	ta.SetWidth(80)  // Default width, will be updated on WindowSizeMsg
	ta.SetHeight(1)  // Start at 1 line, grows dynamically
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)
	// Disable built-in arrow key handling for history navigation
	ta.KeyMap.LineNext.SetEnabled(false)
	ta.KeyMap.LinePrevious.SetEnabled(false)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	// Initialize markdown renderer (ignore error, will fallback to plain text)
	mdRenderer, _ := NewMarkdownRenderer(80)

	m := chatModel{
		textarea:     ta,
		spinner:      sp,
		apiKey:       apiKey,
		modelName:    modelName,
		messages:     []Message{},
		historyIndex: -1,
		mdRenderer:   mdRenderer,
	}

	// Load existing session or create new one
	if existingSession != nil {
		m.session = existingSession
		m.isResumed = true
		// Restore messages from session
		for _, msg := range existingSession.Messages {
			m.messages = append(m.messages, Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	} else {
		m.session = config.NewSession()
		m.session.Model = modelName
	}

	return m
}

func (m chatModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle model picker mode
	if m.showingModelPicker && m.modelPickerModel != nil {
		return m.updateModelPicker(msg)
	}

	// Handle session picker mode
	if m.showingPicker && m.pickerModel != nil {
		return m.updatePicker(msg)
	}

	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle autocomplete navigation when visible
		if m.showingAutocomplete {
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
			// Empty textarea: just quit
			if strings.TrimSpace(m.textarea.Value()) == "" {
				return m, tea.Quit
			}

			now := time.Now()
			if m.escTimeoutActive && now.Sub(m.escPressedAt) < 2*time.Second {
				// Second ESC - clear input
				m.textarea.Reset()
				m.updateTextareaState()
				m.escTimeoutActive = false
				m.historyIndex = -1
				m.currentDraft = ""
				return m, nil
			}

			// First ESC - show prompt, start timer
			m.escPressedAt = now
			m.escTimeoutActive = true
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return escTimeoutMsg{}
			})
		case tea.KeyCtrlU:
			// Unix standard: clear line
			m.textarea.Reset()
			m.updateTextareaState()
			m.historyIndex = -1
			m.currentDraft = ""
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
				if strings.TrimSpace(m.textarea.Value()) == "" || m.historyIndex >= 0 {
					// Navigate history when empty or already browsing history
					m.navigateHistoryUp()
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
				if strings.TrimSpace(m.textarea.Value()) == "" || m.historyIndex >= 0 {
					// Navigate history when empty or already browsing history
					m.navigateHistoryDown()
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

			// Handle /resume command
			if userInput == "/resume" {
				m.textarea.Reset()
				m.updateTextareaState()
				return m.showSessionPicker()
			}

			// Handle /models command
			if userInput == "/models" {
				m.textarea.Reset()
				m.updateTextareaState()
				return m.showModelPicker()
			}

			// Handle /quit and /exit commands
			if userInput == "/quit" || userInput == "/exit" {
				return m, tea.Quit
			}

			// Save to history (skip consecutive duplicates)
			historyLen := len(m.session.History)
			if historyLen == 0 || m.session.History[historyLen-1] != userInput {
				m.session.AppendHistory(userInput)
			}
			m.historyIndex = -1
			m.currentDraft = ""

			// Save user message to session for resume
			m.session.AppendMessage("user", userInput)

			m.messages = append(m.messages, Message{Role: "user", Content: userInput})
			m.textarea.Reset()
			m.updateTextareaState()
			m.streaming = true
			m.currentContent = ""
			m.err = nil

			m.updateViewportContent()

			return m, tea.Batch(m.startStream(), m.spinner.Tick)
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

	case streamChunkMsg:
		m.currentContent += string(msg)
		m.updateViewportContent()
		return m, waitForChunk

	case streamDoneMsg:
		if m.currentContent != "" {
			m.messages = append(m.messages, Message{Role: "assistant", Content: m.currentContent})
			// Save assistant message to session for resume
			m.session.AppendMessage("assistant", m.currentContent)
		}
		m.streaming = false
		m.currentContent = ""
		m.updateViewportContent()
		return m, nil

	case streamErrMsg:
		m.err = msg.err
		m.streaming = false
		m.currentContent = ""
		m.updateViewportContent()
		return m, nil

	case escTimeoutMsg:
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
		m.updateAutocompleteState()
		m.updateTextareaState()
	}
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m *chatModel) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	return strings.TrimRight(wrapped, "\n")
}

func (m *chatModel) navigateHistoryUp() {
	if len(m.session.History) == 0 {
		return
	}

	// First press: save current draft and start at most recent
	if m.historyIndex == -1 {
		m.currentDraft = m.textarea.Value()
		m.historyIndex = len(m.session.History) - 1
	} else if m.historyIndex > 0 {
		// Move to older entry
		m.historyIndex--
	}

	m.textarea.SetValue(m.session.History[m.historyIndex])
}

func (m *chatModel) navigateHistoryDown() {
	if m.historyIndex == -1 {
		return
	}

	if m.historyIndex < len(m.session.History)-1 {
		// Move to newer entry
		m.historyIndex++
		m.textarea.SetValue(m.session.History[m.historyIndex])
	} else {
		// At bottom of history, restore draft
		m.historyIndex = -1
		m.textarea.SetValue(m.currentDraft)
	}
}

// calculateVisualLines calculates how many visual rows the content takes
func (m *chatModel) calculateVisualLines() int {
	content := m.textarea.Value()
	if content == "" {
		return 1
	}

	// Use a slightly smaller width than the textarea to be conservative
	// This accounts for any internal padding the textarea might use
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

// updateTextareaState updates textarea height and summary state
func (m *chatModel) updateTextareaState() {
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

// updateAutocompleteState updates autocomplete visibility based on input
func (m *chatModel) updateAutocompleteState() {
	input := m.textarea.Value()

	// Only show autocomplete for input starting with / and no space
	if !strings.HasPrefix(input, "/") || strings.Contains(input, " ") {
		m.showingAutocomplete = false
		m.filteredCommands = nil
		m.autocompleteIndex = 0
		return
	}

	m.filteredCommands = FilterCommands(input)

	// Don't show autocomplete if input exactly matches a command
	exactMatch := false
	for _, cmd := range m.filteredCommands {
		if strings.EqualFold(cmd.Name, input) {
			exactMatch = true
			break
		}
	}

	m.showingAutocomplete = len(m.filteredCommands) > 0 && !exactMatch

	// Clamp index to valid range
	if m.autocompleteIndex >= len(m.filteredCommands) {
		m.autocompleteIndex = max(0, len(m.filteredCommands)-1)
	}
}

// updateAutocomplete handles key events when autocomplete is visible
func (m chatModel) updateAutocomplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Hide autocomplete but don't quit
		m.showingAutocomplete = false
		return m, nil

	case tea.KeyUp:
		if m.autocompleteIndex > 0 {
			m.autocompleteIndex--
		}
		return m, nil

	case tea.KeyDown:
		if m.autocompleteIndex < len(m.filteredCommands)-1 {
			m.autocompleteIndex++
		}
		return m, nil

	case tea.KeyEnter:
		// Fill selected command into textarea
		if m.autocompleteIndex < len(m.filteredCommands) {
			m.textarea.SetValue(m.filteredCommands[m.autocompleteIndex].Name)
			m.updateTextareaState()
		}
		m.showingAutocomplete = false
		return m, nil

	default:
		// Pass to textarea then update state
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.updateAutocompleteState()
		m.updateTextareaState()
		return m, cmd
	}
}

// renderAutocomplete renders the autocomplete dropdown
func (m *chatModel) renderAutocomplete() string {
	var items []string
	for i, cmd := range m.filteredCommands {
		var line string
		if i == m.autocompleteIndex {
			line = autocompleteSelectedStyle.Render("> " + cmd.Name)
		} else {
			line = autocompleteItemStyle.Render(cmd.Name)
		}
		line += " " + autocompleteDescStyle.Render(cmd.Description)
		items = append(items, line)
	}
	content := strings.Join(items, "\n")
	return autocompleteBoxStyle.Render(content)
}

func (m *chatModel) updateViewportContent() {
	var sb strings.Builder
	contentWidth := m.width - 2
	if contentWidth < 10 {
		contentWidth = 80
	}

	for _, msg := range m.messages {
		if msg.Role == "user" {
			sb.WriteString(userStyle.Render("You: "))
			sb.WriteString(m.wrapText(msg.Content, contentWidth-5))
		} else {
			sb.WriteString(assistantStyle.Render("Assistant: "))
			sb.WriteString(m.renderMarkdown(msg.Content, contentWidth-11))
		}
		sb.WriteString("\n\n")
	}

	if m.streaming {
		sb.WriteString(assistantStyle.Render("Assistant: "))
		if m.currentContent != "" {
			sb.WriteString(m.renderMarkdown(m.currentContent, contentWidth-11))
		}
		sb.WriteString("▋")
	}

	if m.err != nil {
		sb.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

// renderMarkdown renders content as markdown, falling back to plain text on error
func (m *chatModel) renderMarkdown(content string, width int) string {
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

func (m chatModel) showSessionPicker() (tea.Model, tea.Cmd) {
	summaries, err := config.ListSessions()
	if err != nil || len(summaries) == 0 {
		m.err = fmt.Errorf("no saved sessions found")
		m.updateViewportContent()
		return m, nil
	}

	picker := newSessionPickerModel(summaries)
	picker.list.SetWidth(m.width)
	picker.list.SetHeight(m.height - 2)
	m.pickerModel = &picker
	m.showingPicker = true
	return m, nil
}

func (m chatModel) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.pickerModel.list.SetWidth(msg.Width)
		m.pickerModel.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Return to chat without selecting
			m.showingPicker = false
			m.pickerModel = nil
			return m, nil

		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			if i, ok := m.pickerModel.list.SelectedItem().(sessionItem); ok {
				// Load the selected session
				session, err := config.LoadSession(i.summary.ID)
				if err != nil {
					m.err = fmt.Errorf("failed to load session: %w", err)
					m.showingPicker = false
					m.pickerModel = nil
					m.updateViewportContent()
					return m, nil
				}

				// Replace current session with loaded one
				m.session = session
				m.isResumed = true
				m.messages = []Message{}
				for _, msg := range session.Messages {
					m.messages = append(m.messages, Message{
						Role:    msg.Role,
						Content: msg.Content,
					})
				}

				// Update model if session has one
				if session.Model != "" {
					m.modelName = session.Model
				}

				m.historyIndex = -1
				m.currentDraft = ""
				m.showingPicker = false
				m.pickerModel = nil
				m.updateViewportContent()
				return m, nil
			}
		}
	}

	// Delegate to picker
	newPicker, cmd := m.pickerModel.Update(msg)
	if p, ok := newPicker.(sessionPickerModel); ok {
		m.pickerModel = &p
	}
	return m, cmd
}

func (m chatModel) showModelPicker() (tea.Model, tea.Cmd) {
	picker := newModelPickerModel(m.width, m.height)
	m.modelPickerModel = &picker
	m.showingModelPicker = true
	return m, tea.Batch(loadModelsCmd(m.apiKey), m.modelPickerModel.spinner.Tick)
}

func (m chatModel) updateModelPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.modelPickerModel != nil {
			m.modelPickerModel.width = msg.Width
			m.modelPickerModel.height = msg.Height
			if !m.modelPickerModel.loading {
				m.modelPickerModel.list.SetWidth(msg.Width)
				m.modelPickerModel.list.SetHeight(msg.Height - 2)
			}
		}
		return m, nil

	case modelsLoadedMsg, modelsLoadErrorMsg:
		// Delegate to picker
		newPicker, cmd := m.modelPickerModel.Update(msg)
		m.modelPickerModel = &newPicker
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// If filtering is active, let the list handle it
			if !m.modelPickerModel.loading && m.modelPickerModel.list.FilterState() == list.Filtering {
				newPicker, cmd := m.modelPickerModel.Update(msg)
				m.modelPickerModel = &newPicker
				return m, cmd
			}
			// Return to chat without selecting
			m.showingModelPicker = false
			m.modelPickerModel = nil
			return m, nil

		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			if m.modelPickerModel.loading {
				return m, nil
			}
			if i, ok := m.modelPickerModel.list.SelectedItem().(modelItem); ok {
				// Update the model
				m.modelName = i.model.ID
				m.session.Model = i.model.ID
				m.showingModelPicker = false
				m.modelPickerModel = nil
				return m, nil
			}
		}
	}

	// Delegate to picker
	newPicker, cmd := m.modelPickerModel.Update(msg)
	m.modelPickerModel = &newPicker
	return m, cmd
}

func (m chatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Show model picker if active
	if m.showingModelPicker && m.modelPickerModel != nil {
		return m.modelPickerModel.View() + "\n" + helpStyle.Render("Enter: select | Esc: cancel | /: filter")
	}

	// Show session picker if active
	if m.showingPicker && m.pickerModel != nil {
		return m.pickerModel.View() + "\n" + helpStyle.Render("Enter: select | Esc: cancel | /: filter")
	}

	// Header - minimal, just show resumed status if applicable
	var header string
	if m.isResumed {
		header = helpStyle.Render("(Resumed session)")
	}

	// Footer - show model name and status
	var footer string
	modelInfo := dimHelpStyle.Render(m.modelName)
	sep := dimHelpStyle.Render(" • ")

	if m.streaming {
		if m.currentContent == "" {
			footer = modelInfo + sep + m.spinner.View() + " Thinking..."
		} else {
			footer = modelInfo + sep + m.spinner.View() + " Streaming..."
		}
	} else if m.escTimeoutActive {
		// Warning state
		footer = modelInfo + sep + escWarningStyle.Render("Press ⎋ again to clear input")
	} else if m.historyIndex >= 0 {
		// History browsing mode
		historyPos := fmt.Sprintf("browsing history (%d/%d)",
			len(m.session.History)-m.historyIndex, len(m.session.History))
		footer = modelInfo + sep + historyModeStyle.Render(historyPos) +
			sep + dimHelpStyle.Render("↑↓: navigate • Enter: use • ⎋: cancel")
	} else {
		// Normal state with styled hints
		hints := []string{
			keyHintStyle.Render("Enter") + dimHelpStyle.Render(": send"),
			keyHintStyle.Render("↑↓") + dimHelpStyle.Render(": history"),
			keyHintStyle.Render("/") + dimHelpStyle.Render(": commands"),
		}
		footer = modelInfo + sep + strings.Join(hints, sep)
	}

	// Render autocomplete if showing
	var autocompleteView string
	if m.showingAutocomplete && len(m.filteredCommands) > 0 {
		autocompleteView = m.renderAutocomplete()
	}

	// Style the input box - change border color based on state
	currentInputStyle := inputBoxStyle
	if m.escTimeoutActive {
		currentInputStyle = escWarningBoxStyle
	} else if m.historyIndex >= 0 {
		currentInputStyle = historyBorderStyle
	}

	// Render input box - show summary for very long text
	var inputBox string
	if m.showingSummary {
		visualLines := m.calculateVisualLines()
		summaryText := dimHelpStyle.Render(fmt.Sprintf("[text input: %d lines] ", visualLines)) + "Enter: send | Backspace: clear"
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

type streamState struct {
	chunks  chan string
	errChan chan error
	done    bool
}

var activeStream *streamState

func (m chatModel) startStream() tea.Cmd {
	return func() tea.Msg {
		chunks := make(chan string, 100)
		errChan := make(chan error, 1)

		activeStream = &streamState{
			chunks:  chunks,
			errChan: errChan,
		}

		go func() {
			err := streamChat(m.apiKey, m.modelName, m.messages, chunks)
			if err != nil {
				errChan <- err
			}
			close(errChan)
		}()

		return waitForChunk()
	}
}

func waitForChunk() tea.Msg {
	if activeStream == nil {
		return nil
	}

	select {
	case chunk, ok := <-activeStream.chunks:
		if !ok {
			// Channel closed, check for errors
			select {
			case err := <-activeStream.errChan:
				if err != nil {
					return streamErrMsg{err: err}
				}
			default:
			}
			activeStream = nil
			return streamDoneMsg("")
		}
		return streamChunkMsg(chunk)
	case err := <-activeStream.errChan:
		if err != nil {
			activeStream = nil
			return streamErrMsg{err: err}
		}
		return waitForChunk()
	}
}

func runChat(apiKey, modelName string) error {
	return runChatWithSession(apiKey, modelName, nil)
}

func runChatWithSession(apiKey, modelName string, session *config.Session) error {
	p := tea.NewProgram(
		newChatModel(apiKey, modelName, session),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Enable mouse to handle scroll wheel properly
	)

	_, err := p.Run()
	return err
}
