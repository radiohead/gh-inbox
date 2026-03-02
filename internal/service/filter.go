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

// modeToSource maps a filter Mode to its corresponding Source.
func modeToSource(m Mode) Source {
	switch m {
	case ModeDirect:
		return SourceDirect
	case ModeTeam:
		return SourceTeam
	case ModeCodeowner:
		return SourceCodeowner
	default:
		return "" // ModeAll handled before this is called
	}
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
