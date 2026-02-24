package github

import (
	"encoding/json"
	"testing"
)

func TestPullRequestUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  PullRequest
	}{
		{
			name: "basic PR",
			input: `{
				"number": 42,
				"title": "Fix bug",
				"url": "https://github.com/org/repo/pull/42",
				"repository": {"owner": "org", "name": "repo"},
				"reviewRequests": {"nodes": []}
			}`,
			want: PullRequest{
				Number:         42,
				Title:          "Fix bug",
				URL:            "https://github.com/org/repo/pull/42",
				Repository:     Repository{Owner: "org", Name: "repo"},
				ReviewRequests: ReviewRequestConnection{Nodes: []ReviewRequest{}},
			},
		},
		{
			name: "PR with user review request",
			input: `{
				"number": 1,
				"title": "Add feature",
				"url": "https://github.com/org/repo/pull/1",
				"repository": {"owner": "org", "name": "repo"},
				"reviewRequests": {
					"nodes": [
						{
							"asCodeOwner": false,
							"requestedReviewer": {"type": "User", "login": "alice"}
						}
					]
				}
			}`,
			want: PullRequest{
				Number:     1,
				Title:      "Add feature",
				URL:        "https://github.com/org/repo/pull/1",
				Repository: Repository{Owner: "org", Name: "repo"},
				ReviewRequests: ReviewRequestConnection{
					Nodes: []ReviewRequest{
						{
							AsCodeOwner:       false,
							RequestedReviewer: RequestedReviewer{Type: "User", Login: "alice"},
						},
					},
				},
			},
		},
		{
			name: "PR with CODEOWNERS review request",
			input: `{
				"number": 2,
				"title": "Update docs",
				"url": "https://github.com/org/repo/pull/2",
				"repository": {"owner": "org", "name": "repo"},
				"reviewRequests": {
					"nodes": [
						{
							"asCodeOwner": true,
							"requestedReviewer": {"type": "User", "login": "bob"}
						}
					]
				}
			}`,
			want: PullRequest{
				Number:     2,
				Title:      "Update docs",
				URL:        "https://github.com/org/repo/pull/2",
				Repository: Repository{Owner: "org", Name: "repo"},
				ReviewRequests: ReviewRequestConnection{
					Nodes: []ReviewRequest{
						{
							AsCodeOwner:       true,
							RequestedReviewer: RequestedReviewer{Type: "User", Login: "bob"},
						},
					},
				},
			},
		},
		{
			name: "PR with team review request",
			input: `{
				"number": 3,
				"title": "Refactor",
				"url": "https://github.com/org/repo/pull/3",
				"repository": {"owner": "org", "name": "repo"},
				"reviewRequests": {
					"nodes": [
						{
							"asCodeOwner": true,
							"requestedReviewer": {"type": "Team", "login": "backend-team"}
						}
					]
				}
			}`,
			want: PullRequest{
				Number:     3,
				Title:      "Refactor",
				URL:        "https://github.com/org/repo/pull/3",
				Repository: Repository{Owner: "org", Name: "repo"},
				ReviewRequests: ReviewRequestConnection{
					Nodes: []ReviewRequest{
						{
							AsCodeOwner:       true,
							RequestedReviewer: RequestedReviewer{Type: "Team", Login: "backend-team"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got PullRequest
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got.Number != tt.want.Number {
				t.Errorf("Number = %d, want %d", got.Number, tt.want.Number)
			}
			if got.Title != tt.want.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.want.Title)
			}
			if got.URL != tt.want.URL {
				t.Errorf("URL = %q, want %q", got.URL, tt.want.URL)
			}
			if got.Repository != tt.want.Repository {
				t.Errorf("Repository = %+v, want %+v", got.Repository, tt.want.Repository)
			}
			if len(got.ReviewRequests.Nodes) != len(tt.want.ReviewRequests.Nodes) {
				t.Fatalf("ReviewRequests len = %d, want %d", len(got.ReviewRequests.Nodes), len(tt.want.ReviewRequests.Nodes))
			}
			for i, node := range got.ReviewRequests.Nodes {
				wantNode := tt.want.ReviewRequests.Nodes[i]
				if node.AsCodeOwner != wantNode.AsCodeOwner {
					t.Errorf("node[%d].AsCodeOwner = %v, want %v", i, node.AsCodeOwner, wantNode.AsCodeOwner)
				}
				if node.RequestedReviewer != wantNode.RequestedReviewer {
					t.Errorf("node[%d].RequestedReviewer = %+v, want %+v", i, node.RequestedReviewer, wantNode.RequestedReviewer)
				}
			}
		})
	}
}
