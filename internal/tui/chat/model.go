package chat

import (
	"context"

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

	// State machine
	state          ChatState
	escState       escState
	currentContent string
	err            error
	ready          bool
	width          int
	height         int

	// Messages
	messages []api.Message

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
	ta.CharLimit = 0                              // No limit
	ta.SetWidth(config.DefaultTerminalWidth)      // Default width, will be updated on WindowSizeMsg
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
	mdRenderer, _ := tui.NewMarkdownRenderer(config.DefaultTerminalWidth)

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
// It creates a StreamState and returns a command that reads from the API.
// The StreamState is stored in m.activeStream before the command runs.
func (m *Model) StartStream() tea.Cmd {
	// Create the stream state NOW, before returning
	// This ensures it's part of the Model that gets returned from Update
	m.activeStream = NewStreamState()

	// Capture what we need in local variables to avoid pointer issues
	stream := m.activeStream
	client := m.client
	modelName := m.modelName
	messages := m.messages

	return func() tea.Msg {
		go func() {
			ctx := context.Background()
			reader, err := client.ChatStream(ctx, &api.ChatRequest{
				Model:    modelName,
				Messages: messages,
				Stream:   true,
			})
			if err != nil {
				stream.SendError(err)
				stream.Close()
				return
			}
			stream.SetReader(reader)

			for {
				chunk, err := reader.Next()
				if err != nil {
					stream.SendError(err)
					break
				}
				if chunk == nil || chunk.Done {
					break
				}
				if chunk.Content != "" {
					stream.SendChunk(chunk.Content)
				}
			}
			stream.Close()
		}()

		return waitForChunk(stream)
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
	// Capture the stream directly to avoid pointer issues with m
	stream := m.activeStream
	return func() tea.Msg {
		return waitForChunk(stream)
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
