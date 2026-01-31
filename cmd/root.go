package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "openrouter",
	Short: "A CLI for interacting with OpenRouter API",
	Long: `OpenRouter CLI - Interact with AI models via the OpenRouter API.

Commands:
  chat      Start interactive chat or send a single prompt
  image     Generate images with image-capable models
  models    List and explore available models
  resume    Continue a previous chat session

Examples:
  openrouter chat                       # Interactive chat mode
  openrouter chat -p "Hello"            # Single-turn query
  openrouter models --details           # List available models
  openrouter image -p "..." -f out.png  # Generate an image`,
}

func init() {
	// No flags on root - all commands define their own
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

// printFirstRunHelp prints the help message shown after first-run API key setup.
func printFirstRunHelp() {
	configPath, _ := config.GetConfigPath()
	fmt.Printf("\nAPI key saved to %s\n", configPath)
	fmt.Println("\nYou're all set! Try running:")
	fmt.Println("  openrouter chat                       # Interactive chat")
	fmt.Println("  openrouter chat -p \"Hello, world!\"    # Single-turn mode")
	fmt.Println("  openrouter models                     # List available models")
	fmt.Println("  openrouter image -p \"...\" -f out.png  # Generate images")
}
