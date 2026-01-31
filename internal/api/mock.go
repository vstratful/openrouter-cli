package api

import (
	"context"
)

// MockClient is a mock implementation of the Client interface for testing.
type MockClient struct {
	// ChatFunc is called when Chat is invoked.
	ChatFunc func(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStreamFunc is called when ChatStream is invoked.
	ChatStreamFunc func(ctx context.Context, req *ChatRequest) (*StreamReader, error)

	// ListModelsFunc is called when ListModels is invoked.
	ListModelsFunc func(ctx context.Context, opts *ListModelsOptions) ([]Model, error)

	// ChatCalls records all calls to Chat.
	ChatCalls []ChatCall

	// ChatStreamCalls records all calls to ChatStream.
	ChatStreamCalls []ChatStreamCall

	// ListModelsCalls records all calls to ListModels.
	ListModelsCalls []ListModelsCall
}

// ChatCall records a call to Chat.
type ChatCall struct {
	Ctx context.Context
	Req *ChatRequest
}

// ChatStreamCall records a call to ChatStream.
type ChatStreamCall struct {
	Ctx context.Context
	Req *ChatRequest
}

// ListModelsCall records a call to ListModels.
type ListModelsCall struct {
	Ctx  context.Context
	Opts *ListModelsOptions
}

// NewMockClient creates a new MockClient with default implementations.
func NewMockClient() *MockClient {
	return &MockClient{
		ChatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{
				Choices: []Choice{
					{
						Message: struct {
							Content string `json:"content"`
						}{
							Content: "mock response",
						},
					},
				},
			}, nil
		},
		ChatStreamFunc: func(ctx context.Context, req *ChatRequest) (*StreamReader, error) {
			return nil, &StreamError{Message: "mock streaming not implemented"}
		},
		ListModelsFunc: func(ctx context.Context, opts *ListModelsOptions) ([]Model, error) {
			return []Model{
				{ID: "mock-model", Name: "Mock Model"},
			}, nil
		},
	}
}

// Chat implements Client.Chat.
func (m *MockClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	m.ChatCalls = append(m.ChatCalls, ChatCall{Ctx: ctx, Req: req})
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, req)
	}
	return nil, nil
}

// ChatStream implements Client.ChatStream.
func (m *MockClient) ChatStream(ctx context.Context, req *ChatRequest) (*StreamReader, error) {
	m.ChatStreamCalls = append(m.ChatStreamCalls, ChatStreamCall{Ctx: ctx, Req: req})
	if m.ChatStreamFunc != nil {
		return m.ChatStreamFunc(ctx, req)
	}
	return nil, nil
}

// ListModels implements Client.ListModels.
func (m *MockClient) ListModels(ctx context.Context, opts *ListModelsOptions) ([]Model, error) {
	m.ListModelsCalls = append(m.ListModelsCalls, ListModelsCall{Ctx: ctx, Opts: opts})
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx, opts)
	}
	return nil, nil
}

// Reset clears all recorded calls.
func (m *MockClient) Reset() {
	m.ChatCalls = nil
	m.ChatStreamCalls = nil
	m.ListModelsCalls = nil
}
