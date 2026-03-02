# Design Doc: `gh-inbox` - GitHub Productivity CLI

## Context

No existing tool covers the full set of requirements. Here's why:

| Tool | PRs for review | CODEOWNERS filter | My PRs need response | Issues | Discussions |
|------|:-:|:-:|:-:|:-:|:-:|
| **gh-dash** (10k stars) | Yes (via search filters) | No | No | Partial | No |
| **gh-notify** (325 stars) | Yes (notifications) | No | No | No | No |
| **gh CLI native** | `review-requested:@me` | No | No | Basic | No |

The critical gap: **CODEOWNERS vs explicit assignment** and **"do I need to respond?"** logic require GraphQL + client-side processing that none of these tools implement.

---

## Proposal: `gh-inbox` - a `gh` extension in Go

### Why a gh extension (vs standalone tool)?

| | gh extension | Standalone binary | Shell scripts |
|---|---|---|---|
| **Auth** | Free - delegates to `gh auth token` | Must implement OAuth/token management | Uses `gh api` (indirect) |
| **Distribution** | `gh extension install user/gh-inbox` | Manual binary download or brew | Copy scripts around |
| **Output control** | Full - it's just a Go binary | Full | Limited (string manipulation) |
| **Complex logic** | Go: easy for GraphQL parsing, filtering | Same | Painful for nested JSON + filtering |
| **Ecosystem** | Discoverable via `gh extension search` | Separate discovery | Not discoverable |
| **Dependencies** | Only requires `gh` (already installed) | Standalone | Requires `gh` + `jq` + maybe `python` |
| **Multi-format output** | `--json`, `--table`, `--jq` trivial in Go | Same | Hard to maintain |

The key reasons to choose gh extension over standalone:
1. **Zero auth setup** - reuses `gh`'s existing auth, which already has the right scopes
2. **Distribution story** - `gh extension install` is one command, auto-updates work
3. **Composability** - fits naturally in `gh` workflows (`gh inbox | gh pr view ...`)
4. **Go is the right language** - the complex client-side filtering (CODEOWNERS, response detection) is painful in shell but natural in Go. And Go is what `gh` itself and `gh-dash` use.

### Distribution (zero-friction)

No registry or approval process. The workflow:
1. Create a GitHub repo named `gh-inbox` with the `gh-extension` topic
2. Scaffold with `gh extension create --precompiled=go` (generates cross-compile CI)
3. `git tag v0.1.0 && git push --tags` -> CI builds binaries for all platforms
4. Users install with `gh extension install radiohead/gh-inbox`
5. Users update with `gh extension upgrade gh-inbox`
6. Discoverable via `gh extension search inbox`

A gh CLI extension that shows your GitHub action items across PRs, Issues, and Discussions.

```
gh inbox                          # show all action items (table)
gh inbox prs review --org grafana # PRs awaiting my review
gh inbox prs review --org grafana --filter direct    # hide CODEOWNERS noise
gh inbox prs review --org grafana --filter codeowner # CODEOWNERS-only PRs
gh inbox prs authored             # my PRs needing attention
gh inbox issues                   # issues needing action
gh inbox discussions              # discussions needing response
```

### Output formats (by priority)

- **P0**: Plain table (human-friendly, like `gh pr list`)
- **P1**: JSON (`--json` flag for agent/automation consumption)
- **P2 (maybe)**: TUI mode (`--tui` or separate `gh inbox-tui`)

### Configuration

```yaml
# ~/.config/gh-inbox/config.yml
orgs:
  - grafana
  - grafana-labs
username: radiohead    # auto-detected from gh auth status
```

---

## Feature Breakdown

### a) PRs Awaiting My Review (`gh inbox prs review`)

**Logic**:
1. Query: `is:open is:pr review-requested:@me org:{org}` via GraphQL search
2. For each PR, fetch `reviewRequests` with reviewer type and login
3. Fetch authenticated user's teams via REST (`GET /user/teams`)
4. Client-side filtering using **teammate detection** (see below)

**Filter modes** (`--filter`):

