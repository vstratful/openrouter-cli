package cmd

import (
	"fmt"
	"strings"

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
}

func newChatModel(apiKey, modelName string) chatModel {
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

	return chatModel{
		textarea:     ta,
		spinner:      sp,
		apiKey:       apiKey,
		modelName:    modelName,
		messages:     []Message{},
		session:      config.NewSession(),
		historyIndex: -1,
	}
}

func (m chatModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m chatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	header := fmt.Sprintf("Chat with %s", m.modelName)

	var footer string
	if m.streaming {
		if m.currentContent == "" {
			footer = fmt.Sprintf("%s Thinking...", m.spinner.View())
		} else {
			footer = fmt.Sprintf("%s Streaming...", m.spinner.View())
		}
	} else {
		footer = helpStyle.Render("Enter: send | ↑/↓: history | Esc/Ctrl+C: quit")
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
	p := tea.NewProgram(
		newChatModel(apiKey, modelName),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
