package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestModel_IsImageModel(t *testing.T) {
	tests := []struct {
		name             string
		outputModalities []string
		want             bool
	}{
		{
			name:             "image only",
			outputModalities: []string{"image"},
			want:             true,
		},
		{
			name:             "text and image",
			outputModalities: []string{"text", "image"},
			want:             true,
		},
		{
			name:             "image first",
			outputModalities: []string{"image", "text"},
			want:             true,
		},
		{
			name:             "text only",
			outputModalities: []string{"text"},
			want:             false,
		},
		{
			name:             "empty modalities",
			outputModalities: []string{},
			want:             false,
		},
		{
			name:             "nil modalities",
			outputModalities: nil,
			want:             false,
		},
		{
			name:             "audio and text",
			outputModalities: []string{"audio", "text"},
			want:             false,
		},
		{
			name:             "multiple including image",
			outputModalities: []string{"text", "audio", "image"},
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				Architecture: ModelArchitecture{
					OutputModalities: tt.outputModalities,
				},
			}
			if got := m.IsImageModel(); got != tt.want {
				t.Errorf("IsImageModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_SupportsImageInput(t *testing.T) {
	tests := []struct {
		name            string
		inputModalities []string
		want            bool
	}{
		{
			name:            "image only",
			inputModalities: []string{"image"},
			want:            true,
		},
		{
			name:            "text and image",
			inputModalities: []string{"text", "image"},
			want:            true,
		},
		{
			name:            "text only",
			inputModalities: []string{"text"},
			want:            false,
		},
		{
			name:            "empty modalities",
			inputModalities: []string{},
			want:            false,
		},
		{
			name:            "nil modalities",
			inputModalities: nil,
			want:            false,
		},
		{
			name:            "multiple including image",
			inputModalities: []string{"text", "audio", "image"},
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				Architecture: ModelArchitecture{
					InputModalities: tt.inputModalities,
				},
			}
			if got := m.SupportsImageInput(); got != tt.want {
				t.Errorf("SupportsImageInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_MarshalJSON_StringContent(t *testing.T) {
	msg := Message{Role: "user", Content: "hello"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if raw["role"] != "user" {
		t.Errorf("role = %v, want user", raw["role"])
	}
	if raw["content"] != "hello" {
		t.Errorf("content = %v, want hello", raw["content"])
	}
}

func TestMessage_MarshalJSON_ContentParts(t *testing.T) {
	msg := Message{
		Role: "user",
		ContentParts: []ContentPart{
			{Type: "text", Text: "describe this"},
			{Type: "image_url", ImageURL: &ImageURL{URL: "data:image/png;base64,abc"}},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	parts, ok := raw["content"].([]interface{})
	if !ok {
		t.Fatalf("content is not an array: %T", raw["content"])
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	first := parts[0].(map[string]interface{})
	if first["type"] != "text" {
		t.Errorf("first part type = %v, want text", first["type"])
	}
	if first["text"] != "describe this" {
		t.Errorf("first part text = %v, want 'describe this'", first["text"])
	}
	second := parts[1].(map[string]interface{})
	if second["type"] != "image_url" {
		t.Errorf("second part type = %v, want image_url", second["type"])
	}
	imgURL := second["image_url"].(map[string]interface{})
	if imgURL["url"] != "data:image/png;base64,abc" {
		t.Errorf("image_url.url = %v, want data:image/png;base64,abc", imgURL["url"])
	}
}

func TestMessage_MarshalJSON_ContentPartsOverridesContent(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "this should be ignored",
		ContentParts: []ContentPart{
			{Type: "text", Text: "this wins"},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	// content should be an array, not a string
	if _, ok := raw["content"].([]interface{}); !ok {
		t.Fatalf("content should be array when ContentParts is set, got %T", raw["content"])
	}
}

func TestMessage_UnmarshalJSON_StringContent(t *testing.T) {
	input := `{"role":"assistant","content":"hello world"}`
	var msg Message
	if err := json.Unmarshal([]byte(input), &msg); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if msg.Role != "assistant" {
		t.Errorf("Role = %v, want assistant", msg.Role)
	}
	if msg.Content != "hello world" {
		t.Errorf("Content = %v, want 'hello world'", msg.Content)
	}
	if msg.ContentParts != nil {
		t.Errorf("ContentParts should be nil for string content, got %v", msg.ContentParts)
	}
}

func TestMessage_UnmarshalJSON_ArrayContent(t *testing.T) {
	input := `{"role":"user","content":[{"type":"text","text":"describe"},{"type":"image_url","image_url":{"url":"data:image/png;base64,abc"}}]}`
	var msg Message
	if err := json.Unmarshal([]byte(input), &msg); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if msg.Role != "user" {
		t.Errorf("Role = %v, want user", msg.Role)
	}
	if len(msg.ContentParts) != 2 {
		t.Fatalf("expected 2 ContentParts, got %d", len(msg.ContentParts))
	}
	if msg.ContentParts[0].Type != "text" || msg.ContentParts[0].Text != "describe" {
		t.Errorf("first part = %+v, want text/describe", msg.ContentParts[0])
	}
	if msg.ContentParts[1].Type != "image_url" || msg.ContentParts[1].ImageURL.URL != "data:image/png;base64,abc" {
		t.Errorf("second part = %+v", msg.ContentParts[1])
	}
	// Content should be extracted from text part
	if msg.Content != "describe" {
		t.Errorf("Content = %v, want 'describe'", msg.Content)
	}
}

func TestMessage_MarshalUnmarshal_RoundTrip(t *testing.T) {
	t.Run("string content", func(t *testing.T) {
		original := Message{Role: "user", Content: "hello"}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error = %v", err)
		}
		var decoded Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal error = %v", err)
		}
		if decoded.Role != original.Role || decoded.Content != original.Content {
			t.Errorf("round trip mismatch: got %+v, want %+v", decoded, original)
		}
	})

	t.Run("array content", func(t *testing.T) {
		original := Message{
			Role: "user",
			ContentParts: []ContentPart{
				{Type: "text", Text: "hello"},
				{Type: "image_url", ImageURL: &ImageURL{URL: "data:image/png;base64,abc"}},
			},
		}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error = %v", err)
		}
		var decoded Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal error = %v", err)
		}
		if !reflect.DeepEqual(decoded.ContentParts, original.ContentParts) {
			t.Errorf("ContentParts mismatch:\ngot  %+v\nwant %+v", decoded.ContentParts, original.ContentParts)
		}
	})
}
