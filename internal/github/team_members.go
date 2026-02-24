package github

import (
	"encoding/json"
	"fmt"
)

// userResponse is the REST response for GET /user.
type userResponse struct {
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

// FetchTeamMembers retrieves all members of the given org/slug team via REST.
// If a Cacher is configured on the Client, results are read from and written
// to the cache to avoid redundant API calls.
func (c *Client) FetchTeamMembers(org, slug string) ([]TeamMember, error) {
	cacheKey := "team:" + org + "/" + slug

	if c.cache != nil {
		data, found, err := c.cache.Get(cacheKey)
		if err == nil && found {
			var cached []TeamMember
			if jsonErr := json.Unmarshal(data, &cached); jsonErr == nil {
				return cached, nil
			}
		}
	}

	var members []TeamMember
	path := fmt.Sprintf("orgs/%s/teams/%s/members?per_page=100", org, slug)
	if err := c.rest.Get(path, &members); err != nil {
		return nil, fmt.Errorf("fetching team members for %s/%s: %w", org, slug, err)
	}

	if c.cache != nil {
		if data, err := json.Marshal(members); err == nil {
			_ = c.cache.Set(cacheKey, data)
		}
	}

	return members, nil
}
