package chat

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vstratful/openrouter-cli/internal/api"
	"github.com/vstratful/openrouter-cli/internal/config"
	"github.com/vstratful/openrouter-cli/internal/tui"
)

const maxTextareaHeight = 5

// Message types for tea.Msg
type (
	StreamChunkMsg string
	StreamDoneMsg  string
	StreamErrMsg   struct{ Err error }
	EscTimeoutMsg  struct{}
)

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// API client
	client api.Client

	// State
	messages       []api.Message
	streaming      bool
	currentContent string
	err            error
	ready          bool
	width          int
	height         int

	// Session
	session   *config.Session
	modelName string
	isResumed bool

	// History navigation
	history *HistoryNavigator

	// Autocomplete
	autocomplete *AutocompleteState

	// Input summary mode (for very long text)
	showingSummary bool

	// ESC double-press state
	escPressedAt     time.Time
	escTimeoutActive bool
	escActionIsExit  bool

	// Markdown renderer
	mdRenderer *tui.MarkdownRenderer

	// Stream state
	activeStream *StreamState

	// Picker state (managed by parent)
	ShowingPicker      bool
	ShowingModelPicker bool
}

// Config holds configuration for creating a new chat model.
type Config struct {
	Client          api.Client
	ModelName       string
	ExistingSession *config.Session
}

// New creates a new chat Model.
func New(cfg Config) Model {
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
	mdRenderer, _ := tui.NewMarkdownRenderer(80)

	m := Model{
		textarea:     ta,
		spinner:      sp,
		client:       cfg.Client,
		modelName:    cfg.ModelName,
		messages:     []api.Message{},
		history:      NewHistoryNavigator(),
		autocomplete: NewAutocompleteState(),
		mdRenderer:   mdRenderer,
	}

	// Load existing session or create new one
	if cfg.ExistingSession != nil {
		m.session = cfg.ExistingSession
		m.isResumed = true
		// Restore messages from session
		for _, msg := range cfg.ExistingSession.Messages {
			m.messages = append(m.messages, api.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
		// Set history from session
		m.history.SetHistory(cfg.ExistingSession.History)
	} else {
		m.session = config.NewSession()
		m.session.Model = cfg.ModelName
	}

	return m
}

// Init initializes the chat model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

// Session returns the current session.
func (m *Model) Session() *config.Session {
	return m.session
}

// ModelName returns the current model name.
func (m *Model) ModelName() string {
	return m.modelName
}

// SetModelName sets the model name and updates the session.
func (m *Model) SetModelName(name string) {
	m.modelName = name
	m.session.Model = name
}

// IsResumed returns whether this is a resumed session.
func (m *Model) IsResumed() bool {
	return m.isResumed
}

// Messages returns the current messages.
func (m *Model) Messages() []api.Message {
	return m.messages
}

// SetMessages sets the messages (used when resuming).
func (m *Model) SetMessages(messages []api.Message) {
	m.messages = messages
}

// SetSession sets a new session.
func (m *Model) SetSession(session *config.Session) {
	m.session = session
	m.isResumed = true
	m.messages = []api.Message{}
	for _, msg := range session.Messages {
		m.messages = append(m.messages, api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	m.history.SetHistory(session.History)
	if session.Model != "" {
		m.modelName = session.Model
	}
}

// Err returns the last error.
func (m *Model) Err() error {
	return m.err
}

// SetErr sets an error.
func (m *Model) SetErr(err error) {
	m.err = err
}

// StartStream starts a new streaming request.
func (m *Model) StartStream() tea.Cmd {
	return func() tea.Msg {
		m.activeStream = NewStreamState()

		go func() {
			ctx := context.Background()
			reader, err := m.client.ChatStream(ctx, &api.ChatRequest{
				Model:    m.modelName,
				Messages: m.messages,
				Stream:   true,
			})
			if err != nil {
				m.activeStream.SendError(err)
				m.activeStream.Close()
				return
			}
			m.activeStream.SetReader(reader)

			for {
				chunk, err := reader.Next()
				if err != nil {
					m.activeStream.SendError(err)
					break
				}
				if chunk == nil || chunk.Done {
					break
				}
				if chunk.Content != "" {
					m.activeStream.SendChunk(chunk.Content)
				}
			}
			m.activeStream.Close()
		}()

		return waitForChunk(m.activeStream)
	}
}

func waitForChunk(stream *StreamState) tea.Msg {
	if stream == nil || stream.IsDone() {
		return StreamDoneMsg("")
	}

	select {
	case chunk, ok := <-stream.Chunks():
		if !ok {
			// Channel closed, check for errors
			select {
			case err := <-stream.ErrChan():
				if err != nil {
					return StreamErrMsg{Err: err}
				}
			default:
			}
			return StreamDoneMsg("")
		}
		return StreamChunkMsg(chunk)
	case err := <-stream.ErrChan():
		if err != nil {
			return StreamErrMsg{Err: err}
		}
		return waitForChunk(stream)
	}
}

// WaitForChunk returns a command to wait for the next stream chunk.
func (m *Model) WaitForChunk() tea.Cmd {
	return func() tea.Msg {
		return waitForChunk(m.activeStream)
	}
}

// Run starts the chat TUI.
func Run(client api.Client, modelName string, session *config.Session) error {
	m := New(Config{
		Client:          client,
		ModelName:       modelName,
		ExistingSession: session,
	})

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
