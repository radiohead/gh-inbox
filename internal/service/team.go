package service

import "github.com/radiohead/gh-inbox/internal/github"

// TeamMemberFetcher can retrieve team members and the authenticated user's teams.
// github.Client satisfies this interface implicitly.
type TeamMemberFetcher interface {
	FetchTeamMembers(org, slug string) ([]github.TeamMember, error)
	FetchMyTeams() ([]github.UserTeam, error)
	FetchChildTeams(org, parentSlug string) ([]github.ChildTeam, error)
	FetchIsOrgMember(org, login string) (bool, error)
}

// TeamService provides team membership queries with in-process caching.
// It wraps a TeamMemberFetcher and lazily caches results per org/slug pair.
type TeamService struct {
	fetcher  TeamMemberFetcher
	cache    map[string]map[string]bool // "org/slug" -> login set
	myTeams  []github.UserTeam          // cached result of FetchMyTeams
	myLoaded bool                       // whether myTeams has been fetched
	myErr    error                      // error from FetchMyTeams (sticky)
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

// SharesTeamWith reports whether otherLogin is a member of any team the
// authenticated user belongs to within the given org. On fetch error the
// function returns false (fail-closed: no overlap assumed, PR stays visible
// in direct mode).
func (s *TeamService) SharesTeamWith(org, otherLogin string) bool {
	teams := s.loadMyTeams()
	if teams == nil {
		return false // fail-closed
	}
	for _, t := range teams {
		if t.Organization.Login != org {
			continue
		}
		if s.IsTeamMember(org, t.Slug, otherLogin) {
			return true
		}
	}
	return false
}

// IsSiblingTeamMember reports whether login is a member of any sibling team
// (child of the same parent) of any team the authenticated user belongs to
// within org. Returns false on any error (fail-closed).
func (s *TeamService) IsSiblingTeamMember(org, login string) bool {
	teams := s.loadMyTeams()
	if teams == nil {
		return false
	}
	for _, t := range teams {
		if t.Organization.Login != org || t.Parent == nil {
			continue
		}
		children, err := s.fetcher.FetchChildTeams(org, t.Parent.Slug)
		if err != nil {
			return false // fail-closed
		}
		for _, child := range children {
			if child.Slug == t.Slug {
				continue // skip own team
			}
			if s.IsTeamMember(org, child.Slug, login) {
				return true
			}
		}
	}
	return false
}

// IsOrgMember reports whether login is a member of the given organization.
// Returns false on any error (fail-closed).
func (s *TeamService) IsOrgMember(org, login string) bool {
	ok, err := s.fetcher.FetchIsOrgMember(org, login)
	return err == nil && ok
}

// loadMyTeams fetches and caches the authenticated user's teams.
func (s *TeamService) loadMyTeams() []github.UserTeam {
	if !s.myLoaded {
		s.myTeams, s.myErr = s.fetcher.FetchMyTeams()
		s.myLoaded = true
	}
	return s.myTeams
}
