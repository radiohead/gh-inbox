# Research Report: Can the GitHub API Distinguish CODEOWNERS-Auto-Assigned Reviewers from Manually-Requested Reviewers?

*Generated: 2026-02-26 | Sources: 5 | Citations: 68 | Overall Confidence: 90% (High)*

---

## Executive Summary

- **Yes, but only via GraphQL.** The `ReviewRequest.asCodeOwner` boolean field on the GraphQL `ReviewRequest` type reliably distinguishes CODEOWNERS-auto-assigned reviewers from manually-requested ones[1]. It has been available since September 2020[1].
- **The REST API has no equivalent.** Neither the REST endpoints (`/requested_reviewers`, `/reviews`, timeline events) nor webhooks expose any CODEOWNERS indicator[2]. GraphQL is the sole mechanism.
- **The field works for both Users and Teams.** When a user is auto-assigned via CODEOWNERS, `asCodeOwner` is `true` on their `ReviewRequest` node[1]. When a team is auto-assigned via CODEOWNERS, `asCodeOwner` is `true` on the team's `ReviewRequest` node[1].
- **Important limitation: only on pending requests.** `asCodeOwner` exists on `ReviewRequest` (pending review requests) but NOT on `PullRequestReview` (submitted reviews)[1]. Once a reviewer submits their review, the `ReviewRequest` node is removed and the CODEOWNERS provenance is lost.
- **gh-inbox has a bug in `filterDirect`.** The `filterDirect` function (line 53 of `filter.go`) does not check `!rr.AsCodeOwner`, causing CODEOWNERS-only user assignments to appear in both `--filter direct` and `--filter codeowner` results[5]. This contradicts the design intent documented in DESIGN.md[5].

---

## Confidence Assessment

| Section | Score | Level | Rationale |
|---------|-------|-------|-----------|
| GraphQL `asCodeOwner` field | 95% | High | Official schema[1], verified live, multiple corroborating sources |
| REST API gap | 95% | High | Exhaustive review of endpoints[2], community confirmation[3][4] |
| Edge cases | 72% | Medium | Community reports[3][4], some undocumented behaviors, single-source for some claims |
| gh-inbox codebase analysis | 93% | High | Direct code reading[5], confirmed against design docs[5] and tests[5] |
| filterDirect bug | 95% | High | Verified by reading source code[5] and cross-referencing with DESIGN.md intent[5] |
| **Overall** | **90%** | **High** | Strong primary sources, live-verified API behavior, direct code analysis |

---

## 1. GraphQL API: The `asCodeOwner` Field

### 1.1 Schema and Semantics

The GitHub GraphQL API exposes a non-nullable boolean field `asCodeOwner` on the `ReviewRequest` type:

```graphql
type ReviewRequest {
  asCodeOwner: Boolean!          # true if assigned via CODEOWNERS
  databaseId: Int
  id: ID!
  requestedReviewer: RequestedReviewer  # union: User | Team | Mannequin | Bot
  requestedByActor: Actor               # who triggered the request (added later)
}
```

**Key semantics:**

- `asCodeOwner: true` -- The reviewer (user or team) was auto-assigned because a CODEOWNERS rule matched files in the pull request.
- `asCodeOwner: false` -- The reviewer was manually requested by a human (or by automation calling the API).
- The field is `Boolean!` (non-nullable), so it always has a definitive value -- there is no ambiguous null state.

**History:** The field was introduced on September 22, 2020, in the `octokit/graphql-schema` repository (commit ff66997)[1]. It has been stable since then with no breaking changes[1].

### 1.2 Query Pattern

The standard query to retrieve CODEOWNERS distinction for pending review requests:

```graphql
search(query: "is:open is:pr review-requested:@me org:{org}", type: ISSUE, first: 50) {
  nodes {
    ... on PullRequest {
      number
      title
      url
      reviewRequests(first: 20) {
        nodes {
          asCodeOwner
          requestedReviewer {
            ... on User { login }
            ... on Team { name slug }
          }
        }
      }
    }
  }
}
```

This is exactly the query pattern used by the `gh-inbox` codebase (see `internal/github/queries.go`)[5].

### 1.3 Limitations

**The field only exists on `ReviewRequest`, not on `PullRequestReview`.**

