package api

import (
	"io"
	"strings"
	"testing"
)

func TestStreamReader_ReadAll(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "simple response",
			input:   "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\ndata: [DONE]\n",
			want:    "Hello world",
			wantErr: false,
		},
		{
			name:    "empty response",
			input:   "data: [DONE]\n",
			want:    "",
			wantErr: false,
		},
		{
			name:    "with comments and empty lines",
			input:   ": comment\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\ndata: [DONE]\n",
			want:    "test",
			wantErr: false,
		},
		{
			name:    "malformed json skipped",
			input:   "data: {invalid}\ndata: {\"choices\":[{\"delta\":{\"content\":\"valid\"}}]}\ndata: [DONE]\n",
			want:    "valid",
			wantErr: false,
		},
		{
			name:    "api error in stream",
			input:   "data: {\"error\":{\"message\":\"rate limit\"}}\n",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewStreamReader(io.NopCloser(strings.NewReader(tt.input)))
			defer reader.Close()

			got, err := reader.ReadAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadAll() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStreamReader_Next(t *testing.T) {
	input := "data: {\"choices\":[{\"delta\":{\"content\":\"A\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"B\"}}]}\n\ndata: [DONE]\n"

	reader := NewStreamReader(io.NopCloser(strings.NewReader(input)))
	defer reader.Close()

	// First chunk
	chunk, err := reader.Next()
	if err != nil {
		t.Fatalf("First Next() error = %v", err)
	}
	if chunk.Content != "A" {
		t.Errorf("First chunk content = %q, want %q", chunk.Content, "A")
	}
	if chunk.Done {
		t.Error("First chunk should not be done")
	}

	// Second chunk
	chunk, err = reader.Next()
	if err != nil {
		t.Fatalf("Second Next() error = %v", err)
	}
	if chunk.Content != "B" {
		t.Errorf("Second chunk content = %q, want %q", chunk.Content, "B")
	}
	if chunk.Done {
		t.Error("Second chunk should not be done")
	}

	// Done signal
	chunk, err = reader.Next()
	if err != nil {
		t.Fatalf("Third Next() error = %v", err)
	}
	if !chunk.Done {
		t.Error("Third chunk should be done")
	}

	// After done
	chunk, err = reader.Next()
	if err != nil {
		t.Fatalf("Fourth Next() error = %v", err)
	}
	if chunk != nil {
		t.Error("Fourth Next() should return nil")
	}
}

func TestStreamReader_Close(t *testing.T) {
	input := "data: {\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\ndata: [DONE]\n"
	reader := NewStreamReader(io.NopCloser(strings.NewReader(input)))

	// Close before reading
	if err := reader.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// After close, Next should return nil
	chunk, err := reader.Next()
	if err != nil {
		t.Errorf("Next() after close error = %v", err)
	}
	if chunk != nil {
		t.Error("Next() after close should return nil")
	}
}
