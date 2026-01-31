package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/config"
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