| Mode | Semantics |
|------|-----------|
| `all` (default) | No filtering — show all PRs |
| `direct` | I'm a User reviewer AND no other User reviewer shares any of my teams |
| `team` | My team is requested AND no individual User reviewer shares any of my teams |
| `codeowner` | Residual — PRs matching neither `direct` nor `team` (I review alongside teammates) |

**Why teammate detection instead of `asCodeOwner`?**

The GraphQL `asCodeOwner` field is unreliable for User review requests. When
GitHub's CODEOWNERS team auto-assign expands a team into individual users,
those users get `asCodeOwner=false` even though the review was
CODEOWNERS-originated. This made `--filter=direct` show CODEOWNERS
auto-assigns and `--filter=codeowner` return zero results.

The teammate-based approach uses `SharesTeamWith` to detect whether other User
reviewers are on any of my teams. CODEOWNERS auto-assign always produces
multiple reviewers from the same team, so this signal reliably distinguishes
direct requests from CODEOWNERS fan-out.

**Key verified GraphQL query** (tested live against grafana org):
```graphql
search(query: "is:open is:pr review-requested:@me org:grafana", type: ISSUE, first: 50) {
  nodes {
    ... on PullRequest {
      number, title, url
      repository { nameWithOwner }
      reviewRequests(first: 20) {
        nodes {
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

**Client-side filtering logic**:

```
# Data: my teams fetched via GET /user/teams (cached per session)
# SharesTeamWith(org, login) = login is in any of my teams within org

matchesDirect(pr):
  for each reviewer in pr.reviewRequests:
    skip if not a User
    if reviewer == me:  meRequested = true
    else if SharesTeamWith(org, reviewer):  return false   # teammate → not direct
  return meRequested

matchesTeam(pr):
  for each reviewer in pr.reviewRequests:
    if Team and I'm a member:  hasMyTeam = true
    if User and != me and SharesTeamWith(org, reviewer):  return false
  return hasMyTeam

--filter=direct:    matchesDirect(pr)
--filter=team:      matchesTeam(pr)
--filter=codeowner: !matchesDirect(pr) AND !matchesTeam(pr)
```

**Fail modes**: `SharesTeamWith` is fail-closed (returns false on API error →
PR stays visible in direct mode). `IsTeamMember` is fail-open (returns true on
error → avoids hiding PRs due to transient failures).

### b) My PRs Needing Attention (`gh inbox prs authored`)

**Logic**:
1. Query: `is:open is:pr author:@me org:{org}`
2. For each PR, fetch `reviewThreads` with `isResolved`
3. For unresolved threads, check if last comment is NOT by me -> I need to respond
4. Also check: PR has `CHANGES_REQUESTED` review state and I haven't pushed since

**Key verified GraphQL fields**:
- `PullRequestReviewThread.isResolved` - confirmed in schema
- `PullRequestReviewThread.comments` - get author of last comment
- `PullRequest.reviewDecision` - APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED

**Output columns**: Repo | PR# | Title | Unresolved Threads | Review State | Last Updated

### c) Issues Needing Action (`gh inbox issues`)

**Two sub-queries**:

1. **Assigned + In Progress**: Query Projects v2 API for items with Status="In Progress" assigned to me
2. **Mentioned but not responded**: Search `is:issue mentions:@me is:open`, then check if I've commented after the most recent mention of my username

**Caveat**: The "needs action" heuristic for in-progress issues is inherently fuzzy. V1 can simply list in-progress issues; "staleness" detection (no update in X days) could be a V2 enhancement.

### d) Discussions Needing Response (`gh inbox discussions`)

**Logic**:
1. Search: `mentions:@me type:discussion` via GraphQL
2. For each discussion, check if I've commented after the mention
3. Also show discussions where I'm the author and there are unanswered replies

**Confirmed GraphQL fields**: `Discussion.comments`, `Discussion.answer`, `Discussion.isAnswered`

---

## Architecture

```
gh-inbox (Go binary)
  |
  +-- config.go          # load ~/.config/gh-inbox/config.yml
  +-- github/
  |     +-- client.go    # GraphQL + REST + Cacher interfaces, Client struct
  |     +-- queries.go   # all GraphQL query strings
  |     +-- prs.go       # PR fetching
  |     +-- team_members.go  # FetchCurrentUser, FetchTeamMembers (+ Cacher wiring)
  |     +-- issues.go    # issue fetching + mention-response detection
  |     +-- discussions.go
  +-- service/
  |     +-- team.go      # TeamService — membership cache, SharesTeamWith, fail-open/closed
  |     +-- filter.go    # Filter dispatcher + matchesDirect + matchesTeam predicates + filterDirect + filterCodeowner + filterTeam
  +-- output/
  |     +-- table.go     # human-friendly table output
  |     +-- json.go      # machine-readable JSON output
  +-- cmd/
        +-- root.go      # `gh inbox` root command
        +-- prs/
              +-- prs.go     # `prs` parent command, exports Cmd
              +-- review.go  # `prs review` subcommand, --org + --filter
              +-- authored.go# `prs authored` subcommand (placeholder)
