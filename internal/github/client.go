package github

import (
	"github.com/cli/go-gh/v2/pkg/api"
)

// graphQLDoer is the interface used to execute GraphQL queries.
// It matches the shurcooL-graphql calling convention used by go-gh.
type graphQLDoer interface {
	Query(name string, q interface{}, variables map[string]interface{}) error
}

// restDoer is the interface used to execute REST API calls.
type restDoer interface {
	Get(path string, resp interface{}) error
}

// Cacher is an optional key-value cache used by Client to avoid redundant API calls.
type Cacher interface {
	Get(key string) ([]byte, bool, error) // data, found, error
	Set(key string, data []byte) error
}

// Client wraps a GraphQL client for GitHub API access.
type Client struct {
	gql              graphQLDoer
	rest             restDoer
	cache            Cacher // optional; nil means no caching
	prCache          Cacher // optional; separate cache for PR data (different TTL)
	skipPRCacheRead  bool   // when true, FetchReviewRequestedPRs skips cache read
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithCache sets the Cacher used by the Client for persistent caching.
func WithCache(c Cacher) ClientOption {
	return func(cl *Client) {
		cl.cache = c
	}
}

// WithPRCache sets the dedicated PR Cacher used by FetchReviewRequestedPRs.
// This should be a separate DiskCacher instance with a shorter TTL than the
// main cache (e.g. 5 minutes vs 4 hours for team/user data).
func WithPRCache(c Cacher) ClientOption {
	return func(cl *Client) {
		cl.prCache = c
	}
}

// WithRefresh configures the Client to skip the PR cache read in
// FetchReviewRequestedPRs, forcing a fresh GraphQL fetch. The result is still
// written to cache after a successful fetch.
func WithRefresh() ClientOption {
	return func(cl *Client) {
		cl.skipPRCacheRead = true
	}
}

// NewClient creates a Client using the default gh CLI authentication.
func NewClient(opts ...ClientOption) (*Client, error) {
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	c := &Client{gql: gql, rest: rest}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// NewClientWithDoer creates a Client with the provided graphQLDoer.
// Intended for testing. REST client is set to nil.
func NewClientWithDoer(doer graphQLDoer) *Client {
	return &Client{gql: doer}
}

// NewClientWithDoers creates a Client with the provided graphQLDoer and restDoer.
// Intended for testing.
func NewClientWithDoers(gql graphQLDoer, rest restDoer) *Client {
	return &Client{gql: gql, rest: rest}
}
