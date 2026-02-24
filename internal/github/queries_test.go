package github

import "testing"

func TestBuildReviewRequestedSearchQuery(t *testing.T) {
	tests := []struct {
		name string
		org  string
		want string
	}{
		{
			name: "no org — all orgs",
			org:  "",
			want: "is:open is:pr review-requested:@me",
		},
		{
			name: "simple org name",
			org:  "grafana",
			want: "is:open is:pr review-requested:@me org:grafana",
		},
		{
			name: "org with hyphen",
			org:  "grafana-labs",
			want: "is:open is:pr review-requested:@me org:grafana-labs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildReviewRequestedSearchQuery(tt.org)
			if got != tt.want {
				t.Errorf("buildReviewRequestedSearchQuery(%q) = %q, want %q", tt.org, got, tt.want)
			}
		})
	}
}
