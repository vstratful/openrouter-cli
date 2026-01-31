package picker

import (
	"testing"
	"time"

	"github.com/vstratful/openrouter-cli/internal/api"
	"github.com/vstratful/openrouter-cli/internal/config"
)

func TestNewSessionPicker(t *testing.T) {
	summaries := []config.SessionSummary{
		{
			ID:           "session-1",
			Model:        "gpt-4",
			UpdatedAt:    time.Now(),
			MessageCount: 5,
			Preview:      "Hello, world!",
		},
		{
			ID:           "session-2",
			UpdatedAt:    time.Now().Add(-time.Hour),
			MessageCount: 10,
			Preview:      "Another session",
		},
	}

	m := NewSessionPicker(summaries, 80, 24)

	if m.Loading {
		t.Error("NewSessionPicker() should not be in loading state")
	}

	if len(m.List.Items()) != 2 {
		t.Errorf("NewSessionPicker() list has %d items, want 2", len(m.List.Items()))
	}

	if m.List.Title != "Resume a previous session" {
		t.Errorf("NewSessionPicker() title = %q, want %q", m.List.Title, "Resume a previous session")
	}
}

func TestSessionItem(t *testing.T) {
	summary := config.SessionSummary{
		ID:           "session-1",
		Model:        "gpt-4",
		UpdatedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		MessageCount: 5,
		Preview:      "Hello, world!",
	}

	item := SessionItem{Summary: summary}

	// Test Title
	expectedTitle := "Jan 15, 10:30"
	if item.Title() != expectedTitle {
		t.Errorf("SessionItem.Title() = %q, want %q", item.Title(), expectedTitle)
	}

	// Test Description with model
	desc := item.Description()
	if desc == "" {
		t.Error("SessionItem.Description() should not be empty")
	}
	if !contains(desc, "gpt-4") {
		t.Errorf("SessionItem.Description() = %q, should contain model name", desc)
	}
	if !contains(desc, "5 messages") {
		t.Errorf("SessionItem.Description() = %q, should contain message count", desc)
	}

	// Test Description without model
	summary.Model = ""
	item = SessionItem{Summary: summary}
	desc = item.Description()
	if contains(desc, "[") {
		t.Errorf("SessionItem.Description() = %q, should not have model brackets when model is empty", desc)
	}

	// Test FilterValue
	if item.FilterValue() != "Hello, world!" {
		t.Errorf("SessionItem.FilterValue() = %q, want %q", item.FilterValue(), "Hello, world!")
	}
}

func TestGetSessionSummary(t *testing.T) {
	summary := config.SessionSummary{ID: "test-session"}
	item := SessionItem{Summary: summary}

	t.Run("returns summary from SessionItem", func(t *testing.T) {
		result := GetSessionSummary(item)
		if result == nil {
			t.Fatal("GetSessionSummary() returned nil, want summary")
		}
		if result.ID != "test-session" {
			t.Errorf("GetSessionSummary().ID = %q, want %q", result.ID, "test-session")
		}
	})

	t.Run("returns nil for non-SessionItem", func(t *testing.T) {
		// Use a ModelItem instead
		modelItem := ModelItem{Model: api.Model{ID: "test-model"}}
		result := GetSessionSummary(modelItem)
		if result != nil {
			t.Errorf("GetSessionSummary() returned %v, want nil", result)
		}
	})
}

func TestNewModelPicker(t *testing.T) {
	m := NewModelPicker(80, 24)

	if !m.Loading {
		t.Error("NewModelPicker() should be in loading state")
	}

	if m.Width != 80 {
		t.Errorf("NewModelPicker().Width = %d, want 80", m.Width)
	}

	if m.Height != 24 {
		t.Errorf("NewModelPicker().Height = %d, want 24", m.Height)
	}
}

func TestSetModels(t *testing.T) {
	m := NewModelPicker(80, 24)
	contextLen := 128000

	models := []api.Model{
		{
			ID:   "gpt-4",
			Name: "GPT-4",
			Pricing: api.ModelPricing{
				Prompt:     "0.00003",
				Completion: "0.00006",
			},
			ContextLength: &contextLen,
		},
		{
			ID:   "claude-3",
			Name: "Claude 3",
		},
	}

	SetModels(&m, models)

	if m.Loading {
		t.Error("SetModels() should clear loading state")
	}

	if len(m.List.Items()) != 2 {
		t.Errorf("SetModels() list has %d items, want 2", len(m.List.Items()))
	}

	if m.List.Title != "Select a model" {
		t.Errorf("SetModels() title = %q, want %q", m.List.Title, "Select a model")
	}
}

