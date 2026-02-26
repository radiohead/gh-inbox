package service

import "github.com/radiohead/gh-inbox/internal/github"

// Mode controls which PRs are included in the output.
type Mode int

const (
	// ModeAll is the default — no filtering, all PRs are shown.
	ModeAll Mode = iota

	// ModeDirect shows PRs where I'm requested as a User AND no other
	// User reviewer shares any of my teams.
	ModeDirect

	// ModeCodeowner shows PRs that match neither direct nor team — the
	// residual bucket where I'm reviewing alongside teammates.
	ModeCodeowner

	// ModeTeam shows PRs where my team is requested AND no individual
	// User reviewer shares any of my teams.
	ModeTeam
)

// Filter dispatches to the appropriate filter based on mode.
func Filter(prs []github.PullRequest, myLogin string, teams *TeamService, mode Mode) []github.PullRequest {
	switch mode {
	case ModeDirect:
		return filterDirect(prs, myLogin, teams)
	case ModeCodeowner:
		return filterCodeowner(prs, myLogin, teams)
	case ModeTeam:
		return filterTeam(prs, myLogin, teams)
	default:
		return prs
	}
}

// MatchesDirect is the exported predicate for direct-mode matching.
// Used by the output package to label PR sources.
func MatchesDirect(pr github.PullRequest, myLogin string, teams *TeamService) bool {
	return matchesDirect(pr, myLogin, teams)
}

// MatchesTeam is the exported predicate for team-mode matching.
// Used by the output package to label PR sources.
func MatchesTeam(pr github.PullRequest, myLogin string, teams *TeamService) bool {
	return matchesTeam(pr, myLogin, teams)
}

// matchesDirect reports whether myLogin is requested as a User AND no other
// User reviewer shares any of the authenticated user's teams.
func matchesDirect(pr github.PullRequest, myLogin string, teams *TeamService) bool {
	org := pr.Repository.Owner
	meRequested := false
	for _, rr := range pr.ReviewRequests.Nodes {
		if rr.RequestedReviewer.Type != "User" {
			continue
		}
		if rr.RequestedReviewer.Login == myLogin {
			meRequested = true
		} else if teams.SharesTeamWith(org, rr.RequestedReviewer.Login) {
			return false // teammate found → not direct
		}
	}
	return meRequested
}

// matchesTeam reports whether at least one of the authenticated user's teams
// is requested AND no individual User reviewer shares any of those teams.
func matchesTeam(pr github.PullRequest, myLogin string, teams *TeamService) bool {
	org := pr.Repository.Owner
	hasMyTeam := false
	for _, rr := range pr.ReviewRequests.Nodes {
		switch rr.RequestedReviewer.Type {
		case "Team":
			if teams.IsTeamMember(org, rr.RequestedReviewer.Login, myLogin) {
				hasMyTeam = true
			}
		case "User":
			if rr.RequestedReviewer.Login != myLogin &&
				teams.SharesTeamWith(org, rr.RequestedReviewer.Login) {
				return false // individual teammate → not pure team
			}
		}
	}
	return hasMyTeam
}

// filterDirect includes a PR when myLogin is requested as a User AND no other
// User reviewer shares any of the authenticated user's teams.
func filterDirect(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if matchesDirect(pr, myLogin, teams) {
			result = append(result, pr)
		}
	}
	return result
}

// filterCodeowner includes a PR when it is NOT direct AND NOT team — the
// residual bucket for PRs where the user reviews alongside teammates.
func filterCodeowner(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if !matchesDirect(pr, myLogin, teams) && !matchesTeam(pr, myLogin, teams) {
			result = append(result, pr)
		}
	}
	return result
}

// filterTeam includes a PR when at least one of the authenticated user's teams
// is requested AND no individual User reviewer shares any of those teams.
func filterTeam(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if matchesTeam(pr, myLogin, teams) {
			result = append(result, pr)
		}
	}
	return result
}
