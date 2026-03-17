package github

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
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
											Team     struct{ Slug string } `graphql:"... on Team"`
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
											Team     struct{ Slug string } `graphql:"... on Team"`
											TypeName string                     `graphql:"__typename"`
										}{
											Team:     struct{ Slug string }{Slug: "backend-team"},
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
											Team     struct{ Slug string } `graphql:"... on Team"`
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
			name: "SAML partial data: valid nodes returned, zero-value nodes skipped, no error",
			org:  "saml-org",
			queryFunc: func(_ string, q interface{}, _ map[string]interface{}) error {
				result := q.(*searchReviewRequestedQuery)
				result.Search.Nodes = []struct {
					PullRequest searchPRNode `graphql:"... on PullRequest"`
				}{
					{
						PullRequest: searchPRNode{
							Number: 42,
							Title:  "Accessible PR",
							URL:    "https://github.com/saml-org/repo/pull/42",
							Author: struct{ Login string }{Login: "alice"},
							Repository: struct{ NameWithOwner string }{
								NameWithOwner: "saml-org/repo",
							},
						},
					},
					{
						// Zero-valued node from SAML-blocked result.
						PullRequest: searchPRNode{},
					},
				}
				return &api.GraphQLError{
					Errors: []api.GraphQLErrorItem{
						{Message: "Resource protected by organization SAML enforcement"},
					},
				}
			},
			want: []PullRequest{
				{
					Number:     42,
					Title:      "Accessible PR",
					URL:        "https://github.com/saml-org/repo/pull/42",
					Author:     "alice",
					Repository: Repository{Owner: "saml-org", Name: "repo"},
					ReviewRequests: ReviewRequestConnection{
						Nodes: []ReviewRequest{},
					},
				},
			},
		},
		{
			name: "non-SAML GraphQL error propagates",
			org:  "test-org",
			queryFunc: func(_ string, _ interface{}, _ map[string]interface{}) error {
				return &api.GraphQLError{
					Errors: []api.GraphQLErrorItem{
						{Message: "Could not resolve to a Repository with the name 'test-org/missing'."},
					},
				}
			},
			wantErr: true,
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

func TestConvertSearchPRNode_Reviews(t *testing.T) {
	tests := []struct {
		name        string
		reviewNodes []searchReviewNode
		wantReviews []Review
	}{
		{
			name: "all five review states mapped correctly",
			reviewNodes: []searchReviewNode{
				{Author: struct{ Login string }{Login: "alice"}, State: ReviewStateApproved},
				{Author: struct{ Login string }{Login: "bob"}, State: ReviewStateChangesRequested},
				{Author: struct{ Login string }{Login: "carol"}, State: ReviewStateCommented},
				{Author: struct{ Login string }{Login: "dave"}, State: ReviewStatePending},
				{Author: struct{ Login string }{Login: "eve"}, State: ReviewStateDismissed},
			},
			wantReviews: []Review{
				{Author: "alice", State: ReviewStateApproved},
				{Author: "bob", State: ReviewStateChangesRequested},
				{Author: "carol", State: ReviewStateCommented},
				{Author: "dave", State: ReviewStatePending},
				{Author: "eve", State: ReviewStateDismissed},
			},
		},
		{
			name:        "zero reviews produces empty slice not nil",
			reviewNodes: nil,
			wantReviews: []Review{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := searchPRNode{
				Number: 1,
				Title:  "Test PR",
				URL:    "https://github.com/org/repo/pull/1",
				Repository: struct{ NameWithOwner string }{
					NameWithOwner: "org/repo",
				},
				Reviews: struct {
					Nodes []searchReviewNode
				}{
					Nodes: tt.reviewNodes,
				},
			}

			pr := convertSearchPRNode(node)

			if len(pr.Reviews) != len(tt.wantReviews) {
				t.Fatalf("Reviews len = %d, want %d", len(pr.Reviews), len(tt.wantReviews))
			}
			for i, rv := range pr.Reviews {
				w := tt.wantReviews[i]
				if rv.Author != w.Author {
					t.Errorf("[%d] Author = %q, want %q", i, rv.Author, w.Author)
				}
				if rv.State != w.State {
					t.Errorf("[%d] State = %q, want %q", i, rv.State, w.State)
				}
			}
		})
	}
}

// makeSinglePRQueryFunc returns a queryFunc that populates q with one PR.
func makeSinglePRQueryFunc(number int, url, org string) func(string, interface{}, map[string]interface{}) error {
	return func(_ string, q interface{}, _ map[string]interface{}) error {
		result := q.(*searchReviewRequestedQuery)
		result.Search.Nodes = []struct {
			PullRequest searchPRNode `graphql:"... on PullRequest"`
		}{
			{
				PullRequest: searchPRNode{
					Number: number,
					Title:  "Test PR",
					URL:    url,
					Repository: struct{ NameWithOwner string }{
						NameWithOwner: org + "/repo",
					},
				},
			},
		}
		return nil
	}
}

func TestFetchReviewRequestedPRsWithPRCache(t *testing.T) {
	const org = "myorg"
	const cacheKey = "review-prs:myorg"
	const prURL = "https://github.com/myorg/repo/pull/1"

	t.Run("cache miss: calls GraphQL and writes result to cache", func(t *testing.T) {
		gqlCalled := false
		gql := &mockGraphQLDoer{queryFunc: func(name string, q interface{}, vars map[string]interface{}) error {
			gqlCalled = true
			return makeSinglePRQueryFunc(1, prURL, org)(name, q, vars)
		}}
		cacher := newMockCacher()
		client := NewClientWithDoer(gql)
		client.prCache = cacher

		prs, err := client.FetchReviewRequestedPRs(org)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(prs) != 1 {
			t.Fatalf("got %d PRs, want 1", len(prs))
		}
		if !gqlCalled {
			t.Error("expected GraphQL to be called on cache miss")
		}
		data, found := cacher.store[cacheKey]
		if !found {
			t.Fatal("expected cache entry to be written after miss")
		}
		var cached []PullRequest
		if err := json.Unmarshal(data, &cached); err != nil {
			t.Fatalf("cache value not valid JSON: %v", err)
		}
		if len(cached) != 1 || cached[0].URL != prURL {
			t.Errorf("cached PR URL = %q, want %q", cached[0].URL, prURL)
		}
	})

	t.Run("cache hit: returns cached PRs without calling GraphQL", func(t *testing.T) {
		gqlCalled := false
		gql := &mockGraphQLDoer{queryFunc: func(_ string, _ interface{}, _ map[string]interface{}) error {
			gqlCalled = true
			return nil
		}}
		cacher := newMockCacher()
		cachedPRs := []PullRequest{{Number: 99, URL: prURL, Repository: Repository{Owner: org, Name: "repo"}}}
		data, _ := json.Marshal(cachedPRs)
		cacher.store[cacheKey] = data

		client := NewClientWithDoer(gql)
		client.prCache = cacher

		prs, err := client.FetchReviewRequestedPRs(org)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gqlCalled {
			t.Error("expected GraphQL NOT to be called on cache hit")
		}
		if len(prs) != 1 || prs[0].Number != 99 {
			t.Errorf("got PR number %d, want 99", prs[0].Number)
		}
	})

	t.Run("refresh bypass: skips cache read, calls GraphQL, overwrites cache", func(t *testing.T) {
		gqlCalled := false
		gql := &mockGraphQLDoer{queryFunc: func(name string, q interface{}, vars map[string]interface{}) error {
			gqlCalled = true
			return makeSinglePRQueryFunc(2, prURL, org)(name, q, vars)
		}}
		cacher := newMockCacher()
		// Pre-populate cache with stale data.
		stalePRs := []PullRequest{{Number: 99, URL: "https://github.com/myorg/repo/pull/99"}}
		staleData, _ := json.Marshal(stalePRs)
		cacher.store[cacheKey] = staleData

		client := NewClientWithDoer(gql)
		client.prCache = cacher
		client.skipPRCacheRead = true

		prs, err := client.FetchReviewRequestedPRs(org)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gqlCalled {
			t.Error("expected GraphQL to be called when skipPRCacheRead is true")
		}
		if len(prs) != 1 || prs[0].Number != 2 {
			t.Errorf("got PR number %d, want 2", prs[0].Number)
		}
		// Cache should be overwritten with fresh data.
		newData, found := cacher.store[cacheKey]
		if !found {
			t.Fatal("expected cache entry to be written after refresh")
		}
		var newCached []PullRequest
		if err := json.Unmarshal(newData, &newCached); err != nil {
			t.Fatalf("cache value not valid JSON: %v", err)
		}
		if len(newCached) != 1 || newCached[0].Number != 2 {
			t.Errorf("cache entry PR number = %d, want 2", newCached[0].Number)
		}
	})

	t.Run("cache Get error: proceeds to GraphQL", func(t *testing.T) {
		gqlCalled := false
		gql := &mockGraphQLDoer{queryFunc: func(name string, q interface{}, vars map[string]interface{}) error {
			gqlCalled = true
			return makeSinglePRQueryFunc(1, prURL, org)(name, q, vars)
		}}
		cacher := newMockCacher()
		cacher.getErr = errors.New("cache read failure")

		client := NewClientWithDoer(gql)
		client.prCache = cacher

		prs, err := client.FetchReviewRequestedPRs(org)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gqlCalled {
			t.Error("expected GraphQL to be called when cache.Get returns error")
		}
		if len(prs) != 1 {
			t.Fatalf("got %d PRs, want 1", len(prs))
		}
	})

	t.Run("SAML partial data: partial results returned and cached", func(t *testing.T) {
		gql := &mockGraphQLDoer{queryFunc: func(_ string, q interface{}, _ map[string]interface{}) error {
			result := q.(*searchReviewRequestedQuery)
			result.Search.Nodes = []struct {
				PullRequest searchPRNode `graphql:"... on PullRequest"`
			}{
				{
					PullRequest: searchPRNode{
						Number: 42,
						Title:  "Accessible PR",
						URL:    prURL,
						Author: struct{ Login string }{Login: "alice"},
						Repository: struct{ NameWithOwner string }{
							NameWithOwner: "myorg/repo",
						},
					},
				},
				{
					// Zero-valued node from SAML-blocked result.
					PullRequest: searchPRNode{},
				},
			}
			return &api.GraphQLError{
				Errors: []api.GraphQLErrorItem{
					{Message: "Resource protected by organization SAML enforcement"},
				},
			}
		}}
		cacher := newMockCacher()
		client := NewClientWithDoer(gql)
		client.prCache = cacher

		prs, err := client.FetchReviewRequestedPRs(org)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(prs) != 1 || prs[0].Number != 42 {
			t.Fatalf("got %d PRs, want 1 with number 42", len(prs))
		}
		// Partial data should be cached.
		data, found := cacher.store[cacheKey]
		if !found {
			t.Fatal("expected cache entry to be written with partial SAML data")
		}
		var cached []PullRequest
		if err := json.Unmarshal(data, &cached); err != nil {
			t.Fatalf("cache value not valid JSON: %v", err)
		}
		if len(cached) != 1 || cached[0].Number != 42 {
			t.Errorf("cached PR number = %d, want 42", cached[0].Number)
		}
	})
}
