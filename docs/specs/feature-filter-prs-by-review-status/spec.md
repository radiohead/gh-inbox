---
type: feature-spec
title: "Filter PRs by Review Status (--filter-status flag)"
status: approved
beads_id: inbox-dqa
created: 2026-03-11
---

# Filter PRs by Review Status (--filter-status flag)

## Problem Statement

The `gh inbox prs review` command surfaces all PRs where a review is requested from the current user (`review-requested:@me`). It has no awareness of whether a review has already been submitted by the user or any teammate. This means PRs that have already been reviewed by someone on the user's team still appear in the inbox, creating noise and making it difficult to identify PRs that genuinely need attention.

The current workaround is manual: the user must open each PR on GitHub to check whether a teammate has already reviewed it, then mentally skip it. There is no way to filter the output by review progress (open, in-review, approved).

## Scope

### In Scope

- New `--filter-status` CLI flag with values `open`, `in_review`, `approved`, `all`
- Extending the GraphQL query to fetch submitted reviews (`reviews(last: 20)`) alongside existing review requests
- New domain types for submitted reviews (`Review`, `ReviewConnection`, `ReviewState` enum)
- Review status classification logic using `TeamService` to determine if a teammate has reviewed
- Re-request detection heuristic: treating a pending review request as overriding a previously submitted review
- Adding `ReviewStatus` as a new filtering dimension in `FilterCriteria`
- Updating `--filter` presets (`focus`, `nearby`, `org`, `all`) to declare an explicit `ReviewStatus` default
- Mutual exclusivity rules between `--filter-status` and `--filter` (or composability if appropriate)
- Unit tests for all new classification and filtering logic
- Integration with existing JSON and table output formats

### Out of Scope

- **Pagination of reviews beyond a single `last: N` window** — The initial implementation will fetch the last N reviews (e.g., 20). Full review history pagination is a separate concern.
- **Timeline-based re-request detection via the GitHub Timeline API** — The heuristic will use the presence of a pending review request to infer re-request, not timeline events. Precise timeline tracking is deferred.
- **New output columns showing review status** — Table/JSON output changes (e.g., adding a "Status" column) are not required by this spec. The feature filters rows; it does not add visible metadata.
- **Changes to the GitHub search query itself** — The search query remains `review-requested:@me`. Status filtering happens post-fetch in the pipeline.
- **Review status for non-team reviewers** — The `open` status check considers only reviewers who are members of the authenticated user's teams. External reviewer status is not tracked.

## Key Decisions

| Decision | Chosen | Rationale | Source |
|----------|--------|-----------|--------|
| Where status filtering occurs in pipeline | Post-fetch, as a new dimension in `FilterCriteria` / `CriteriaFilter` | Consistent with existing ReviewType and AuthorSource dimensions; keeps Fetcher unchanged | Codebase pattern in `filter.go` |
| Re-request detection mechanism | Presence of pending review request for the user implies re-request | GitHub GraphQL API does not expose review-request timestamps; checking if a review request exists after a submitted review is the most reliable heuristic available | GitHub API constraint |
| Default value of `--filter-status` | `open` | The primary use case is seeing PRs that still need attention; already-reviewed PRs are noise by default | Bead task specification |
| Reviews fetch depth | `last: 20` on each PR node | Sufficient to capture recent review state for typical PRs; avoids excessive API payload | Performance constraint |
| `--filter-status` composability with `--filter` preset | Presets declare a default status; `--filter-status` overrides it when both are specified | Allows presets to have sensible defaults while letting the user refine further | Flexibility for users |
| Review state mapping for `in_review` | PRs lacking an APPROVED review, OR having a CHANGES_REQUESTED review without a subsequent APPROVED | Matches the semantic meaning of "not yet fully approved" | Bead task specification |

## Review Status Definitions

Each PR is classified into exactly one review status based on its review state.
Classification considers only reviews with state APPROVED, CHANGES_REQUESTED, or
COMMENTED. Reviews with state PENDING or DISMISSED are ignored.

