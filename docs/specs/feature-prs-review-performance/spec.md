---
type: feature-spec
title: "Optimize prs review command performance (~8s to <3s)"
status: approved
beads_id: inbox-2b5
created: 2026-03-17
---

# Optimize prs review command performance (~8s to <3s)

## Problem Statement

The `gh inbox prs review` command takes approximately 7.8 seconds to complete for a typical workload (50 PRs, 5 teams, 10 unique authors). This latency makes the tool frustrating for interactive use, where sub-3-second response times are expected.

The time budget breaks down as follows:

```
FetchCurrentUser()           ~300ms  (REST, cached after first run)
FetchReviewRequestedPRs()    ~1-2s   (GraphQL, uncached)
ClassifyAll() total:         ~3-4s   (sequential REST calls)
  - FetchMyTeams()           ~300ms  (first call only, in-process cached)
  - FetchTeamMembers() x5    ~1.5s   (300ms x 5 teams, disk-cached)
  - FetchIsOrgMember() x10   ~2s     (200ms x 10 authors, NO disk cache)
Filter.Apply()               <100ms  (in-process)
```

Two root causes dominate: (1) `FetchIsOrgMember` makes per-author REST calls with no disk caching, and (2) the PR fetch and team pre-loading run sequentially when they could overlap. A third, minor cause is that the GraphQL query fetches a field (`Team.Name`) that is never consumed.

The current workaround is to wait. There is no `--no-cache` or warm-cache workflow.

## Scope

### In Scope

- **Track 1 — Caching:** Wire disk caching into `FetchIsOrgMember` and `FetchReviewRequestedPRs`
- **Track 2 — Parallelism:** Pre-fetch team data concurrently with the PR query
- **Track 3 — Query optimization:** Remove unused GraphQL fields from the PR search query
- **`--refresh` flag** to force-invalidate the PR cache and fetch fresh data
- Unit tests for all new caching paths
- Preserving existing error-handling semantics (cache failures silently ignored, fail-open/fail-closed behavior unchanged)

### Out of Scope

- **Query splitting** (fetching PRs and reviews separately) — investigated and ruled out; adds request count with no net improvement
- **Filter-before-classify optimization** — impossible because all filters (ReviewType, AuthorSource, ReviewStatus) depend on classification output
- **Batch REST endpoints** — GitHub API does not support batch org membership checks
- **Cache eviction UI or cache management subcommand** — users can delete the cache directory manually
- **Pagination beyond 50 PRs** — existing limit is sufficient for current usage

## Key Decisions

| Decision | Chosen | Rationale | Source |
|----------|--------|-----------|--------|
| TTL for org membership cache | 4 hours (same as team data) | Org membership changes rarely; matches existing `DiskCacher` default TTL | Investigation finding |
| TTL for PR data cache | 5 minutes | PR data changes moderately; 5-minute TTL deduplicates typical interactive sessions while keeping data reasonably fresh | User feedback |
| Force-invalidate flag | `--refresh` on `prs review` | Users need an escape hatch when they know data has changed (e.g., just approved a PR) | User feedback |
| Separate DiskCacher instance for PR data | Yes — distinct cache directory or prefix | Prevents 5-min-TTL PR cache from sharing a 4h-TTL instance with team/org data | DiskCacher design (one TTL per instance) |
| Cache key for org membership | `"org-member:{org}:{login}"` | Matches existing key patterns (`"team:{org}/{slug}"`, `"child-teams:{org}/{slug}"`) | Codebase convention |
| Cache key for PR data | `"review-prs:{org}"` | One cached result set per org, matching the single-org query pattern | Codebase convention |
| Parallelism strategy | goroutines with errgroup | FetchMyTeams and FetchReviewRequestedPRs are independent after FetchCurrentUser completes | Data flow analysis |
| GraphQL field removal | Remove `Team.Name` only | `Team.Name` is fetched but never read in Go code; `AsCodeOwner` IS used by classify and filter logic | Code grep verification |

## Functional Requirements

### Track 1: Caching

**FR-001** `FetchIsOrgMember` MUST read from and write to the disk cache when a `Cacher` is configured on the `Client`, following the same pattern as `FetchTeamMembers` (check cache, on miss call API, write result to cache, silently ignore cache errors).

**FR-002** The cache key for org membership MUST be `"org-member:{org}:{login}"` where `{org}` and `{login}` are the function parameters.

**FR-003** The cached value for org membership MUST encode the boolean result as a JSON-marshalable value (e.g., `[]byte("true")` / `[]byte("false")`).

