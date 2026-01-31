package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/internal/config"
)

var (
	chatModel  string
	chatPrompt string
	chatStream bool
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat or send a single prompt",
	Long: `Start a conversation with an AI model.

Modes:
  Interactive (default): Full TUI chat interface with history and model switching
  Single-turn (--prompt): Send one message, get response, and exit

Examples:
  openrouter chat                                 # Interactive chat
  openrouter chat -m anthropic/claude-3.5-sonnet  # With specific model
  openrouter chat -p "Explain Go concurrency"     # Single-turn mode
  openrouter chat -p "Hello" --stream=false       # Without streaming`,
	RunE: runChatCommand,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "", "Model to use (default: "+config.DefaultModel+")")
	chatCmd.Flags().StringVarP(&chatPrompt, "prompt", "p", "", "Prompt for single-turn mode (omit for interactive chat)")
	chatCmd.Flags().BoolVarP(&chatStream, "stream", "s", true, "Stream the response (default: true)")
}

func runChatCommand(cmd *cobra.Command, args []string) error {
	// Get API key first - nothing else matters without it
	apiKey, cfg, isFirstRun, err := getAPIKey()
	if err != nil {
		return err
	}

	// If this was first-run setup, show success and exit
	if isFirstRun {
		printFirstRunHelp()
		return nil
	}

	// Use default model if not specified
	modelName := chatModel
	if modelName == "" {
		modelName = cfg.DefaultModel
	}

	// Interactive chat mode when no prompt provided
	if chatPrompt == "" {
		return runChat(apiKey, modelName)
	}

	// Single-turn mode
	return runPrompt(apiKey, modelName, chatPrompt, chatStream)
}
