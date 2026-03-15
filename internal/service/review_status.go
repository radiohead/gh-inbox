package service

import "github.com/radiohead/gh-inbox/internal/github"

// ReviewStatus represents the review state of a pull request from the
// perspective of the filter (FR-001).
type ReviewStatus string

const (
	// ReviewStatusOpen means no reviews have been submitted yet.
	ReviewStatusOpen ReviewStatus = "open"
	// ReviewStatusInReview means at least one review has been submitted but
	// no approval has been given.
	ReviewStatusInReview ReviewStatus = "in_review"
	// ReviewStatusApproved means at least one approving review has been submitted.
	ReviewStatusApproved ReviewStatus = "approved"
)

// ReviewStatusSet is a set of ReviewStatus values for filter matching.
type ReviewStatusSet map[ReviewStatus]bool

// ClassifyReviewStatus returns the ReviewStatus for a pull request from the
// perspective of the authenticated user (myLogin).
//
// Classification rules:
//   - Reviews with state PENDING or DISMISSED are ignored.
//   - Only reviews from team members (members of the authenticated user's teams,
//     including the user themselves) are considered when determining open status.
//   - Re-request heuristic: if the user has at least one submitted review AND a
//     pending ReviewRequest node, the PR is treated as re-requested → open.
//   - open: no team member has submitted an actionable review (after re-request check).
//   - approved: at least one APPROVED review exists among all reviews, and no
//     subsequent review has state CHANGES_REQUESTED.
//   - in_review: otherwise (at least one actionable review, but not approved).
func ClassifyReviewStatus(pr github.PullRequest, myLogin string, teams *TeamService) ReviewStatus {
	org := pr.Repository.Owner

	// Collect actionable reviews (excluding PENDING and DISMISSED).
	type actionableReview struct {
		author string
		state  github.ReviewState
	}
	var actionable []actionableReview
	for _, r := range pr.Reviews {
		if r.State == github.ReviewStatePending || r.State == github.ReviewStateDismissed {
			continue
		}
		actionable = append(actionable, actionableReview{author: r.Author, state: r.State})
	}

	// Re-request heuristic: user has a submitted review AND a pending ReviewRequest.
	userHasReview := false
	for _, r := range actionable {
		if r.author == myLogin {
			userHasReview = true
			break
		}
	}
	userHasPendingRequest := false
	for _, rr := range pr.ReviewRequests.Nodes {
		if rr.RequestedReviewer.Type == "User" && rr.RequestedReviewer.Login == myLogin {
			userHasPendingRequest = true
			break
		}
	}
	if userHasReview && userHasPendingRequest {
		return ReviewStatusOpen
	}

	// Check if any team member has submitted an actionable review.
	// "Team member" means: the user themselves, or a member of any team the user belongs to.
	teamMemberReviewExists := false
	for _, r := range actionable {
		if r.author == myLogin || teams.SharesTeamWith(org, r.author) {
			teamMemberReviewExists = true
			break
		}
	}
	if !teamMemberReviewExists {
		return ReviewStatusOpen
	}

	// Determine approved vs in_review using all actionable reviews.
	// approved: at least one APPROVED and no subsequent CHANGES_REQUESTED.
	// We scan from the end: find the last APPROVED index and check if any
	// CHANGES_REQUESTED appears after it.
	lastApprovedIdx := -1
	for i, r := range actionable {
		if r.state == github.ReviewStateApproved {
			lastApprovedIdx = i
		}
	}
	if lastApprovedIdx == -1 {
		return ReviewStatusInReview
	}
	for _, r := range actionable[lastApprovedIdx+1:] {
		if r.state == github.ReviewStateChangesRequested {
			return ReviewStatusInReview
		}
	}
	return ReviewStatusApproved
}
