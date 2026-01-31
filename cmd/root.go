package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/internal/config"
)

var (
	model  string
	prompt string
	stream bool
)

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
		apiKey, cfg, isFirstRun, err := getAPIKey()
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
			model = cfg.DefaultModel
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
	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "", "Model to use (default: "+config.DefaultModel+")")
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
// Returns the key, the loaded config, a boolean indicating if this was first-run setup, and any error
func getAPIKey() (string, *config.Config, bool, error) {
	// 1. Load config first (we need it regardless)
	cfg, err := config.Load()
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Check environment variable
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		return key, cfg, false, nil
	}

	// 3. Use config file if API key exists
	if cfg.APIKey != "" {
		return cfg.APIKey, cfg, false, nil
	}

	// 4. First-run: prompt user and save with defaults
	key, err := config.PromptForAPIKey()
	if err != nil {
		return "", nil, false, err
	}

	cfg.APIKey = key
	cfg.DefaultModel = config.DefaultModel
	cfg.DefaultImageModel = config.DefaultImageModel
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	}

	return key, cfg, true, nil
}
