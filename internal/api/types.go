// Package api provides the OpenRouter API client.
package api

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the chat completions API.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Choice represents a completion choice in the response.
type Choice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	FinishReason *string `json:"finish_reason"`
}

// ChatResponse represents the response from the chat completions API.
type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ModelPricing represents pricing information for a model.
type ModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Request    string `json:"request"`
	Image      string `json:"image"`
	Web        string `json:"web_search,omitempty"`
	Audio      string `json:"input_audio,omitempty"`
}

// ModelArchitecture represents the architecture of a model.
type ModelArchitecture struct {
	Tokenizer        string   `json:"tokenizer"`
	InstructType     *string  `json:"instruct_type"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
}

// TopProviderInfo represents information about the top provider.
type TopProviderInfo struct {
	ContextLength       *int `json:"context_length"`
	MaxCompletionTokens *int `json:"max_completion_tokens"`
	IsModerated         bool `json:"is_moderated"`
}

// PerRequestLimits represents per-request token limits.
type PerRequestLimits struct {
	PromptTokens     *int `json:"prompt_tokens"`
	CompletionTokens *int `json:"completion_tokens"`
}

// Model represents an OpenRouter model.
type Model struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Created             int64             `json:"created"`
	Description         string            `json:"description"`
	ContextLength       *int              `json:"context_length"`
	Pricing             ModelPricing      `json:"pricing"`
	Architecture        ModelArchitecture `json:"architecture"`
	TopProvider         TopProviderInfo   `json:"top_provider"`
	PerRequestLimits    *PerRequestLimits `json:"per_request_limits"`
	SupportedParameters []string          `json:"supported_parameters"`
}

// ModelsResponse represents the response from the models API.
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// ListModelsOptions represents options for listing models.
type ListModelsOptions struct {
	Category            string
	SupportedParameters string
}
