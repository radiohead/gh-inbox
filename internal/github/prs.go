package github

// FetchReviewRequestedPRs fetches open PRs in org where review has been
// requested from the current user, returning unfiltered results.
func (c *Client) FetchReviewRequestedPRs(org string) ([]PullRequest, error) {
	var q searchReviewRequestedQuery
	variables := map[string]interface{}{
		"query": buildReviewRequestedSearchQuery(org),
		"first": 50,
	}
	if err := c.gql.Query("SearchReviewRequestedPRs", &q, variables); err != nil {
		return nil, err
	}

	prs := make([]PullRequest, 0, len(q.Search.Nodes))
	for _, node := range q.Search.Nodes {
		prs = append(prs, convertSearchPRNode(node.PullRequest))
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

	return PullRequest{
		Number: n.Number,
		Title:  n.Title,
		URL:    n.URL,
		Repository: Repository{
			NameWithOwner: n.Repository.NameWithOwner,
		},
		ReviewRequests: ReviewRequestConnection{
			Nodes: requests,
		},
	}
}
