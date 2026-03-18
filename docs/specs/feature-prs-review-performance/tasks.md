---
type: feature-tasks
title: "Optimize prs review command performance (~8s to <3s)"
status: draft
spec: docs/specs/feature-prs-review-performance/spec.md
plan: docs/specs/feature-prs-review-performance/plan.md
created: 2026-03-17
---

# Implementation Tasks

## Dependency Graph

```
T1 (org-member cache) ──┐
                        ├──► T4 (wire caches + parallelism in review.go)
T2 (PR cache)    ───────┤
                        │
T3 (GraphQL trim) ──────┘
```

T1, T2, T3 are independent and form Wave 1.
T4 depends on all three and forms Wave 2.

## Wave 1: Core Caching and Query Optimization

### T1: Add disk caching to FetchIsOrgMember

**Priority**: P0
**Effort**: Small
**Depends on**: none
**Type**: task

Add disk cache read/write to `FetchIsOrgMember` in `internal/github/team_members.go`, following the same pattern used by `FetchTeamMembers`, `FetchMyTeams`, and `FetchChildTeams`. The cache key MUST be `"org-member:{org}:{login}"`. The boolean result MUST be stored as a JSON-marshalable value (`json.Marshal(bool)` producing `"true"` or `"false"` bytes). Cache errors MUST be silently ignored (do not return them to the caller). Existing fail-closed behavior on API errors MUST be preserved: on API error, return `false, err` without writing to cache.

**Deliverables:**
- `internal/github/team_members.go` — updated `FetchIsOrgMember`
- `internal/github/team_members_test.go` — new test cases for cache hit, cache miss, cache error, and API error paths

**Acceptance criteria:**
- GIVEN a Client with a Cacher configured and no cached entry for `"org-member:myorg:alice"`
  WHEN `FetchIsOrgMember("myorg", "alice")` is called and the API returns 204
  THEN the function returns `(true, nil)` AND the Cacher contains key `"org-member:myorg:alice"` with value `"true"`
- GIVEN a Client with a Cacher containing `"org-member:myorg:alice"` = `"true"`
  WHEN `FetchIsOrgMember("myorg", "alice")` is called
  THEN the function returns `(true, nil)` without making a REST API call
- GIVEN a Client with a Cacher containing `"org-member:myorg:bob"` = `"false"`
  WHEN `FetchIsOrgMember("myorg", "bob")` is called
  THEN the function returns `(false, nil)` without making a REST API call
- GIVEN a Client with a Cacher that returns an error on Get
  WHEN `FetchIsOrgMember` is called
  THEN the function proceeds to make the REST API call (cache error silently ignored)
- GIVEN a Client with a Cacher that returns an error on Set
  WHEN `FetchIsOrgMember` successfully queries the API
  THEN the function returns the correct result and the Set error is silently ignored

---

### T2: Add disk caching to FetchReviewRequestedPRs with --refresh support

**Priority**: P0
**Effort**: Medium
**Depends on**: none
**Type**: task

Add a second `prCache Cacher` field to the `Client` struct in `internal/github/client.go`, with a `WithPRCache(Cacher)` client option. Wire cache read/write into `FetchReviewRequestedPRs` in `internal/github/prs.go` using key `"review-prs:{org}"` (empty org uses `"review-prs:"` as key). The cached payload is the JSON-serialized `[]PullRequest` slice. Add a `skipPRCacheRead` bool field on Client and a `WithRefresh()` client option that sets it; when true, FetchReviewRequestedPRs skips cache read but still writes to cache after a successful fetch. Cache errors MUST be silently ignored. SAML error handling MUST remain unchanged.

**Deliverables:**
- `internal/github/client.go` — `prCache Cacher` field, `WithPRCache(Cacher)` option, `skipPRCacheRead` bool, `WithRefresh()` option
- `internal/github/prs.go` — cache read/write in `FetchReviewRequestedPRs`
- `internal/github/prs_test.go` — new test cases for PR cache hit, miss, refresh bypass, cache error, and SAML partial data

**Acceptance criteria:**
- GIVEN a Client with a prCache configured and no cached entry
  WHEN `FetchReviewRequestedPRs("myorg")` is called and GraphQL returns results
  THEN the function returns the PRs AND writes them to prCache under key `"review-prs:myorg"`
