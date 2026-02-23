package github

// PullRequest represents a GitHub pull request with review request information.
type PullRequest struct {
	Number         int                      `json:"number"`
	Title          string                   `json:"title"`
	URL            string                   `json:"url"`
	Repository     Repository               `json:"repository"`
	ReviewRequests ReviewRequestConnection  `json:"reviewRequests"`
}

// Repository identifies the repository a PR belongs to.
type Repository struct {
	NameWithOwner string `json:"nameWithOwner"`
}

// ReviewRequestConnection holds the list of review requests on a PR.
type ReviewRequestConnection struct {
	Nodes []ReviewRequest `json:"nodes"`
}

// ReviewRequest represents a request for a specific reviewer on a PR.
type ReviewRequest struct {
	AsCodeOwner       bool              `json:"asCodeOwner"`
	RequestedReviewer RequestedReviewer `json:"requestedReviewer"`
}

// RequestedReviewer is a union type for User or Team reviewers.
// Type is populated from GraphQL __typename ("User" or "Team").
type RequestedReviewer struct {
	Type  string `json:"type,omitempty"`
	Login string `json:"login,omitempty"` // User only
	Name  string `json:"name,omitempty"`  // Team only
	Slug  string `json:"slug,omitempty"`  // Team only
}
