package service

import (
	"errors"
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

// mockFetchFunc implements Fetcher for tests.
type mockFetchFunc struct {
	fn func(org string) ([]github.PullRequest, error)
}

func (m *mockFetchFunc) Fetch(org string) ([]github.PullRequest, error) {
	return m.fn(org)
}

func TestSourceClassifier(t *testing.T) {
	// alice sole reviewer → direct
	// alice + bob (teammate) → codeowner
	// backend team only → team
	members := map[string][]github.TeamMember{
		"org/myteam":  {{Login: "alice"}, {Login: "bob"}},
		"org/backend": {{Login: "alice"}},
	}
	myTeams := []github.UserTeam{
		{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
		{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
	}
	svc := newMockTeamService(members, myTeams)
	c := &SourceClassifier{Login: "alice", Teams: svc}

	prs := []github.PullRequest{
		buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
		buildPR("org/repo", []github.ReviewRequest{userReq("alice"), userReq("bob")}),
		buildPR("org/repo", []github.ReviewRequest{teamReq("backend")}),
	}
	got := c.ClassifyAll(prs)
	if len(got) != 3 {
		t.Fatalf("ClassifyAll returned %d results, want 3", len(got))
	}

	cases := []ReviewType{ReviewTypeDirect, ReviewTypeCodeowner, ReviewTypeTeam}
	for i, want := range cases {
		if got[i].ReviewType != want {
			t.Errorf("[%d] ReviewType = %q, want %q", i, got[i].ReviewType, want)
		}
		if got[i].PR.Number != prs[i].Number {
			t.Errorf("[%d] PR.Number = %d, want %d", i, got[i].PR.Number, prs[i].Number)
		}
	}
}

func TestPassthroughClassifier(t *testing.T) {
	prs := []github.PullRequest{
		buildPR("org/a", []github.ReviewRequest{userReq("alice")}),
		buildPR("org/b", []github.ReviewRequest{teamReq("backend")}),
	}
	got := PassthroughClassifier{}.ClassifyAll(prs)
	if len(got) != 2 {
		t.Fatalf("ClassifyAll returned %d results, want 2", len(got))
	}
	for i, cp := range got {
		if cp.ReviewType != "" {
			t.Errorf("[%d] ReviewType = %q, want empty", i, cp.ReviewType)
		}
	}
}

func TestPipelineRun(t *testing.T) {
	prs := []github.PullRequest{
		buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
		buildPR("org/repo", []github.ReviewRequest{teamReq("backend")}),
	}
	members := map[string][]github.TeamMember{
		"org/backend": {{Login: "alice"}},
	}
	myTeams := []github.UserTeam{
		{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
	}
	svc := newMockTeamService(members, myTeams)

	fetcher := &mockFetchFunc{fn: func(_ string) ([]github.PullRequest, error) {
		return prs, nil
	}}
	classifier := &SourceClassifier{Login: "alice", Teams: svc}
	filter := &CriteriaFilter{Criteria: FilterCriteria{ReviewTypes: ReviewTypeSet{ReviewTypeDirect: true}}}

	p := NewPipeline(fetcher, classifier, filter)
	got, err := p.Run("org")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Run() returned %d PRs, want 1", len(got))
	}
	if got[0].ReviewType != ReviewTypeDirect {
		t.Errorf("Run() got[0].ReviewType = %q, want %q", got[0].ReviewType, ReviewTypeDirect)
	}
}

func TestPipelineRunFetchError(t *testing.T) {
	fetchErr := errors.New("network error")
	fetcher := &mockFetchFunc{fn: func(_ string) ([]github.PullRequest, error) {
		return nil, fetchErr
	}}
	p := NewPipeline(fetcher, PassthroughClassifier{}, &CriteriaFilter{})
	_, err := p.Run("org")
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
	if !errors.Is(err, fetchErr) {
		t.Errorf("Run() error = %v, want wrapping %v", err, fetchErr)
	}
}