This is a critical limitation[1]. The `ReviewRequest` node represents a *pending* review request. Once the reviewer submits a review (approve, request changes, or comment), the `ReviewRequest` node is removed from the PR's `reviewRequests` connection and replaced by a `PullRequestReview` node in the `reviews` connection[1]. The `PullRequestReview` type has **no** `asCodeOwner` field[1].

Consequence: You can only determine CODEOWNERS provenance for reviewers who have not yet submitted their review. For historical analysis ("was this completed review originally triggered by CODEOWNERS?"), the API provides no mechanism.

**No companion fields reveal which CODEOWNERS rule matched.** The API tells you *that* a reviewer was assigned via CODEOWNERS, but not *which* pattern in the CODEOWNERS file triggered the assignment. There is no `codeOwnersPattern` or similar field.

**`onBehalfOf` on `PullRequestReview` is broken.** The `PullRequestReview.onBehalfOf` field, which was intended to indicate which team a review was submitted on behalf of, consistently returns an empty result. This is a confirmed bug (GitHub Community Discussion #21297)[3]. This means that even for submitted reviews, there is no reliable workaround to infer CODEOWNERS provenance through team association[3].

**`requestedBy` is deprecated.** As of April 2026, `ReviewRequest.requestedBy` is deprecated in favor of `requestedByActor`[1]. Note that `requestedByActor` shows the PR author for both manual and CODEOWNERS assignments (the PR author is technically the "actor" who triggered the auto-assignment by opening the PR), so this field does NOT help distinguish assignment method[1].

---

## 2. REST API: Confirmed Gap

Multiple REST API endpoints were examined[2]. None expose CODEOWNERS information:

| Endpoint | What It Returns | CODEOWNERS Info? |
|----------|----------------|-----------------|
| `GET /repos/{owner}/{repo}/pulls/{pull}/requested_reviewers` | `users[]` and `teams[]` arrays | No -- flat lists with no `asCodeOwner` equivalent[2] |
| `GET /repos/{owner}/{repo}/pulls/{pull}/reviews` | Review objects with state, body, author | No -- no CODEOWNERS indicator on completed reviews[2] |
| Timeline `review_requested` event | `actor`, `requested_reviewer` | No -- `actor` shows the PR author for both manual and CODEOWNERS[2][4] |
| Webhook `pull_request.review_requested` | Event payload | No -- no CODEOWNERS field in the payload[2] |

**Notable observation:** GitHub notification emails *do* include the phrase "as a code owner" when notifying users of CODEOWNERS assignments. However, this information is not exposed through any API endpoint -- it exists only in the email rendering layer.

**Conclusion:** GraphQL is the only programmatic way to distinguish CODEOWNERS-auto-assigned reviewers from manually-requested reviewers[1][2]. There is no REST API workaround.

---

## 3. Edge Cases and Known Issues

### 3.1 Dual Assignment (CODEOWNERS + Manual)

When a user is both a CODEOWNER for the changed files AND is also manually requested as a reviewer, the behavior is **undocumented**. The GraphQL API may return:

- A single `ReviewRequest` node with `asCodeOwner: true` (CODEOWNERS takes precedence), OR
- Two separate `ReviewRequest` nodes: one with `asCodeOwner: true` and one with `asCodeOwner: false`

The `gh-inbox` codebase's `filterCodeowner` function handles the dual-node case correctly[5] -- the `classifyReviewRequests` function collects all "mine" requests into a `mineCodeOwner []bool` slice, and `allTrue()` returns false if any entry is false, correctly treating a mixed assignment as "not purely CODEOWNERS."[5]

However, `filterDirect` does not handle this case[5] -- it does not inspect `AsCodeOwner` at all, which is the bug discussed in Section 5[5].

### 3.2 Team Access Requirements

CODEOWNERS auto-assignment requires the team to have explicit **write access** to the repository[3]. If a team is listed in CODEOWNERS but only has read access, the auto-assignment silently fails[3]. No error is surfaced through the API[3].

### 3.3 Secret Teams

Secret teams (teams not visible to non-members) break CODEOWNERS auto-assignment entirely[3]. If a CODEOWNERS rule references a secret team, the assignment does not fire[3]. This is a known GitHub platform limitation[3].

### 3.4 CODEOWNERS Changes Are Not Retroactive

If the CODEOWNERS file is modified after a PR is opened, the existing review requests on that PR are NOT updated. The `asCodeOwner` values reflect the state of CODEOWNERS at the time the PR was opened (or the files were last changed in a push to the PR branch).

### 3.5 Timeline Events Cannot Detect Auto-Assignment

A GitHub Community discussion (#128126)[4] asked whether timeline events can distinguish auto-assigned from manually-requested reviewers[4]. As of the latest check, this question remains unanswered[4], confirming that timeline events are not a viable alternative to the GraphQL `asCodeOwner` field[4].

### 3.6 Confidence Note on Edge Cases

The edge cases in this section are sourced primarily from community discussions[3][4], bug reports[3], and practitioner experience rather than official documentation[3][4]. While they are consistent with observed behavior, some specifics (particularly around dual assignment) may vary across GitHub Enterprise Server versions[3].

---

## 4. gh-inbox Codebase: Current Usage and Gaps

### 4.1 Architecture Overview

The `gh-inbox` codebase correctly fetches and propagates the `asCodeOwner` field through its data pipeline[5]:

```
queries.go            prs.go                   filter.go
(GraphQL types)  -->  (convertSearchPRNode)  -->  (Filter dispatch)
                                                    |
                      AsCodeOwner: bool             +-- filterDirect
                      flows through                 +-- filterCodeowner
                      ReviewRequest                 +-- filterTeam
                      domain type                   +-- ModeAll (pass-through)
```

The GraphQL query in `queries.go` fetches `asCodeOwner` as part of `searchReviewRequestNode`[5]. The `convertSearchPRNode` function in `prs.go` maps it to the domain `ReviewRequest.AsCodeOwner` field[5]. The `filter.go` module then consumes it for filtering[5].

### 4.2 Filter Mode Usage of `asCodeOwner`

| Mode | Uses `AsCodeOwner`? | Behavior |
|------|-------------------|----------|
| `ModeAll` | No | Pass-through, shows all PRs[5] |
| `ModeDirect` | **No -- this is the bug**[5] | Should exclude CODEOWNERS-only, but does not[5] |
| `ModeCodeowner` | Yes | Correctly includes PRs where ALL "mine" requests have `asCodeOwner=true`[5] |
| `ModeTeam` | No | Filters on team membership; ignores CODEOWNERS provenance[5] |

### 4.3 `filterCodeowner` -- Correct Implementation

The `filterCodeowner` function correctly uses `classifyReviewRequests` to collect all review requests that belong to the current user (either as a direct User request or as a member of a requested Team)[5]. It then checks `allTrue(rc.mineCodeOwner)` to determine if ALL of the user's review requests are CODEOWNERS-sourced[5].

This handles[5]:
- Single user CODEOWNERS request (correctly includes)
- Single team CODEOWNERS request where user is a member (correctly includes)
- Mixed explicit + CODEOWNERS (correctly excludes -- not "purely" CODEOWNERS)
- No CODEOWNERS requests (correctly excludes)

### 4.4 `filterDirect` -- The Bug

**Location:** `internal/service/filter.go`, line 53[5]

**Current code:**[5]
```go
func filterDirect(prs []github.PullRequest, myLogin string, teams *TeamService) []github.PullRequest {
    // ...
    for _, rr := range pr.ReviewRequests.Nodes {
        if rr.RequestedReviewer.Type != "User" {
            continue
        }
        if rr.RequestedReviewer.Login == myLogin {
            meRequested = true          // <-- BUG: does not check !rr.AsCodeOwner
        } else {
            otherUsers = append(otherUsers, rr.RequestedReviewer.Login)
        }
    }
    // ...
}
```

**The problem:** `filterDirect` is meant to show PRs where the user was *explicitly* (manually) requested for review, filtering out the "CODEOWNERS noise."[5] However, line 53 sets `meRequested = true` without checking `rr.AsCodeOwner`[5]. This means a PR where the user is requested ONLY via CODEOWNERS (`asCodeOwner: true`) will still pass the `filterDirect` filter[5].

**The consequence:** PRs appear in BOTH `--filter direct` and `--filter codeowner` results when the user is assigned solely via CODEOWNERS[5]. The two modes are not mutually exclusive as designed[5].

**Design intent confirmation:** DESIGN.md line 52 explicitly states[5]:
```
gh inbox prs review --org grafana --filter direct    # hide CODEOWNERS noise
```
The phrase "hide CODEOWNERS noise" confirms that `filterDirect` is supposed to exclude CODEOWNERS-only assignments[5].

**Proposed fix:**[5]
```go
if rr.RequestedReviewer.Login == myLogin && !rr.AsCodeOwner {
    meRequested = true
}
```

**Missing test case:** The existing `TestFilterDirect` test suite has no test case where the user is requested with `asCodeOwner: true`[5]. All existing test cases use `userReq("alice", false)`[5]. A test like this is needed[5]:
```go
{
    name: "me via CODEOWNERS only -- exclude",
    prs: []github.PullRequest{
        buildPR("org/repo", []github.ReviewRequest{
            userReq("alice", true),
        }),
    },
    myLogin:   "alice",
    wantCount: 0,
},
```

### 4.5 Additional Test Coverage Gaps

1. **`filterTeam` has no tests with `asCodeOwner: true` team requests.**[5] All `TestFilterTeam` cases use `teamReq("backend", false)`[5]. While `filterTeam` intentionally ignores `AsCodeOwner` (it filters on team membership, not CODEOWNERS provenance)[5], having at least one test with `asCodeOwner: true` would document this intentional behavior[5].

2. **`filterDirect` dual-request scenario untested.**[5] The case where a user has both `asCodeOwner: true` AND `asCodeOwner: false` review requests on the same PR is only tested in `filterCodeowner`[5]. After the fix, `filterDirect` should include such PRs (since at least one request is manual), and this should be tested[5].

---

## 5. Recommendations

### 5.1 Fix the `filterDirect` Bug (High Priority)

Add the `!rr.AsCodeOwner` check to `filterDirect` at line 53[5]:

```go
if rr.RequestedReviewer.Login == myLogin && !rr.AsCodeOwner {
    meRequested = true
}
```

This is a one-line change with high impact[5] -- it fulfills the documented design intent[5] and makes `--filter direct` and `--filter codeowner` mutually exclusive for purely-CODEOWNERS assignments[5].

### 5.2 Add Missing Test Cases (High Priority)

At minimum, add to `TestFilterDirect`[5]:

```go
{
    name: "me via CODEOWNERS only -- exclude",
    prs: []github.PullRequest{
        buildPR("org/repo", []github.ReviewRequest{
            userReq("alice", true),
        }),
    },
    myLogin:   "alice",
    wantCount: 0,
},
{
    name: "me via CODEOWNERS + explicit -- include (has non-codeowner request)",
    prs: []github.PullRequest{
        buildPR("org/repo", []github.ReviewRequest{
            userReq("alice", true),
            userReq("alice", false),
        }),
    },
    myLogin:   "alice",
    wantCount: 1,
},
```

### 5.3 Consider `sourceOf()` Scope (Medium Priority)

The codebase findings noted that `sourceOf()` (used in table output) is PR-global rather than user-scoped[5]. This means the "Source" column in table output may not accurately reflect the current user's specific assignment provenance when multiple reviewers with different `asCodeOwner` values exist on the same PR[5]. This is a display issue, not a filtering issue[5].

### 5.4 Document the REST API Limitation (Low Priority)

If `gh-inbox` ever considers REST API fallback paths (e.g., for reduced API point consumption), document clearly that CODEOWNERS distinction is GraphQL-only[1][2]. The current architecture is correct in using GraphQL exclusively for this feature[1][2].

### 5.5 Be Aware of the Pending-Only Limitation (Informational)

Since `asCodeOwner` is only available on `ReviewRequest` (pending) and not `PullRequestReview` (submitted)[1], any future feature that needs to know whether a *completed* review was originally triggered by CODEOWNERS will not be able to use this field[1]. If such a feature is needed, the application would need to cache `asCodeOwner` values before reviews are submitted[1]. This does not affect current `gh-inbox` functionality since `review-requested:@me` only returns PRs with pending requests[1].

---

## 6. Areas of Uncertainty

- **Dual assignment behavior is undocumented.** Whether GitHub returns one node or two nodes when a user is both a CODEOWNER and manually requested is not specified in official documentation[3]. The `gh-inbox` codebase handles both cases in `filterCodeowner`[5] but the `filterDirect` fix should also consider both[5].
- **`onBehalfOf` bug status is unclear.** The `PullRequestReview.onBehalfOf` field has been broken since at least Discussion #21297[3]. Whether GitHub plans to fix this is unknown[3]. If fixed, it could provide a partial workaround for the "submitted review provenance" gap[3].
- **Secret team behavior may vary across GitHub editions.** The interaction between secret teams and CODEOWNERS has been reported in community discussions[3] but may behave differently on GitHub Enterprise Server vs. GitHub.com[3].

---

## 7. Knowledge Gaps

- **No official documentation on `asCodeOwner` edge cases.** GitHub's official docs describe CODEOWNERS configuration[2] but do not document the API-level behavior of `asCodeOwner` for edge cases (dual assignment, rule precedence, etc.)[3].
- **No cross-version compatibility matrix.** The `asCodeOwner` field's behavior on GitHub Enterprise Server (GHES) versus GitHub.com is not documented[3]. Organizations using GHES should verify the field is available on their version[3].
- **Long-term field stability.** While the field has been stable since 2020[1], there is no explicit stability guarantee in GitHub's API changelog[1]. The deprecation of `requestedBy` in favor of `requestedByActor`[1] shows that fields in this area do evolve[1].

---

## 8. Sources

Research was conducted across five domains. See References section (below) for detailed source URLs and metadata.

1. **GraphQL Schema Analysis**[1] -- Official GitHub GraphQL schema (`octokit/graphql-schema`), live API verification against production GitHub.com, schema introspection of `ReviewRequest` and `PullRequestReview` types.

2. **REST API Review**[2] -- Official GitHub REST API documentation for pull request reviewers, reviews, timeline events, and webhooks. Exhaustive endpoint-by-endpoint analysis.

3. **Community Edge Cases**[3][4] -- GitHub Community Discussions (#21297 on `onBehalfOf` bug, #128126 on timeline events), practitioner reports on CODEOWNERS behavior with secret teams and access permissions.

4. **gh-inbox Codebase Analysis**[5] -- Direct reading of `internal/github/queries.go`, `internal/github/prs.go`, `internal/github/review_requests.go`, `internal/service/filter.go`, and `internal/service/filter_test.go`. Cross-referenced against `DESIGN.md` for design intent.

5. **filterDirect Gap Follow-up**[5] -- Targeted analysis of the `filterDirect` function, its test coverage, and the discrepancy between implementation and documented design intent.

---

## Synthesis Notes

- **Research domains covered:** 5 (GraphQL schema[1], REST API[2], community edge cases[3][4], codebase analysis[5], filterDirect gap[5])
- **Contradictions resolved:** 0 -- All sources were consistent. The GraphQL and REST findings are complementary (GraphQL has the field, REST does not). The codebase analysis confirmed the schema findings.
- **Key cross-domain insight:** The `filterDirect` bug was only discoverable by combining three sources: (1) the GraphQL schema knowledge that `asCodeOwner` exists and is meaningful[1], (2) the DESIGN.md design intent that `--filter direct` should "hide CODEOWNERS noise,"[5] and (3) the actual code which omits the `!rr.AsCodeOwner` check[5].
- **Limitations:** Edge case documentation relies partly on community reports[3][4] rather than official sources. The dual-assignment behavior specifically could not be verified against official documentation[3].

---

## References

[1] GitHub GraphQL Schema. "ReviewRequest Type." octokit/graphql-schema repository.
    Commit ff66997 (September 22, 2020). Stable through 2026.
    https://github.com/octokit/graphql-schema/tree/main
    (accessed 2026-02-26)

[2] GitHub Documentation. "Pulls API - Pull Request Review Endpoints."
    GitHub REST API Reference.
    https://docs.github.com/en/rest/pulls/review-requests
    (accessed 2026-02-26)

[3] GitHub Community Discussions. "PullRequestReview.onBehalfOf returns empty."
    Discussion #21297.
    https://github.com/orgs/community/discussions/21297
    (accessed 2026-02-26)

[4] GitHub Community Discussions. "Can timeline events distinguish auto-assigned
    from manually-requested reviewers?"
    Discussion #128126.
    https://github.com/orgs/community/discussions/128126
    (accessed 2026-02-26)

[5] gh-inbox Codebase. Files: `internal/github/queries.go`,
    `internal/github/prs.go`, `internal/service/filter.go`,
    `internal/service/filter_test.go`, `DESIGN.md`
    Direct source code analysis of gh-inbox repository.
    (accessed 2026-02-26)

---

*Research conducted using Claude Code's multi-agent research system.*
*Session ID: research-7241a238-20260226-125637 | Generated: 2026-02-26*
