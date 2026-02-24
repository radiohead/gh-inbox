# System Architecture

## Pipeline Overview

gh-inbox fetches GitHub data via GraphQL and REST APIs, filters it with
client-side logic, and renders results as a human-friendly table or JSON.

```
  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
  │  GitHub API  │────>│  github/     │────>│  filter/ +   │────>│   output/    │
  │  GraphQL/REST│     │  fetch data  │     │  service/    │     │  table / JSON│
  └──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
```

## Package Responsibilities

### `cmd/`

CLI entry point. Parses subcommands (`prs`, `issues`, `discussions`) and
flags, then wires fetch + filter + output.

```
cmd/
  root.go         → root command, registers subcommands
  prs/
    prs.go        → "prs" parent command, exports Cmd
    review.go     → "prs review" subcommand with --org + --filter
    authored.go   → "prs authored" subcommand (placeholder)
    review_test.go
```

### `github/`

All API interaction. Key files:

| File | Responsibility | Status |
|------|----------------|--------|
| `client.go` | `graphQLDoer`/`restDoer`/`Cacher` interfaces, `Client` struct, constructors | ✅ implemented |
| `queries.go` | GraphQL query structs + `buildReviewRequestedSearchQuery()` | ✅ implemented |
| `prs.go` | `FetchReviewRequestedPRs()`, `convertSearchPRNode()` | ✅ implemented |
| `team_members.go` | `FetchCurrentUser()`, `FetchTeamMembers()` with optional `Cacher` wiring | ✅ implemented |
| `types.go` | Shared public types: `PullRequest`, `Repository`, `TeamMember` | ✅ implemented |
| `issues.go` | Issue fetching + mention-response detection | planned |
| `discussions.go` | Discussion fetching + unanswered-reply detection | planned |

### `service/`

Business logic layer, independent of API and rendering details.

| File | Responsibility | Status |
|------|----------------|--------|
| `team.go` | `TeamService` — lazy in-process team membership cache with fail-open semantics | ✅ implemented |

`TeamService` accepts a `TeamMemberFetcher` interface, which `github.Client`
satisfies implicitly. This decouples the cache logic from the HTTP layer.

### `filter/`

Client-side filtering logic, independent of API details.

| File | Responsibility | Status |
|------|----------------|--------|
| `filter.go` | Top-level `Filter()` dispatcher + `FilterDirect()` + `FilterCodeowner()` | ✅ implemented |

**Modes:**

| Mode | Behavior |
|------|----------|
| `ModeAll` | No filtering — all PRs shown |
| `ModeDirect` | Hide PRs where my requests are CODEOWNERS-only and others are assigned |
| `ModeCodeowner` | Show only PRs where I'm the sole CODEOWNERS reviewer |

### `output/`

Rendering layer, independent of business logic.

| File | Responsibility |
|------|----------------|
| `table.go` | Human-friendly table output (like `gh pr list`) |
| `json.go` | Machine-readable JSON output |

### `config.go`

Loads `~/.config/gh-inbox/config.yml`. Provides org list and username
(falls back to `gh auth status` if not configured).

## Configuration

Config file location: `~/.config/gh-inbox/config.yml`

```yaml
orgs:
  - grafana
  - grafana-labs
username: radiohead    # auto-detected from gh auth status if omitted
```

## Auth Strategy

Delegates entirely to `gh auth token` — no separate OAuth or token management.
The `github.Client` calls `gh auth token` once at startup and reuses the result.

## API Strategy

- GraphQL for all structured data (PRs, issues, discussions, review threads)
- REST for Notifications (`/notifications`) and team membership (`/orgs/{org}/teams/{slug}/members`)
- Single batched query per subcommand where possible to minimize API calls
- GraphQL rate limit: 5000 points/hr — well within budget for typical usage