func TestModelItem(t *testing.T) {
	contextLen := 128000
	model := api.Model{
		ID:   "gpt-4",
		Name: "GPT-4",
		Pricing: api.ModelPricing{
			Prompt:     "0.00003",
			Completion: "0.00006",
		},
		ContextLength: &contextLen,
	}

	item := ModelItem{Model: model}

	// Test Title
	if item.Title() != "gpt-4" {
		t.Errorf("ModelItem.Title() = %q, want %q", item.Title(), "gpt-4")
	}

	// Test Description
	desc := item.Description()
	if !contains(desc, "GPT-4") {
		t.Errorf("ModelItem.Description() = %q, should contain name", desc)
	}
	if !contains(desc, "128k ctx") {
		t.Errorf("ModelItem.Description() = %q, should contain context length", desc)
	}
	if !contains(desc, "per 1M tokens") {
		t.Errorf("ModelItem.Description() = %q, should contain pricing", desc)
	}

	// Test FilterValue
	if !contains(item.FilterValue(), "gpt-4") {
		t.Errorf("ModelItem.FilterValue() = %q, should contain ID", item.FilterValue())
	}
	if !contains(item.FilterValue(), "GPT-4") {
		t.Errorf("ModelItem.FilterValue() = %q, should contain name", item.FilterValue())
	}
}

func TestGetModel(t *testing.T) {
	model := api.Model{ID: "test-model"}
	item := ModelItem{Model: model}

	t.Run("returns model from ModelItem", func(t *testing.T) {
		result := GetModel(item)
		if result == nil {
			t.Fatal("GetModel() returned nil, want model")
		}
		if result.ID != "test-model" {
			t.Errorf("GetModel().ID = %q, want %q", result.ID, "test-model")
		}
	})

	t.Run("returns nil for non-ModelItem", func(t *testing.T) {
		sessionItem := SessionItem{Summary: config.SessionSummary{ID: "test"}}
		result := GetModel(sessionItem)
		if result != nil {
			t.Errorf("GetModel() returned %v, want nil", result)
		}
	})
}

func TestFormatPricePerMillion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.00003", "30.00"},
		{"0.00006", "60.00"},
		{"0.000001", "1.00"},
		{"0.0000001", "0.10"},
		{"0.00000001", "0.01"},
		{"0", "0"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FormatPricePerMillion(tt.input)
			if result != tt.expected {
				t.Errorf("FormatPricePerMillion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHasTextModality(t *testing.T) {
	tests := []struct {
		name       string
		modalities []string
		expected   bool
	}{
		{"has text", []string{"text", "image"}, true},
		{"text only", []string{"text"}, true},
		{"no text", []string{"image", "audio"}, false},
		{"empty", []string{}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasTextModality(tt.modalities)
			if result != tt.expected {
				t.Errorf("HasTextModality(%v) = %v, want %v", tt.modalities, result, tt.expected)
			}
		})
	}
}

func TestFilterTextModels(t *testing.T) {
	models := []api.Model{
		{
			ID: "text-model",
			Architecture: api.ModelArchitecture{
				InputModalities:  []string{"text"},
				OutputModalities: []string{"text"},
			},
		},
		{
			ID: "image-model",
			Architecture: api.ModelArchitecture{
				InputModalities:  []string{"image"},
				OutputModalities: []string{"text"},
			},
		},
		{
			ID: "multimodal-model",
			Architecture: api.ModelArchitecture{
				InputModalities:  []string{"text", "image"},
				OutputModalities: []string{"text", "image"},
			},
		},
	}

	filtered := FilterTextModels(models)

	if len(filtered) != 2 {
		t.Errorf("FilterTextModels() returned %d models, want 2", len(filtered))
	}

	// Check that the right models were kept
	hasTextModel := false
	hasMultimodal := false
	for _, m := range filtered {
		if m.ID == "text-model" {
			hasTextModel = true
		}
		if m.ID == "multimodal-model" {
			hasMultimodal = true
		}
	}

	if !hasTextModel {
		t.Error("FilterTextModels() should include text-model")
	}
	if !hasMultimodal {
		t.Error("FilterTextModels() should include multimodal-model")
	}
}

func TestPickerModel(t *testing.T) {
	t.Run("New creates picker with items", func(t *testing.T) {
		summaries := []config.SessionSummary{
			{ID: "1", Preview: "test"},
		}
		items := make([]any, len(summaries))
		for i, s := range summaries {
			items[i] = SessionItem{Summary: s}
		}

		m := NewSessionPicker(summaries, 80, 24)

		if m.Loading {
			t.Error("New() should not be in loading state")
		}
		if m.Width != 80 {
			t.Errorf("New().Width = %d, want 80", m.Width)
		}
	})

	t.Run("NewLoading creates picker in loading state", func(t *testing.T) {
		m := NewLoading(80, 24)

		if !m.Loading {
			t.Error("NewLoading() should be in loading state")
		}
	})

	t.Run("SetError clears loading and sets error", func(t *testing.T) {
		m := NewLoading(80, 24)
		testErr := &testError{msg: "test error"}
		m.SetError(testErr)

		if m.Loading {
			t.Error("SetError() should clear loading state")
		}
		if m.Err != testErr {
			t.Errorf("SetError() error = %v, want %v", m.Err, testErr)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
