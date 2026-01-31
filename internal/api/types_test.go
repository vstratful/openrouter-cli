package api

import "testing"

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
