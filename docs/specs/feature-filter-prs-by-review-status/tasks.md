---
type: feature-tasks
title: "Filter PRs by Review Status (--filter-status flag)"
status: draft
spec: docs/specs/feature-filter-prs-by-review-status/spec.md
plan: docs/specs/feature-filter-prs-by-review-status/plan.md
created: 2026-03-13
---

# Implementation Tasks

## Dependency Graph

```
T1 (domain types) ──┬──► T2 (GraphQL + fetch) ──┬──► T4 (CLI flag wiring)
                    │                            │
                    └──► T3 (classification +    ┘
                           filter logic) ────────┘
```

## Wave 1: Domain Types and Review Data Model

### T1: Add Review domain types and extend PullRequest

**Priority**: P0
**Effort**: Small
**Depends on**: none
**Type**: task

Add the `Review` struct, `ReviewState` enum, and `ReviewConnection` type to the
`github` package. Add a `Reviews` field to the `PullRequest` domain type. Add
`ReviewStatus` enum and `ReviewStatusSet` to the `service` package. Add
`ReviewStatuses` field to `FilterCriteria` and update `Matches()` to check it.

**Deliverables:**
- `internal/github/types.go` — `Review`, `ReviewConnection`, `ReviewState` constants
- `internal/service/review_status.go` — `ReviewStatus` enum, `ReviewStatusSet` type, constants (`ReviewStatusOpen`, `ReviewStatusInReview`, `ReviewStatusApproved`)
- `internal/service/filter.go` — `ReviewStatuses ReviewStatusSet` field in `FilterCriteria`, updated `Matches()` method
- `internal/service/pipeline.go` — `ReviewStatus` field in `ClassifiedPR`

**Acceptance criteria:**
- GIVEN the `github.PullRequest` struct
  WHEN inspected
  THEN it contains a `Reviews []Review` field where each `Review` has `Author` (string) and `State` (ReviewState) fields
- GIVEN a `FilterCriteria` with a non-nil `ReviewStatuses` set
  WHEN `Matches()` is called with a `ClassifiedPR` whose `ReviewStatus` is not in the set
  THEN `Matches()` returns false
- GIVEN a `FilterCriteria` with a nil `ReviewStatuses` set
  WHEN `Matches()` is called with any `ClassifiedPR`
  THEN the `ReviewStatuses` dimension does not filter it out (nil-means-all convention, FR-009)
- GIVEN the `ReviewStatus` type
  WHEN its constants are enumerated
  THEN exactly three values exist: `open`, `in_review`, `approved` (FR-001 values minus `all` which means "no filter")

---

## Wave 2: GraphQL Query Extension and Classification Logic

### T2: Extend GraphQL query to fetch reviews

**Priority**: P0
**Effort**: Medium
**Depends on**: T1
**Type**: task

Extend `searchPRNode` in `queries.go` to include `reviews(last: 20)` with
`author { login }` and `state` fields. Update `convertSearchPRNode` in `prs.go`
to populate the new `Reviews` field on the domain `PullRequest`. Add tests for
the conversion logic.

**Deliverables:**
- `internal/github/queries.go` — new `searchReviewNode` struct, `Reviews` field on `searchPRNode`
- `internal/github/prs.go` — updated `convertSearchPRNode` to map review nodes
- `internal/github/prs_test.go` — tests for review conversion

**Acceptance criteria:**
- GIVEN the `searchPRNode` struct
  WHEN the GraphQL query executes
  THEN it fetches `reviews(last: 20)` on each PR node, retrieving each review's `author.login` and `state` (FR-003)
- GIVEN a GraphQL response with reviews containing states APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING, and DISMISSED
  WHEN `convertSearchPRNode` processes the response
  THEN each review is mapped to a `github.Review` with the correct `Author` and `State` values
- GIVEN a PR with zero reviews in the GraphQL response
  WHEN `convertSearchPRNode` processes it
  THEN `PullRequest.Reviews` is an empty slice (not nil)

### T3: Implement ReviewStatus classification logic

**Priority**: P0
**Effort**: Medium
**Depends on**: T1
**Type**: task

Implement `ClassifyReviewStatus()` in a new `review_status.go` file. The
function takes a `PullRequest`, `myLogin`, and `*TeamService` and returns a
`ReviewStatus`. It implements the three-state classification (open, in_review,
approved) including the re-request heuristic. Update `SourceClassifier.ClassifyAll()`
to populate the new `ReviewStatus` field.

