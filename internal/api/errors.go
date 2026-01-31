package api

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for common API error conditions.
var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrRateLimited        = errors.New("rate limited")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrStreamClosed       = errors.New("stream closed")
)

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

// Unwrap returns the underlying sentinel error based on status code.
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return nil
	}
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
