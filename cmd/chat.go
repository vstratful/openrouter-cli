package cmd

import (
	"fmt"
	"strings"

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
)

type streamChunkMsg string
type streamDoneMsg string
type streamErrMsg struct{ err error }

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
}

func newChatModel(apiKey, modelName string, existingSession *config.Session) chatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = ""
	ta.CharLimit = 4096
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)
	// Disable built-in arrow key handling for history navigation
	ta.KeyMap.LineNext.SetEnabled(false)
	ta.KeyMap.LinePrevious.SetEnabled(false)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	m := chatModel{
		textarea:     ta,
		spinner:      sp,
		apiKey:       apiKey,
		modelName:    modelName,
		messages:     []Message{},
		historyIndex: -1,
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
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyUp:
			if !m.streaming {
				m.navigateHistoryUp()
			}
			return m, nil
		case tea.KeyDown:
			if !m.streaming {
				m.navigateHistoryDown()
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
				return m.showSessionPicker()
			}

			// Handle /model command
			if userInput == "/model" {
				m.textarea.Reset()
				return m.showModelPicker()
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
			m.streaming = true
			m.currentContent = ""
			m.err = nil

			m.updateViewportContent()

			return m, tea.Batch(m.startStream(), m.spinner.Tick)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Layout: header(1) + viewport + inputBox(3 + 2 border) + footer(1)
		headerHeight := 1
		inputBoxHeight := 5 // textarea(3) + border(2)
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

		// Account for border and padding in textarea width
		m.textarea.SetWidth(msg.Width - 8)
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

	case spinner.TickMsg:
		if m.streaming {
			m.spinner, spCmd = m.spinner.Update(msg)
			return m, spCmd
		}
	}

	if !m.streaming {
		m.textarea, tiCmd = m.textarea.Update(msg)
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
			sb.WriteString(m.wrapText(msg.Content, contentWidth-11))
		}
		sb.WriteString("\n\n")
	}

	if m.streaming {
		sb.WriteString(assistantStyle.Render("Assistant: "))
		if m.currentContent != "" {
			sb.WriteString(m.wrapText(m.currentContent, contentWidth-11))
		}
		sb.WriteString("▋")
	}

	if m.err != nil {
		sb.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
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

	header := fmt.Sprintf("Chat with %s", m.modelName)
	if m.isResumed {
		header += " (Resumed)"
	}

	var footer string
	if m.streaming {
		if m.currentContent == "" {
			footer = fmt.Sprintf("%s Thinking...", m.spinner.View())
		} else {
			footer = fmt.Sprintf("%s Streaming...", m.spinner.View())
		}
	} else {
		footer = helpStyle.Render("Enter: send | ↑/↓: history | /resume: switch session | /model: change model | Esc: quit")
	}

	// Style the input box
	inputBox := inputBoxStyle.Width(m.width - 4).Render(m.textarea.View())

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
	)

	_, err := p.Run()
	return err
}
