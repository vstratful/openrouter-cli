package api

import "fmt"

// APIError represents an error from the OpenRouter API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

// StreamError represents an error that occurred during streaming.
type StreamError struct {
	Message string
	Cause   error
}

func (e *StreamError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("stream error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("stream error: %s", e.Message)
}

func (e *StreamError) Unwrap() error {
	return e.Cause
}
