package service

import (
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

func TestClassifyReviewStatus(t *testing.T) {
	// newPR builds a PullRequest for org "org" with the given reviews and review requests.
	newPR := func(reviews []github.Review, reviewRequests []github.ReviewRequest) github.PullRequest {
		return github.PullRequest{
			Repository:     github.Repository{Owner: "org", Name: "repo"},
			Reviews:        reviews,
			ReviewRequests: github.ReviewRequestConnection{Nodes: reviewRequests},
		}
	}

	// newTeamService builds a TeamService where "alice" is in team "backend".
	newTeamService := func(teamMembers []github.TeamMember) *TeamService {
		return NewTeamService(&mockFetcher{
			fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
				if org == "org" && slug == "backend" {
					return teamMembers, nil
				}
				return nil, nil
			},
			myTeamsFunc: func() ([]github.UserTeam, error) {
				return []github.UserTeam{
					{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
				}, nil
			},
		})
	}

	// defaultTeam has alice and bob as team members.
	defaultTeam := []github.TeamMember{{Login: "alice"}, {Login: "bob"}}

	tests := []struct {
		name    string
		pr      github.PullRequest
		myLogin string
		teams   *TeamService
		want    ReviewStatus
	}{
		{
			name:    "no reviews at all → open",
			pr:      newPR(nil, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusOpen,
		},
		{
			name: "only PENDING reviews → open",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStatePending},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusOpen,
		},
		{
			name: "only DISMISSED reviews → open",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStateDismissed},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusOpen,
		},
		{
			name: "non-team member COMMENTED → open (only team members count for open check)",
			pr: newPR([]github.Review{
				{Author: "external", State: github.ReviewStateCommented},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam), // only alice and bob are team members
			want:    ReviewStatusOpen,
		},
		{
			name: "team member COMMENTED, no APPROVED → in_review",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStateCommented},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusInReview,
		},
		{
			name: "team member CHANGES_REQUESTED, no APPROVED → in_review",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStateChangesRequested},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusInReview,
		},
		{
			name: "APPROVED with no subsequent CHANGES_REQUESTED → approved",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStateApproved},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusApproved,
		},
		{
			name: "APPROVED then CHANGES_REQUESTED → in_review",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStateApproved},
				{Author: "bob", State: github.ReviewStateChangesRequested},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusInReview,
		},
		{
			name: "CHANGES_REQUESTED then APPROVED → approved",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStateChangesRequested},
				{Author: "bob", State: github.ReviewStateApproved},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusApproved,
		},
		{
			name: "re-request heuristic: user reviewed + pending request → open",
			pr: newPR(
				[]github.Review{
					{Author: "alice", State: github.ReviewStateCommented},
				},
				[]github.ReviewRequest{
					{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
				},
			),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusOpen,
		},
		{
			name: "user reviewed but no pending request → in_review (not re-request)",
			pr: newPR(
				[]github.Review{
					{Author: "alice", State: github.ReviewStateCommented},
				},
				nil,
			),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusInReview,
		},
		{
			name: "pending request exists but user has no review → not re-request, open if no team review",
			pr: newPR(
				nil,
				[]github.ReviewRequest{
					{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
				},
			),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusOpen,
		},
		{
			name: "PENDING and DISMISSED reviews are ignored in approved calculation",
			pr: newPR([]github.Review{
				{Author: "bob", State: github.ReviewStateApproved},
				{Author: "bob", State: github.ReviewStatePending},
				{Author: "bob", State: github.ReviewStateDismissed},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusApproved,
		},
		{
			name: "user is themselves a team member and approves → approved",
			pr: newPR([]github.Review{
				{Author: "alice", State: github.ReviewStateApproved},
			}, nil),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusApproved,
		},
		{
			name: "re-request with APPROVED: re-request takes precedence → open",
			pr: newPR(
				[]github.Review{
					{Author: "alice", State: github.ReviewStateApproved},
				},
				[]github.ReviewRequest{
					{RequestedReviewer: github.RequestedReviewer{Type: "User", Login: "alice"}},
				},
			),
			myLogin: "alice",
			teams:   newTeamService(defaultTeam),
			want:    ReviewStatusOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyReviewStatus(tt.pr, tt.myLogin, tt.teams)
			if got != tt.want {
				t.Errorf("ClassifyReviewStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
