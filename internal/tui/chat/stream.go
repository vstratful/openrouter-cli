package chat

import (
	"sync"

	"github.com/vstratful/openrouter-cli/internal/api"
)

// StreamState manages the state of an active stream.
// This replaces the global activeStream variable for better encapsulation.
type StreamState struct {
	mu      sync.Mutex
	chunks  chan string
	errChan chan error
	done    bool
	reader  *api.StreamReader
}

// NewStreamState creates a new StreamState.
func NewStreamState() *StreamState {
	return &StreamState{
		chunks:  make(chan string, 100),
		errChan: make(chan error, 1),
	}
}

// Chunks returns the channel for receiving stream chunks.
func (s *StreamState) Chunks() <-chan string {
	return s.chunks
}

// ErrChan returns the channel for receiving stream errors.
func (s *StreamState) ErrChan() <-chan error {
	return s.errChan
}

// SetReader sets the stream reader.
func (s *StreamState) SetReader(reader *api.StreamReader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reader = reader
}

// SendChunk sends a chunk to the chunks channel.
func (s *StreamState) SendChunk(chunk string) {
	s.chunks <- chunk
}

// SendError sends an error to the error channel.
func (s *StreamState) SendError(err error) {
	select {
	case s.errChan <- err:
	default:
	}
}

// Close marks the stream as done and closes channels.
func (s *StreamState) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.done {
		s.done = true
		close(s.chunks)
		close(s.errChan)
		if s.reader != nil {
			s.reader.Close()
		}
	}
}

// IsDone returns whether the stream is done.
func (s *StreamState) IsDone() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

// Cancel cancels the stream by closing the reader.
func (s *StreamState) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reader != nil && !s.done {
		s.reader.Close()
	}
}
