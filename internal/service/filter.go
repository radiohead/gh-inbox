package service

import "github.com/radiohead/gh-inbox/internal/github"

// Mode controls which PRs are included in the output.
type Mode int

const (
	// ModeAll is the default — no filtering, all PRs are shown.
	ModeAll Mode = iota

	// ModeDirect shows PRs where I'm requested as a User AND no other User
	// reviewer shares any team with me.
	ModeDirect

	// ModeCodeowner shows PRs where ALL my review requests come from
	// CODEOWNERS (asCodeOwner=true for every "mine" request).
	ModeCodeowner

	// ModeTeam shows PRs where my team is requested AND no User reviewer
	// is a member of any of my requested teams.
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

// filterDirect includes a PR when:
//   - I'm requested as a User (requestedReviewer.Login == myLogin, type == "User")
//   - No other User reviewer shares any team with me
func filterDirect(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		org := pr.Repository.Owner
		meRequested := false
		var otherUsers []string

		for _, rr := range pr.ReviewRequests.Nodes {
			if rr.RequestedReviewer.Type != "User" {
				continue
			}
			if rr.RequestedReviewer.Login == myLogin {
				meRequested = true
			} else {
				otherUsers = append(otherUsers, rr.RequestedReviewer.Login)
			}
		}

		if !meRequested {
			continue
		}

		sharesTeam := false
		for _, other := range otherUsers {
			if teams.SharesTeamWith(org, other) {
				sharesTeam = true
				break
			}
		}
		if !sharesTeam {
			result = append(result, pr)
		}
	}
	return result
}

// filterCodeowner includes a PR when all my review requests have asCodeOwner=true.
func filterCodeowner(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
	result := make([]github.PullRequest, 0, len(prs))
	for _, pr := range prs {
		org := pr.Repository.Owner
		rc := classifyReviewRequests(pr, myLogin, org, teams)
		if len(rc.mineCodeOwner) == 0 {
			continue
		}
		if allTrue(rc.mineCodeOwner) {
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

// reviewClassification holds the result of classifying review requests for a PR.
type reviewClassification struct {
	mineCodeOwner []bool
}

// classifyReviewRequests classifies each review request as "mine" or not.
func classifyReviewRequests(pr github.PullRequest, myLogin, org string, teams *TeamService) reviewClassification {
	var rc reviewClassification
	for _, rr := range pr.ReviewRequests.Nodes {
		isMine := false
		switch rr.RequestedReviewer.Type {
		case "User":
			isMine = rr.RequestedReviewer.Login == myLogin
		case "Team":
			isMine = teams.IsTeamMember(org, rr.RequestedReviewer.Login, myLogin)
		}
		if isMine {
			rc.mineCodeOwner = append(rc.mineCodeOwner, rr.AsCodeOwner)
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
