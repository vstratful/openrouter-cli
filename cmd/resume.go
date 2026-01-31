package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/internal/config"
	"github.com/vstratful/openrouter-cli/internal/tui"
	"github.com/vstratful/openrouter-cli/internal/tui/picker"
)

var lastSession bool

var resumeCmd = &cobra.Command{
	Use:   "resume [session-id]",
	Short: "Resume a previous chat session",
	Long: `Resume a previous chat session.

Usage:
  openrouter resume           # Opens session picker TUI
  openrouter resume <id>      # Resumes session directly by ID
  openrouter resume --last    # Resumes most recent session`,
	RunE: runResume,
}

func init() {
	rootCmd.AddCommand(resumeCmd)
	resumeCmd.Flags().BoolVar(&lastSession, "last", false, "Resume most recent session")
}

func runResume(cmd *cobra.Command, args []string) error {
	// Get API key
	apiKey, isFirstRun, err := getAPIKey()
	if err != nil {
		return err
	}
	if isFirstRun {
		configPath, _ := config.GetConfigPath()
		fmt.Printf("\nAPI key saved to %s\n", configPath)
		fmt.Println("\nYou're all set! Try running:")
		fmt.Println("  openrouter                    # Interactive chat")
		fmt.Println("  openrouter -p \"Hello, world!\" # Single-turn mode")
		return nil
	}

	var session *config.Session

	// Determine which session to resume
	if len(args) > 0 {
		// Direct ID provided
		session, err = config.LoadSession(args[0])
		if err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}
	} else if lastSession {
		// --last flag: get most recent
		session, err = config.GetLatestSession()
		if err != nil {
			return fmt.Errorf("no sessions found")
		}
	} else {
		// Show picker TUI
		summary, err := runSessionPicker()
		if err != nil {
			return fmt.Errorf("failed to show session picker: %w", err)
		}
		if summary == nil {
			// User quit or no sessions
			summaries, _ := config.ListSessions()
			if len(summaries) == 0 {
				fmt.Println("No saved sessions found.")
				fmt.Println("Start a new chat with: openrouter")
				return nil
			}
			// User quit without selecting
			return nil
		}
		session, err = config.LoadSession(summary.ID)
		if err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}
	}

	// Determine model: if user provided -m flag, use that; otherwise use session's model
	modelName := model
	if modelName == "" {
		modelName = session.Model
	}
	if modelName == "" {
		modelName = defaultModel
	}

	return runChatWithSession(apiKey, modelName, session)
}

// sessionPickerModel is a standalone picker for the resume command.
type sessionPickerModel struct {
	picker   picker.Model
	selected *config.SessionSummary
}

func (m sessionPickerModel) Init() tea.Cmd {
	return m.picker.Init()
}

func (m sessionPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if summary := picker.GetSessionSummary(m.picker.SelectedItem()); summary != nil {
				m.selected = summary
			}
			return m, tea.Quit

		case "esc":
			if m.picker.IsFiltering() {
				var cmd tea.Cmd
				m.picker, cmd = m.picker.Update(msg)
				return m, cmd
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m sessionPickerModel) View() string {
	return m.picker.View() + "\n" + tui.HelpStyle.Render("Enter: select | Esc/q: cancel | /: filter")
}

// runSessionPicker shows the session picker TUI and returns the selected session
func runSessionPicker() (*config.SessionSummary, error) {
	summaries, err := config.ListSessions()
	if err != nil {
		return nil, err
	}

	if len(summaries) == 0 {
		return nil, nil
	}

	m := sessionPickerModel{
		picker: picker.NewSessionPicker(summaries, 0, 0),
	}
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	if fm, ok := finalModel.(sessionPickerModel); ok {
		return fm.selected, nil
	}

	return nil, nil
}
