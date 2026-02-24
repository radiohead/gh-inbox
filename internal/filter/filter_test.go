package filter

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

// userReq builds a user ReviewRequest.
func userReq(login string, asCodeOwner bool) github.ReviewRequest {
	return github.ReviewRequest{
		AsCodeOwner: asCodeOwner,
		RequestedReviewer: github.RequestedReviewer{
			Type:  "User",
			Login: login,
		},
	}
}

// teamReq builds a team ReviewRequest.
func teamReq(slug string, asCodeOwner bool) github.ReviewRequest {
	return github.ReviewRequest{
		AsCodeOwner: asCodeOwner,
		RequestedReviewer: github.RequestedReviewer{
			Type:  "Team",
			Login: slug,
		},
	}
}

func TestFilter(t *testing.T) {
	alwaysMember := func(org, slug string) bool { return true }
	neverMember := func(org, slug string) bool { return false }

	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		isMyTeam  IsMyTeamFunc
		mode      Mode
		wantCount int
	}{
		{
			name: "1: explicit request, direct mode — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDirect,
			wantCount: 1,
		},
		{
			name: "2: codeowners-only with other reviewers, direct — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // my request, asCodeOwner=true
					userReq("bob", false),  // other reviewer
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDirect,
			wantCount: 0,
		},
		{
			name: "3: codeowners-only, sole reviewer, direct — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // my request via codeowners, no others
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDirect,
			wantCount: 1,
		},
		{
			name: "4: mixed explicit + codeowner — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false), // explicit
					userReq("alice", true),  // also codeowner (edge case: same user twice)
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDirect,
			wantCount: 1,
		},
		{
			name: "5: all mode shows everything",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // my codeowner request
					userReq("bob", false),  // other reviewer — direct mode would skip
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeAll,
			wantCount: 1,
		},
		{
			name: "6: codeowner mode, sole codeowner — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // sole reviewer via codeowners
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeCodeowner,
			wantCount: 1,
		},
		{
			name: "7: codeowner mode, codeowner with others — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // my codeowner request
					userReq("bob", false),  // other reviewer — disqualifies codeowner mode
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeCodeowner,
			wantCount: 0,
		},
		{
			name: "8: codeowner mode, explicit request (not codeowner) — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false), // explicit, not via codeowners
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeCodeowner,
			wantCount: 0,
		},
		{
			name: "9: team reviewer, user is member, not codeowner — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", false), // team request, user is member
				}),
			},
			myLogin:   "alice",
			isMyTeam:  alwaysMember,
			mode:      ModeDirect,
			wantCount: 1,
		},
		{
			name: "10: team reviewer, user not member — treated as other",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					teamReq("backend", false), // team request, user is NOT member
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDirect,
			wantCount: 0,
		},
		{
			name: "11: empty review requests — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDirect,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(tt.prs, tt.myLogin, tt.isMyTeam, tt.mode)
			if len(got) != tt.wantCount {
				t.Errorf("Filter returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}
