package service

import (
	"strings"
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

// buildPR constructs a PullRequest for testing with the given review requests.
// nameWithOwner should be in "owner/repo" format.
func buildPR(nameWithOwner string, requests []github.ReviewRequest) github.PullRequest {
	parts := strings.SplitN(nameWithOwner, "/", 2)
	owner, name := parts[0], parts[1]
	return github.PullRequest{
		Number: 1,
		Title:  "Test PR",
		URL:    "https://github.com/" + nameWithOwner + "/pull/1",
		Repository: github.Repository{
			Owner: owner,
			Name:  name,
		},
		ReviewRequests: github.ReviewRequestConnection{
			Nodes: requests,
		},
	}
}

// userReq builds a User ReviewRequest.
func userReq(login string, asCodeOwner bool) github.ReviewRequest {
	return github.ReviewRequest{
		AsCodeOwner: asCodeOwner,
		RequestedReviewer: github.RequestedReviewer{
			Type:  "User",
			Login: login,
		},
	}
}

// teamReq builds a Team ReviewRequest.
func teamReq(slug string, asCodeOwner bool) github.ReviewRequest {
	return github.ReviewRequest{
		AsCodeOwner: asCodeOwner,
		RequestedReviewer: github.RequestedReviewer{
			Type:  "Team",
			Login: slug,
		},
	}
}

// newMockTeamService builds a TeamService backed by a mockFetcher with
// configurable team membership and user teams.
func newMockTeamService(
	members map[string][]github.TeamMember,
	myTeams []github.UserTeam,
) *TeamService {
	fetcher := &mockFetcher{
		fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
			key := org + "/" + slug
			if m, ok := members[key]; ok {
				return m, nil
			}
			return nil, nil
		},
		myTeamsFunc: func() ([]github.UserTeam, error) {
			return myTeams, nil
		},
	}
	return NewTeamService(fetcher)
}

func TestFilterDirect(t *testing.T) {
	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		members   map[string][]github.TeamMember
		myTeams   []github.UserTeam
		wantCount int
	}{
		{
			name: "only me requested as User — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "me + teammate (shares team) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
					userReq("john", false),
				}),
			},
			myLogin: "alice",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 0,
		},
		{
			name: "me + non-teammate (no shared teams) — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
					userReq("bob", false),
				}),
			},
			myLogin: "alice",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			wantCount: 1,
		},
		{
			name: "me + mixed (one shares team) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
					userReq("bob", false),
					userReq("john", false),
				}),
			},
			myLogin: "alice",
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 0,
		},
		{
			name: "me via team only (not as User) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "empty requests — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "me as User + team also requested — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
					teamReq("backend", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, tt.myTeams)
			got := Filter(tt.prs, tt.myLogin, svc, ModeDirect)
			if len(got) != tt.wantCount {
				t.Errorf("filterDirect returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterTeam(t *testing.T) {
	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		members   map[string][]github.TeamMember
		myTeams   []github.UserTeam
		wantCount int
	}{
		{
			name: "my team requested, no individuals — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 1,
		},
		{
			name: "my team + member individually tagged — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", false),
					userReq("john", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 0,
		},
		{
			name: "my team + non-member individually tagged — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", false),
					userReq("carol", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 1,
		},
		{
			name: "other team (not mine) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("frontend", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/frontend": {{Login: "bob"}, {Login: "carol"}},
			},
			wantCount: 0,
		},
		{
			name: "only User reviewers (no teams) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "multiple teams, one mine, clean — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("frontend", false),
					teamReq("backend", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/frontend": {{Login: "bob"}},
				"org/backend":  {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 1,
		},
		{
			name: "my team + member of that team tagged — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("foo", false),
					userReq("john", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/foo": {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, tt.myTeams)
			got := Filter(tt.prs, tt.myLogin, svc, ModeTeam)
			if len(got) != tt.wantCount {
				t.Errorf("filterTeam returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterCodeowner(t *testing.T) {
	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		members   map[string][]github.TeamMember
		wantCount int
	}{
		{
			name: "sole codeowner — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "codeowner with other reviewers — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
					userReq("bob", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "explicit request (not codeowner) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "mixed explicit + codeowner — exclude (not all codeowner)",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
					userReq("alice", true),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "team codeowner, user is member — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", true),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			wantCount: 1,
		},
		{
			name: "empty requests — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, nil)
			got := Filter(tt.prs, tt.myLogin, svc, ModeCodeowner)
			if len(got) != tt.wantCount {
				t.Errorf("filterCodeowner returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterAll(t *testing.T) {
	prs := []github.PullRequest{
		buildPR("org/repo", []github.ReviewRequest{
			userReq("alice", true),
		}),
		buildPR("org/repo2", []github.ReviewRequest{
			userReq("alice", false),
		}),
	}
	svc := newMockTeamService(nil, nil)
	got := Filter(prs, "alice", svc, ModeAll)
	if len(got) != 2 {
		t.Errorf("ModeAll returned %d PRs, want 2", len(got))
	}
}