**FR-004** `FetchReviewRequestedPRs` MUST read from and write to a dedicated PR cache when available, using the same cache-check / API-call / cache-write pattern as other cached methods.

**FR-005** The PR data cache MUST use a separate `DiskCacher` instance with a 5-minute TTL, distinct from the team/org data cache (4-hour TTL).

**FR-006** The cache key for PR data MUST be `"review-prs:{org}"` where `{org}` is the org parameter (empty string when no org filter is applied).

**FR-007** The `Client` MUST accept a new option `WithPRCache(Cacher)` that sets the PR-specific cache, stored in a separate field from the existing `cache` field.

**FR-008** `cmd/prs/review.go` MUST create two `DiskCacher` instances: one with default TTL (4h) for team/org data, one with 5-minute TTL for PR data, and pass both via `WithCache` and `WithPRCache`.

### Track 1b: Cache Refresh Flag

**FR-009** The `prs review` command MUST accept a `--refresh` flag that, when set, bypasses the PR cache read (forces a fresh GraphQL fetch) and overwrites any existing cache entry with the new result.

**FR-010** The `--refresh` flag MUST only affect the PR data cache. Team/org data caching (FetchTeamMembers, FetchMyTeams, FetchIsOrgMember) MUST NOT be affected by `--refresh`.

### Track 2: Parallelism

**FR-011** After `FetchCurrentUser()` completes, `FetchReviewRequestedPRs` and `TeamService.loadMyTeams` (via `FetchMyTeams`) MUST execute concurrently, not sequentially.

**FR-012** The parallelism MUST NOT change the observable output — classified and filtered results MUST be identical to sequential execution.

**FR-013** If either concurrent operation fails, the error MUST propagate to the caller with the same error wrapping as today.

### Track 3: Query Optimization

**FR-014** The `searchReviewRequestNode` GraphQL struct MUST NOT include the `Team.Name` field, since it is fetched but never consumed by any Go code.

**FR-015** All existing fields that ARE consumed (`AsCodeOwner`, `Team.Slug`, `User.Login`, `__typename`) MUST remain in the query.

## Acceptance Criteria

### Track 1: Caching

- GIVEN a `Client` with a `Cacher` configured
  WHEN `FetchIsOrgMember(org, login)` is called for the first time for a given org/login pair
  THEN the result is fetched from the GitHub API AND written to the cache under key `"org-member:{org}:{login}"`

- GIVEN a `Client` with a `Cacher` configured AND a cached entry exists for `"org-member:{org}:{login}"` that is within TTL
  WHEN `FetchIsOrgMember(org, login)` is called
  THEN the cached value is returned WITHOUT making a GitHub API call

- GIVEN a `Client` with a `Cacher` configured AND a cached entry exists for `"org-member:{org}:{login}"` that is beyond TTL
  WHEN `FetchIsOrgMember(org, login)` is called
  THEN the result is fetched from the GitHub API (cache miss behavior)

- GIVEN a `Client` with NO `Cacher` configured (cache is nil)
  WHEN `FetchIsOrgMember(org, login)` is called
  THEN behavior is identical to today (direct API call, no cache interaction)

- GIVEN a `Client` with a PR cache configured via `WithPRCache`
  WHEN `FetchReviewRequestedPRs(org)` is called for the first time
  THEN the result is fetched from the GraphQL API AND the serialized PR list is written to the PR cache under key `"review-prs:{org}"`

- GIVEN a `Client` with a PR cache configured AND a cached PR entry exists within 5-minute TTL
  WHEN `FetchReviewRequestedPRs(org)` is called
  THEN the cached PR list is returned WITHOUT making a GraphQL API call

- GIVEN a `Client` with a PR cache configured AND a cached PR entry is older than 5 minutes
  WHEN `FetchReviewRequestedPRs(org)` is called
  THEN the result is fetched from the GraphQL API (cache miss)

- GIVEN `cmd/prs/review.go` executes
  WHEN the disk cache initializes successfully
  THEN two separate `DiskCacher` instances are created: one with default TTL for `WithCache`, one with 5-minute TTL for `WithPRCache`

### Track 1b: Cache Refresh Flag

- GIVEN the `prs review` command is invoked with `--refresh`
  WHEN `FetchReviewRequestedPRs` executes
  THEN the PR cache read is skipped, a fresh GraphQL query is made, and the result is written to the PR cache (overwriting any existing entry)

