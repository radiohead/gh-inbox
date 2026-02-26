package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/radiohead/gh-inbox/internal/github"
	"github.com/radiohead/gh-inbox/internal/service"
)

// noopFetcher satisfies service.TeamMemberFetcher with empty responses.
type noopFetcher struct{}

func (noopFetcher) FetchTeamMembers(_, _ string) ([]github.TeamMember, error) {
	return nil, nil
}
func (noopFetcher) FetchMyTeams() ([]github.UserTeam, error) { return nil, nil }

// tableTestFetcher satisfies service.TeamMemberFetcher with configurable data.
type tableTestFetcher struct {
	members map[string][]github.TeamMember
	teams   []github.UserTeam
}

func (f tableTestFetcher) FetchTeamMembers(org, slug string) ([]github.TeamMember, error) {
	return f.members[org+"/"+slug], nil
}
func (f tableTestFetcher) FetchMyTeams() ([]github.UserTeam, error) { return f.teams, nil }

func TestWriteTable_empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTable(&buf, nil, "", service.NewTeamService(noopFetcher{})); err != nil {
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
			URL:       "https://github.com/acme/api/pull/42",
			CreatedAt: now.Add(-2 * 24 * time.Hour),
			Repository: github.Repository{Owner: "acme", Name: "api"},
			ReviewRequests: github.ReviewRequestConnection{
				Nodes: []github.ReviewRequest{
					{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteTable(&buf, prs, "alice", service.NewTeamService(noopFetcher{})); err != nil {
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
	if !strings.Contains(out, "https://github.com/acme/api/pull/42") {
		t.Errorf("expected URL in output, got: %q", out)
	}
	if !strings.Contains(out, "direct") {
		t.Errorf("expected source=direct in output, got: %q", out)
	}
	// Age should be "2d"
	if !strings.Contains(out, "2d") {
		t.Errorf("expected age=2d in output, got: %q", out)
	}
}

func TestWriteTable_sortsByAgeOldestFirst(t *testing.T) {
	now := time.Now()
	prs := []github.PullRequest{
		{Number: 1, Title: "newest", URL: "https://github.com/acme/api/pull/1", CreatedAt: now.Add(-1 * 24 * time.Hour), Repository: github.Repository{Owner: "acme", Name: "api"}},
		{Number: 3, Title: "oldest", URL: "https://github.com/acme/api/pull/3", CreatedAt: now.Add(-10 * 24 * time.Hour), Repository: github.Repository{Owner: "acme", Name: "api"}},
		{Number: 2, Title: "middle", URL: "https://github.com/acme/api/pull/2", CreatedAt: now.Add(-5 * 24 * time.Hour), Repository: github.Repository{Owner: "acme", Name: "api"}},
	}

	var buf bytes.Buffer
	if err := WriteTable(&buf, prs, "", service.NewTeamService(noopFetcher{})); err != nil {
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

func TestSourceOf(t *testing.T) {
	noop := service.NewTeamService(noopFetcher{})

	tests := []struct {
		name     string
		login    string
		requests []github.ReviewRequest
		teams    *service.TeamService
		want     string
	}{
		{
			name:  "sole User reviewer → direct",
			login: "alice",
			requests: []github.ReviewRequest{
				{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
			},
			teams: noop,
			want:  "direct",
		},
		{
			// carol is not in alice's teams (noop fetcher returns empty) → not a teammate
			name:  "User + non-teammate → direct",
			login: "alice",
			requests: []github.ReviewRequest{
				{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
				{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "carol"}},
			},
			teams: noop,
			want:  "direct",
		},
		{
			// bob is alice's teammate → not direct, not team → codeowner
			name:  "User + teammate → codeowner",
			login: "alice",
			requests: []github.ReviewRequest{
				{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
				{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "bob"}},
			},
			teams: service.NewTeamService(tableTestFetcher{
				members: map[string][]github.TeamMember{
					"org/myteam": {{Login: "alice"}, {Login: "bob"}},
				},
				teams: []github.UserTeam{
					{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
				},
			}),
			want: "codeowner",
		},
		{
			// alice's team requested, no individual teammates → team
			name:  "Team (my team), no individuals → team",
			login: "alice",
			requests: []github.ReviewRequest{
				{RequestedReviewer: github.RequestedReviewer{Type: "Team", Login: "backend"}},
			},
			teams: service.NewTeamService(tableTestFetcher{
				members: map[string][]github.TeamMember{
					"org/backend": {{Login: "alice"}},
				},
			}),
			want: "team",
		},
		{
			name:     "empty requests → codeowner",
			login:    "alice",
			requests: nil,
			teams:    noop,
			want:     "codeowner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := github.PullRequest{
				Repository:     github.Repository{Owner: "org"},
				ReviewRequests: github.ReviewRequestConnection{Nodes: tt.requests},
			}
			got := sourceOf(pr, tt.login, tt.teams)
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
