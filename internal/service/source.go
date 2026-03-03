package service

import "github.com/radiohead/gh-inbox/internal/github"

// AuthorSource classifies PR authors by their organizational relationship
// to the authenticated user.
type AuthorSource string

const (
	AuthorSourceTeam  AuthorSource = "TEAM"
	AuthorSourceGroup AuthorSource = "GROUP"
	AuthorSourceOrg   AuthorSource = "ORG"
	AuthorSourceOther AuthorSource = "OTHER"
)

// ClassifyAuthorSource returns the author source classification for a PR.
//
// Precedence:
//  1. Empty author or self -> TEAM (own PRs are treated as team-level).
//  2. Author shares a direct team -> TEAM.
//  3. Author is in a sibling team (child of the same parent) -> GROUP.
//  4. Author is in the same org -> ORG.
//  5. Otherwise -> OTHER.
func ClassifyAuthorSource(pr github.PullRequest, myLogin string, teams *TeamService) AuthorSource {
	author := pr.Author
	org := pr.Repository.Owner

	if author == "" || author == myLogin {
		return AuthorSourceTeam
	}
	if teams.SharesTeamWith(org, author) {
		return AuthorSourceTeam
	}
	if teams.IsSiblingTeamMember(org, author) {
		return AuthorSourceGroup
	}
	if teams.IsOrgMember(org, author) {
		return AuthorSourceOrg
	}
	return AuthorSourceOther
}
