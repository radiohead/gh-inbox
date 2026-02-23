package filter

import (
	"strings"

	"github.com/radiohead/gh-inbox/internal/github"
)

// Mode controls which CODEOWNERS-related PRs are included in the output.
type Mode int

const (
	// ModeDefault hides PRs where all my review requests come solely from
	// CODEOWNERS and there are other explicit human reviewers on the PR.
	ModeDefault Mode = iota

	// ModeIncludeAll disables CODEOWNERS filtering: all PRs where I'm a
	// reviewer are shown regardless of how the review was requested.
	ModeIncludeAll

	// ModeSolo shows only PRs where I'm the sole reviewer via CODEOWNERS
	// (useful for batch-approving low-noise code changes).
	ModeSolo
)

// IsMyTeamFunc reports whether the authenticated user is a member of the team
// identified by org and slug.
type IsMyTeamFunc func(org, slug string) bool

// CodeOwners filters prs according to CODEOWNERS membership logic.
//
// Algorithm per PR:
//  1. Extract org from PR.Repository.NameWithOwner (split on "/").
//  2. Classify each review request as "mine" or "other":
//     - "mine" if Type=="User" and Login==myLogin
//     - "mine" if Type=="Team" and isMyTeam(org, Slug) returns true
//     - otherwise "other"
//  3. If mode == ModeIncludeAll: always include.
//  4. If no "mine" requests: skip.
//  5. allMineAreCodeowner = every "mine" request has AsCodeOwner==true.
//  6. If mode == ModeSolo: include only if allMineAreCodeowner && len(others)==0.
//  7. If mode == ModeDefault:
//     - If allMineAreCodeowner && len(others)>0: skip (noise from automated CODEOWNERS).
//     - Otherwise: include.
func CodeOwners(prs []github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc, mode Mode) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))

	for _, pr := range prs {
		if includePR(pr, myLogin, isMyTeam, mode) {
			result = append(result, pr)
		}
	}

	return result
}

func includePR(pr github.PullRequest, myLogin string, isMyTeam IsMyTeamFunc, mode Mode) bool {
	// Extract org from "org/repo".
	parts := strings.SplitN(pr.Repository.NameWithOwner, "/", 2)
	org := ""
	if len(parts) == 2 {
		org = parts[0]
	}

	// If include-all mode, show everything.
	if mode == ModeIncludeAll {
		return true
	}

	var mineCodeOwner []bool // asCodeOwner per "mine" request
	otherCount := 0

	for _, rr := range pr.ReviewRequests.Nodes {
		isMine := false
		switch rr.RequestedReviewer.Type {
		case "User":
			isMine = rr.RequestedReviewer.Login == myLogin
		case "Team":
			isMine = isMyTeam(org, rr.RequestedReviewer.Slug)
		}

		if isMine {
			mineCodeOwner = append(mineCodeOwner, rr.AsCodeOwner)
		} else {
			otherCount++
		}
	}

	// No requests directed at me — skip.
	if len(mineCodeOwner) == 0 {
		return false
	}

	allMineAreCodeowner := true
	for _, co := range mineCodeOwner {
		if !co {
			allMineAreCodeowner = false
			break
		}
	}

	switch mode {
	case ModeSolo:
		return allMineAreCodeowner && otherCount == 0
	default: // ModeDefault
		if allMineAreCodeowner && otherCount > 0 {
			return false
		}
		return true
	}
}
