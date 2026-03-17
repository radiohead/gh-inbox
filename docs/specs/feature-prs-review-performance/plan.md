---
type: feature-plan
title: "Optimize prs review command performance (~8s to <3s)"
status: draft
spec: docs/specs/feature-prs-review-performance/spec.md
created: 2026-03-17
---

# Architecture and Design Decisions

## Pipeline Architecture

Current sequential flow:

```
cmd/prs/review.go
  │
  ├─ NewDiskCacher()          ── single 4h-TTL cacher
  ├─ NewClient(WithCache)
  ├─ FetchCurrentUser()       ── ~300ms (REST, cached)
  │
  ├─ Pipeline.Run(org)
  │    ├─ FetchReviewRequestedPRs(org)    ── ~1-2s (GraphQL, NO cache)
  │    ├─ ClassifyAll(prs)                ── sequential
  │    │    ├─ Classify()
  │    │    │    └─ loadMyTeams()          ── ~300ms (REST, first call)
  │    │    ├─ ClassifyAuthorSource()
  │    │    │    ├─ SharesTeamWith()       ── FetchTeamMembers x5 (disk cached)
  │    │    │    ├─ IsSiblingTeamMember()  ── FetchChildTeams (disk cached)
  │    │    │    └─ IsOrgMember()          ── FetchIsOrgMember x10 (NO disk cache)
  │    │    └─ ClassifyReviewStatus()
  │    └─ Filter.Apply()                  ── <100ms
  │
  └─ output (table/json)
```

Target flow after optimization:

```
cmd/prs/review.go
  │
  ├─ NewDiskCacher("", 0)         ── 4h-TTL cacher (teams, org-member, user)
  ├─ NewDiskCacher("", 5min)      ── 5min-TTL cacher (PR data)
  ├─ NewClient(WithCache, WithPRCache)
  ├─ FetchCurrentUser()           ── ~300ms (REST, cached)
  │
  ├─ ┌── errgroup ──────────────────────────────┐
  │  │ goroutine 1: FetchReviewRequestedPRs(org) │  ← reads/writes PR cache
  │  │ goroutine 2: FetchMyTeams()                │  ← pre-loads team list
  │  └──────────────────────────────────────────────┘
  │        │                          │
  │        ▼                          ▼
  │   []PullRequest              []UserTeam (in TeamService cache)
  │
  ├─ ClassifyAll(prs)              ── teams pre-loaded, org-member disk-cached
  │    ├─ Classify()               ── loadMyTeams() returns instantly
  │    ├─ ClassifyAuthorSource()
  │    │    └─ IsOrgMember()       ── FetchIsOrgMember now disk-cached
  │    └─ ClassifyReviewStatus()
  │
  ├─ Filter.Apply()
  └─ output (table/json)

Cache topology:
  ┌──────────────────────────────────────────────┐
  │  DiskCacher (4h TTL)                         │
  │    current-user, my-teams, team:org/slug,    │
  │    child-teams:org/slug,                     │
  │    org-member:org:login  ← NEW               │
  ├──────────────────────────────────────────────┤
  │  DiskCacher (5min TTL)  ← NEW                │
  │    review-prs:org                            │
  └──────────────────────────────────────────────┘
```

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Add `prCache Cacher` field to Client alongside existing `cache Cacher` | FR-005/FR-008: PR data needs a separate 5-min TTL. The existing `cache` field uses 4h TTL for team/user data. Two distinct DiskCacher instances in review.go, two fields in Client. |
| `WithPRCache(Cacher)` client option | FR-007: Follows the existing `WithCache` pattern. Clean opt-in; callers that do not set it get no PR caching. |
| Boolean encoding as `"true"`/`"false"` bytes for org-member cache | FR-003: `json.Marshal(bool)` produces `"true"` or `"false"` — simple, consistent with existing JSON marshal pattern used for team members. |
| Cache key `"org-member:{org}:{login}"` with colon separator | FR-002: Matches existing cache key patterns (`"team:org/slug"`, `"child-teams:org/slug"`). Uses colon to distinguish from path-like keys. |
| Pre-load `FetchMyTeams()` via `TeamService.PreloadTeams()` method | FR-011: The Pipeline currently calls `loadMyTeams()` lazily inside ClassifyAll. To run it concurrently, TeamService needs an explicit public method that the command layer can call in a goroutine. |
| `--refresh` flag bypasses PR cache read only | FR-009/FR-010: When `--refresh` is set, `FetchReviewRequestedPRs` skips cache read but still writes. Team/org caches are unaffected. Implemented via a `skipPRCacheRead` bool on Client. |
| `errgroup` for parallel fetch | FR-011/FR-013: `golang.org/x/sync/errgroup` provides structured concurrency with error propagation. Both goroutines are independent after FetchCurrentUser completes. |
| Remove `Name string` from `Team` struct in `searchReviewRequestNode` | FR-014: `Team.Name` is fetched by GraphQL but never read by `convertSearchPRNode` — only `Team.Slug` is used. Removing it reduces query payload. |

## Compatibility

**Unchanged:**
- All existing CLI flags and their behavior
- Table and JSON output formats
- Filter presets and granular filter flags
- Error handling semantics (fail-open for team membership, fail-closed for org membership and author source)
- SAML graceful degradation in FetchReviewRequestedPRs
- Existing disk cache paths and TTLs for team/user data

**Newly available:**
- `--refresh` flag on `prs review` to force-invalidate PR cache
- Disk caching for `FetchIsOrgMember` (transparent to callers)
- Disk caching for `FetchReviewRequestedPRs` (transparent to callers)

**Deprecated:**
- Nothing
