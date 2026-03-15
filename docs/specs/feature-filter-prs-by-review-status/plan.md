---
type: feature-plan
title: "Filter PRs by Review Status (--filter-status flag)"
status: draft
spec: docs/specs/feature-filter-prs-by-review-status/spec.md
created: 2026-03-13
---

# Architecture and Design Decisions

## Pipeline Architecture

The feature extends the existing Fetch → Classify → Filter pipeline. The GraphQL
query gains a `reviews(last: 20)` field. Classification populates a new
`ReviewStatus` field on `ClassifiedPR`. Filtering gains a new `ReviewStatuses`
dimension in `FilterCriteria`.

```
                         ┌──────────────────────────────────┐
                         │         GitHub GraphQL API        │
                         │  search(review-requested:@me)     │
                         │    └─ reviewRequests(first:20)    │
                         │    └─ reviews(last:20)  ← NEW    │
                         └──────────────┬───────────────────┘
                                        │
                              ┌─────────▼─────────┐
                              │      Fetcher       │
                              │  (FetchFunc wraps  │
                              │   Client method)   │
                              └─────────┬──────────┘
                                        │ []github.PullRequest
                                        │   (now includes Reviews field)
                              ┌─────────▼──────────┐
                              │   SourceClassifier  │
                              │  .ClassifyAll()     │
                              │  ├─ ReviewType      │
                              │  ├─ AuthorSource    │
                              │  └─ ReviewStatus ←NEW│
                              └─────────┬──────────┘
                                        │ []ClassifiedPR
                              ┌─────────▼──────────┐
                              │   CriteriaFilter    │
                              │  ReviewTypes    set │
                              │  AuthorSources  set │
                              │  ReviewStatuses set ← NEW
                              └─────────┬──────────┘
                                        │ []ClassifiedPR (filtered)
                              ┌─────────▼──────────┐
                              │   Output (table/json)│
                              └─────────────────────┘
```

### CLI Flag Resolution Flow

```
--filter-status flag ─┐
                      ├──► resolveFilter() ──► FilterCriteria
--filter preset ──────┘         │                  │
                                │  preset provides │
                                │  default status  │
                                │  --filter-status │
                                │  overrides it    │
                                ▼
                         CriteriaFilter.Apply()
```

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Add `Reviews` field to `github.PullRequest` domain type as `[]Review` (not a connection wrapper) | Keeps the domain type flat and easy to consume. The GraphQL connection is unwrapped during conversion in `convertSearchPRNode`, matching the existing `ReviewRequests` pattern (FR-004). |
| Add `ReviewState` as a string enum in `github` package | States (APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING, DISMISSED) come from the GitHub API. Defining them in `github` keeps API concerns in the API layer. (FR-003) |
| Add `ReviewStatus` enum and `ReviewStatusSet` in `service` package | Follows the existing pattern of `ReviewType`/`ReviewTypeSet` and `AuthorSource`/`AuthorSourceSet`. (FR-009, FR-014) |
| New `ClassifyReviewStatus()` function in a dedicated `review_status.go` file | Keeps the classification logic isolated and testable, similar to how `Classify()` lives in `classify.go` and `ClassifyAuthorSource()` lives in `source.go`. (FR-005, FR-006, FR-007) |
| `ClassifyReviewStatus` takes `myLogin`, `*TeamService`, and PR as inputs | Needs team membership to determine if a team member has reviewed (for `open` status), and needs `myLogin` for re-request detection. (FR-005, FR-013) |
| Extend `searchPRNode` with `Reviews` GraphQL field | The shurcooL-graphql library drives the query from the struct shape. Adding the field to the existing query struct avoids a second network request. (FR-003, negative constraint: no new network requests) |
| `--filter-status` is NOT mutually exclusive with `--filter` | Per spec, presets declare a default status and `--filter-status` overrides it. This differs from `--filter-type`/`--filter-source` which ARE mutually exclusive with `--filter`. (FR-010, FR-011) |
| Default filter changes from showing all PRs to showing `open` PRs only when no flags provided | When no `--filter` and no `--filter-status` are provided, the default behavior is to show `open` PRs only. This changes the current default (which shows all). The change is intentional per FR-002. |
| `resolveFilter` gains a `filterStatus` parameter | Keeps the flag resolution logic centralized. The function builds the `ReviewStatuses` set by combining preset defaults with explicit overrides. |

## Compatibility

**Unchanged behavior:**
- All existing flags (`--org`, `--filter`, `--filter-type`, `--filter-source`, `--output`) continue to work identically
- `--filter-type` and `--filter-source` remain mutually exclusive with `--filter`
- Table and JSON output formats are unchanged (no new columns per Out of Scope)
- The GitHub search query (`review-requested:@me`) is unchanged
- `PassthroughClassifier` continues to work (empty `ReviewStatus` matches all when `ReviewStatuses` is nil)

**Changed behavior:**
- Default output now shows only `open` PRs (previously showed all). Users can restore the old behavior with `--filter-status=all`
- PRs now carry review data in the domain type (additional API payload from `reviews(last: 20)`)

**Newly available:**
- `--filter-status=open` — PRs needing team attention (default)
- `--filter-status=in_review` — PRs with reviews but not yet approved
- `--filter-status=approved` — PRs with approval
- `--filter-status=all` — no status filtering (old default behavior)
