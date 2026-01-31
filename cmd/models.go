package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	category            string
	supportedParameters string
	showDetails         bool
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models from OpenRouter",
	Long: `List all available models from the OpenRouter API.

Examples:
  openrouter models                              # List all models
  openrouter models --category programming       # Filter by category
  openrouter models --details                    # Show detailed info`,
	RunE: runModels,
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.Flags().StringVar(&category, "category", "", "Filter by category (e.g., programming, roleplay, marketing)")
	modelsCmd.Flags().StringVar(&supportedParameters, "supported-parameters", "", "Filter by supported parameters")
	modelsCmd.Flags().BoolVar(&showDetails, "details", false, "Show detailed model information")
}

func runModels(cmd *cobra.Command, args []string) error {
	apiKey, isFirstRun, err := getAPIKey()
	if err != nil {
		return err
	}
	if isFirstRun {
		fmt.Println("\nAPI key saved. Run the command again to list models.")
		return nil
	}

	opts := &GetModelsOptions{
		Category:            category,
		SupportedParameters: supportedParameters,
	}

	models, err := GetModels(apiKey, opts)
	if err != nil {
		return err
	}

	if len(models) == 0 {
		fmt.Println("No models found.")
		return nil
	}

	fmt.Printf("Found %d models:\n\n", len(models))

	for _, m := range models {
		if showDetails {
			printModelDetails(m)
		} else {
			printModelSummary(m)
		}
	}

	return nil
}

func printModelSummary(m Model) {
	fmt.Printf("%-50s %s\n", m.ID, m.Name)
}

func printModelDetails(m Model) {
	fmt.Printf("ID: %s\n", m.ID)
	fmt.Printf("Name: %s\n", m.Name)

	if m.ContextLength != nil {
		fmt.Printf("Context Length: %d tokens\n", *m.ContextLength)
	}

	fmt.Printf("Pricing: prompt=$%s/1M tokens, completion=$%s/1M tokens\n",
		formatPricePerMillion(m.Pricing.Prompt), formatPricePerMillion(m.Pricing.Completion))

	if len(m.Architecture.InputModalities) > 0 {
		fmt.Printf("Input: %s\n", strings.Join(m.Architecture.InputModalities, ", "))
	}
	if len(m.Architecture.OutputModalities) > 0 {
		fmt.Printf("Output: %s\n", strings.Join(m.Architecture.OutputModalities, ", "))
	}

	if m.Description != "" {
		desc := m.Description
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		fmt.Printf("Description: %s\n", desc)
	}

	fmt.Println(strings.Repeat("-", 60))
}
