package github

import (
	"time"
)

// searchReviewRequestedQuery is the GraphQL query struct for fetching PRs
// where review has been requested from the current user.
type searchReviewRequestedQuery struct {
	Search struct {
		IssueCount int
		Nodes      []struct {
			PullRequest searchPRNode `graphql:"... on PullRequest"`
		}
	} `graphql:"search(query: $query, type: ISSUE, first: $first)"`
}

// searchPRNode represents a pull request node returned in search results.
type searchPRNode struct {
	Number    int
	Title     string
	URL       string
	CreatedAt time.Time
	Repository struct {
		NameWithOwner string
	}
	ReviewRequests struct {
		Nodes []searchReviewRequestNode
	} `graphql:"reviewRequests(first: 20)"`
}

// searchReviewRequestNode represents a single review request on a PR.
type searchReviewRequestNode struct {
	AsCodeOwner       bool
	RequestedReviewer struct {
		User struct {
			Login string
		} `graphql:"... on User"`
		Team struct {
			Name string
			Slug string
		} `graphql:"... on Team"`
		TypeName string `graphql:"__typename"`
	}
}

// buildReviewRequestedSearchQuery returns a GitHub search query string that
// finds open PRs where review has been requested from the current user.
// If org is non-empty, results are filtered to that organization.
func buildReviewRequestedSearchQuery(org string) string {
	q := "is:open is:pr review-requested:@me"
	if org != "" {
		q += " org:" + org
	}
	return q
}
