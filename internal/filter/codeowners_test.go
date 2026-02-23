package filter

import (
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

// buildPR constructs a PullRequest for testing with the given review requests.
func buildPR(nameWithOwner string, requests []github.ReviewRequest) github.PullRequest {
	return github.PullRequest{
		Number: 1,
		Title:  "Test PR",
		URL:    "https://github.com/" + nameWithOwner + "/pull/1",
		Repository: github.Repository{
			NameWithOwner: nameWithOwner,
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

func TestCodeOwners(t *testing.T) {
	alwaysMember := func(org, slug string) bool { return true }
	neverMember := func(org, slug string) bool { return false }

	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		isMyTeam  IsMyTeamFunc
		mode      Mode
		wantCount int
		wantNums  []int // PR numbers expected in output, in order
	}{
		{
			name: "1: explicit request, default mode — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false),
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDefault,
			wantCount: 1,
		},
		{
			name: "2: codeowners-only with other reviewers, default — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),   // my request, asCodeOwner=true
					userReq("bob", false),    // other reviewer
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDefault,
			wantCount: 0,
		},
		{
			name: "3: codeowners-only, sole reviewer, default — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // my request via codeowners, no others
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDefault,
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
			mode:      ModeDefault,
			wantCount: 1,
		},
		{
			name: "5: include-all shows everything",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true),  // my codeowner request
					userReq("bob", false),   // other reviewer — default mode would skip
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeIncludeAll,
			wantCount: 1,
		},
		{
			name: "6: solo mode, sole codeowner — show",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // sole reviewer via codeowners
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeSolo,
			wantCount: 1,
		},
		{
			name: "7: solo mode, codeowner with others — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", true), // my codeowner request
					userReq("bob", false),  // other reviewer — disqualifies solo mode
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeSolo,
			wantCount: 0,
		},
		{
			name: "8: solo mode, explicit request (not codeowner) — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{
					userReq("alice", false), // explicit, not via codeowners
				}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeSolo,
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
			mode:      ModeDefault,
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
			mode:      ModeDefault,
			wantCount: 0,
		},
		{
			name: "11: empty review requests — skip",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{}),
			},
			myLogin:   "alice",
			isMyTeam:  neverMember,
			mode:      ModeDefault,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CodeOwners(tt.prs, tt.myLogin, tt.isMyTeam, tt.mode)
			if len(got) != tt.wantCount {
				t.Errorf("CodeOwners returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}
