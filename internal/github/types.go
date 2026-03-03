package github

import "time"

// PullRequest represents a GitHub pull request with review request information.
type PullRequest struct {
	Number         int                     `json:"number"`
	Title          string                  `json:"title"`
	URL            string                  `json:"url"`
	Author         string                  `json:"author"`
	CreatedAt      time.Time               `json:"createdAt"`
	Repository     Repository              `json:"repository"`
	ReviewRequests ReviewRequestConnection `json:"reviewRequests"`
}

// Repository identifies the repository a PR belongs to.
type Repository struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

// TeamMember represents a member of a GitHub team.
type TeamMember struct {
	Login string `json:"login"`
}

// UserTeam represents a team the authenticated user belongs to.
type UserTeam struct {
	Slug         string           `json:"slug"`
	Organization TeamOrganization `json:"organization"`
	Parent       *ParentTeam      `json:"parent"`
}

// ParentTeam identifies a team's parent in a nested team hierarchy.
type ParentTeam struct {
	Slug string `json:"slug"`
}

// ChildTeam represents a child team within a parent team hierarchy.
type ChildTeam struct {
	Slug string `json:"slug"`
}

// TeamOrganization identifies the org a team belongs to.
type TeamOrganization struct {
	Login string `json:"login"`
}
