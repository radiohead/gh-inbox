package service

import "github.com/radiohead/gh-inbox/internal/github"

// Source represents the canonical classification of why a PR appears in the
// user's review queue. The classification is exclusive: each PR gets exactly
// one source, with direct taking precedence over team.
type Source string

const (
	SourceDirect    Source = "direct"
	SourceTeam      Source = "team"
	SourceCodeowner Source = "codeowner"
)

// Classify returns the canonical source for a single PR.
// Precedence: direct > team > codeowner.
func Classify(pr github.PullRequest, myLogin string, teams *TeamService) Source {
	if matchesDirect(pr, myLogin, teams) {
		return SourceDirect
	}
	if matchesTeam(pr, myLogin, teams) {
		return SourceTeam
	}
	return SourceCodeowner
}