```

**Auth**: Delegates to `gh auth token` - no separate auth config needed.

**API strategy**: GraphQL for everything except Notifications (REST-only). Single batched query per subcommand where possible to minimize API calls.

---

## Key API Findings (Verified Live)

| Capability | API | Verified |
|---|---|---|
| `ReviewRequest.asCodeOwner` | GraphQL `ReviewRequest` type | Yes - tested, but unreliable for User requests after team auto-assign expansion. Not used for filtering; teammate detection via `SharesTeamWith` used instead. |
| `PullRequestReviewThread.isResolved` | GraphQL | Yes - schema confirmed |
| `Discussion` search + comments | GraphQL | Yes - schema confirmed |
| Projects v2 field values | GraphQL `ProjectV2Item` | Schema confirmed |
| Notifications by reason | REST `/notifications` | Yes - tested, returns `reason` field |

---

## Risks & Open Questions

1. **"Have I responded?" is heuristic**: No API tracks this. We scan comment authors, which works ~90% of the time but misses cases like responding in a different thread or resolving via commit.

2. **CODEOWNERS team membership**: If I'm assigned via a team in CODEOWNERS, the `requestedReviewer` is a `Team`, not a `User`. We need to resolve team membership to know if I'm the reviewer. This requires an additional API call: `GET /orgs/{org}/teams/{slug}/members`.

3. **Rate limits**: GraphQL has 5000 points/hr. Each search query costs ~1 point per node. With 3 orgs x 4 queries x 50 results, we're well within limits.

4. **Projects v2 scope**: Querying project status requires knowing the project number. Config may need to include project IDs, or we discover them via the org's `projectsV2` connection.

---

## Implementation Phases

| Phase | Scope | Effort |
|---|---|---|
| **V0.1** | `gh inbox prs --review` with CODEOWNERS filtering | Small |
| **V0.2** | `gh inbox prs --authored` with unresolved thread detection | Small |
| **V0.3** | `gh inbox issues` (assigned, mentioned) | Medium |
| **V0.4** | `gh inbox discussions` | Small |
| **V0.5** | `--json` output mode | Small |
| **V1.0** | Config file, multiple orgs, polish | Medium |

---

## Verification

To validate the design before building:
```bash
# 1. Verify asCodeOwner works for your PRs
gh api graphql -f query='{ search(query: "is:open is:pr review-requested:@me org:grafana", type: ISSUE, first: 5) { nodes { ... on PullRequest { number title reviewRequests(first: 10) { nodes { asCodeOwner requestedReviewer { ... on User { login } ... on Team { name } } } } } } } }'

# 2. Verify review threads on your authored PRs
gh api graphql -f query='{ search(query: "is:open is:pr author:@me org:grafana", type: ISSUE, first: 5) { nodes { ... on PullRequest { number title reviewThreads(first: 20) { nodes { isResolved comments(last: 1) { nodes { author { login } } } } } } } } }'

# 3. Verify discussion search
gh api graphql -f query='{ search(query: "mentions:@me type:discussion org:grafana", type: DISCUSSION, first: 5) { nodes { ... on Discussion { number title repository { nameWithOwner } } } } }'
```
