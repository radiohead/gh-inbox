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

func TestNeedsUserContext(t *testing.T) {
	tests := []struct {
		name   string
		mode   service.Mode
		output string
		want   bool
	}{
		{name: "mode all json", mode: service.ModeAll, output: "json", want: false},
		{name: "mode all table", mode: service.ModeAll, output: "table", want: true},
		{name: "mode all empty output defaults table", mode: service.ModeAll, output: "", want: true},
		{name: "mode direct json", mode: service.ModeDirect, output: "json", want: true},
		{name: "mode team json", mode: service.ModeTeam, output: "json", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsUserContext(tt.mode, tt.output)
			if got != tt.want {
				t.Errorf("needsUserContext(%v, %q) = %v, want %v", tt.mode, tt.output, got, tt.want)
			}
		})
	}
}
