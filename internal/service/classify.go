package service

import "github.com/radiohead/gh-inbox/internal/github"

// ReviewType represents the canonical classification of why a PR appears in the
// user's review queue. The classification is exclusive: each PR gets exactly
// one review type, with direct taking precedence over team.
type ReviewType string

const (
	ReviewTypeDirect    ReviewType = "direct"
	ReviewTypeTeam      ReviewType = "team"
	ReviewTypeCodeowner ReviewType = "codeowner"
)

// Classify returns the canonical review type for a single PR.
// Precedence: direct > team > codeowner.
func Classify(pr github.PullRequest, myLogin string, teams *TeamService) ReviewType {
	if matchesDirect(pr, myLogin, teams) {
		return ReviewTypeDirect
	}
	if matchesTeam(pr, myLogin, teams) {
		return ReviewTypeTeam
	}
	return ReviewTypeCodeowner
}
