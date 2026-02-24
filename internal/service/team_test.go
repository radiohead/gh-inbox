package service

import (
	"errors"
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

// mockFetcher is a test double for TeamMemberFetcher.
type mockFetcher struct {
	fetchFunc func(org, slug string) ([]github.TeamMember, error)
	callCount int
}

func (m *mockFetcher) FetchTeamMembers(org, slug string) ([]github.TeamMember, error) {
	m.callCount++
	return m.fetchFunc(org, slug)
}

func TestIsTeamMember(t *testing.T) {
	tests := []struct {
		name     string
		members  []github.TeamMember
		fetchErr error
		login    string
		want     bool
	}{
		{
			name:    "cache miss, user IS member",
			members: []github.TeamMember{{Login: "alice"}, {Login: "bob"}},
			login:   "alice",
			want:    true,
		},
		{
			name:    "cache miss, user is NOT member",
			members: []github.TeamMember{{Login: "carol"}, {Login: "dave"}},
			login:   "alice",
			want:    false,
		},
		{
			name:     "fetch error: fail-open returns true",
			fetchErr: errors.New("team not found"),
			login:    "alice",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockFetcher{
				fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
					if tt.fetchErr != nil {
						return nil, tt.fetchErr
					}
					return tt.members, nil
				},
			}
			svc := NewTeamService(fetcher)
			got := svc.IsTeamMember("test-org", "backend", tt.login)
			if got != tt.want {
				t.Errorf("IsTeamMember = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTeamMemberCacheHit(t *testing.T) {
	fetcher := &mockFetcher{
		fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
			return []github.TeamMember{{Login: "alice"}}, nil
		},
	}
	svc := NewTeamService(fetcher)

	// First call — cache miss, should fetch from REST.
	got1 := svc.IsTeamMember("test-org", "backend", "alice")
	if !got1 {
		t.Errorf("first call: IsTeamMember = false, want true")
	}

	// Second call with same org/slug — should use cache, not call fetcher again.
	got2 := svc.IsTeamMember("test-org", "backend", "alice")
	if !got2 {
		t.Errorf("second call: IsTeamMember = false, want true")
	}

	if fetcher.callCount != 1 {
		t.Errorf("fetcher called %d times, want 1 (cache should prevent second call)", fetcher.callCount)
	}
}
