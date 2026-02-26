package service

import "github.com/radiohead/gh-inbox/internal/github"

// Mode controls which PRs are included in the output.
type Mode int

const (
	// ModeAll is the default — no filtering, all PRs are shown.
	ModeAll Mode = iota

	// ModeDirect shows PRs where I'm requested as a User AND either I have an
	// explicit (non-codeowner) request, or I'm the sole reviewer on the PR.
	ModeDirect

	// ModeCodeowner shows PRs that are neither direct nor team — the pure
	// fallback bucket covering codeowner-only review requests not handled by
	// the other two modes.
	ModeCodeowner

	// ModeTeam shows PRs where my team is requested AND no User reviewer
	// is a member of any of my requested teams.
	ModeTeam
)

// Filter dispatches to the appropriate filter based on mode.
func Filter(prs []github.PullRequest, myLogin string, teams *TeamService, mode Mode) []github.PullRequest {
	switch mode {
	case ModeDirect:
		return filterDirect(prs, myLogin)
	case ModeCodeowner:
		return filterCodeowner(prs, myLogin, teams)
	case ModeTeam:
		return filterTeam(prs, myLogin, teams)
	default:
		return prs
	}
}

// matchesDirect reports whether myLogin is requested as a User AND either
// has an explicit (non-codeowner) request OR is the sole reviewer.
func matchesDirect(pr github.PullRequest, myLogin string) bool {
	isMine, meExplicit, hasOthers := false, false, false
	for _, rr := range pr.ReviewRequests.Nodes {
		if rr.RequestedReviewer.Type == "User" && rr.RequestedReviewer.Login == myLogin {
			isMine = true
			if !rr.AsCodeOwner {
				meExplicit = true
			}
		} else {
			hasOthers = true
		}
	}
	return isMine && (meExplicit || !hasOthers)
}

// matchesTeam reports whether at least one of myLogin's teams is requested.
func matchesTeam(pr github.PullRequest, myLogin string, teams *TeamService) bool {
	org := pr.Repository.Owner
	for _, rr := range pr.ReviewRequests.Nodes {
		if rr.RequestedReviewer.Type == "Team" && teams.IsTeamMember(org, rr.RequestedReviewer.Login, myLogin) {
			return true
		}
	}
	return false
}

// filterDirect includes a PR when:
//   - I'm requested as a User (requestedReviewer.Login == myLogin, type == "User")
//   - AND at least one of my requests has AsCodeOwner=false (explicit), OR I'm
//     the sole reviewer (no other reviewers at all)
func filterDirect(prs []github.PullRequest, myLogin string) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if matchesDirect(pr, myLogin) {
			result = append(result, pr)
		}
	}
	return result
}

// filterCodeowner includes a PR when it is NOT direct AND NOT team — the
// fallback bucket for codeowner-only review requests not claimed by the
// other two modes.
func filterCodeowner(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if !matchesDirect(pr, myLogin) && !matchesTeam(pr, myLogin, teams) {
			result = append(result, pr)
		}
	}
	return result
}

// filterTeam includes a PR when:
//   - At least one of my teams is requested (Team-type request where I'm a member)
//   - No User reviewer is a member of any of my requested teams
func filterTeam(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		org := pr.Repository.Owner

		// Collect my team slugs and all User reviewer logins.
		var myTeamSlugs []string
		var userLogins []string

		for _, rr := range pr.ReviewRequests.Nodes {
			switch rr.RequestedReviewer.Type {
			case "Team":
				if teams.IsTeamMember(org, rr.RequestedReviewer.Login, myLogin) {
					myTeamSlugs = append(myTeamSlugs, rr.RequestedReviewer.Login)
				}
			case "User":
				userLogins = append(userLogins, rr.RequestedReviewer.Login)
			}
		}

		if len(myTeamSlugs) == 0 {
			continue
		}

		// Exclude if any User reviewer is a member of any of my requested teams.
		anyMember := false
		for _, login := range userLogins {
			for _, slug := range myTeamSlugs {
				if teams.IsTeamMember(org, slug, login) {
					anyMember = true
					break
				}
			}
			if anyMember {
				break
			}
		}

		if !anyMember {
			result = append(result, pr)
		}
	}
	return result
}
