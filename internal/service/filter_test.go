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
func userReq(login string) github.ReviewRequest {
	return github.ReviewRequest{
		RequestedReviewer: github.RequestedReviewer{
			Type:  "User",
			Login: login,
		},
	}
}

// teamReq builds a Team ReviewRequest.
func teamReq(slug string) github.ReviewRequest {
	return github.ReviewRequest{
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

func TestClassify(t *testing.T) {
	tests := []struct {
		name    string
		pr      github.PullRequest
		myLogin string
		members map[string][]github.TeamMember
		myTeams []github.UserTeam
		want    Source
	}{
		{
			name:    "sole User reviewer — SourceDirect",
			pr:      buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
			myLogin: "alice",
			want:    SourceDirect,
		},
		{
			// KEY CASE: direct wins over team — me(User) + my team both requested
			name: "User(me) + Team(my-team) — SourceDirect (direct wins)",
			pr: buildPR("org/repo", []github.ReviewRequest{
				userReq("alice"), teamReq("backend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: SourceDirect,
		},
		{
			// only team requested, no individual user me → SourceTeam
			name: "only Team (mine), no User me — SourceTeam",
			pr: buildPR("org/repo", []github.ReviewRequest{
				teamReq("backend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: SourceTeam,
		},
		{
			// me + teammate(User) → not direct (teammate present), not team (no team req) → codeowner
			name: "User(me) + teammate(User) — SourceCodeowner",
			pr: buildPR("org/repo", []github.ReviewRequest{
				userReq("alice"), userReq("bob"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: SourceCodeowner,
		},
		{
			name:    "empty requests — SourceCodeowner",
			pr:      buildPR("org/repo", []github.ReviewRequest{}),
			myLogin: "alice",
			want:    SourceCodeowner,
		},
		{
			// me + non-teammate + my team → me(User) direct (carol not a teammate) → SourceDirect
			name: "User(me) + non-teammate + Team(mine) — SourceDirect",
			pr: buildPR("org/repo", []github.ReviewRequest{
				userReq("alice"), userReq("carol"), teamReq("backend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
				"org/myteam":  {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: SourceDirect,
		},
		{
			// only a team that is NOT mine → neither direct nor team → SourceCodeowner
			name: "only Team (not mine) — SourceCodeowner",
			pr: buildPR("org/repo", []github.ReviewRequest{
				teamReq("frontend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/frontend": {{Login: "bob"}},
			},
			want: SourceCodeowner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, tt.myTeams)
			got := Classify(tt.pr, tt.myLogin, svc)
			if got != tt.want {
				t.Errorf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}
