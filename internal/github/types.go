package github

// PullRequest represents a GitHub pull request with review request information.
type PullRequest struct {
	Number         int                     `json:"number"`
	Title          string                  `json:"title"`
	URL            string                  `json:"url"`
	Repository     Repository              `json:"repository"`
	ReviewRequests ReviewRequestConnection `json:"reviewRequests"`
}

// Repository identifies the repository a PR belongs to.
type Repository struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}