- GIVEN a Client with prCache containing valid cached PRs for `"review-prs:myorg"`
  WHEN `FetchReviewRequestedPRs("myorg")` is called
  THEN the function returns the cached PRs without executing a GraphQL query
- GIVEN a Client with prCache containing valid cached PRs AND `skipPRCacheRead` set to true
  WHEN `FetchReviewRequestedPRs("myorg")` is called
  THEN the function skips the cache read, executes the GraphQL query, and overwrites the cache entry
- GIVEN a Client with prCache configured
  WHEN `FetchReviewRequestedPRs` encounters a SAML warning
  THEN SAML handling proceeds identically to the uncached path (partial data returned and cached)
- GIVEN a Client with prCache that returns error on Get
  WHEN `FetchReviewRequestedPRs` is called
  THEN the function proceeds to query GraphQL (cache error silently ignored)

---

### T3: Remove unused Team.Name field from GraphQL query struct

**Priority**: P2
**Effort**: Small
**Depends on**: none
**Type**: chore

Remove the `Name string` field from the `Team` struct inside `searchReviewRequestNode.RequestedReviewer` in `internal/github/queries.go`. Verify that `convertSearchPRNode` in `prs.go` only reads `Team.Slug` (already confirmed). Ensure all existing tests pass unchanged.

**Deliverables:**
- `internal/github/queries.go` — `Team` struct field removal

**Acceptance criteria:**
- GIVEN the updated `searchReviewRequestNode` struct
  WHEN the GraphQL query executes
  THEN the `Team` fragment requests only `Slug` and `__typename`, not `Name`
- GIVEN the updated struct
  WHEN `convertSearchPRNode` processes a Team review request
  THEN `RequestedReviewer.Login` is correctly set to `Team.Slug` (no behavioral change)
- The `AsCodeOwner` field MUST remain present and functional

---

## Wave 2: Parallelism and Command Wiring

### T4: Wire dual caches, --refresh flag, and parallel fetch into review command

**Priority**: P0
**Effort**: Medium
**Depends on**: T1, T2, T3
**Type**: task

Update `cmd/prs/review.go` to: (1) create two DiskCacher instances — the existing one (4h TTL, default) for team/org/user data and a new one (5-minute TTL) for PR data; (2) pass both to `NewClient` via `WithCache` and `WithPRCache`; (3) add a `--refresh` bool flag that passes `WithRefresh()` to the client; (4) after `FetchCurrentUser`, run `FetchReviewRequestedPRs` and `TeamService.PreloadTeams` concurrently using `errgroup`. Add a `PreloadTeams()` method to `TeamService` in `internal/service/team.go` that calls `loadMyTeams()` and returns its error. The pipeline's `Run` method is no longer used for the fetch step — instead, the command layer fetches PRs and pre-loads teams, then calls `ClassifyAll` and `Filter.Apply` directly. Observable output MUST remain identical.

**Deliverables:**
- `cmd/prs/review.go` — dual cacher setup, `--refresh` flag, errgroup-based parallel fetch
- `cmd/prs/review_test.go` — test that `--refresh` flag is registered and wired
- `internal/service/team.go` — `PreloadTeams() error` method
- `internal/service/team_test.go` — test for `PreloadTeams`
- `go.mod` / `go.sum` — `golang.org/x/sync` dependency (if not already present)

**Acceptance criteria:**
- GIVEN the review command is invoked without `--refresh`
  WHEN PR data exists in the 5-min cache
  THEN the cached PR data is used (no GraphQL call) AND team/org caches function at 4h TTL
- GIVEN the review command is invoked with `--refresh`
  WHEN PR data exists in the 5-min cache
  THEN the cache is bypassed, a fresh GraphQL call is made, and the cache is overwritten
- GIVEN the review command is invoked with `--refresh`
  WHEN the command runs
  THEN team and org-member caches are NOT invalidated (only PR cache read is skipped)
- GIVEN FetchReviewRequestedPRs and PreloadTeams are launched concurrently
  WHEN both succeed
  THEN the command produces identical output to the sequential version
- GIVEN FetchReviewRequestedPRs fails with an error
  WHEN running concurrently
  THEN the error is propagated with the same wrapping as the current sequential path
- GIVEN PreloadTeams fails with an error
  WHEN ClassifyAll runs subsequently
  THEN classification falls back gracefully (same behavior as current lazy-load failure)
- GIVEN the updated command
  WHEN `make test` and `make lint` are run
  THEN all checks pass with no regressions