For the `open` status, only reviews from **team members** (members of the
authenticated user's teams, including the user) are considered. For `in_review`
and `approved`, **all reviews** are considered regardless of reviewer team
membership — these statuses reflect the PR's overall review progress, not
the user's team responsibility.

| Status | Condition | Semantics |
|--------|-----------|-----------|
| `open` | No team member has submitted a review (APPROVED, CHANGES_REQUESTED, or COMMENTED). **Exception:** if the user has both a submitted review AND a pending review request, the PR is treated as re-requested and remains `open`. | "Needs attention from my team" |
| `in_review` | At least one review exists, but the PR is not yet approved. Specifically: (a) no review has state APPROVED, OR (b) the most recent actionable review has state CHANGES_REQUESTED (even if an earlier APPROVED exists). | "Someone looked at it, but it's not done yet" |
| `approved` | At least one review has state APPROVED, and no subsequent review has state CHANGES_REQUESTED. | "Ready to merge" |

A PR that has no reviews at all is `open` (trivially — no team member has reviewed).

### Re-Request Heuristic

A re-request is detected when ALL of these are true for the authenticated user:
1. The user has at least one submitted review on the PR (any state except PENDING/DISMISSED)
2. The user has a pending `ReviewRequest` node in the PR's `reviewRequests`

When re-request is detected, the PR is classified as `open` even though a team
member (the user) has reviewed it. This ensures re-requested reviews surface in
the default inbox.

### Filter Preset Defaults

Each `--filter` preset declares a default `ReviewStatus`. When `--filter-status`
is explicitly provided, it overrides the preset default.

| Preset | ReviewStatus Default | Rationale |
|--------|---------------------|-----------|
| `focus` | `open` | Focus = PRs that need my attention; already-reviewed PRs are noise |
| `nearby` | `open` | Nearby = PRs from my team's area; same reasoning as focus |
| `org` | `all` | Org-wide view; show everything regardless of review progress |
| `all` | `all` | No filtering; show everything |

When no `--filter` preset is used and `--filter-status` is not provided, the
default is `open`.

## Functional Requirements

- **FR-001**: The system MUST accept a `--filter-status` flag on `prs review` with exactly four valid values: `open`, `in_review`, `approved`, `all`.
- **FR-002**: The system MUST default `--filter-status` to `open` when the flag is not provided and no `--filter` preset overrides it.
- **FR-003**: The GraphQL query MUST fetch `reviews(last: 20)` on each PR node, retrieving each review's `author.login` and `state` (APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED, PENDING).
- **FR-004**: The `PullRequest` domain type MUST include an optional `Reviews` field containing the fetched review data.
- **FR-005**: When `--filter-status=open`, the system MUST exclude PRs where any member of the authenticated user's teams (including the user) has submitted a review with state APPROVED, CHANGES_REQUESTED, or COMMENTED — unless the user currently has a pending review request (indicating re-request).
- **FR-006**: When `--filter-status=in_review`, the system MUST show only PRs where at least one review exists (from any reviewer, not team-scoped) but the PR is not yet approved — specifically: no APPROVED review exists, OR the most recent actionable review is CHANGES_REQUESTED.
- **FR-007**: When `--filter-status=approved`, the system MUST show only PRs where at least one review (from any reviewer, not team-scoped) has state APPROVED and no subsequent review has state CHANGES_REQUESTED.
- **FR-008**: When `--filter-status=all`, the system MUST NOT apply any review-status filtering.
- **FR-009**: `FilterCriteria` MUST include a `ReviewStatuses` field (type `ReviewStatusSet`) that follows the existing nil-means-all convention.
- **FR-010**: Each `--filter` preset MUST declare a default `ReviewStatus` value: `focus` MUST default to `open`, `nearby` MUST default to `open`, `org` MUST default to `all`, `all` MUST default to `all`.
- **FR-011**: When `--filter-status` is explicitly provided alongside `--filter`, the explicit `--filter-status` value MUST override the preset's default status.
- **FR-012**: The system MUST reject invalid `--filter-status` values with a clear error message listing the valid options.
- **FR-013**: The re-request heuristic MUST treat the authenticated user as having a pending re-request if and only if a `ReviewRequest` node exists for the user in the PR's `reviewRequests` AND the user has at least one submitted review in the PR's `reviews`.
- **FR-014**: The `ClassifiedPR` struct MUST include a `ReviewStatus` field populated during classification.

## Acceptance Criteria

- GIVEN the user runs `prs review` without `--filter-status`
  WHEN results are returned
  THEN PRs where any teammate has already submitted a review (APPROVED, CHANGES_REQUESTED, or COMMENTED) are excluded, unless the user has a pending re-request on that PR

- GIVEN the user runs `prs review --filter-status=open`
  WHEN a PR has a COMMENTED review from a teammate and no pending review request for the user
  THEN that PR is excluded from results

- GIVEN the user runs `prs review --filter-status=open`
  WHEN a PR has a COMMENTED review from the user but the user also has a pending review request
  THEN that PR is included in results (re-request detected)

- GIVEN the user runs `prs review --filter-status=in_review`
  WHEN a PR has reviews but none with state APPROVED
  THEN that PR is included in results

- GIVEN the user runs `prs review --filter-status=in_review`
  WHEN a PR has an APPROVED review and no subsequent CHANGES_REQUESTED
  THEN that PR is excluded from results

- GIVEN the user runs `prs review --filter-status=approved`
  WHEN a PR has at least one APPROVED review and no subsequent CHANGES_REQUESTED
  THEN that PR is included in results

- GIVEN the user runs `prs review --filter-status=approved`
  WHEN a PR has no APPROVED reviews
  THEN that PR is excluded from results

- GIVEN the user runs `prs review --filter-status=all`
  WHEN results are returned
  THEN all PRs matching other filter dimensions are included regardless of review status

- GIVEN the user runs `prs review --filter=focus`
  WHEN `--filter-status` is not provided
  THEN the `open` status filter is applied by default

- GIVEN the user runs `prs review --filter=focus --filter-status=approved`
  WHEN results are returned
  THEN the focus preset's ReviewType and AuthorSource filters apply, but the status filter is `approved` (overriding the preset default of `open`)

- GIVEN the user runs `prs review --filter-status=invalid`
  WHEN the command is parsed
  THEN an error is returned listing valid values: open, in_review, approved, all

- GIVEN the GraphQL response includes `reviews(last: 20)` data
  WHEN the response is parsed
  THEN each `PullRequest` domain object contains the review author logins and states

- GIVEN a PR with reviews only from users outside the authenticated user's teams
  WHEN `--filter-status=open` is active
  THEN that PR is included (non-team reviews do not count as "reviewed by my team")

- The system SHALL pass `make test` with all new and existing tests succeeding

## Negative Constraints

- NEVER change the GitHub search query string (`review-requested:@me`); status filtering MUST happen post-fetch in the pipeline.
- NEVER treat a review with state PENDING (draft/uncommitted review) as a submitted review for status classification purposes.
- NEVER treat a review with state DISMISSED as an active review; dismissed reviews MUST be ignored when determining current review status.
- DO NOT add new network requests beyond extending the existing GraphQL query with the `reviews` field. The feature MUST NOT introduce additional API calls per PR.
- NEVER expose GraphQL error details to the user; wrap API errors with user-facing context messages.
- DO NOT break existing `--filter`, `--filter-type`, or `--filter-source` flag behavior when `--filter-status` is not provided.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| `reviews(last: 20)` window misses older relevant reviews, causing incorrect status classification | Medium — PRs with >20 reviews could be misclassified | Document the limitation; 20 is sufficient for typical PRs. Future work addresses full review history. |
| Re-request heuristic produces false positives (user has pending request but was never actually re-requested) | Low — edge case where review request was never cleared | Acceptable trade-off; the heuristic errs on the side of showing the PR (false inclusion is better than false exclusion) |
| Increased GraphQL response payload size from fetching reviews | Low — additional ~500 bytes per PR for 20 reviews | Already within API rate limits; payload increase is modest |
| Preset default status changes could surprise users who rely on current `--filter=focus` behavior | Medium — existing users see fewer PRs by default | The previous behavior was equivalent to `all` status; the change is intentional and documented. Users can opt out with `--filter-status=all`. |
| Team membership cache staleness causes incorrect "teammate reviewed" classification | Low — team membership cache already exists with TTL | Existing `TeamService` caching and fail-open semantics apply; no new risk beyond current behavior |

## Open Questions

- [DEFERRED]: Should `--filter-status=in_review` consider the number of required approvals from branch protection rules? — Will address if needed based on user feedback; initial implementation uses simple "has APPROVED" check.
- [DEFERRED]: Should the `nearby` preset default to `open` or `in_review`? — Specified as `open` for now; can be adjusted based on usage patterns.
- [RESOLVED]: How to detect re-requests without timeline API? — Use the presence of a pending review request for a user who has already submitted a review as the heuristic.
- [RESOLVED]: Should `--filter-status` be mutually exclusive with `--filter`? — No; presets declare a default status that `--filter-status` can override, enabling composability.
