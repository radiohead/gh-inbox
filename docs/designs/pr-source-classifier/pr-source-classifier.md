# PR Source Classifier — Implementation Plan

## Context

The current output column `SOURCE` describes the **review type** (direct/team/codeowner) — how the user ended up as a reviewer. Users also want to know the **organizational distance** between themselves and the PR author. This feature:

1. Renames the existing `SOURCE` column to `TYPE` (review type)
2. Adds a new `SOURCE` column that classifies where the PR author comes from organizationally

The 4-level source hierarchy:
- **TEAM** — Author is a direct teammate (member of a team I'm directly in)
- **GROUP** — Author belongs to a sibling team under my team's parent group
- **ORG** — Author belongs to any team in an org I belong to
- **OTHER** — Everything else

## Requirements

**In scope:**
- Rename `SOURCE` → `TYPE` in table and `source` → `reviewType` in JSON
- Add `SOURCE` column (table) / `source` field (JSON) for author origin
- GitHub REST API calls: child teams listing, org membership check
- Tests for the new classifier and updated output

**Out of scope:**
- Filtering by author source
- Sorting by source
- New caching layer (reuse existing TeamService cache)

**Acceptance criteria:**
1. Table shows `TYPE` column with values: direct, team, codeowner
2. Table shows `SOURCE` column with values: TEAM, GROUP, ORG, OTHER
3. JSON includes `reviewType` and `source` fields
4. TEAM = author is member of a team where user is a direct member
5. GROUP = author is member of a sibling team under the same parent
6. ORG = author belongs to an org the user belongs to
7. OTHER = none of the above
8. `make test` passes

## Architecture

### Pipeline Extension (Fetch → Classify → Filter)

The existing 3-stage pipeline is extended — not replaced. Each stage gains
new responsibilities while preserving the existing contract.

```
CURRENT PIPELINE:

  FETCH                         CLASSIFY                        FILTER
  ─────                         ────────                        ──────
  FetchReviewRequestedPRs       SourceClassifier.ClassifyAll    ModeFilter.Apply
  (GQL: PRs + review reqs)     (→ Source: direct|team|co)      (keep matching Source)
       │                              │                              │
       v                              v                              v
  []PullRequest{                []ClassifiedPR{                 []ClassifiedPR
    Number, Title, URL,           PR: ...,                        (subset)
    ReviewRequests,               Source: "direct",
  }                             }


PROPOSED PIPELINE:

  FETCH (enriched)              CLASSIFY (multi-dimension)      FILTER
  ────────────────              ──────────────────────────      ──────
  FetchReviewRequestedPRs       SourceClassifier.ClassifyAll    ModeFilter.Apply
  + author { login }            ├→ ReviewType: direct|team|co   (keep matching
  (GQL enrichment)              └→ AuthorSource: TEAM|GROUP|     ReviewType)
                                     ORG|OTHER
       │                              │                              │
       v                              v                              v
  []PullRequest{                []ClassifiedPR{                 []ClassifiedPR
    ...,                          PR: ...,                        (subset)
    Author: "alice",              ReviewType:   "direct",
  }                               AuthorSource: "TEAM",
                                }
```

### User Context in the Classify Stage

The classifier needs enriched context about the current user — not just their
login, but their team memberships and org relationships.

```
User Context (built during pipeline setup, used by classify stage):

  SourceClassifier {
    Login: "igor"                          ← who am I?
    Teams: *TeamService {
      myTeams: []UserTeam [                ← my direct teams (with parent)
        {Slug: "app-platform-squad",
         Org: "grafana",
         Parent: {Slug: "backend-group"}}, ← enables GROUP resolution
        {Slug: "some-other-team",
         Org: "grafana",
         Parent: nil},                     ← top-level, no GROUP check
      ]
      cache: {                             ← lazy-loaded team member sets
        "grafana/app-platform-squad": {"igor", "alice", "bob"},
        "grafana/frontend-squad": {"carol", "dave"},
      }
      fetcher: TeamMemberFetcher {         ← API access for hierarchy
        FetchTeamMembers(org, slug)        ← existing
        FetchMyTeams()                     ← existing (now with Parent)
        FetchChildTeams(org, parentSlug)   ← NEW: sibling discovery
        FetchIsOrgMember(org, login)       ← NEW: org membership check
      }
    }
  }
```

The classifier uses this context to run both classifiers:

```
SourceClassifier.ClassifyAll(prs):
  For each PR:
    ├→ Classify(pr, login, teams)             → ReviewType
    │   Uses: pr.ReviewRequests, login
    │   Calls: teams.SharesTeamWith, teams.IsTeamMember
    │
    └→ ClassifyAuthorSource(pr, login, teams) → AuthorSource
        Uses: pr.Author, login
        Resolution chain (short-circuit on first match):
          1. TEAM:  teams.SharesTeamWith(org, pr.Author)
          2. GROUP: teams.IsSiblingTeamMember(org, pr.Author)  ← NEW
          3. ORG:   teams.IsOrgMember(org, pr.Author)          ← NEW
          4. OTHER: fallback
```

### AuthorSource Resolution Detail

```
IsSiblingTeamMember(org, authorLogin):
  myTeams = loadMyTeams()    ← cached
  For each myTeam in org:
    if myTeam.Parent == nil: skip
    children = FetchChildTeams(org, myTeam.Parent.Slug)  ← cached
    For each sibling in children:
      if sibling.Slug == myTeam.Slug: skip   ← don't re-check own team
      if IsTeamMember(org, sibling.Slug, authorLogin): return true  ← cached
  return false

IsOrgMember(org, login):
  GET /orgs/{org}/members/{login}
  204 → true, 404 → false
```

Visual example:
```
        [backend-group]              ← parent
        /             \
  [app-platform]   [frontend]       ← my team + sibling
    |                |
  igor, alice      carol, dave

PR author = carol  → GROUP (in sibling under same parent)
PR author = alice  → TEAM  (in my direct team)
PR author = eve    → ORG   (in grafana org, not in my hierarchy)
PR author = frank  → OTHER (external)
```

### Component Layout

```
internal/github/
  types.go         — (FETCH) Add Author to PullRequest, Parent to UserTeam
  queries.go       — (FETCH) Add author{login} to GQL query
  prs.go           — (FETCH) Map author from GQL response
  team_members.go  — (FETCH) Add FetchChildTeams, FetchIsOrgMember

internal/service/
  classify.go      — (CLASSIFY) Rename Source→ReviewType
  source.go        — (CLASSIFY) NEW: AuthorSource classifier
  pipeline.go      — (CLASSIFY) ClassifiedPR gains AuthorSource;
                     SourceClassifier.ClassifyAll runs both classifiers
  team.go          — (CLASSIFY) TeamService gains IsSiblingTeamMember, IsOrgMember;
                     TeamMemberFetcher gains FetchChildTeams, FetchIsOrgMember
  filter.go        — (FILTER) Rename Source→ReviewType references, unchanged semantics

internal/output/
  table.go         — Rename SOURCE→TYPE, add SOURCE column for AuthorSource
```

### Key Design Decisions

| Decision | Options | Chosen | Why |
|----------|---------|--------|-----|
| Table column name for review type | `REVIEW TYPE` vs `TYPE` | `TYPE` | Shorter, fits table width; JSON: `reviewType` |
| Author source column | `SOURCE` vs `ORIGIN` | `SOURCE` | Reuses familiar column name, now with new meaning |
| Org membership check | Paginated list vs HEAD check | `GET /orgs/{org}/members/{login}` (204/404) | Single request, O(1) |
| Fail strategy for author source | Fail-open (TEAM) vs fail-closed (OTHER) | Fail to OTHER | Conservative: unknown distance = OTHER; consistent with classification being best-effort |
| Where hierarchy resolution lives | Fetch stage vs Classify stage | TeamService (used by Classify) | TeamService already holds user context; hierarchy is a classification concern, not a data fetch concern |

### New REST API Endpoints Needed

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `GET /orgs/{org}/teams/{slug}/teams?per_page=100` | List child teams of a parent | `[]Team{slug, name}` |
| `GET /orgs/{org}/members/{login}` | Check org membership | 204=member, 404=not |

## Implementation Tasks (4 tasks)

### Task 1: Enrich FETCH stage — PR author + team hierarchy data

**Goal:** Every fetched PullRequest includes the author's login, and UserTeam
includes parent team info. The fetch stage produces enriched data for classify.

**Depends on:** none

**Files to modify:**
- `internal/github/types.go` — Add `Author string` to `PullRequest`; add `Parent *ParentTeam` to `UserTeam`; add `ParentTeam` and `ChildTeam` structs
- `internal/github/queries.go` — Add `Author struct { Login string }` to `searchPRNode`
- `internal/github/prs.go` — Map `n.Author.Login` to `pr.Author` in `convertSearchPRNode`
- `internal/github/team_members.go` — Add `FetchChildTeams(org, parentSlug)` and `FetchIsOrgMember(org, login)` methods with caching

**Details:**

Types (`types.go`):
```go
type PullRequest struct {
    // ... existing fields ...
    Author string `json:"author"` // PR author's login
}

type UserTeam struct {
    Slug         string           `json:"slug"`
    Organization TeamOrganization `json:"organization"`
    Parent       *ParentTeam      `json:"parent"` // nil for top-level teams
}

type ParentTeam struct {
    Slug string `json:"slug"`
}

type ChildTeam struct {
    Slug string `json:"slug"`
}
```

GQL query enrichment (`queries.go`):
```go
// Add to searchPRNode:
Author struct {
    Login string
} `graphql:"author"`
```

New API methods (`team_members.go`):
```go
func (c *Client) FetchChildTeams(org, parentSlug string) ([]ChildTeam, error) {
    cacheKey := "child-teams:" + org + "/" + parentSlug
    // Same cache pattern as FetchTeamMembers
    // GET /orgs/{org}/teams/{parentSlug}/teams?per_page=100
}

func (c *Client) FetchIsOrgMember(org, login string) (bool, error) {
    // GET /orgs/{org}/members/{login}
    // 204 = true, 404 = false
    // NOTE: restDoer.Get() may not handle 204 No Content.
    // Options: extend restDoer interface, use go-gh's Do() directly,
    // or check membership via org team list as fallback.
}
```

**Tests:**
- `internal/github/prs_test.go` — Add author login to existing `convertSearchPRNode` test cases
- `internal/github/team_members_test.go` — Test `FetchChildTeams` with mock REST; test `FetchIsOrgMember` 204/404

**Visible deliverable:**
`go test ./internal/github/...` passes with author data flowing through conversion and new API methods tested.

**Verification steps:**
1. `make test` passes
2. `go vet ./...` passes

---

### Task 2: Extend CLASSIFY stage — rename Source→ReviewType, add AuthorSource stub

**Goal:** The classify stage becomes multi-dimensional. `Source` is renamed
to `ReviewType`, and `ClassifiedPR` gains `AuthorSource` (stub: always OTHER).
Table output shows `TYPE` + `SOURCE` columns.

**Depends on:** Task 1

**Files to modify:**
- `internal/service/classify.go` — Rename `Source` → `ReviewType`, constants: `SourceDirect` → `ReviewTypeDirect`, etc.
- `internal/service/pipeline.go` — `ClassifiedPR` gains `AuthorSource`; rename `Source` → `ReviewType`; `SourceClassifier.ClassifyAll` calls stub `ClassifyAuthorSource`
- `internal/service/filter.go` — Rename `Source` → `ReviewType` in references; `modeToSource` → `modeToReviewType`
- `internal/output/table.go` — Header: `SOURCE` → `TYPE`; add `SOURCE` column for `AuthorSource`; colors for both columns
- `cmd/prs/review.go` — Update references to renamed types

**Files to create:**
- `internal/service/source.go` — `AuthorSource` type + constants + stub `ClassifyAuthorSource()` returning `AuthorSourceOther`

**Details:**

`classify.go`:
```go
type ReviewType string

const (
    ReviewTypeDirect    ReviewType = "direct"
    ReviewTypeTeam      ReviewType = "team"
    ReviewTypeCodeowner ReviewType = "codeowner"
)

func Classify(...) ReviewType { ... }
```

`pipeline.go` — ClassifiedPR becomes multi-dimensional:
```go
type ClassifiedPR struct {
    PR           github.PullRequest
    ReviewType   ReviewType
    AuthorSource AuthorSource
}
```

`source.go` — stub:
```go
type AuthorSource string

const (
    AuthorSourceTeam  AuthorSource = "TEAM"
    AuthorSourceGroup AuthorSource = "GROUP"
    AuthorSourceOrg   AuthorSource = "ORG"
    AuthorSourceOther AuthorSource = "OTHER"
)

func ClassifyAuthorSource(pr github.PullRequest, myLogin string, teams *TeamService) AuthorSource {
    return AuthorSourceOther // stub — wired in Task 3
}
```

`table.go` — two-column update:
```go
tp.AddHeader([]string{"REPO", "PR", "TITLE", "URL", "TYPE", "SOURCE", "AGE"})
// ...
tp.AddField(reviewTypeLabel(cp.ReviewType), tableprinter.WithColor(reviewTypeColor(...)))
tp.AddField(authorSourceLabel(cp.AuthorSource), tableprinter.WithColor(authorSourceColor(...)))
```

Author source colors: TEAM=green, GROUP=yellow, ORG=cyan, OTHER=default.

**Tests:**
- `internal/service/filter_test.go` — Update all `Source:` → `ReviewType:`
- `internal/output/table_test.go` — Update headers; add `AuthorSource` to test data
- `internal/service/source_test.go` — Test stub returns OTHER

**Visible deliverable:**
```
REPO        PR    TITLE                    URL    TYPE        SOURCE    AGE
grafana/g   #123  Fix dashboard loading    ...    direct      OTHER     2d
grafana/g   #456  Add new panel            ...    codeowner   OTHER     5d
```

**Verification steps:**
1. `make test` passes
2. Table headers show TYPE and SOURCE (not old SOURCE)

---

### Task 3: Wire real AuthorSource classifier + TeamService hierarchy

**Goal:** The classify stage produces real TEAM/GROUP/ORG/OTHER values by
extending TeamService with hierarchy resolution and wiring `ClassifyAuthorSource`.

**Depends on:** Task 1, Task 2

**Files to modify:**
- `internal/service/team.go` — Extend `TeamMemberFetcher` interface with `FetchChildTeams` + `FetchIsOrgMember`; add `IsSiblingTeamMember()` and `IsOrgMember()` to `TeamService`
- `internal/service/source.go` — Replace stub with real classification logic

**Details:**

Extend `TeamMemberFetcher` (`team.go`):
```go
type TeamMemberFetcher interface {
    FetchTeamMembers(org, slug string) ([]github.TeamMember, error)
    FetchMyTeams() ([]github.UserTeam, error)
    FetchChildTeams(org, parentSlug string) ([]github.ChildTeam, error)
    FetchIsOrgMember(org, login string) (bool, error)
}
```

New TeamService methods (`team.go`):
```go
func (s *TeamService) IsSiblingTeamMember(org, login string) bool {
    teams := s.loadMyTeams()
    if teams == nil { return false }
    for _, t := range teams {
        if t.Organization.Login != org || t.Parent == nil { continue }
        children, err := s.fetcher.FetchChildTeams(org, t.Parent.Slug)
        if err != nil { return false } // fail-closed
        for _, child := range children {
            if child.Slug == t.Slug { continue } // skip own team
            if s.IsTeamMember(org, child.Slug, login) { return true }
        }
    }
    return false
}

func (s *TeamService) IsOrgMember(org, login string) bool {
    ok, err := s.fetcher.FetchIsOrgMember(org, login)
    return err == nil && ok
}
```

Real classifier (`source.go`):
```go
func ClassifyAuthorSource(pr github.PullRequest, myLogin string, teams *TeamService) AuthorSource {
    author := pr.Author
    org := pr.Repository.Owner

    if author == myLogin {
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
```

**Tests:**
- `internal/service/team_test.go` — Table-driven tests for `IsSiblingTeamMember`:
  - Author in sibling team → true
  - Author in my team → false (handled by SharesTeamWith, not this method)
  - My team has no parent → false
  - FetchChildTeams error → false
  - Test `IsOrgMember`: member → true, non-member → false, error → false
- `internal/service/source_test.go` — Table-driven tests for `ClassifyAuthorSource`:
  - Author is myself → TEAM
  - Author in my direct team → TEAM
  - Author in sibling team → GROUP
  - Author in org but not hierarchy → ORG
  - External author → OTHER
  - API error → OTHER

**Visible deliverable:**
```
REPO             PR     TITLE                     URL    TYPE        SOURCE    AGE
grafana/grafana  #1234  Fix auth flow             ...    direct      TEAM      2h
grafana/grafana  #5678  Update panel rendering    ...    codeowner   GROUP     1d
grafana/loki     #910   Add log parser            ...    team        ORG       3d
external/repo    #111   Some external PR          ...    codeowner   OTHER     5d
```

**Verification steps:**
1. `make test` passes
2. `go vet ./...` passes
3. Run `gh inbox prs review --org grafana` — verify SOURCE column shows real values
4. Run `gh inbox prs review --org grafana --json` — verify `reviewType` and `source` fields

---

### Task 4: Polish + JSON output + edge cases

**Goal:** JSON output includes `reviewType` and `source` fields. Edge cases
are handled. End-to-end verification passes.

**Depends on:** Task 3

**Files to modify:**
- `cmd/prs/review.go` — Update JSON output to include review type and author source (currently strips classification)
- `internal/output/json.go` — May need a JSON-specific output struct that includes classification fields

**Details:**

Currently `review.go` strips classification for JSON output:
```go
case "json":
    prs := make([]github.PullRequest, len(results))
    for i, cp := range results {
        prs[i] = cp.PR
    }
    return output.WriteJSON(cmd.OutOrStdout(), prs)
```

This needs to include classification:
```go
case "json":
    return output.WriteJSON(cmd.OutOrStdout(), results)
    // or a dedicated JSON struct with reviewType + source fields
```

Define a JSON output struct if needed:
```go
type jsonPR struct {
    github.PullRequest
    ReviewType   string `json:"reviewType,omitempty"`
    Source       string `json:"source,omitempty"`
}
```

Edge cases to handle:
- `PassthroughClassifier` produces empty ReviewType and AuthorSource — table renders as "-"
- Author is empty string (shouldn't happen, but defensive) → OTHER
- PR from an org the user doesn't belong to → OTHER

**Tests:**
- `cmd/prs/review_test.go` or `internal/output/json_test.go` — Verify JSON includes reviewType and source
- Edge case tests for empty author, passthrough classifier

**Visible deliverable:**
```json
[
  {
    "number": 1234,
    "title": "Fix auth flow",
    "url": "https://github.com/grafana/grafana/pull/1234",
    "author": "alice",
    "reviewType": "direct",
    "source": "TEAM"
  }
]
```

**Verification steps:**
1. `make test` passes
2. `gh inbox prs review --org grafana` — table output correct
3. `gh inbox prs review --org grafana --json` — JSON includes reviewType and source
4. `gh inbox prs review --org grafana --filter direct` — filter still works
5. `gh inbox prs review --org grafana --filter all` — all PRs shown with correct SOURCE

## Task Dependency Graph

```
T1 (FETCH: author in GQL, parent in UserTeam, new API methods)
│
├──→ T2 (CLASSIFY: rename Source→ReviewType, add AuthorSource stub, update output)
│     │
│     └──→ T3 (CLASSIFY: real AuthorSource logic + TeamService hierarchy)
│           │
│           └──→ T4 (Polish: JSON output, edge cases, end-to-end)
```

T1 → T2 → T3 → T4 (sequential — each builds on the previous)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `restDoer.Get()` doesn't handle 204 No Content (org member check) | Medium | Extend restDoer with a status-check method, or use go-gh's raw HTTP client for this one call |
| `GET /user/teams` may not include `parent` field in response | High | Verify empirically in Task 1; if missing, use `GET /orgs/{org}/teams/{slug}` per team to fetch parent |
| Rate limits for large orgs with many sibling teams | Medium | TeamService caches team members per slug; child team lists are small (<10); cache child team lists too |
| GraphQL `author` field might differ from REST login format | Low | GitHub is consistent; verify in Task 1 by comparing GQL author with REST team members |
| Mock complexity: extending TeamMemberFetcher interface | Low | Update all existing test mocks to satisfy new interface; keep new methods as no-ops in unrelated tests |

## Verification Plan

**Automated checks:**
- `make test` — all unit tests pass
- `make build` — binary compiles
- `go vet ./...` — no issues

**Manual smoke tests:**
1. Run `gh inbox prs review --org grafana` → table shows TYPE and SOURCE columns
2. Verify a PR from a known teammate shows `SOURCE=TEAM`
3. Verify a PR from an org member (non-teammate) shows `SOURCE=ORG`
4. Run with `--json` → output includes `reviewType` and `source` fields
5. Run with `--filter direct` → filtering still works on review type

**Edge cases to verify:**
- PR author is the viewer themselves → TEAM
- Viewer has no parent team (top-level team) → GROUP check skipped gracefully
- API error during team fetch → fallback to OTHER
- Org with no teams → all authors show as ORG or OTHER
- Empty author field → OTHER
