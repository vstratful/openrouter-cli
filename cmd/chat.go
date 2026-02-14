package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vstratful/openrouter-cli/internal/api"
	"github.com/vstratful/openrouter-cli/internal/config"
	"github.com/vstratful/openrouter-cli/internal/tui"
	"github.com/vstratful/openrouter-cli/internal/tui/chat"
	"github.com/vstratful/openrouter-cli/internal/tui/picker"
)

// Message types for async model loading
type modelsLoadedMsg struct {
	models []api.Model
}

type modelsLoadErrorMsg struct {
	err error
}

// loadModelsCmd fetches models asynchronously from the API
func loadModelsCmd(apiKey string) tea.Cmd {
	return func() tea.Msg {
		client := api.DefaultClient(apiKey, timeout)
		models, err := client.ListModels(context.Background(), nil)
		if err != nil {
			return modelsLoadErrorMsg{err: err}
		}
		return modelsLoadedMsg{models: picker.FilterTextModels(models)}
	}
}

// chatWrapper wraps the internal chat model and handles pickers.
type chatWrapper struct {
	chat               chat.Model
	apiKey             string
	showingPicker      bool
	pickerModel        picker.Model
	showingModelPicker bool
	modelPickerModel   picker.Model
	width              int
	height             int
}

func newChatWrapper(apiKey, modelName string, existingSession *config.Session) chatWrapper {
	client := api.DefaultClient(apiKey, timeout)
	chatModel := chat.New(chat.Config{
		Client:          client,
		ModelName:       modelName,
		ExistingSession: existingSession,
	})

	return chatWrapper{
		chat:   chatModel,
		apiKey: apiKey,
	}
}

func (m chatWrapper) Init() tea.Cmd {
	return m.chat.Init()
}

func (m chatWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle model picker mode
	if m.showingModelPicker {
		return m.updateModelPicker(msg)
	}

	// Handle session picker mode
	if m.showingPicker {
		return m.updatePicker(msg)
	}

	// Update window size tracking
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
	}

	// Delegate to chat model
	updatedChat, cmd := m.chat.Update(msg)
	m.chat = updatedChat.(chat.Model)

	// Check if chat wants to show pickers
	if m.chat.ShowingPicker {
		m.chat.ShowingPicker = false
		return m.showSessionPicker()
	}
	if m.chat.ShowingModelPicker {
		m.chat.ShowingModelPicker = false
		return m.showModelPicker()
	}

	return m, cmd
}

func (m chatWrapper) View() string {
	// Show model picker if active
	if m.showingModelPicker {
		return m.modelPickerModel.View() + "\n" + tui.HelpStyle.Render("Enter: select | Esc: cancel | /: filter")
	}

	// Show session picker if active
	if m.showingPicker {
		return m.pickerModel.View() + "\n" + tui.HelpStyle.Render("Enter: select | Esc: cancel | /: filter")
	}

	return m.chat.View()
}

func (m chatWrapper) showSessionPicker() (tea.Model, tea.Cmd) {
	summaries, err := config.ListSessions()
	if err != nil || len(summaries) == 0 {
		m.chat.SetErr(fmt.Errorf("no saved sessions found"))
		return m, nil
	}

	m.pickerModel = picker.NewSessionPicker(summaries, m.width, m.height)
	m.showingPicker = true
	return m, nil
}

func (m chatWrapper) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		var cmd tea.Cmd
		m.pickerModel, cmd = m.pickerModel.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// If filtering is active, let the picker handle it
			if m.pickerModel.IsFiltering() {
				var cmd tea.Cmd
				m.pickerModel, cmd = m.pickerModel.Update(msg)
				return m, cmd
			}
			// Return to chat without selecting
			m.showingPicker = false
			return m, nil

		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			if summary := picker.GetSessionSummary(m.pickerModel.SelectedItem()); summary != nil {
				// Load the selected session
				session, err := config.LoadSession(summary.ID)
				if err != nil {
					m.chat.SetErr(fmt.Errorf("failed to load session: %w", err))
					m.showingPicker = false
					return m, nil
				}

				// Update chat with loaded session
				m.chat.SetSession(session)
				m.showingPicker = false
				return m, nil
			}
		}
	}

	// Delegate to picker
	var cmd tea.Cmd
	m.pickerModel, cmd = m.pickerModel.Update(msg)
	return m, cmd
}

func (m chatWrapper) showModelPicker() (tea.Model, tea.Cmd) {
	m.modelPickerModel = picker.NewModelPicker(m.width, m.height)
	m.showingModelPicker = true
	return m, tea.Batch(loadModelsCmd(m.apiKey), m.modelPickerModel.Spinner.Tick)
}

func (m chatWrapper) updateModelPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		var cmd tea.Cmd
		m.modelPickerModel, cmd = m.modelPickerModel.Update(msg)
		return m, cmd

	case modelsLoadedMsg:
		picker.SetModels(&m.modelPickerModel, msg.models)
		return m, nil

	case modelsLoadErrorMsg:
		m.modelPickerModel.SetError(msg.err)
		return m, nil

	case spinner.TickMsg:
		if m.modelPickerModel.Loading {
			var cmd tea.Cmd
			m.modelPickerModel, cmd = m.modelPickerModel.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// If filtering is active, let the picker handle it
			if !m.modelPickerModel.Loading && m.modelPickerModel.IsFiltering() {
				var cmd tea.Cmd
				m.modelPickerModel, cmd = m.modelPickerModel.Update(msg)
				return m, cmd
			}
			// Return to chat without selecting
			m.showingModelPicker = false
			return m, nil

		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			if m.modelPickerModel.Loading {
				return m, nil
			}
			if model := picker.GetModel(m.modelPickerModel.SelectedItem()); model != nil {
				// Update the model
				m.chat.SetModelName(model.ID)
				m.showingModelPicker = false
				return m, nil
			}
		}
	}

	// Delegate to picker
	var cmd tea.Cmd
	m.modelPickerModel, cmd = m.modelPickerModel.Update(msg)
	return m, cmd
}

func runChat(apiKey, modelName string) error {
	return runChatWithSession(apiKey, modelName, nil)
}

func runChatWithSession(apiKey, modelName string, session *config.Session) error {
	p := tea.NewProgram(
		newChatWrapper(apiKey, modelName, session),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Enable mouse to handle scroll wheel properly
	)

	_, err := p.Run()
	return err
}
