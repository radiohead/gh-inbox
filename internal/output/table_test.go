package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/radiohead/gh-inbox/internal/github"
	"github.com/radiohead/gh-inbox/internal/service"
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
	prs := []service.ClassifiedPR{
		{
			ReviewType: service.ReviewTypeDirect,
			PR: github.PullRequest{
				Number:    42,
				Title:     "Fix the bug",
				URL:       "https://github.com/acme/api/pull/42",
				CreatedAt: now.Add(-2 * 24 * time.Hour),
				Repository: github.Repository{Owner: "acme", Name: "api"},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteTable(&buf, prs); err != nil {
		t.Fatalf("WriteTable unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "acme/api") {
		t.Errorf("expected repo in output, got: %q", out)
	}
	if !strings.Contains(out, "#42") {
		t.Errorf("expected PR number in output, got: %q", out)
	}
	if !strings.Contains(out, "Fix the bug") {
		t.Errorf("expected title in output, got: %q", out)
	}
	if !strings.Contains(out, "https://github.com/acme/api/pull/42") {
		t.Errorf("expected URL in output, got: %q", out)
	}
	if !strings.Contains(out, "direct") {
		t.Errorf("expected reviewType=direct in output, got: %q", out)
	}
	if !strings.Contains(out, "2d") {
		t.Errorf("expected age=2d in output, got: %q", out)
	}
}

func TestWriteTable_emptySourceRenderedAsDash(t *testing.T) {
	prs := []service.ClassifiedPR{
		{
			// Source is empty — PassthroughClassifier path
			PR: github.PullRequest{
				Number:     1,
				Title:      "Some PR",
				URL:        "https://github.com/acme/api/pull/1",
				CreatedAt:  time.Now().Add(-1 * time.Hour),
				Repository: github.Repository{Owner: "acme", Name: "api"},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteTable(&buf, prs); err != nil {
		t.Fatalf("WriteTable unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "-") {
		t.Errorf("expected reviewType=- for empty ReviewType, got: %q", buf.String())
	}
}

func TestWriteTable_sortsByAgeOldestFirst(t *testing.T) {
	now := time.Now()
	prs := []service.ClassifiedPR{
		{ReviewType: service.ReviewTypeDirect, PR: github.PullRequest{Number: 1, Title: "newest", URL: "https://github.com/acme/api/pull/1", CreatedAt: now.Add(-1 * 24 * time.Hour), Repository: github.Repository{Owner: "acme", Name: "api"}}},
		{ReviewType: service.ReviewTypeDirect, PR: github.PullRequest{Number: 3, Title: "oldest", URL: "https://github.com/acme/api/pull/3", CreatedAt: now.Add(-10 * 24 * time.Hour), Repository: github.Repository{Owner: "acme", Name: "api"}}},
		{ReviewType: service.ReviewTypeDirect, PR: github.PullRequest{Number: 2, Title: "middle", URL: "https://github.com/acme/api/pull/2", CreatedAt: now.Add(-5 * 24 * time.Hour), Repository: github.Repository{Owner: "acme", Name: "api"}}},
	}

	var buf bytes.Buffer
	if err := WriteTable(&buf, prs); err != nil {
		t.Fatalf("WriteTable unexpected error: %v", err)
	}

	out := buf.String()
	oldestPos := strings.Index(out, "oldest")
	middlePos := strings.Index(out, "middle")
	newestPos := strings.Index(out, "newest")
	if oldestPos >= middlePos || middlePos >= newestPos {
		t.Errorf("expected oldest < middle < newest in output, got positions: oldest=%d middle=%d newest=%d\noutput: %q", oldestPos, middlePos, newestPos, out)
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
