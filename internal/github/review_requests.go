package github

// ReviewRequestConnection holds the list of review requests on a PR.
type ReviewRequestConnection struct {
	Nodes []ReviewRequest `json:"nodes"`
}

// ReviewRequest represents a request for a specific reviewer on a PR.
type ReviewRequest struct {
	AsCodeOwner       bool              `json:"asCodeOwner"`
	RequestedReviewer RequestedReviewer `json:"requestedReviewer"`
}

// RequestedReviewer identifies who was asked to review (User or Team).
// Type is "User" or "Team"; Login holds the user's login or the team's slug.
type RequestedReviewer struct {
	Type  string `json:"type,omitempty"`
	Login string `json:"login,omitempty"`
}
