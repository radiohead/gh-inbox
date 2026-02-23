package github

import (
	"github.com/cli/go-gh/v2/pkg/api"
)

// graphQLDoer is the interface used to execute GraphQL queries.
// It matches the shurcooL-graphql calling convention used by go-gh.
type graphQLDoer interface {
	Query(name string, q interface{}, variables map[string]interface{}) error
}

// Client wraps a GraphQL client for GitHub API access.
type Client struct {
	gql graphQLDoer
}

// NewClient creates a Client using the default gh CLI authentication.
func NewClient() (*Client, error) {
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}
	return &Client{gql: gql}, nil
}

// NewClientWithDoer creates a Client with the provided graphQLDoer.
// Intended for testing.
func NewClientWithDoer(doer graphQLDoer) *Client {
	return &Client{gql: doer}
}
