package github

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

// userResponse is the REST response for GET /user.
type userResponse struct {
	Login string `json:"login"`
}

// FetchCurrentUser returns the login of the authenticated GitHub user via REST.
// If a Cacher is configured on the Client, results are read from and written
// to the cache to avoid redundant API calls.
func (c *Client) FetchCurrentUser() (string, error) {
	const cacheKey = "current-user"

	if c.cache != nil {
		data, found, err := c.cache.Get(cacheKey)
		if err == nil && found {
			return string(data), nil
		}
	}

	var u userResponse
	if err := c.rest.Get("user", &u); err != nil {
		return "", fmt.Errorf("fetching current user: %w", err)
	}

	if c.cache != nil {
		_ = c.cache.Set(cacheKey, []byte(u.Login))
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

// FetchMyTeams returns all teams the authenticated user belongs to via REST.
// If a Cacher is configured on the Client, results are read from and written
// to the cache to avoid redundant API calls.
func (c *Client) FetchMyTeams() ([]UserTeam, error) {
	cacheKey := "my-teams"

	if c.cache != nil {
		data, found, err := c.cache.Get(cacheKey)
		if err == nil && found {
			var cached []UserTeam
			if jsonErr := json.Unmarshal(data, &cached); jsonErr == nil {
				return cached, nil
			}
		}
	}

	var teams []UserTeam
	if err := c.rest.Get("user/teams?per_page=100", &teams); err != nil {
		return nil, fmt.Errorf("fetching user teams: %w", err)
	}

	if c.cache != nil {
		if data, err := json.Marshal(teams); err == nil {
			_ = c.cache.Set(cacheKey, data)
		}
	}

	return teams, nil
}

// FetchChildTeams retrieves all child teams of the given parent team via REST.
// If a Cacher is configured on the Client, results are read from and written
// to the cache to avoid redundant API calls.
func (c *Client) FetchChildTeams(org, parentSlug string) ([]ChildTeam, error) {
	cacheKey := "child-teams:" + org + "/" + parentSlug

	if c.cache != nil {
		data, found, err := c.cache.Get(cacheKey)
		if err == nil && found {
			var cached []ChildTeam
			if jsonErr := json.Unmarshal(data, &cached); jsonErr == nil {
				return cached, nil
			}
		}
	}

	var children []ChildTeam
	path := fmt.Sprintf("orgs/%s/teams/%s/teams?per_page=100", org, parentSlug)
	if err := c.rest.Get(path, &children); err != nil {
		return nil, fmt.Errorf("fetching child teams for %s/%s: %w", org, parentSlug, err)
	}

	if c.cache != nil {
		if data, err := json.Marshal(children); err == nil {
			_ = c.cache.Set(cacheKey, data)
		}
	}

	return children, nil
}

// FetchIsOrgMember reports whether login is a member of the given organization.
// The GitHub REST API returns 204 for members and 404 for non-members.
// If a Cacher is configured on the Client, results are read from and written
// to the cache to avoid redundant API calls. Cache errors are silently ignored.
// On API error (non-404), the function returns false, err without writing to cache.
func (c *Client) FetchIsOrgMember(org, login string) (bool, error) {
	cacheKey := "org-member:" + org + ":" + login

	if c.cache != nil {
		data, found, err := c.cache.Get(cacheKey)
		if err == nil && found {
			var cached bool
			if jsonErr := json.Unmarshal(data, &cached); jsonErr == nil {
				return cached, nil
			}
		}
	}

	path := fmt.Sprintf("orgs/%s/members/%s", org, login)
	err := c.rest.Get(path, nil)
	if err == nil {
		if c.cache != nil {
			if data, jsonErr := json.Marshal(true); jsonErr == nil {
				_ = c.cache.Set(cacheKey, data)
			}
		}
		return true, nil
	}
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
		if c.cache != nil {
			if data, jsonErr := json.Marshal(false); jsonErr == nil {
				_ = c.cache.Set(cacheKey, data)
			}
		}
		return false, nil
	}
	return false, fmt.Errorf("checking org membership for %s/%s: %w", org, login, err)
}
