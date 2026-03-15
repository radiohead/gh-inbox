package service

import (
	"strings"
	"testing"
	"time"

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

func TestClassify(t *testing.T) {
	tests := []struct {
		name    string
		pr      github.PullRequest
		myLogin string
		members map[string][]github.TeamMember
		myTeams []github.UserTeam
		want    ReviewType
	}{
		{
			name:    "sole User reviewer — ReviewTypeDirect",
			pr:      buildPR("org/repo", []github.ReviewRequest{userReq("alice")}),
			myLogin: "alice",
			want:    ReviewTypeDirect,
		},
		{
			// KEY CASE: direct wins over team — me(User) + my team both requested
			name: "User(me) + Team(my-team) — ReviewTypeDirect (direct wins)",
			pr: buildPR("org/repo", []github.ReviewRequest{
				userReq("alice"), teamReq("backend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: ReviewTypeDirect,
		},
		{
			// only team requested, no individual user me → ReviewTypeTeam
			name: "only Team (mine), no User me — ReviewTypeTeam",
			pr: buildPR("org/repo", []github.ReviewRequest{
				teamReq("backend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "backend", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: ReviewTypeTeam,
		},
		{
			// me + teammate(User) → not direct (teammate present), not team (no team req) → codeowner
			name: "User(me) + teammate(User) — ReviewTypeCodeowner",
			pr: buildPR("org/repo", []github.ReviewRequest{
				userReq("alice"), userReq("bob"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/myteam": {{Login: "alice"}, {Login: "bob"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: ReviewTypeCodeowner,
		},
		{
			name:    "empty requests — ReviewTypeCodeowner",
			pr:      buildPR("org/repo", []github.ReviewRequest{}),
			myLogin: "alice",
			want:    ReviewTypeCodeowner,
		},
		{
			// me + non-teammate + my team → me(User) direct (carol not a teammate) → ReviewTypeDirect
			name: "User(me) + non-teammate + Team(mine) — ReviewTypeDirect",
			pr: buildPR("org/repo", []github.ReviewRequest{
				userReq("alice"), userReq("carol"), teamReq("backend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/backend": {{Login: "alice"}},
				"org/myteam":  {{Login: "alice"}},
			},
			myTeams: []github.UserTeam{
				{Slug: "myteam", Organization: github.TeamOrganization{Login: "org"}},
			},
			want: ReviewTypeDirect,
		},
		{
			// only a team that is NOT mine → neither direct nor team → ReviewTypeCodeowner
			name: "only Team (not mine) — ReviewTypeCodeowner",
			pr: buildPR("org/repo", []github.ReviewRequest{
				teamReq("frontend"),
			}),
			myLogin: "alice",
			members: map[string][]github.TeamMember{
				"org/frontend": {{Login: "bob"}},
			},
			want: ReviewTypeCodeowner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockTeamService(tt.members, tt.myTeams)
			got := Classify(tt.pr, tt.myLogin, svc)
			if got != tt.want {
				t.Errorf("Classify() = %q, want %q", got, tt.want)
			}
		})
	}
}

// buildClassifiedPR constructs a ClassifiedPR with the given type and source for filtering tests.
func buildClassifiedPR(rt ReviewType, as AuthorSource) ClassifiedPR {
	return buildClassifiedPRWithStatus(rt, as, ReviewStatusOpen)
}

// buildClassifiedPRWithStatus constructs a ClassifiedPR with type, source, and review status.
func buildClassifiedPRWithStatus(rt ReviewType, as AuthorSource, rs ReviewStatus) ClassifiedPR {
	return ClassifiedPR{
		PR:           github.PullRequest{Number: 1, CreatedAt: time.Now()},
		ReviewType:   rt,
		AuthorSource: as,
		ReviewStatus: rs,
	}
}

func TestCriteriaFilter(t *testing.T) {
	direct := buildClassifiedPR(ReviewTypeDirect, AuthorSourceTeam)
	codeowner := buildClassifiedPR(ReviewTypeCodeowner, AuthorSourceGroup)
	team := buildClassifiedPR(ReviewTypeTeam, AuthorSourceOrg)
	external := buildClassifiedPR(ReviewTypeCodeowner, AuthorSourceOther)
	all := []ClassifiedPR{direct, codeowner, team, external}

	// PRs with different review statuses for status-dimension tests.
	openPR := buildClassifiedPRWithStatus(ReviewTypeDirect, AuthorSourceTeam, ReviewStatusOpen)
	inReviewPR := buildClassifiedPRWithStatus(ReviewTypeDirect, AuthorSourceTeam, ReviewStatusInReview)
	approvedPR := buildClassifiedPRWithStatus(ReviewTypeDirect, AuthorSourceTeam, ReviewStatusApproved)
	statusAll := []ClassifiedPR{openPR, inReviewPR, approvedPR}

	tests := []struct {
		name     string
		input    []ClassifiedPR
		criteria FilterCriteria
		want     []ClassifiedPR
	}{
		{
			name:     "nil criteria matches all",
			input:    all,
			criteria: FilterCriteria{},
			want:     all,
		},
		{
			name:     "type-only: direct",
			input:    all,
			criteria: FilterCriteria{ReviewTypes: ReviewTypeSet{ReviewTypeDirect: true}},
			want:     []ClassifiedPR{direct},
		},
		{
			name:     "type-only: direct and codeowner",
			input:    all,
			criteria: FilterCriteria{ReviewTypes: ReviewTypeSet{ReviewTypeDirect: true, ReviewTypeCodeowner: true}},
			want:     []ClassifiedPR{direct, codeowner, external},
		},
		{
			name:     "source-only: TEAM",
			input:    all,
			criteria: FilterCriteria{AuthorSources: AuthorSourceSet{AuthorSourceTeam: true}},
			want:     []ClassifiedPR{direct},
		},
		{
			name:  "combined: direct type AND TEAM source",
			input: all,
			criteria: FilterCriteria{
				ReviewTypes:   ReviewTypeSet{ReviewTypeDirect: true},
				AuthorSources: AuthorSourceSet{AuthorSourceTeam: true},
			},
			want: []ClassifiedPR{direct},
		},
		{
			name:  "combined: direct+codeowner AND TEAM+GROUP",
			input: all,
			criteria: FilterCriteria{
				ReviewTypes:   ReviewTypeSet{ReviewTypeDirect: true, ReviewTypeCodeowner: true},
				AuthorSources: AuthorSourceSet{AuthorSourceTeam: true, AuthorSourceGroup: true},
			},
			want: []ClassifiedPR{direct, codeowner},
		},
		{
			name:     "no match",
			input:    all,
			criteria: FilterCriteria{ReviewTypes: ReviewTypeSet{ReviewTypeTeam: true}, AuthorSources: AuthorSourceSet{AuthorSourceTeam: true}},
			want:     []ClassifiedPR{},
		},
		// ReviewStatus dimension
		{
			name:     "status-only: open",
			input:    statusAll,
			criteria: FilterCriteria{ReviewStatuses: ReviewStatusSet{ReviewStatusOpen: true}},
			want:     []ClassifiedPR{openPR},
		},
		{
			name:     "status-only: in_review",
			input:    statusAll,
			criteria: FilterCriteria{ReviewStatuses: ReviewStatusSet{ReviewStatusInReview: true}},
			want:     []ClassifiedPR{inReviewPR},
		},
		{
			name:     "status-only: approved",
			input:    statusAll,
			criteria: FilterCriteria{ReviewStatuses: ReviewStatusSet{ReviewStatusApproved: true}},
			want:     []ClassifiedPR{approvedPR},
		},
		{
			name:     "status nil matches all statuses",
			input:    statusAll,
			criteria: FilterCriteria{},
			want:     statusAll,
		},
		{
			name:  "status open AND type direct",
			input: statusAll,
			criteria: FilterCriteria{
				ReviewTypes:    ReviewTypeSet{ReviewTypeDirect: true},
				ReviewStatuses: ReviewStatusSet{ReviewStatusOpen: true},
			},
			want: []ClassifiedPR{openPR},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &CriteriaFilter{Criteria: tt.criteria}
			got := f.Apply(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("Apply() returned %d PRs, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].ReviewType != tt.want[i].ReviewType || got[i].AuthorSource != tt.want[i].AuthorSource {
					t.Errorf("[%d] got {ReviewType:%q AuthorSource:%q}, want {ReviewType:%q AuthorSource:%q}",
						i, got[i].ReviewType, got[i].AuthorSource,
						tt.want[i].ReviewType, tt.want[i].AuthorSource)
				}
			}
		})
	}
}

func TestPresetCriteria(t *testing.T) {
	direct := buildClassifiedPR(ReviewTypeDirect, AuthorSourceTeam)
	codeownTeam := buildClassifiedPR(ReviewTypeCodeowner, AuthorSourceTeam)
	codeownGroup := buildClassifiedPR(ReviewTypeCodeowner, AuthorSourceGroup)
	teamReview := buildClassifiedPR(ReviewTypeTeam, AuthorSourceOrg)
	external := buildClassifiedPR(ReviewTypeDirect, AuthorSourceOther)
	all := []ClassifiedPR{direct, codeownTeam, codeownGroup, teamReview, external}

	tests := []struct {
		preset    Preset
		wantCount int
		check     func([]ClassifiedPR) bool
	}{
		{
			preset:    PresetAll,
			wantCount: 5,
		},
		{
			preset:    PresetFocus,
			wantCount: 2, // direct+TEAM, codeowner+TEAM
		},
		{
			preset:    PresetNearby,
			wantCount: 3, // direct+TEAM, codeowner+TEAM, codeowner+GROUP
		},
		{
			preset:    PresetOrg,
			wantCount: 4, // all except OTHER
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.preset), func(t *testing.T) {
			f := &CriteriaFilter{Criteria: PresetCriteria(tt.preset)}
			got := f.Apply(all)
			if len(got) != tt.wantCount {
				t.Errorf("PresetCriteria(%q).Apply() = %d PRs, want %d", tt.preset, len(got), tt.wantCount)
			}
		})
	}
}
