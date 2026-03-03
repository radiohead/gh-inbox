package github

import (
	"errors"
	"testing"

	graphql "github.com/cli/shurcooL-graphql"
)

// mockGraphQLDoer is a test double for graphQLDoer. queryFunc receives the
// query struct and populates it in place, matching shurcooL-graphql semantics.
type mockGraphQLDoer struct {
	queryFunc func(name string, q interface{}, variables map[string]interface{}) error
}

func (m *mockGraphQLDoer) Query(name string, q interface{}, variables map[string]interface{}) error {
	return m.queryFunc(name, q, variables)
}

func TestFetchReviewRequestedPRs(t *testing.T) {
	tests := []struct {
		name      string
		org       string
		queryFunc func(name string, q interface{}, variables map[string]interface{}) error
		want      []PullRequest
		wantErr   bool
	}{
		{
			name: "happy path: mixed User and Team reviewers",
			org:  "test-org",
			queryFunc: func(_ string, q interface{}, variables map[string]interface{}) error {
				result := q.(*searchReviewRequestedQuery)
				result.Search.Nodes = []struct {
					PullRequest searchPRNode `graphql:"... on PullRequest"`
				}{
					{
						PullRequest: searchPRNode{
							Number: 101,
							Title:  "Add feature",
							URL:    "https://github.com/test-org/repo/pull/101",
							Author: struct{ Login string }{Login: "someuser"},
							Repository: struct{ NameWithOwner string }{
								NameWithOwner: "test-org/repo",
							},
							ReviewRequests: struct {
								Nodes []searchReviewRequestNode
							}{
								Nodes: []searchReviewRequestNode{
									{
										AsCodeOwner: false,
										RequestedReviewer: struct {
											User     struct{ Login string } `graphql:"... on User"`
											Team     struct{ Name, Slug string } `graphql:"... on Team"`
											TypeName string                     `graphql:"__typename"`
										}{
											User:     struct{ Login string }{Login: "alice"},
											TypeName: "User",
										},
									},
									{
										AsCodeOwner: true,
										RequestedReviewer: struct {
											User     struct{ Login string } `graphql:"... on User"`
											Team     struct{ Name, Slug string } `graphql:"... on Team"`
											TypeName string                     `graphql:"__typename"`
										}{
											Team:     struct{ Name, Slug string }{Name: "backend", Slug: "backend-team"},
											TypeName: "Team",
										},
									},
								},
							},
						},
					},
				}
				return nil
			},
			want: []PullRequest{
				{
					Number: 101,
					Title:  "Add feature",
					URL:    "https://github.com/test-org/repo/pull/101",
					Author: "someuser",
					Repository: Repository{Owner: "test-org", Name: "repo"},
					ReviewRequests: ReviewRequestConnection{
						Nodes: []ReviewRequest{
							{AsCodeOwner: false, RequestedReviewer: RequestedReviewer{Type: "User", Login: "alice"}},
							{AsCodeOwner: true, RequestedReviewer: RequestedReviewer{Type: "Team", Login: "backend-team"}},
						},
					},
				},
			},
		},
		{
			name: "empty result",
			org:  "empty-org",
			queryFunc: func(_ string, q interface{}, _ map[string]interface{}) error {
				result := q.(*searchReviewRequestedQuery)
				result.Search.Nodes = nil
				return nil
			},
			want: []PullRequest{},
		},
		{
			name: "API error",
			org:  "test-org",
			queryFunc: func(_ string, _ interface{}, _ map[string]interface{}) error {
				return errors.New("API rate limit exceeded")
			},
			wantErr: true,
		},
		{
			name: "CODEOWNERS-only review request preserved",
			org:  "test-org",
			queryFunc: func(_ string, q interface{}, _ map[string]interface{}) error {
				result := q.(*searchReviewRequestedQuery)
				result.Search.Nodes = []struct {
					PullRequest searchPRNode `graphql:"... on PullRequest"`
				}{
					{
						PullRequest: searchPRNode{
							Number: 202,
							Title:  "Update docs",
							URL:    "https://github.com/test-org/repo/pull/202",
							Repository: struct{ NameWithOwner string }{
								NameWithOwner: "test-org/repo",
							},
							ReviewRequests: struct {
								Nodes []searchReviewRequestNode
							}{
								Nodes: []searchReviewRequestNode{
									{
										AsCodeOwner: true,
										RequestedReviewer: struct {
											User     struct{ Login string } `graphql:"... on User"`
											Team     struct{ Name, Slug string } `graphql:"... on Team"`
											TypeName string                     `graphql:"__typename"`
										}{
											User:     struct{ Login string }{Login: "bob"},
											TypeName: "User",
										},
									},
								},
							},
						},
					},
				}
				return nil
			},
			want: []PullRequest{
				{
					Number:     202,
					Title:      "Update docs",
					URL:        "https://github.com/test-org/repo/pull/202",
					Repository: Repository{Owner: "test-org", Name: "repo"},
					ReviewRequests: ReviewRequestConnection{
						Nodes: []ReviewRequest{
							{AsCodeOwner: true, RequestedReviewer: RequestedReviewer{Type: "User", Login: "bob"}},
						},
					},
				},
			},
		},
		{
			name: "correct query string sent for org",
			org:  "my-org",
			queryFunc: func(_ string, q interface{}, variables map[string]interface{}) error {
				wantQuery := "is:open is:pr review-requested:@me org:my-org"
				gotQuery, ok := variables["query"].(graphql.String)
				if !ok {
					return errors.New("missing query variable")
				}
				if string(gotQuery) != wantQuery {
					return errors.New("wrong query: got " + string(gotQuery))
				}
				result := q.(*searchReviewRequestedQuery)
				result.Search.Nodes = nil
				return nil
			},
			want: []PullRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithDoer(&mockGraphQLDoer{queryFunc: tt.queryFunc})
			got, err := client.FetchReviewRequestedPRs(tt.org)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i, pr := range got {
				w := tt.want[i]
				if pr.Number != w.Number {
					t.Errorf("[%d] Number = %d, want %d", i, pr.Number, w.Number)
				}
				if pr.Title != w.Title {
					t.Errorf("[%d] Title = %q, want %q", i, pr.Title, w.Title)
				}
				if pr.URL != w.URL {
					t.Errorf("[%d] URL = %q, want %q", i, pr.URL, w.URL)
				}
				if pr.Author != w.Author {
					t.Errorf("[%d] Author = %q, want %q", i, pr.Author, w.Author)
				}
				if pr.Repository != w.Repository {
					t.Errorf("[%d] Repo = %+v, want %+v", i, pr.Repository, w.Repository)
				}
				if len(pr.ReviewRequests.Nodes) != len(w.ReviewRequests.Nodes) {
					t.Fatalf("[%d] ReviewRequests len = %d, want %d", i, len(pr.ReviewRequests.Nodes), len(w.ReviewRequests.Nodes))
				}
				for j, rr := range pr.ReviewRequests.Nodes {
					wr := w.ReviewRequests.Nodes[j]
					if rr.AsCodeOwner != wr.AsCodeOwner {
						t.Errorf("[%d][%d] AsCodeOwner = %v, want %v", i, j, rr.AsCodeOwner, wr.AsCodeOwner)
					}
					if rr.RequestedReviewer != wr.RequestedReviewer {
						t.Errorf("[%d][%d] RequestedReviewer = %+v, want %+v", i, j, rr.RequestedReviewer, wr.RequestedReviewer)
					}
				}
			}
		})
	}
}
