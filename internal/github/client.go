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

// Client wraps a GraphQL client for GitHub API access.
type Client struct {
	gql         graphQLDoer
	rest        restDoer
	teamMembers map[string]map[string]bool // cache: "org/slug" -> set of member logins
}

// NewClient creates a Client using the default gh CLI authentication.
func NewClient() (*Client, error) {
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	return &Client{gql: gql, rest: rest}, nil
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
