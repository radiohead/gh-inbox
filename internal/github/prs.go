package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	graphql "github.com/cli/shurcooL-graphql"

	gherrors "github.com/radiohead/gh-inbox/internal/errors"
)

// FetchReviewRequestedPRs fetches open PRs in org where review has been
// requested from the current user, returning unfiltered results.
//
// If the GraphQL query returns a SAML-enforcement error, the function logs a
// warning to stderr and returns whatever partial data GitHub provided.
func (c *Client) FetchReviewRequestedPRs(org string) ([]PullRequest, error) {
	cacheKey := "review-prs:" + org

	if c.prCache != nil && !c.skipPRCacheRead {
		data, found, err := c.prCache.Get(cacheKey)
		if err == nil && found {
			var cached []PullRequest
			if jsonErr := json.Unmarshal(data, &cached); jsonErr == nil {
				return cached, nil
			}
		}
	}

	var q searchReviewRequestedQuery
	variables := map[string]interface{}{
		"query": graphql.String(buildReviewRequestedSearchQuery(org)),
		"first": graphql.Int(50),
	}
	if err := c.gql.Query("SearchReviewRequestedPRs", &q, variables); err != nil {
		classified := gherrors.Classify(err, gherrors.GitHubClassifiers...)
		switch classified.Severity() {
		case gherrors.SeverityWarning:
			fmt.Fprintf(os.Stderr, "warning: %s\n", classified.Summary())
			// fall through — partial data may be available in q
		case gherrors.SeveritySilent:
			// fall through — ignore silently
		default:
			return nil, err
		}
	}

	prs := make([]PullRequest, 0, len(q.Search.Nodes))
	for _, node := range q.Search.Nodes {
		// SAML-blocked nodes are zero-valued (empty URL); skip them.
		if node.PullRequest.URL == "" {
			continue
		}
		prs = append(prs, convertSearchPRNode(node.PullRequest))
	}

	if c.prCache != nil {
		if data, err := json.Marshal(prs); err == nil {
			_ = c.prCache.Set(cacheKey, data)
		}
	}

	return prs, nil
}

// convertSearchPRNode maps a searchPRNode from the GraphQL response into
// the domain PullRequest type.
func convertSearchPRNode(n searchPRNode) PullRequest {
	requests := make([]ReviewRequest, 0, len(n.ReviewRequests.Nodes))
	for _, rr := range n.ReviewRequests.Nodes {
		var reviewer RequestedReviewer
		switch rr.RequestedReviewer.TypeName {
		case "User":
			reviewer = RequestedReviewer{
				Type:  "User",
				Login: rr.RequestedReviewer.User.Login,
			}
		case "Team":
			reviewer = RequestedReviewer{
				Type:  "Team",
				Login: rr.RequestedReviewer.Team.Slug,
			}
		default:
			reviewer = RequestedReviewer{Type: rr.RequestedReviewer.TypeName}
		}
		requests = append(requests, ReviewRequest{
			AsCodeOwner:       rr.AsCodeOwner,
			RequestedReviewer: reviewer,
		})
	}

	reviews := make([]Review, 0, len(n.Reviews.Nodes))
	for _, rv := range n.Reviews.Nodes {
		reviews = append(reviews, Review{
			Author: rv.Author.Login,
			State:  rv.State,
		})
	}

	owner, name := splitNameWithOwner(n.Repository.NameWithOwner)
	return PullRequest{
		Number:    n.Number,
		Title:     n.Title,
		URL:       n.URL,
		Author:    n.Author.Login,
		CreatedAt: n.CreatedAt,
		Repository: Repository{
			Owner: owner,
			Name:  name,
		},
		ReviewRequests: ReviewRequestConnection{
			Nodes: requests,
		},
		Reviews: reviews,
	}
}

// splitNameWithOwner splits a GitHub "owner/name" string into its two parts.
func splitNameWithOwner(nameWithOwner string) (owner, name string) {
	parts := strings.SplitN(nameWithOwner, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return nameWithOwner, ""
}
