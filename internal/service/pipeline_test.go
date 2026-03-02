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

	cases := []Source{SourceDirect, SourceCodeowner, SourceTeam}
	for i, want := range cases {
		if got[i].Source != want {
			t.Errorf("[%d] Source = %q, want %q", i, got[i].Source, want)
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
		if cp.Source != "" {
			t.Errorf("[%d] Source = %q, want empty", i, cp.Source)
		}
	}
}

func TestModeFilter(t *testing.T) {
	direct := ClassifiedPR{PR: buildPR("org/a", nil), Source: SourceDirect}
	team := ClassifiedPR{PR: buildPR("org/b", nil), Source: SourceTeam}
	codeowner := ClassifiedPR{PR: buildPR("org/c", nil), Source: SourceCodeowner}
	all := []ClassifiedPR{direct, team, codeowner}

	tests := []struct {
		mode      Mode
		wantCount int
		wantSrc   Source
	}{
		{ModeAll, 3, ""},
		{ModeDirect, 1, SourceDirect},
		{ModeTeam, 1, SourceTeam},
		{ModeCodeowner, 1, SourceCodeowner},
	}

	for _, tt := range tests {
		f := &ModeFilter{Mode: tt.mode}
		got := f.Apply(all)
		if len(got) != tt.wantCount {
			t.Errorf("ModeFilter{%v}.Apply() returned %d PRs, want %d", tt.mode, len(got), tt.wantCount)
			continue
		}
		if tt.mode != ModeAll {
			for _, cp := range got {
				if cp.Source != tt.wantSrc {
					t.Errorf("ModeFilter{%v}: got Source=%q, want %q", tt.mode, cp.Source, tt.wantSrc)
				}
			}
		}
	}
}

func TestModeFilterPartition(t *testing.T) {
	// The three non-all modes partition the input exhaustively and non-overlapping.
	direct := ClassifiedPR{PR: buildPR("org/a", nil), Source: SourceDirect}
	team := ClassifiedPR{PR: buildPR("org/b", nil), Source: SourceTeam}
	codeowner := ClassifiedPR{PR: buildPR("org/c", nil), Source: SourceCodeowner}
	all := []ClassifiedPR{direct, team, codeowner}

	d := (&ModeFilter{Mode: ModeDirect}).Apply(all)
	tm := (&ModeFilter{Mode: ModeTeam}).Apply(all)
	co := (&ModeFilter{Mode: ModeCodeowner}).Apply(all)

	total := len(d) + len(tm) + len(co)
	if total != len(all) {
		t.Errorf("partition total = %d, want %d (non-overlapping, exhaustive)", total, len(all))
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
	filter := &ModeFilter{Mode: ModeDirect}

	p := NewPipeline(fetcher, classifier, filter)
	got, err := p.Run("org")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Run() returned %d PRs, want 1", len(got))
	}
	if got[0].Source != SourceDirect {
		t.Errorf("Run() got[0].Source = %q, want %q", got[0].Source, SourceDirect)
	}
}

func TestPipelineRunFetchError(t *testing.T) {
	fetchErr := errors.New("network error")
	fetcher := &mockFetchFunc{fn: func(_ string) ([]github.PullRequest, error) {
		return nil, fetchErr
	}}
	p := NewPipeline(fetcher, PassthroughClassifier{}, &ModeFilter{Mode: ModeAll})
	_, err := p.Run("org")
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
	if !errors.Is(err, fetchErr) {
		t.Errorf("Run() error = %v, want wrapping %v", err, fetchErr)
	}
}
