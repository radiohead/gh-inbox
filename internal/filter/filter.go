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

	// ModeDirect shows only PRs where at least one of my review requests is NOT
	// from CODEOWNERS (I was explicitly requested as a reviewer).
	ModeDirect

	// ModeCodeowner shows only PRs where ALL my review requests come from
	// CODEOWNERS (none were explicitly requested), regardless of other reviewers.
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

// FilterDirect shows only PRs where at least one of my review requests is NOT
// from CODEOWNERS (I was explicitly requested as a reviewer).
func FilterDirect(prs []github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		rc := classifyReviewRequests(pr, myLogin, isMyTeam)
		if len(rc.mineCodeOwner) == 0 {
			continue
		}
		if !allTrue(rc.mineCodeOwner) {
			result = append(result, pr)
		}
	}
	return result
}

// FilterCodeowner shows only PRs where ALL my review requests come from
// CODEOWNERS (none were explicitly requested), regardless of other reviewers.
func FilterCodeowner(prs []github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		rc := classifyReviewRequests(pr, myLogin, isMyTeam)
		if len(rc.mineCodeOwner) == 0 {
			continue
		}
		if allTrue(rc.mineCodeOwner) {
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
