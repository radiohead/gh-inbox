package github

import "fmt"

// userResponse is the REST response for GET /user.
type userResponse struct {
	Login string `json:"login"`
}

// teamMemberResponse is a single entry in the REST response for GET /orgs/{org}/teams/{slug}/members.
type teamMemberResponse struct {
	Login string `json:"login"`
}

// FetchCurrentUser returns the login of the authenticated GitHub user via REST.
func (c *Client) FetchCurrentUser() (string, error) {
	var u userResponse
	if err := c.rest.Get("user", &u); err != nil {
		return "", fmt.Errorf("fetching current user: %w", err)
	}
	return u.Login, nil
}

// IsTeamMember reports whether login is a member of the given org/slug team.
// Results are lazily cached per team (org+slug key). On REST error the function
// returns true (fail-open) to avoid hiding PRs due to transient API failures.
func (c *Client) IsTeamMember(org, slug, login string) bool {
	if c.teamMembers == nil {
		c.teamMembers = make(map[string]map[string]bool)
	}

	key := org + "/" + slug
	if _, cached := c.teamMembers[key]; !cached {
		members, err := c.fetchTeamMembers(org, slug)
		if err != nil {
			// Fail-open: treat user as member to avoid hiding PRs.
			return true
		}
		set := make(map[string]bool, len(members))
		for _, m := range members {
			set[m.Login] = true
		}
		c.teamMembers[key] = set
	}

	return c.teamMembers[key][login]
}

// fetchTeamMembers retrieves all members of the given org/slug team via REST.
func (c *Client) fetchTeamMembers(org, slug string) ([]teamMemberResponse, error) {
	var members []teamMemberResponse
	path := fmt.Sprintf("orgs/%s/teams/%s/members?per_page=100", org, slug)
	if err := c.rest.Get(path, &members); err != nil {
		return nil, fmt.Errorf("fetching team members for %s/%s: %w", org, slug, err)
	}
	return members, nil
}
