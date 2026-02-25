package prs

import (
	"testing"

	"github.com/radiohead/gh-inbox/internal/service"
)

func TestParseFilterMode(t *testing.T) {
	tests := []struct {
		input    string
		wantMode service.Mode
		wantErr  bool
	}{
		{input: "all", wantMode: service.ModeAll},
		{input: "", wantMode: service.ModeAll},
		{input: "direct", wantMode: service.ModeDirect},
		{input: "codeowner", wantMode: service.ModeCodeowner},
		{input: "team", wantMode: service.ModeTeam},
		{input: "invalid", wantErr: true},
		{input: "DIRECT", wantErr: true},
		{input: "solo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseFilterMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseFilterMode(%q) = nil error, want error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFilterMode(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.wantMode {
				t.Errorf("parseFilterMode(%q) = %v, want %v", tt.input, got, tt.wantMode)
			}
		})
	}
}
