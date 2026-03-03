package service

import (
	"errors"
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

// mockFetcher is a test double for TeamMemberFetcher.
type mockFetcher struct {
	fetchFunc       func(org, slug string) ([]github.TeamMember, error)
	myTeamsFunc     func() ([]github.UserTeam, error)
	childTeamsFunc  func(org, parentSlug string) ([]github.ChildTeam, error)
	isOrgMemberFunc func(org, login string) (bool, error)
	callCount       int
}

func (m *mockFetcher) FetchTeamMembers(org, slug string) ([]github.TeamMember, error) {
	m.callCount++
	return m.fetchFunc(org, slug)
}

func (m *mockFetcher) FetchMyTeams() ([]github.UserTeam, error) {
	if m.myTeamsFunc != nil {
		return m.myTeamsFunc()
	}
	return nil, nil
}

func (m *mockFetcher) FetchChildTeams(org, parentSlug string) ([]github.ChildTeam, error) {
	if m.childTeamsFunc != nil {
		return m.childTeamsFunc(org, parentSlug)
	}
	return nil, nil
}

func (m *mockFetcher) FetchIsOrgMember(org, login string) (bool, error) {
	if m.isOrgMemberFunc != nil {
		return m.isOrgMemberFunc(org, login)
	}
	return false, nil
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

func TestSharesTeamWith(t *testing.T) {
	tests := []struct {
		name       string
		myTeams    []github.UserTeam
		myTeamsErr error
		members    map[string][]github.TeamMember // "org/slug" -> members
		org        string
		other      string
		want       bool
	}{
		{
			name: "overlap: other is in my team",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "acme"}},
			},
			members: map[string][]github.TeamMember{
				"acme/backend": {{Login: "alice"}, {Login: "john"}},
			},
			org:   "acme",
			other: "john",
			want:  true,
		},
		{
			name: "no overlap: other is not in my team",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "acme"}},
			},
			members: map[string][]github.TeamMember{
				"acme/backend": {{Login: "alice"}},
			},
			org:   "acme",
			other: "bob",
			want:  false,
		},
		{
			name: "different org team ignored",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "other-org"}},
			},
			members: map[string][]github.TeamMember{},
			org:     "acme",
			other:   "john",
			want:    false,
		},
		{
			name:       "fetch error: fail-closed returns false",
			myTeamsErr: errors.New("unauthorized"),
			org:        "acme",
			other:      "john",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockFetcher{
				fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
					key := org + "/" + slug
					if m, ok := tt.members[key]; ok {
						return m, nil
					}
					return nil, nil
				},
				myTeamsFunc: func() ([]github.UserTeam, error) {
					if tt.myTeamsErr != nil {
						return nil, tt.myTeamsErr
					}
					return tt.myTeams, nil
				},
			}
			svc := NewTeamService(fetcher)
			got := svc.SharesTeamWith(tt.org, tt.other)
			if got != tt.want {
				t.Errorf("SharesTeamWith(%q, %q) = %v, want %v", tt.org, tt.other, got, tt.want)
			}
		})
	}
}

func TestSharesTeamWithCaching(t *testing.T) {
	callCount := 0
	fetcher := &mockFetcher{
		fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
			return []github.TeamMember{{Login: "alice"}, {Login: "john"}}, nil
		},
		myTeamsFunc: func() ([]github.UserTeam, error) {
			callCount++
			return []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "acme"}},
			}, nil
		},
	}
	svc := NewTeamService(fetcher)

	// Two calls — FetchMyTeams should only be called once.
	svc.SharesTeamWith("acme", "john")
	svc.SharesTeamWith("acme", "bob")

	if callCount != 1 {
		t.Errorf("FetchMyTeams called %d times, want 1", callCount)
	}
}

func TestIsSiblingTeamMember(t *testing.T) {
	tests := []struct {
		name       string
		myTeams    []github.UserTeam
		members    map[string][]github.TeamMember
		childTeams map[string][]github.ChildTeam // "org/parentSlug" -> children
		org        string
		login      string
		want       bool
	}{
		{
			name: "author in sibling team",
			myTeams: []github.UserTeam{
				{
					Slug:         "backend",
					Organization: github.TeamOrganization{Login: "acme"},
					Parent:       &github.ParentTeam{Slug: "platform"},
				},
			},
			members: map[string][]github.TeamMember{
				"acme/frontend": {{Login: "bob"}},
			},
			childTeams: map[string][]github.ChildTeam{
				"acme/platform": {{Slug: "backend"}, {Slug: "frontend"}},
			},
			org:   "acme",
			login: "bob",
			want:  true,
		},
		{
			name: "my team has no parent",
			myTeams: []github.UserTeam{
				{
					Slug:         "backend",
					Organization: github.TeamOrganization{Login: "acme"},
					// no Parent
				},
			},
			org:   "acme",
			login: "bob",
			want:  false,
		},
		{
			name: "FetchChildTeams error: fail-closed",
			myTeams: []github.UserTeam{
				{
					Slug:         "backend",
					Organization: github.TeamOrganization{Login: "acme"},
					Parent:       &github.ParentTeam{Slug: "platform"},
				},
			},
			childTeams: nil, // will cause error path
			org:        "acme",
			login:      "bob",
			want:       false,
		},
		{
			name: "author not in any sibling team",
			myTeams: []github.UserTeam{
				{
					Slug:         "backend",
					Organization: github.TeamOrganization{Login: "acme"},
					Parent:       &github.ParentTeam{Slug: "platform"},
				},
			},
			members: map[string][]github.TeamMember{
				"acme/frontend": {{Login: "carol"}},
			},
			childTeams: map[string][]github.ChildTeam{
				"acme/platform": {{Slug: "backend"}, {Slug: "frontend"}},
			},
			org:   "acme",
			login: "bob",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockFetcher{
				fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
					key := org + "/" + slug
					if m, ok := tt.members[key]; ok {
						return m, nil
					}
					return nil, nil
				},
				myTeamsFunc: func() ([]github.UserTeam, error) {
					return tt.myTeams, nil
				},
				childTeamsFunc: func(org, parentSlug string) ([]github.ChildTeam, error) {
					key := org + "/" + parentSlug
					if tt.childTeams == nil {
						return nil, errors.New("child teams error")
					}
					if c, ok := tt.childTeams[key]; ok {
						return c, nil
					}
					return nil, nil
				},
			}
			svc := NewTeamService(fetcher)
			got := svc.IsSiblingTeamMember(tt.org, tt.login)
			if got != tt.want {
				t.Errorf("IsSiblingTeamMember(%q, %q) = %v, want %v", tt.org, tt.login, got, tt.want)
			}
		})
	}
}

func TestIsOrgMember(t *testing.T) {
	tests := []struct {
		name string
		fn   func(org, login string) (bool, error)
		want bool
	}{
		{
			name: "member",
			fn:   func(org, login string) (bool, error) { return true, nil },
			want: true,
		},
		{
			name: "non-member",
			fn:   func(org, login string) (bool, error) { return false, nil },
			want: false,
		},
		{
			name: "error: fail-closed",
			fn:   func(org, login string) (bool, error) { return false, errors.New("network") },
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockFetcher{
				fetchFunc:       func(org, slug string) ([]github.TeamMember, error) { return nil, nil },
				isOrgMemberFunc: tt.fn,
			}
			svc := NewTeamService(fetcher)
			got := svc.IsOrgMember("acme", "bob")
			if got != tt.want {
				t.Errorf("IsOrgMember = %v, want %v", got, tt.want)
			}
		})
	}
}