**Deliverables:**
- `internal/service/review_status.go` — `ClassifyReviewStatus()` function (appended to the file created in T1)
- `internal/service/pipeline.go` — updated `SourceClassifier.ClassifyAll()` to call `ClassifyReviewStatus`
- `internal/service/review_status_test.go` — table-driven tests

**Acceptance criteria:**
- GIVEN a PR with no submitted reviews (APPROVED, CHANGES_REQUESTED, or COMMENTED) from any team member
  WHEN `ClassifyReviewStatus` is called
  THEN it returns `ReviewStatusOpen` (FR-005)
- GIVEN a PR where a team member has submitted a COMMENTED review but no APPROVED review exists from anyone
  WHEN `ClassifyReviewStatus` is called
  THEN it returns `ReviewStatusInReview` (FR-006)
- GIVEN a PR with an APPROVED review and no subsequent CHANGES_REQUESTED review
  WHEN `ClassifyReviewStatus` is called
  THEN it returns `ReviewStatusApproved` (FR-007)
- GIVEN a PR with an APPROVED review followed by a CHANGES_REQUESTED review
  WHEN `ClassifyReviewStatus` is called
  THEN it returns `ReviewStatusInReview` (FR-006, not approved because subsequent changes requested)
- GIVEN the authenticated user has a submitted review AND a pending ReviewRequest on the same PR
  WHEN `ClassifyReviewStatus` is called
  THEN it returns `ReviewStatusOpen` (re-request heuristic, FR-013)
- GIVEN a PR with only PENDING or DISMISSED reviews
  WHEN `ClassifyReviewStatus` is called
  THEN those reviews are ignored and the PR is classified as `ReviewStatusOpen` (negative constraint: never treat PENDING/DISMISSED as active)
- GIVEN `SourceClassifier.ClassifyAll()` is called with PRs
  WHEN classification completes
  THEN each `ClassifiedPR` has its `ReviewStatus` field populated (FR-014)

---

## Wave 3: CLI Flag Wiring and Preset Integration

### T4: Wire --filter-status flag and update preset defaults

**Priority**: P0
**Effort**: Medium
**Depends on**: T2, T3
**Type**: task

Add the `--filter-status` flag to the review command. Update `resolveFilter` to
accept the status flag, apply preset defaults for `ReviewStatus`, and handle the
override logic. Update `PresetCriteria` to include `ReviewStatuses` defaults.
Validate the flag value and produce clear error messages for invalid input. Add
tests for flag resolution and preset integration.

**Deliverables:**
- `cmd/prs/review.go` — `--filter-status` flag on `reviewCmd`, updated `reviewOptions`, updated `resolveFilter` signature and logic
- `internal/service/filter.go` — updated `PresetCriteria` to include `ReviewStatuses` defaults per preset
- `cmd/prs/review_test.go` — tests for flag resolution with presets and overrides
- `internal/service/filter_test.go` — updated `TestCriteriaFilter` and `TestPresetCriteria` for the new dimension

**Acceptance criteria:**
- GIVEN the `prs review` command
  WHEN `--filter-status=open` is provided
  THEN only PRs with `ReviewStatus == open` appear in output (FR-001, FR-005)
- GIVEN the `prs review` command
  WHEN `--filter-status=in_review` is provided
  THEN only PRs with `ReviewStatus == in_review` appear in output (FR-001, FR-006)
- GIVEN the `prs review` command
  WHEN `--filter-status=approved` is provided
  THEN only PRs with `ReviewStatus == approved` appear in output (FR-001, FR-007)
- GIVEN the `prs review` command
  WHEN `--filter-status=all` is provided
  THEN no review-status filtering is applied (FR-008)
- GIVEN the `prs review` command
  WHEN neither `--filter` nor `--filter-status` is provided
  THEN the default status filter is `open` (FR-002)
- GIVEN `--filter=focus` without `--filter-status`
  WHEN the command runs
  THEN the `ReviewStatuses` default for `focus` is `{open: true}` (FR-010)
- GIVEN `--filter=org` without `--filter-status`
  WHEN the command runs
  THEN the `ReviewStatuses` default for `org` is nil (all statuses, FR-010)
- GIVEN `--filter=focus --filter-status=approved`
  WHEN the command runs
  THEN `--filter-status=approved` overrides the preset's default `open` status (FR-011)
- GIVEN `--filter-status=invalid`
  WHEN the command runs
  THEN it returns an error message listing valid values: open, in_review, approved, all (FR-012)
- GIVEN the `prs review` command with no new flags
  WHEN `--filter-type` or `--filter-source` is used
  THEN those flags continue to work as before (negative constraint: do not break existing flag behavior)
