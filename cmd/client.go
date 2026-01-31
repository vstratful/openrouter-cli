package cmd

import (
	"context"
	"fmt"

	"github.com/vstratful/openrouter-cli/internal/api"
)

// Re-export types from internal/api for backward compatibility
type (
	Message           = api.Message
	ChatRequest       = api.ChatRequest
	ChatResponse      = api.ChatResponse
	Choice            = api.Choice
	Model             = api.Model
	ModelPricing      = api.ModelPricing
	ModelArchitecture = api.ModelArchitecture
	TopProviderInfo   = api.TopProviderInfo
	PerRequestLimits  = api.PerRequestLimits
	ModelsResponse    = api.ModelsResponse
	GetModelsOptions  = api.ListModelsOptions
)

// GetModels retrieves available models from the OpenRouter API.
func GetModels(apiKey string, opts *GetModelsOptions) ([]Model, error) {
	client := api.DefaultClient(apiKey)
	return client.ListModels(context.Background(), opts)
}

// runPrompt sends a single prompt to the API and prints the response.
func runPrompt(apiKey, model, prompt string, stream bool) error {
	client := api.DefaultClient(apiKey)
	req := &api.ChatRequest{
		Model: model,
		Messages: []api.Message{
			{Role: "user", Content: prompt},
		},
		Stream: stream,
	}

	if stream {
		reader, err := client.ChatStream(context.Background(), req)
		if err != nil {
			return err
		}
		defer reader.Close()

		// Read and print content as it streams
		var fullContent string
		for {
			chunk, err := reader.Next()
			if err != nil {
				return err
			}
			if chunk == nil || chunk.Done {
				break
			}
			fmt.Print(chunk.Content)
			fullContent += chunk.Content
		}

		// Render final markdown
		if fullContent != "" {
			fmt.Print("\r\033[K") // Clear current line
			renderer, err := NewMarkdownRenderer(80)
			if err == nil {
				rendered, renderErr := renderer.Render(fullContent)
				if renderErr == nil {
					fmt.Print(rendered)
					return nil
				}
			}
			fmt.Println(fullContent)
		} else {
			fmt.Println()
		}
		return nil
	}

	// Non-streaming request
	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		content := resp.Choices[0].Message.Content

		renderer, err := NewMarkdownRenderer(80)
		if err == nil {
			rendered, renderErr := renderer.Render(content)
			if renderErr == nil {
				fmt.Print(rendered)
				return nil
			}
		}
		fmt.Println(content)
	}

	return nil
}

// streamChat streams chat responses to a channel for use with Bubble Tea TUI.
func streamChat(apiKey, model string, messages []Message, chunks chan<- string) error {
	defer close(chunks)

	client := api.DefaultClient(apiKey)
	req := &api.ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	reader, err := client.ChatStream(context.Background(), req)
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		chunk, err := reader.Next()
		if err != nil {
			return err
		}
		if chunk == nil || chunk.Done {
			break
		}
		if chunk.Content != "" {
			chunks <- chunk.Content
		}
	}

	return nil
}
