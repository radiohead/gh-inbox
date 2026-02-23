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
			name:    "no flags (missing org)",
			opts:    prsOptions{},
			wantErr: "--org is required",
		},
		{
			name: "org only",
			opts: prsOptions{org: "test-org"},
		},
		{
			name: "review only (missing org)",
			opts: prsOptions{review: true},
			wantErr: "--org is required",
		},
		{
			name: "org with review",
			opts: prsOptions{org: "test-org", review: true},
		},
		{
			name: "org with authored",
			opts: prsOptions{org: "test-org", authored: true},
		},
		{
			name: "org with review and authored",
			opts: prsOptions{org: "test-org", review: true, authored: true},
		},
		{
			name: "org with review and include-codeowners",
			opts: prsOptions{org: "test-org", review: true, includeCodeowners: true},
		},
		{
			name: "org with review and codeowners-solo",
			opts: prsOptions{org: "test-org", review: true, codeownersSolo: true},
		},
		{
			name:    "include-codeowners without review",
			opts:    prsOptions{org: "test-org", includeCodeowners: true},
			wantErr: "--include-codeowners requires --review",
		},
		{
			name:    "codeowners-solo without review",
			opts:    prsOptions{org: "test-org", codeownersSolo: true},
			wantErr: "--codeowners-solo requires --review",
		},
		{
			name:    "include-codeowners and codeowners-solo together",
			opts:    prsOptions{org: "test-org", review: true, includeCodeowners: true, codeownersSolo: true},
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
