package prs

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/radiohead/gh-inbox/internal/github"
	"github.com/radiohead/gh-inbox/internal/output"
	"github.com/radiohead/gh-inbox/internal/service"
)

// Integration tests exercise the full pipeline wiring:
// mock data → TeamService → SourceClassifier → CriteriaFilter → output
//
// These mirror the wiring in reviewCmd.RunE but inject mocks at the
// github.Client level rather than calling the cobra command directly
// (which requires real gh auth).

// testTeamService builds a TeamService with canned team data.
func testTeamService(
	members map[string][]github.TeamMember,
	myTeams []github.UserTeam,
) *service.TeamService {
	fetcher := &stubFetcher{
		fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
			return members[org+"/"+slug], nil
		},
		myTeamsFunc: func() ([]github.UserTeam, error) {
			return myTeams, nil
		},
	}
	svc := service.NewTeamService(fetcher)
	_ = svc.PreloadTeams()
	return svc
}

// stubFetcher satisfies the service.TeamMemberFetcher interface for tests.
type stubFetcher struct {
	fetchFunc       func(org, slug string) ([]github.TeamMember, error)
	myTeamsFunc     func() ([]github.UserTeam, error)
	childTeamsFunc  func(org, parentSlug string) ([]github.ChildTeam, error)
	isOrgMemberFunc func(org, login string) (bool, error)
}

func (s *stubFetcher) FetchTeamMembers(org, slug string) ([]github.TeamMember, error) {
	return s.fetchFunc(org, slug)
}

func (s *stubFetcher) FetchMyTeams() ([]github.UserTeam, error) {
	if s.myTeamsFunc != nil {
		return s.myTeamsFunc()
	}
	return nil, nil
}

func (s *stubFetcher) FetchChildTeams(org, parentSlug string) ([]github.ChildTeam, error) {
	if s.childTeamsFunc != nil {
		return s.childTeamsFunc(org, parentSlug)
	}
	return nil, nil
}

func (s *stubFetcher) FetchIsOrgMember(org, login string) (bool, error) {
	if s.isOrgMemberFunc != nil {
		return s.isOrgMemberFunc(org, login)
	}
	return false, nil
}

// testPR builds a PullRequest with the given reviewers and reviews.
func testPR(num int, repo, author string, reviewers []github.ReviewRequest, reviews []github.Review) github.PullRequest {
	parts := strings.SplitN(repo, "/", 2)
	return github.PullRequest{
		Number:     num,
		Title:      "PR " + repo + " #" + strings.Repeat("0", 0),
		URL:        "https://github.com/" + repo + "/pull/" + string(rune('0'+num)),
		Author:     author,
		CreatedAt:  time.Now().Add(-time.Duration(num) * 24 * time.Hour),
		Repository: github.Repository{Owner: parts[0], Name: parts[1]},
		ReviewRequests: github.ReviewRequestConnection{
			Nodes: reviewers,
		},
		Reviews: reviews,
	}
}

func userReq(login string) github.ReviewRequest {
	return github.ReviewRequest{
		RequestedReviewer: github.RequestedReviewer{Type: "User", Login: login},
	}
}

func teamReq(slug string) github.ReviewRequest {
	return github.ReviewRequest{
		RequestedReviewer: github.RequestedReviewer{Type: "Team", Login: slug},
	}
}

