package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/radiohead/gh-inbox/internal/github"
)

func TestWriteTable_empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTable(&buf, nil); err != nil {
		t.Fatalf("WriteTable(nil) unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No pull requests found") {
		t.Errorf("expected empty message, got: %q", buf.String())
	}
}

func TestWriteTable_tabSeparated(t *testing.T) {
	now := time.Now()
	prs := []github.PullRequest{
		{
			Number:    42,
			Title:     "Fix the bug",
			CreatedAt: now.Add(-2 * 24 * time.Hour),
			Repository: github.Repository{Owner: "acme", Name: "api"},
			ReviewRequests: github.ReviewRequestConnection{
				Nodes: []github.ReviewRequest{
					{AsCodeOwner: false, RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteTable(&buf, prs); err != nil {
		t.Fatalf("WriteTable unexpected error: %v", err)
	}

	out := buf.String()
	// In non-TTY (test) mode the printer outputs tab-separated values.
	if !strings.Contains(out, "acme/api") {
		t.Errorf("expected repo in output, got: %q", out)
	}
	if !strings.Contains(out, "#42") {
		t.Errorf("expected PR number in output, got: %q", out)
	}
	if !strings.Contains(out, "Fix the bug") {
		t.Errorf("expected title in output, got: %q", out)
	}
	if !strings.Contains(out, "direct") {
		t.Errorf("expected source=direct in output, got: %q", out)
	}
	// Age should be "2d"
	if !strings.Contains(out, "2d") {
		t.Errorf("expected age=2d in output, got: %q", out)
	}
}

func TestSourceOf(t *testing.T) {
	tests := []struct {
		name     string
		requests []github.ReviewRequest
		want     string
	}{
		{
			name:     "direct user non-codeowner",
			requests: []github.ReviewRequest{{AsCodeOwner: false, RequestedReviewer: github.RequestedReviewer{Type: "User"}}},
			want:     "direct",
		},
		{
			name:     "team non-codeowner",
			requests: []github.ReviewRequest{{AsCodeOwner: false, RequestedReviewer: github.RequestedReviewer{Type: "Team"}}},
			want:     "team",
		},
		{
			name:     "all codeowner",
			requests: []github.ReviewRequest{{AsCodeOwner: true, RequestedReviewer: github.RequestedReviewer{Type: "User"}}},
			want:     "codeowner",
		},
		{
			name: "direct wins over team",
			requests: []github.ReviewRequest{
				{AsCodeOwner: false, RequestedReviewer: github.RequestedReviewer{Type: "Team"}},
				{AsCodeOwner: false, RequestedReviewer: github.RequestedReviewer{Type: "User"}},
			},
			want: "direct",
		},
		{
			name:     "empty requests",
			requests: nil,
			want:     "codeowner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := github.PullRequest{ReviewRequests: github.ReviewRequestConnection{Nodes: tt.requests}}
			got := sourceOf(pr)
			if got != tt.want {
				t.Errorf("sourceOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHumanAge(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{name: "minutes", t: now.Add(-30 * time.Minute), want: "30m"},
		{name: "hours", t: now.Add(-3 * time.Hour), want: "3h"},
		{name: "days", t: now.Add(-2 * 24 * time.Hour), want: "2d"},
		{name: "weeks", t: now.Add(-14 * 24 * time.Hour), want: "2w"},
		{name: "months", t: now.Add(-60 * 24 * time.Hour), want: "2mo"},
		{name: "years", t: now.Add(-400 * 24 * time.Hour), want: "1y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanAge(tt.t)
			if got != tt.want {
				t.Errorf("humanAge() = %q, want %q", got, tt.want)
			}
		})
	}
}
