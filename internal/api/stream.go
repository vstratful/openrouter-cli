package api

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// StreamReader reads SSE events from a stream.
type StreamReader struct {
	scanner *bufio.Scanner
	body    io.ReadCloser
	done    bool
	err     error
}

// NewStreamReader creates a new StreamReader from an io.ReadCloser.
func NewStreamReader(body io.ReadCloser) *StreamReader {
	return &StreamReader{
		scanner: bufio.NewScanner(body),
		body:    body,
	}
}

// StreamChunk represents a chunk of streamed content.
type StreamChunk struct {
	Content      string
	Done         bool
	FinishReason *string
}

// Next reads the next chunk from the stream.
// Returns nil, nil when the stream is complete.
// Returns nil, error on stream errors.
func (r *StreamReader) Next() (*StreamChunk, error) {
	if r.done {
		return nil, nil
	}

	for r.scanner.Scan() {
		line := r.scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// SSE format: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Stream end signal
		if data == "[DONE]" {
			r.done = true
			return &StreamChunk{Done: true}, nil
		}

		var response ChatResponse
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			// Skip malformed chunks
			continue
		}

		if response.Error != nil {
			r.done = true
			return nil, &APIError{
				Message: response.Error.Message,
			}
		}

		if len(response.Choices) > 0 {
			choice := response.Choices[0]
			return &StreamChunk{
				Content:      choice.Delta.Content,
				FinishReason: choice.FinishReason,
			}, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		r.done = true
		return nil, &StreamError{
			Message: "reading stream",
			Cause:   err,
		}
	}

	// Scanner finished without [DONE] signal
	r.done = true
	return &StreamChunk{Done: true}, nil
}

// Close closes the underlying stream.
func (r *StreamReader) Close() error {
	r.done = true
	return r.body.Close()
}

// ReadAll reads all content from the stream and returns it as a string.
// This is a convenience method for non-TUI usage.
func (r *StreamReader) ReadAll() (string, error) {
	var content strings.Builder

	for {
		chunk, err := r.Next()
		if err != nil {
			return content.String(), err
		}
		if chunk == nil || chunk.Done {
			break
		}
		content.WriteString(chunk.Content)
	}

	return content.String(), nil
}
