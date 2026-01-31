package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// DefaultBaseURL is the default OpenRouter API base URL.
	DefaultBaseURL = "https://openrouter.ai/api/v1"

	// DefaultTimeout is the default HTTP timeout.
	DefaultTimeout = 30 * time.Second

	// DefaultStreamTimeout is the default timeout for streaming requests.
	DefaultStreamTimeout = 5 * time.Minute

	// DefaultMaxRetries is the default maximum number of retries.
	DefaultMaxRetries = 3

	// DefaultInitialBackoff is the default initial backoff duration.
	DefaultInitialBackoff = 500 * time.Millisecond

	// DefaultMaxBackoff is the default maximum backoff duration.
	DefaultMaxBackoff = 5 * time.Second
)

// Client is the interface for interacting with the OpenRouter API.
type Client interface {
	// Chat sends a non-streaming chat request.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream sends a streaming chat request and returns a StreamReader.
	ChatStream(ctx context.Context, req *ChatRequest) (*StreamReader, error)

	// ListModels retrieves available models.
	ListModels(ctx context.Context, opts *ListModelsOptions) ([]Model, error)
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// InitialBackoff is the initial backoff duration.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	MaxBackoff time.Duration
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     DefaultMaxRetries,
		InitialBackoff: DefaultInitialBackoff,
		MaxBackoff:     DefaultMaxBackoff,
	}
}

// ClientConfig contains configuration for the API client.
type ClientConfig struct {
	// APIKey is the OpenRouter API key.
	APIKey string

	// BaseURL is the API base URL. Defaults to DefaultBaseURL.
	BaseURL string

	// Timeout is the HTTP timeout for non-streaming requests.
	// Defaults to DefaultTimeout.
	Timeout time.Duration

	// StreamTimeout is the timeout for streaming requests.
	// Defaults to DefaultStreamTimeout.
	StreamTimeout time.Duration

	// HTTPClient is an optional custom HTTP client.
	// If nil, a new client will be created.
	HTTPClient *http.Client

	// Referer is the HTTP-Referer header value.
	Referer string

	// Title is the X-Title header value.
	Title string

	// Retry configures retry behavior. If nil, retries are disabled.
	Retry *RetryConfig
}

// DefaultClient creates a new client with default configuration.
func DefaultClient(apiKey string) Client {
	retryConfig := DefaultRetryConfig()
	return NewClient(ClientConfig{
		APIKey:  apiKey,
		Referer: "https://github.com/vstratful/openrouter-cli",
		Title:   "OpenRouter CLI",
		Retry:   &retryConfig,
	})
}

// NewClient creates a new API client with the given configuration.
func NewClient(cfg ClientConfig) Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.StreamTimeout == 0 {
		cfg.StreamTimeout = DefaultStreamTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	// Create a separate client for streaming with longer timeout
	streamClient := &http.Client{
		Timeout: cfg.StreamTimeout,
	}

	return &client{
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		httpClient:   httpClient,
		streamClient: streamClient,
		referer:      cfg.Referer,
		title:        cfg.Title,
		retry:        cfg.Retry,
	}
}

type client struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	streamClient *http.Client
	referer      string
	title        string
	retry        *RetryConfig
}

func (c *client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if c.referer != "" {
		req.Header.Set("HTTP-Referer", c.referer)
	}
	if c.title != "" {
		req.Header.Set("X-Title", c.title)
	}
}

// isSuccessStatus returns true if the status code indicates success (2xx).
func isSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

// isRetryableStatus returns true if the status code indicates a retryable error.
func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout ||
		code >= 500
}

// shouldRetry determines if a request should be retried.
func (c *client) shouldRetry(err error, statusCode int, attempt int) bool {
	if c.retry == nil || attempt >= c.retry.MaxRetries {
		return false
	}

	// Retry on network errors
	if err != nil {
		return true
	}

	// Retry on retryable HTTP status codes
	return isRetryableStatus(statusCode)
}

// calculateBackoff calculates the backoff duration for a retry attempt.
func (c *client) calculateBackoff(attempt int) time.Duration {
	if c.retry == nil {
		return 0
	}

	backoff := c.retry.InitialBackoff
	for i := 0; i < attempt; i++ {
		backoff *= 2
	}

	if backoff > c.retry.MaxBackoff {
		backoff = c.retry.MaxBackoff
	}

	return backoff
}

// sleep waits for the specified duration, respecting context cancellation.
func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func (c *client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Ensure stream is false for non-streaming request
	chatReq := *req
	chatReq.Stream = false

	jsonBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	return doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
			if err != nil {
				return nil, fmt.Errorf("creating request: %w", err)
			}
			c.setHeaders(httpReq)
			return c.httpClient.Do(httpReq)
		},
		func(resp *http.Response) (*ChatResponse, error) {
			defer resp.Body.Close()
			var chatResp ChatResponse
			if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
				return nil, fmt.Errorf("decoding response: %w", err)
			}
			if chatResp.Error != nil {
				return nil, &APIError{Message: chatResp.Error.Message}
			}
			return &chatResp, nil
		},
	)
}

func (c *client) ChatStream(ctx context.Context, req *ChatRequest) (*StreamReader, error) {
	// Ensure stream is true for streaming request
	chatReq := *req
	chatReq.Stream = true

	jsonBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	return doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
			if err != nil {
				return nil, fmt.Errorf("creating request: %w", err)
			}
			c.setHeaders(httpReq)
			return c.streamClient.Do(httpReq)
		},
		func(resp *http.Response) (*StreamReader, error) {
			// Note: don't close resp.Body here, StreamReader owns it
			return NewStreamReader(resp.Body), nil
		},
	)
}

func (c *client) ListModels(ctx context.Context, opts *ListModelsOptions) ([]Model, error) {
	return doWithRetry(ctx, c,
		func(ctx context.Context) (*http.Response, error) {
			httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
			if err != nil {
				return nil, fmt.Errorf("creating request: %w", err)
			}
			c.setHeaders(httpReq)

			if opts != nil {
				q := httpReq.URL.Query()
				if opts.Category != "" {
					q.Set("category", opts.Category)
				}
				if opts.SupportedParameters != "" {
					q.Set("supported_parameters", opts.SupportedParameters)
				}
				httpReq.URL.RawQuery = q.Encode()
			}

			return c.httpClient.Do(httpReq)
		},
		func(resp *http.Response) ([]Model, error) {
			defer resp.Body.Close()
			var modelsResp ModelsResponse
			if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
				return nil, fmt.Errorf("decoding response: %w", err)
			}
			return modelsResp.Data, nil
		},
	)
}
