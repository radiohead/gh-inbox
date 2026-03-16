package service

import "github.com/radiohead/gh-inbox/internal/github"

// ReviewTypeSet is a set of ReviewType values for filter matching.
type ReviewTypeSet map[ReviewType]bool

// AuthorSourceSet is a set of AuthorSource values for filter matching.
type AuthorSourceSet map[AuthorSource]bool

// FilterCriteria specifies which PRs to include.
// A nil set matches all values on that dimension.
type FilterCriteria struct {
	ReviewTypes    ReviewTypeSet    // nil = match all
	AuthorSources  AuthorSourceSet  // nil = match all
	ReviewStatuses ReviewStatusSet  // nil = match all
}

// Matches reports whether cp satisfies the criteria.
func (fc FilterCriteria) Matches(cp ClassifiedPR) bool {
	if fc.ReviewTypes != nil && !fc.ReviewTypes[cp.ReviewType] {
		return false
	}
	if fc.AuthorSources != nil && !fc.AuthorSources[cp.AuthorSource] {
		return false
	}
	if fc.ReviewStatuses != nil && !fc.ReviewStatuses[cp.ReviewStatus] {
		return false
	}
	return true
}

// CriteriaFilter keeps only PRs that satisfy the configured FilterCriteria.
type CriteriaFilter struct {
	Criteria FilterCriteria
}

// Apply returns the subset of prs that match the criteria.
func (f *CriteriaFilter) Apply(prs []ClassifiedPR) []ClassifiedPR {
	result := make([]ClassifiedPR, 0, len(prs))
	for _, cp := range prs {
		if f.Criteria.Matches(cp) {
			result = append(result, cp)
		}
	}
	return result
}

// Preset is a named predefined filter combination.
type Preset string

const (
	PresetAll    Preset = "all"
	PresetFocus  Preset = "focus"
	PresetNearby Preset = "nearby"
	PresetOrg    Preset = "org"
)

// PresetCriteria returns the FilterCriteria for the named preset.
func PresetCriteria(p Preset) FilterCriteria {
	switch p {
	case PresetFocus:
		// Highest-priority: my team's PRs where I'm directly responsible.
		return FilterCriteria{
			ReviewTypes:    ReviewTypeSet{ReviewTypeDirect: true, ReviewTypeCodeowner: true},
			AuthorSources:  AuthorSourceSet{AuthorSourceTeam: true},
			ReviewStatuses: ReviewStatusSet{ReviewStatusOpen: true},
		}
	case PresetNearby:
		// Expand to sibling teams.
		return FilterCriteria{
			ReviewTypes:    ReviewTypeSet{ReviewTypeDirect: true, ReviewTypeCodeowner: true},
			AuthorSources:  AuthorSourceSet{AuthorSourceTeam: true, AuthorSourceGroup: true},
			ReviewStatuses: ReviewStatusSet{ReviewStatusOpen: true},
		}
	case PresetOrg:
		// All org PRs, excluding external contributors.
		return FilterCriteria{
			AuthorSources: AuthorSourceSet{
				AuthorSourceTeam:  true,
				AuthorSourceGroup: true,
				AuthorSourceOrg:   true,
			},
			// ReviewStatuses nil = match all
		}
	default: // PresetAll or unknown
		return FilterCriteria{} // nil sets = match all
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
