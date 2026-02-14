package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_Chat(t *testing.T) {
	tests := []struct {
		name       string
		response   ChatResponse
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful response",
			response: ChatResponse{
				Choices: []Choice{
					{
						Message: struct {
							Content string         `json:"content"`
							Images  []ImageContent `json:"images,omitempty"`
						}{
							Content: "Hello, world!",
						},
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "api error in response",
			response: ChatResponse{
				Error: &struct {
					Message string `json:"message"`
				}{
					Message: "rate limit exceeded",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "server error",
			response:   ChatResponse{},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
					t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("Expected Authorization header")
				}

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			resp, err := client.Chat(context.Background(), &ChatRequest{
				Model: "test-model",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("Chat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(resp.Choices) == 0 {
				t.Error("Expected choices in response")
			}
		})
	}
}

func TestClient_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify stream flag in request body
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("Expected Stream to be true")
		}

		// Return SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	reader, err := client.ChatStream(context.Background(), &ChatRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	defer reader.Close()

	content, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if content != "Hello world" {
		t.Errorf("Content = %q, want %q", content, "Hello world")
	}
}

func TestClient_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/models") {
			t.Errorf("Expected /models, got %s", r.URL.Path)
		}

		// Check query parameters
		if r.URL.Query().Get("category") != "programming" {
			t.Errorf("Expected category=programming")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ModelsResponse{
			Data: []Model{
				{ID: "model-1", Name: "Model 1"},
				{ID: "model-2", Name: "Model 2"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	models, err := client.ListModels(context.Background(), &ListModelsOptions{
		Category: "programming",
	})
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}
	if models[0].ID != "model-1" {
		t.Errorf("First model ID = %q, want %q", models[0].ID, "model-1")
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ModelsResponse{
			Data: []Model{{ID: "model-1", Name: "Model 1"}},
		})
	}))
	defer server.Close()

	retryConfig := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
	}

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Retry:   &retryConfig,
	})

	models, err := client.ListModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}
}

func TestClient_RetryExhausted(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	retryConfig := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
	}

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Retry:   &retryConfig,
	})

	_, err := client.ListModels(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error after retry exhaustion")
	}

	// Should have tried initial + 2 retries = 3 attempts
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.ListModels(ctx, nil)
	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}
}

func TestClient_2xxStatusCodes(t *testing.T) {
	statusCodes := []int{200, 201, 202, 204}

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				// 204 has no body
				if code != 204 {
					json.NewEncoder(w).Encode(ModelsResponse{
						Data: []Model{{ID: "model-1"}},
					})
				}
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			models, err := client.ListModels(context.Background(), nil)
			// 204 will fail to decode but shouldn't be an HTTP error
			if code == 204 {
				// Empty response is fine for 204
				return
			}

			if err != nil {
				t.Errorf("ListModels() with status %d error = %v", code, err)
				return
			}

			if len(models) == 0 {
				t.Errorf("Expected models for status %d", code)
			}
		})
	}
}

func TestClient_Chat_MultipartContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&raw)

		var messages []json.RawMessage
		json.Unmarshal(raw["messages"], &messages)
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}

		var msg map[string]json.RawMessage
		json.Unmarshal(messages[0], &msg)

		// content should be an array, not a string
		contentStr := string(msg["content"])
		if contentStr[0] != '[' {
			t.Errorf("expected content to be array, got: %s", contentStr)
		}

		var parts []map[string]interface{}
		json.Unmarshal(msg["content"], &parts)
		if len(parts) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(parts))
		}
		if parts[0]["type"] != "text" {
			t.Errorf("first part type = %v, want text", parts[0]["type"])
		}
		if parts[1]["type"] != "image_url" {
			t.Errorf("second part type = %v, want image_url", parts[1]["type"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ChatResponse{
			Choices: []Choice{
				{
					Message: struct {
						Content string         `json:"content"`
						Images  []ImageContent `json:"images,omitempty"`
					}{
						Content: "Done",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model: "test-model",
		Messages: []Message{
			{
				Role: "user",
				ContentParts: []ContentPart{
					{Type: "text", Text: "describe this"},
					{Type: "image_url", ImageURL: &ImageURL{URL: "data:image/png;base64,abc"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Error("Expected choices in response")
	}
}

func TestDefaultClient(t *testing.T) {
	client := DefaultClient("test-key", 0)
	if client == nil {
		t.Error("DefaultClient returned nil")
	}
}

func TestMockClient(t *testing.T) {
	mock := NewMockClient()

	// Test Chat
	resp, err := mock.Chat(context.Background(), &ChatRequest{
		Model: "test",
	})
	if err != nil {
		t.Errorf("Chat() error = %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Error("Expected choices in mock response")
	}
	if len(mock.ChatCalls) != 1 {
		t.Errorf("Expected 1 Chat call, got %d", len(mock.ChatCalls))
	}

	// Test ListModels
	models, err := mock.ListModels(context.Background(), nil)
	if err != nil {
		t.Errorf("ListModels() error = %v", err)
	}
	if len(models) == 0 {
		t.Error("Expected models in mock response")
	}
	if len(mock.ListModelsCalls) != 1 {
		t.Errorf("Expected 1 ListModels call, got %d", len(mock.ListModelsCalls))
	}

	// Test Reset
	mock.Reset()
	if len(mock.ChatCalls) != 0 {
		t.Error("Expected calls to be cleared after Reset")
	}
}
