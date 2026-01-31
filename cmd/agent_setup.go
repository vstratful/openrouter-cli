package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/internal/config"
)

var agentSetupCmd = &cobra.Command{
	Use:   "agent-setup",
	Short: "Output setup information for AI agents",
	Long: `Output context and setup instructions for AI agents being introduced to this CLI.

This command does NOT require an API key and can be run immediately after installation.

Use this to provide an AI coding assistant with the information it needs to:
- Configure the OpenRouter API key
- Understand available commands
- Generate images

Example:
  openrouter agent-setup`,
	Run: runAgentSetup,
}

func init() {
	rootCmd.AddCommand(agentSetupCmd)
}

func runAgentSetup(cmd *cobra.Command, args []string) {
	// Get the actual resolved config path
	resolvedPath, err := config.GetConfigPath()
	if err != nil {
		resolvedPath = "(unable to determine)"
	}

	// Get OS-specific path description
	configPathDescription := getConfigPathDescription()

	fmt.Printf(`# OpenRouter CLI - Agent Setup Guide

## Configuration

Config file location: %s
(Your resolved path: %s)

Create the config file with:
{
  "api_key": "sk-or-v1-your-key-here",
  "default_model": "%s",
  "default_image_model": "%s"
}

Alternatively, set the OPENROUTER_API_KEY environment variable.

## Commands

Chat (interactive):
  openrouter chat
  openrouter chat -m anthropic/claude-3.5-sonnet

Chat (single-turn):
  openrouter chat -p "Explain Go concurrency"
  openrouter chat -m google/gemini-2.5-flash -p "Hello"

List models:
  openrouter models
  openrouter models --filter claude
  openrouter models --image-only
  openrouter models --details

Resume session:
  openrouter resume
  openrouter resume --last

## Image Generation

Generate images:
  openrouter image -p "A sunset over mountains" -f output.png
  openrouter image -p "A portrait" --aspect-ratio 9:16 -f portrait.png
  openrouter image -p "A landscape" --size 2K -f hd.png
  openrouter image -p "prompt" --base64

Aspect ratios: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
Sizes: 1K, 2K, 4K
`, configPathDescription, resolvedPath, config.DefaultModel, config.DefaultImageModel)
}

// getConfigPathDescription returns the OS-specific human-readable config path.
func getConfigPathDescription() string {
	switch runtime.GOOS {
	case "darwin":
		return "~/Library/Application Support/openrouter/config.json"
	case "windows":
		return `%APPDATA%\openrouter\config.json`
	default:
		return "~/.config/openrouter/config.json"
	}
}
