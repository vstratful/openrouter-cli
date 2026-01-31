package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/config"
)

var (
	model  string
	prompt string
	stream bool
)

// defaultModel uses the constant from the config package.
const defaultModel = config.DefaultModel

var rootCmd = &cobra.Command{
	Use:   "openrouter",
	Short: "A CLI for interacting with OpenRouter API",
	Long: `OpenRouter CLI allows you to interact with various AI models
through the OpenRouter API directly from your terminal.

Examples:
  openrouter                                        # Interactive chat mode
  openrouter --model google/gemini-2.5-flash        # Chat with specific model
  openrouter --prompt "Hello, world!"               # Single-turn mode
  openrouter --model anthropic/claude-3.5-sonnet --prompt "Explain Go concurrency"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get API key first - nothing else matters without it
		apiKey, isFirstRun, err := getAPIKey()
		if err != nil {
			return err
		}

		// If this was first-run setup, show success and exit
		if isFirstRun {
			configPath, _ := config.GetConfigPath()
			fmt.Printf("\nAPI key saved to %s\n", configPath)
			fmt.Println("\nYou're all set! Try running:")
			fmt.Println("  openrouter                    # Interactive chat")
			fmt.Println("  openrouter -p \"Hello, world!\" # Single-turn mode")
			return nil
		}

		// Use default model if not specified
		if model == "" {
			model = defaultModel
		}

		// Interactive chat mode when no prompt provided
		if prompt == "" {
			return runChat(apiKey, model)
		}

		// Single-turn mode
		return runPrompt(apiKey, model, prompt, stream)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "", "Model to use (default: "+defaultModel+")")
	rootCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Prompt for single-turn mode (omit for interactive chat)")
	rootCmd.Flags().BoolVarP(&stream, "stream", "s", true, "Stream the response (default: true)")
}

func Execute() error {
	return rootCmd.Execute()
}

// getAPIKey retrieves the API key using the following precedence:
// 1. OPENROUTER_API_KEY environment variable
// 2. Config file
// 3. Interactive prompt (first-run experience)
// Returns the key, a boolean indicating if this was first-run setup, and any error
func getAPIKey() (string, bool, error) {
	// 1. Check environment variable first
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		return key, false, nil
	}

	// 2. Load from config file
	cfg, err := config.Load()
	if err != nil {
		return "", false, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.APIKey != "" {
		return cfg.APIKey, false, nil
	}

	// 3. First-run: prompt user and save
	key, err := config.PromptForAPIKey()
	if err != nil {
		return "", false, err
	}

	cfg.APIKey = key
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	}

	return key, true, nil
}
