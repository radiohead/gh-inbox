package service

import "github.com/radiohead/gh-inbox/internal/github"

// TeamMemberFetcher can retrieve all members of a GitHub team.
// github.Client satisfies this interface implicitly.
type TeamMemberFetcher interface {
	FetchTeamMembers(org, slug string) ([]github.TeamMember, error)
}

// TeamService provides team membership queries with in-process caching.
// It wraps a TeamMemberFetcher and lazily caches results per org/slug pair.
type TeamService struct {
	fetcher TeamMemberFetcher
	cache   map[string]map[string]bool // "org/slug" -> login set
}

// NewTeamService creates a TeamService backed by the given fetcher.
func NewTeamService(fetcher TeamMemberFetcher) *TeamService {
	return &TeamService{
		fetcher: fetcher,
		cache:   make(map[string]map[string]bool),
	}
}

// IsTeamMember reports whether login is a member of the given org/slug team.
// Results are lazily cached per team (org+slug key). On fetch error the
// function returns true (fail-open) to avoid hiding PRs due to transient API
// failures.
func (s *TeamService) IsTeamMember(org, slug, login string) bool {
	key := org + "/" + slug
	if _, cached := s.cache[key]; !cached {
		members, err := s.fetcher.FetchTeamMembers(org, slug)
		if err != nil {
			return true // fail-open
		}
		set := make(map[string]bool, len(members))
		for _, m := range members {
			set[m.Login] = true
		}
		s.cache[key] = set
	}
	return s.cache[key][login]
}
