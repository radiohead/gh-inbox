package service

import (
	"strings"
	"testing"

	"github.com/radiohead/gh-inbox/internal/github"
)

// buildPR constructs a PullRequest for testing with the given review requests.
// nameWithOwner should be in "owner/repo" format.
func buildPR(nameWithOwner string, requests []github.ReviewRequest) github.PullRequest {
	parts := strings.SplitN(nameWithOwner, "/", 2)
	owner, name := parts[0], parts[1]
	return github.PullRequest{
		Number: 1,
		Title:  "Test PR",
		URL:    "https://github.com/" + nameWithOwner + "/pull/1",
		Repository: github.Repository{
			Owner: owner,
			Name:  name,
		},
		ReviewRequests: github.ReviewRequestConnection{
			Nodes: requests,
		},
	}
}

// userReq builds a User ReviewRequest.
func userReq(login string) github.ReviewRequest {
	return github.ReviewRequest{
		RequestedReviewer: github.RequestedReviewer{
			Type:  "User",
			Login: login,
		},
	}
}

// teamReq builds a Team ReviewRequest.
func teamReq(slug string) github.ReviewRequest {
	return github.ReviewRequest{
		RequestedReviewer: github.RequestedReviewer{
			Type:  "Team",
			Login: slug,
		},
	}
}

// newMockTeamService builds a TeamService backed by a mockFetcher with
// configurable team membership and user teams.
func newMockTeamService(
	members map[string][]github.TeamMember,
	myTeams []github.UserTeam,
) *TeamService {
	fetcher := &mockFetcher{
		fetchFunc: func(org, slug string) ([]github.TeamMember, error) {
			key := org + "/" + slug
			if m, ok := members[key]; ok {
				return m, nil
			}
			return nil, nil
		},
		myTeamsFunc: func() ([]github.UserTeam, error) {
			return myTeams, nil
		},
	}
	return NewTeamService(fetcher)
}

func TestFilterDirect(t *testing.T) {
	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		members   map[string][]github.TeamMember
		myTeams   []github.UserTeam
		wantCount int
	}{
		{
			name: "sole reviewer — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "me + non-teammate — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice"), userReq("carol")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 1,
		},
		{
			name: "me + teammate — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice"), userReq("bob")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 0,
		},
		{
			name: "me + teammate + non-teammate — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice"), userReq("bob"), userReq("carol")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 0,
		},
		{
			name: "only Team request (not me as User) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("backend")}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "me + Team (no User teammate) — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice"), teamReq("backend")}),
			},
			myLogin:   "alice",
			wantCount: 1,
		},
		{
			name: "empty requests — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, tt.myTeams)
			got := Filter(tt.prs, tt.myLogin, svc, ModeDirect)
			if len(got) != tt.wantCount {
				t.Errorf("filterDirect returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterTeam(t *testing.T) {
	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		members   map[string][]github.TeamMember
		myTeams   []github.UserTeam
		wantCount int
	}{
		{
			name: "my team, no individuals — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("backend")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			wantCount: 1,
		},
		{
			name: "my team + individual teammate — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("backend"), userReq("bob")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
				"org/myteam":  {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 0,
		},
		{
			name: "my team + individual non-teammate — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("backend"), userReq("carol")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
				"org/myteam":  {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 1,
		},
		{
			name: "other team (not mine) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("frontend")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/frontend": {{Login: "bob"}},
			},
			wantCount: 0,
		},
		{
			name: "only User reviewers — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
			},
			myLogin:   "alice",
			wantCount: 0,
		},
		{
			name: "multiple teams, one mine, clean — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("frontend"), teamReq("backend")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/frontend": {{Login: "bob"}},
				"org/backend":  {{Login: "alice"}, {Login: "john"}},
			},
			wantCount: 1,
		},
		{
			name: "my team + teammate from different team — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("backend"), userReq("bob")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
				"org/myteam":  {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, tt.myTeams)
			got := Filter(tt.prs, tt.myLogin, svc, ModeTeam)
			if len(got) != tt.wantCount {
				t.Errorf("filterTeam returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterCodeowner(t *testing.T) {
	tests := []struct {
		name      string
		prs       []github.PullRequest
		myLogin   string
		members   map[string][]github.TeamMember
		myTeams   []github.UserTeam
		wantCount int
	}{
		{
			// alice + bob are teammates, no team request → not direct (bob is teammate),
			// not team (no team request) → codeowner
			name: "me + teammate, no team req — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice"), userReq("bob")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 1,
		},
		{
			// alice + bob are teammates, backend team also requested →
			// not direct (bob is teammate), not team (bob is teammate) → codeowner
			name: "me + teammate + team req — include",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice"), userReq("bob"), teamReq("backend")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}, {Login: "bob"}},
				"org/myteam":  {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 1,
		},
		{
			// sole reviewer → matches direct → excluded from codeowner
			name: "sole reviewer (matches direct) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 0,
		},
		{
			// team only, no individuals → matches team → excluded from codeowner
			name: "team only, no individuals (matches team) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{teamReq("backend")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			wantCount: 0,
		},
		{
			// alice + carol (not teammate) → alice is direct → excluded from codeowner
			name: "me + non-teammate (matches direct) — exclude",
			prs: []github.PullRequest{
				buildPR("org/repo", []github.ReviewRequest{userReq("alice"), userReq("carol")}),
			},
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, tt.myTeams)
			got := Filter(tt.prs, tt.myLogin, svc, ModeCodeowner)
			if len(got) != tt.wantCount {
				t.Errorf("filterCodeowner returned %d PRs, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterAll(t *testing.T) {
	prs := []github.PullRequest{
		buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
		buildPR("org/repo2", []github.ReviewRequest{userReq("alice")}),
	}
	svc := newMockTeamService(nil, nil)
	got := Filter(prs, "alice", svc, ModeAll)
	if len(got) != 2 {
		t.Errorf("ModeAll returned %d PRs, want 2", len(got))
	}
}
