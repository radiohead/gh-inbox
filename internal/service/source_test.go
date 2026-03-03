package service

import (
	"errors"
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

func TestClassifyAuthorSource(t *testing.T) {
	tests := []struct {
		name       string
		pr         github.PullRequest
		myLogin    string
		members    map[string][]github.TeamMember
		myTeams    []github.UserTeam
		childTeams map[string][]github.ChildTeam
		orgMember  *bool // nil = not called, true/false = result
		want       AuthorSource
	}{
		{
			name: "author is myself",
			pr: github.PullRequest{
				Author:     "alice",
				Repository: github.Repository{Owner: "org", Name: "repo"},
			},
			myLogin: "alice",
			want:    AuthorSourceTeam,
		},
		{
			name: "empty author",
			pr: github.PullRequest{
				Author:     "",
				Repository: github.Repository{Owner: "org", Name: "repo"},
			},
			myLogin: "alice",
			want:    AuthorSourceOther,
		},
		{
			name: "author shares direct team",
			pr: github.PullRequest{
				Author:     "bob",
				Repository: github.Repository{Owner: "org", Name: "repo"},
			},
			myLogin: "alice",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "bob"}},
			},
			want: AuthorSourceTeam,
		},
		{
			name: "author in sibling team",
			pr: github.PullRequest{
				Author:     "carol",
				Repository: github.Repository{Owner: "org", Name: "repo"},
			},
			myLogin: "alice",
			myTeams: []github.UserTeam{
				{
					Slug:         "backend",
					Organization: github.TeamOrganization{Login: "org"},
					Parent:       &github.ParentTeam{Slug: "platform"},
				},
			},
			members: map[string][]github.TeamMember{
				"org/backend":  {{Login: "alice"}},
				"org/frontend": {{Login: "carol"}},
			},
			childTeams: map[string][]github.ChildTeam{
				"org/platform": {{Slug: "backend"}, {Slug: "frontend"}},
			},
			want: AuthorSourceGroup,
		},
		{
			name: "author in org but not in team hierarchy",
			pr: github.PullRequest{
				Author:     "dave",
				Repository: github.Repository{Owner: "org", Name: "repo"},
			},
			myLogin: "alice",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			orgMember: boolPtr(true),
			want:      AuthorSourceOrg,
		},
		{
			name: "external author",
			pr: github.PullRequest{
				Author:     "external",
				Repository: github.Repository{Owner: "org", Name: "repo"},
			},
			myLogin: "alice",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			orgMember: boolPtr(false),
			want:      AuthorSourceOther,
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
					if tt.childTeams == nil {
						return nil, errors.New("no child teams configured")
					}
					key := org + "/" + parentSlug
					if c, ok := tt.childTeams[key]; ok {
						return c, nil
					}
					return nil, nil
				},
				isOrgMemberFunc: func(org, login string) (bool, error) {
					if tt.orgMember != nil {
						return *tt.orgMember, nil
					}
					return false, nil
				},
			}
			svc := NewTeamService(fetcher)
			got := ClassifyAuthorSource(tt.pr, tt.myLogin, svc)
			if got != tt.want {
				t.Errorf("ClassifyAuthorSource() = %q, want %q", got, tt.want)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