func TestIntegration_PipelineTableOutput(t *testing.T) {
	// Setup: alice is the authenticated user, member of "backend" team with bob.
	// PR1: alice sole reviewer → direct, author=carol (OTHER)
	// PR2: alice + bob (teammates) → codeowner, author=bob (TEAM)
	// PR3: team "backend" requested → team, author=dave (ORG)
	members := map[string][]github.TeamMember{
		"org/backend": {{Login: "alice"}, {Login: "bob"}},
	}
	myTeams := []github.UserTeam{
		{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
	}
	svc := testTeamService(members, myTeams)
	classifier := &service.SourceClassifier{Login: "alice", Teams: svc}

	prs := []github.PullRequest{
		testPR(1, "org/api", "carol", []github.ReviewRequest{userReq("alice")}, nil),
		testPR(2, "org/api", "bob", []github.ReviewRequest{userReq("alice"), userReq("bob")}, nil),
		testPR(3, "org/api", "dave", []github.ReviewRequest{teamReq("backend")}, nil),
	}

	classified := classifier.ClassifyAll(prs)

	// No filter (all) — should return all 3.
	filter := &service.CriteriaFilter{Criteria: service.FilterCriteria{}}
	results := filter.Apply(classified)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify classification.
	wantTypes := []service.ReviewType{service.ReviewTypeDirect, service.ReviewTypeCodeowner, service.ReviewTypeTeam}
	for i, want := range wantTypes {
		if results[i].ReviewType != want {
			t.Errorf("[%d] ReviewType = %q, want %q", i, results[i].ReviewType, want)
		}
	}

	// Render table and verify output contains expected data.
	var buf bytes.Buffer
	if err := output.WriteTable(&buf, results); err != nil {
		t.Fatalf("WriteTable error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"org/api", "#1", "#2", "#3", "direct", "codeowner", "team"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q\noutput:\n%s", want, out)
		}
	}
}

func TestIntegration_PipelineJSONOutput(t *testing.T) {
	members := map[string][]github.TeamMember{
		"org/backend": {{Login: "alice"}, {Login: "bob"}},
	}
	myTeams := []github.UserTeam{
		{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
	}
	svc := testTeamService(members, myTeams)
	classifier := &service.SourceClassifier{Login: "alice", Teams: svc}

	prs := []github.PullRequest{
		testPR(1, "org/api", "carol", []github.ReviewRequest{userReq("alice")}, nil),
	}

	classified := classifier.ClassifyAll(prs)
	filter := &service.CriteriaFilter{Criteria: service.FilterCriteria{}}
	results := filter.Apply(classified)

	// Mirror the JSON output path from reviewCmd.RunE.
	type jsonPR struct {
		github.PullRequest
		ReviewType   string `json:"reviewType,omitempty"`
		Source       string `json:"source,omitempty"`
		ReviewStatus string `json:"reviewStatus,omitempty"`
	}
	jsonOut := make([]jsonPR, len(results))
	for i, cp := range results {
		jsonOut[i] = jsonPR{
			PullRequest:  cp.PR,
			ReviewType:   string(cp.ReviewType),
			Source:       string(cp.AuthorSource),
			ReviewStatus: string(cp.ReviewStatus),
		}
	}

	var buf bytes.Buffer
	if err := output.WriteJSON(&buf, jsonOut); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var decoded []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON decode error: %v\nraw: %s", err, buf.String())
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 JSON object, got %d", len(decoded))
	}
	if got := decoded[0]["reviewType"]; got != "direct" {
		t.Errorf("reviewType = %v, want %q", got, "direct")
	}
	if got := decoded[0]["source"]; got != "OTHER" {
		t.Errorf("source = %v, want %q", got, "OTHER")
	}
	if got := decoded[0]["reviewStatus"]; got != "open" {
		t.Errorf("reviewStatus = %v, want %q", got, "open")
	}
}

func TestIntegration_FilterReducesResults(t *testing.T) {
	members := map[string][]github.TeamMember{
		"org/backend": {{Login: "alice"}, {Login: "bob"}},
	}
	myTeams := []github.UserTeam{
		{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
	}
	svc := testTeamService(members, myTeams)
	classifier := &service.SourceClassifier{Login: "alice", Teams: svc}

	prs := []github.PullRequest{
		testPR(1, "org/api", "carol", []github.ReviewRequest{userReq("alice")}, nil),                   // direct
		testPR(2, "org/api", "bob", []github.ReviewRequest{userReq("alice"), userReq("bob")}, nil),     // codeowner
		testPR(3, "org/api", "dave", []github.ReviewRequest{teamReq("backend")}, nil),                  // team
	}

	classified := classifier.ClassifyAll(prs)

	tests := []struct {
		name     string
		criteria service.FilterCriteria
		wantLen  int
	}{
		{
			name:     "filter=direct only",
			criteria: service.FilterCriteria{ReviewTypes: service.ReviewTypeSet{service.ReviewTypeDirect: true}},
			wantLen:  1,
		},
		{
			name:     "filter=team only",
			criteria: service.FilterCriteria{ReviewTypes: service.ReviewTypeSet{service.ReviewTypeTeam: true}},
			wantLen:  1,
		},
		{
			name:     "focus preset",
			criteria: service.PresetCriteria(service.PresetFocus),
			wantLen:  1, // focus requires ReviewType=direct|codeowner + AuthorSource=TEAM + open; PR2 (bob=TEAM, codeowner) matches
		},
		{
			name:     "all preset",
			criteria: service.PresetCriteria(service.PresetAll),
			wantLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &service.CriteriaFilter{Criteria: tt.criteria}
			got := filter.Apply(classified)
			if len(got) != tt.wantLen {
				t.Errorf("Apply() returned %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestIntegration_EmptyResults(t *testing.T) {
	svc := testTeamService(nil, nil)
	classifier := &service.SourceClassifier{Login: "alice", Teams: svc}

	classified := classifier.ClassifyAll(nil)
	filter := &service.CriteriaFilter{Criteria: service.FilterCriteria{}}
	results := filter.Apply(classified)

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}

	// Table output for empty results.
	var buf bytes.Buffer
	if err := output.WriteTable(&buf, results); err != nil {
		t.Fatalf("WriteTable error: %v", err)
	}
	if !strings.Contains(buf.String(), "No pull requests found") {
		t.Errorf("expected empty message, got: %q", buf.String())
	}

	// JSON output for empty results.
	buf.Reset()
	if err := output.WriteJSON(&buf, results); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Errorf("expected empty JSON array, got: %q", buf.String())
	}
}

func TestIntegration_ReviewStatusClassification(t *testing.T) {
	members := map[string][]github.TeamMember{
		"org/backend": {{Login: "alice"}, {Login: "bob"}},
	}
	myTeams := []github.UserTeam{
		{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
	}
	svc := testTeamService(members, myTeams)
	classifier := &service.SourceClassifier{Login: "alice", Teams: svc}

	prs := []github.PullRequest{
		// No reviews → open
		testPR(1, "org/api", "carol", []github.ReviewRequest{userReq("alice")}, nil),
		// Team member approved → approved
		testPR(2, "org/api", "carol", []github.ReviewRequest{userReq("alice")}, []github.Review{
			{Author: "bob", State: github.ReviewStateApproved},
		}),
		// Teammate (bob) commented → in_review
		testPR(3, "org/api", "carol", []github.ReviewRequest{userReq("alice")}, []github.Review{
			{Author: "bob", State: github.ReviewStateCommented},
		}),
	}

	classified := classifier.ClassifyAll(prs)

	wantStatuses := []service.ReviewStatus{
		service.ReviewStatusOpen,
		service.ReviewStatusApproved,
		service.ReviewStatusInReview,
	}
	for i, want := range wantStatuses {
		if classified[i].ReviewStatus != want {
			t.Errorf("[%d] ReviewStatus = %q, want %q", i, classified[i].ReviewStatus, want)
		}
	}

	// Filter by status=open should return only PR1.
	filter := &service.CriteriaFilter{Criteria: service.FilterCriteria{
		ReviewStatuses: service.ReviewStatusSet{service.ReviewStatusOpen: true},
	}}
	results := filter.Apply(classified)
	if len(results) != 1 {
		t.Fatalf("filter(open) returned %d results, want 1", len(results))
	}
	if results[0].PR.Number != 1 {
		t.Errorf("filter(open) returned PR #%d, want #1", results[0].PR.Number)
	}
}
