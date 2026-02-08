// Package api provides the OpenRouter API client.
package api

import "encoding/json"

// ContentPart represents a single part of a multipart message content.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// Message represents a chat message.
// Content is used for simple string messages. ContentParts is used for
// multipart messages (e.g., text + images). When both are set, ContentParts
// takes precedence during marshaling.
type Message struct {
	Role         string        `json:"role"`
	Content      string        `json:"-"`
	ContentParts []ContentPart `json:"-"`
}

// MarshalJSON implements custom JSON marshaling for Message.
// When ContentParts is populated, content is serialized as an array.
// Otherwise, content is serialized as a string.
func (m Message) MarshalJSON() ([]byte, error) {
	if len(m.ContentParts) > 0 {
		return json.Marshal(struct {
			Role    string        `json:"role"`
			Content []ContentPart `json:"content"`
		}{
			Role:    m.Role,
			Content: m.ContentParts,
		})
	}
	return json.Marshal(struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{
		Role:    m.Role,
		Content: m.Content,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Message.
// It detects whether content is a string or array and populates fields accordingly.
func (m *Message) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role

	if len(raw.Content) == 0 {
		return nil
	}

	// Try string first
	var s string
	if err := json.Unmarshal(raw.Content, &s); err == nil {
		m.Content = s
		return nil
	}

	// Try array of content parts
	var parts []ContentPart
	if err := json.Unmarshal(raw.Content, &parts); err != nil {
		return err
	}
	m.ContentParts = parts
	// Extract text content into Content for convenience
	for _, p := range parts {
		if p.Type == "text" {
			m.Content = p.Text
			break
		}
	}
	return nil
}

// ImageConfig represents configuration for image generation.
type ImageConfig struct {
	AspectRatio string `json:"aspect_ratio,omitempty"` // e.g., "1:1", "16:9"
	Size        string `json:"size,omitempty"`         // e.g., "1K", "2K", "4K"
}

// ChatRequest represents a request to the chat completions API.
type ChatRequest struct {
	Model       string       `json:"model"`
	Messages    []Message    `json:"messages"`
	Stream      bool         `json:"stream"`
	Modalities  []string     `json:"modalities,omitempty"`
	ImageConfig *ImageConfig `json:"image_config,omitempty"`
}

// ImageURL represents an image URL in the response.
type ImageURL struct {
	URL string `json:"url"` // data:image/png;base64,...
}

// ImageContent represents image content in the response.
type ImageContent struct {
	Type     string   `json:"type"`      // "image_url"
	ImageURL ImageURL `json:"image_url"`
}

// Choice represents a completion choice in the response.
type Choice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	Message struct {
		Content string         `json:"content"`
		Images  []ImageContent `json:"images,omitempty"`
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

// IsImageModel returns true if the model supports image output.
func (m *Model) IsImageModel() bool {
	for _, mod := range m.Architecture.OutputModalities {
		if mod == "image" {
			return true
		}
	}
	return false
}

// SupportsImageInput returns true if the model accepts image input.
func (m *Model) SupportsImageInput() bool {
	for _, mod := range m.Architecture.InputModalities {
		if mod == "image" {
			return true
		}
	}
	return false
}

// IsTextOnlyModel returns true if the model supports text output but not image output.
func (m *Model) IsTextOnlyModel() bool {
	hasText := false
	for _, mod := range m.Architecture.OutputModalities {
		if mod == "text" {
			hasText = true
			break
		}
	}
	return hasText && !m.IsImageModel()
}
