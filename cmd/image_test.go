package cmd

import (
	"testing"
)

func TestParseDataURL(t *testing.T) {
	tests := []struct {
		name    string
		dataURL string
		want    string
		wantErr bool
	}{
		{
			name:    "valid png data url",
			dataURL: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			want:    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			wantErr: false,
		},
		{
			name:    "valid jpeg data url",
			dataURL: "data:image/jpeg;base64,/9j/4AAQSkZJRg==",
			want:    "/9j/4AAQSkZJRg==",
			wantErr: false,
		},
		{
			name:    "valid webp data url",
			dataURL: "data:image/webp;base64,UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAfQ//73v/+BiOh/AAA=",
			want:    "UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAfQ//73v/+BiOh/AAA=",
			wantErr: false,
		},
		{
			name:    "missing data prefix",
			dataURL: "image/png;base64,abc123",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing base64 marker",
			dataURL: "data:image/png,abc123",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			dataURL: "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "regular url",
			dataURL: "https://example.com/image.png",
			want:    "",
			wantErr: true,
		},
		{
			name:    "data prefix only",
			dataURL: "data:",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty base64 data",
			dataURL: "data:image/png;base64,",
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDataURL(tt.dataURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDataURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDataURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
