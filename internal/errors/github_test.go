package errors_test

import (
	stderrors "errors"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"

	gherrors "github.com/radiohead/gh-inbox/internal/errors"
)

func TestClassifySAMLGraphQL(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantMatch    bool
		wantSeverity gherrors.Severity
		wantSummary  string // substring check when non-empty
	}{
		{
			name:      "non-GraphQL error does not match",
			err:       stderrors.New("generic error"),
			wantMatch: false,
		},
		{
			name: "all-SAML GraphQL error returns Warning",
			err: &api.GraphQLError{
				Errors: []api.GraphQLErrorItem{
					{Message: "Resource protected by organization SAML enforcement"},
					{Message: "Resource protected by organization SAML enforcement"},
				},
			},
			wantMatch:    true,
			wantSeverity: gherrors.SeverityWarning,
			wantSummary:  "2 result(s) skipped",
		},
		{
			name: "mixed SAML and non-SAML items does not match",
			err: &api.GraphQLError{
				Errors: []api.GraphQLErrorItem{
					{Message: "Resource protected by organization SAML enforcement"},
					{Message: "Could not resolve to a Repository"},
				},
			},
			wantMatch: false,
		},
		{
			name: "single non-SAML GraphQL error does not match",
			err: &api.GraphQLError{
				Errors: []api.GraphQLErrorItem{
					{Message: "Could not resolve to a Repository"},
				},
			},
			wantMatch: false,
		},
		{
			name:      "empty GraphQL error does not match",
			err:       &api.GraphQLError{},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, result := gherrors.ClassifySAMLGraphQL(tt.err)
			if matched != tt.wantMatch {
				t.Fatalf("matched = %v, want %v", matched, tt.wantMatch)
			}
			if !tt.wantMatch {
				return
			}
			if result.Severity() != tt.wantSeverity {
				t.Errorf("Severity() = %v, want %v", result.Severity(), tt.wantSeverity)
			}
			if tt.wantSummary != "" && !strings.Contains(result.Summary(), tt.wantSummary) {
				t.Errorf("Summary() = %q, want to contain %q", result.Summary(), tt.wantSummary)
			}
		})
	}
}

func TestClassifySAMLHTTP(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantMatch    bool
		wantSeverity gherrors.Severity
	}{
		{
			name:      "non-HTTP error does not match",
			err:       stderrors.New("generic error"),
			wantMatch: false,
		},
		{
			name: "403 with SAML in message returns Warning",
			err: &api.HTTPError{
				StatusCode: 403,
				Message:    "Resource protected by organization SAML enforcement. To access this resource, you must use a SAML SSO token.",
			},
			wantMatch:    true,
			wantSeverity: gherrors.SeverityWarning,
		},
		{
			name: "403 without SAML in message does not match",
			err: &api.HTTPError{
				StatusCode: 403,
				Message:    "Forbidden",
			},
			wantMatch: false,
		},
		{
			name: "non-403 HTTP error with SAML does not match",
			err: &api.HTTPError{
				StatusCode: 401,
				Message:    "SAML token required",
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, result := gherrors.ClassifySAMLHTTP(tt.err)
			if matched != tt.wantMatch {
				t.Fatalf("matched = %v, want %v", matched, tt.wantMatch)
			}
			if !tt.wantMatch {
				return
			}
			if result.Severity() != tt.wantSeverity {
				t.Errorf("Severity() = %v, want %v", result.Severity(), tt.wantSeverity)
			}
		})
	}
}

func TestClassify_WithGitHubClassifiers(t *testing.T) {
	samlErr := &api.GraphQLError{
		Errors: []api.GraphQLErrorItem{
			{Message: "Resource protected by organization SAML enforcement"},
		},
	}
	result := gherrors.Classify(samlErr, gherrors.GitHubClassifiers...)
	if result.Severity() != gherrors.SeverityWarning {
		t.Errorf("Severity() = %v, want Warning", result.Severity())
	}
}

