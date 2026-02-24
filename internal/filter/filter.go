package filter

import (
	"github.com/radiohead/gh-inbox/internal/github"
)

// Mode controls which PRs are included in the output.
type Mode int

const (
	// ModeAll is the default — no filtering, all PRs are shown regardless of
	// how the review was requested.
	ModeAll Mode = iota

	// ModeDirect hides PRs where all my review requests come solely from
	// CODEOWNERS and there are other explicit human reviewers on the PR.
	ModeDirect

	// ModeCodeowner shows only PRs where I'm the sole reviewer via CODEOWNERS
	// (useful for batch-approving low-noise code changes).
	ModeCodeowner
)

// IsMyTeamFunc reports whether the authenticated user is a member of the team
// identified by org and slug.
type IsMyTeamFunc func(org, slug string) bool

// Filter dispatches to the appropriate sub-filter based on mode.
// ModeAll returns prs unchanged.
// ModeDirect delegates to FilterDirect.
// ModeCodeowner delegates to FilterCodeowner.
func Filter(prs []github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc, mode Mode) []github.PullRequest {
	switch mode {
	case ModeDirect:
		return FilterDirect(prs, myLogin, isMyTeam)
	case ModeCodeowner:
		return FilterCodeowner(prs, myLogin, isMyTeam)
	default: // ModeAll
		return prs
	}
}

// FilterDirect hides PRs where all my review requests come solely from
// CODEOWNERS and there are other explicit human reviewers on the PR.
func FilterDirect(prs []github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		rc := classifyReviewRequests(pr, myLogin, isMyTeam)
		if len(rc.mineCodeOwner) == 0 {
			continue
		}
		if allTrue(rc.mineCodeOwner) && rc.otherCount > 0 {
			continue
		}
		result = append(result, pr)
	}
	return result
}

// FilterCodeowner shows only PRs where I'm the sole reviewer via CODEOWNERS.
func FilterCodeowner(prs []github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		rc := classifyReviewRequests(pr, myLogin, isMyTeam)
		if len(rc.mineCodeOwner) == 0 {
			continue
		}
		if allTrue(rc.mineCodeOwner) && rc.otherCount == 0 {
			result = append(result, pr)
		}
	}
	return result
}

// reviewClassification holds the result of classifying review requests for a PR.
type reviewClassification struct {
	mineCodeOwner []bool // asCodeOwner value per "mine" review request
	otherCount    int
}

// classifyReviewRequests classifies each review request in pr as "mine" or "other".
func classifyReviewRequests(pr github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc) reviewClassification {
	org := pr.Repository.Owner
	var rc reviewClassification
	for _, rr := range pr.ReviewRequests.Nodes {
		isMine := false
		switch rr.RequestedReviewer.Type {
		case "User":
			isMine = rr.RequestedReviewer.Login == myLogin
		case "Team":
			isMine = isMyTeam(org, rr.RequestedReviewer.Login)
		}
		if isMine {
			rc.mineCodeOwner = append(rc.mineCodeOwner, rr.AsCodeOwner)
		} else {
			rc.otherCount++
		}
	}
	return rc
}

// allTrue reports whether every value in vals is true.
func allTrue(vals []bool) bool {
	for _, v := range vals {
		if !v {
			return false
		}
	}
	return true
}