- GIVEN the `prs review` command is invoked with `--refresh`
  WHEN team/org data methods (`FetchMyTeams`, `FetchTeamMembers`, `FetchIsOrgMember`) execute
  THEN they MUST still read from and write to the team/org cache normally (unaffected by `--refresh`)

- GIVEN the `prs review` command is invoked WITHOUT `--refresh`
  WHEN a valid cached PR entry exists
  THEN the cached entry is served as normal (default behavior unchanged)

### Track 2: Parallelism

- GIVEN the `prs review` command runs with valid arguments
  WHEN `FetchCurrentUser()` completes successfully
  THEN `FetchReviewRequestedPRs` and `FetchMyTeams` execute concurrently (wall-clock time for both is approximately max(PR_fetch_time, teams_fetch_time), not their sum)

- GIVEN one of the concurrent fetches fails
  WHEN the pipeline processes the error
  THEN the error is returned to the caller with the same wrapping format as the current sequential implementation

- GIVEN 50 PRs with 5 teams and 10 unique authors
  WHEN the `prs review` command runs with warm caches (all team/org data cached, PR data expired)
  THEN total wall-clock time MUST be under 3 seconds (down from ~7.8s baseline)

### Track 3: Query Optimization

- GIVEN the GraphQL query struct `searchReviewRequestNode`
  WHEN the query is sent to GitHub
  THEN the `Team.Name` field is NOT included in the query payload

- GIVEN the GraphQL query struct `searchReviewRequestNode`
  WHEN PR data is fetched and converted via `convertSearchPRNode`
  THEN all existing functionality (review type classification, author source classification, review status, CODEOWNERS detection) produces identical results to the pre-change behavior

## Negative Constraints

- **NC-001** Cache failures MUST NEVER cause `FetchIsOrgMember` or `FetchReviewRequestedPRs` to return an error. Cache read/write errors MUST be silently ignored, consistent with all existing cached methods.

- **NC-002** The `FetchIsOrgMember` fail-closed behavior (return `false` on API error, do not cache errors) MUST NOT change. Only successful API responses are cached.

- **NC-003** The PR cache MUST NEVER use the same `DiskCacher` instance as the team/org cache. Mixing TTLs within a single `DiskCacher` is not supported by the current design.

- **NC-004** The parallelism change MUST NEVER alter the order or content of classified/filtered PR results compared to sequential execution.

- **NC-005** The `AsCodeOwner` field MUST NOT be removed from the GraphQL query. It is actively used by classification and filter logic.

- **NC-006** SAML error handling in `FetchReviewRequestedPRs` MUST NOT change when caching is added. Partial results from SAML-degraded responses MUST be cached the same way as full results (the cache stores whatever the function returns successfully).

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| PR cache serves stale data (new PR not visible for up to 5 min) | Low — users may not see a brand-new review request for up to 5 minutes | `--refresh` flag provides an escape hatch; 5-minute TTL is a reasonable trade-off for interactive CLI usage |
| Concurrent goroutine introduces race condition in TeamService | High — data corruption or panic | TeamService is constructed fresh per command invocation; no shared mutable state between goroutines. FetchMyTeams writes to TeamService fields, FetchReviewRequestedPRs writes to separate PR slice |
| Removing `Team.Name` breaks future code that might use it | Low — field is trivial to re-add | Verified via grep that no Go code references `Team.Name`; can be restored in a single line |
| DiskCacher directory collision between PR cache and team cache | Medium — wrong TTL applied to entries | Use distinct subdirectories (e.g., `gh-inbox/pr-cache` vs default `gh-inbox`), or distinct cache key prefixes that prevent collision |

## Open Questions

- [RESOLVED] Is `AsCodeOwner` used in the codebase? — Yes, it is mapped in `convertSearchPRNode` and consumed by classification/filter logic. It MUST NOT be removed.
- [RESOLVED] Is `Team.Name` used in the codebase? — No. Only `Team.Slug` is read in `prs.go` line 65. Safe to remove.
- [RESOLVED] Can FetchMyTeams and FetchReviewRequestedPRs run concurrently safely? — Yes. They write to completely separate data structures (TeamService.myTeams vs the returned PR slice). No shared mutable state.
- [RESOLVED] Should there be a cache bypass flag? — Yes, `--refresh` flag added to force-invalidate PR cache on demand (user feedback).
- [DEFERRED] Should cache warm-up be a separate subcommand? — Out of scope; not justified by current usage patterns.
