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
		wantCount int
	}{
		{
			name: "sole explicit User request — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "sole codeowner User request (no others) — include (sole reviewer fallback)",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "explicit User + other User — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
					userReq("bob", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "codeowner User + other User — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
					userReq("bob", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "codeowner User + Team — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
					teamReq("backend", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "mixed explicit + codeowner (two requests for me) — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
					userReq("alice", true),
				}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "Team request only (not as User) — exclude",
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
			name: "explicit User + Team also requested — include",
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
			svc := newMockTeamService(nil, nil)
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
			// Sole codeowner User → matches direct (sole reviewer fallback) → excluded from codeowner
			name: "sole codeowner User — exclude (matches direct as sole reviewer)",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			// Codeowner + other non-team User → not direct (codeowner only + others), not team → include
			name: "codeowner + other User (not my team) — include (fallback)",
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
			// Explicit User request → matches direct → excluded from codeowner
			name: "explicit only — exclude (matches direct)",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
				}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			// Team codeowner where I'm a member → matches team → excluded from codeowner
			name: "Team codeowner (my team member) — exclude (matches team)",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", true),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			wantCount: 0,
		},
		{
			// Codeowner User + other non-team User + Team that isn't mine → fallback
			name: "codeowner User + other non-team User + Team (not mine) — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
					userReq("bob", false),
					teamReq("frontend", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/frontend": {{Login: "bob"}, {Login: "carol"}},
			},
			wantCount: 1,
		},
		{
			// Empty requests → no direct, no team → include as fallback
			name: "empty requests — include (neither direct nor team)",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			// Codeowner User + Team (my team) → matches team → excluded from codeowner
			name: "codeowner User + Team (my team) — exclude (matches team)",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),
					teamReq("backend", false),
				}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "john"}},
			},
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
