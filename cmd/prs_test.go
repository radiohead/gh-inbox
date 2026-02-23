package cmd

import (
	"testing"
)

func TestValidatePRsFlags(t *testing.T) {
	tests := []struct {
		name    string
		opts    prsOptions
		wantErr string // empty means no error expected
	}{
		{
			name: "no flags",
			opts: prsOptions{},
		},
		{
			name: "review only",
			opts: prsOptions{review: true},
		},
		{
			name: "authored only",
			opts: prsOptions{authored: true},
		},
		{
			name: "review and authored",
			opts: prsOptions{review: true, authored: true},
		},
		{
			name: "review with include-codeowners",
			opts: prsOptions{review: true, includeCodeowners: true},
		},
		{
			name: "review with codeowners-solo",
			opts: prsOptions{review: true, codeownersSolo: true},
		},
		{
			name:    "include-codeowners without review",
			opts:    prsOptions{includeCodeowners: true},
			wantErr: "--include-codeowners requires --review",
		},
		{
			name:    "codeowners-solo without review",
			opts:    prsOptions{codeownersSolo: true},
			wantErr: "--codeowners-solo requires --review",
		},
		{
			name:    "include-codeowners and codeowners-solo together",
			opts:    prsOptions{review: true, includeCodeowners: true, codeownersSolo: true},
			wantErr: "--include-codeowners and --codeowners-solo are mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePRsFlags(tt.opts)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
