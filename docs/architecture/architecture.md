# System Architecture

## Pipeline Overview

gh-inbox fetches GitHub data via GraphQL and REST APIs, classifies and
filters it with client-side logic, and renders results as a
human-friendly table or JSON.

```
  ┌──────────────┐     ┌──────────────┐     ┌────────────────────┐     ┌──────────────┐
  │  GitHub API  │────>│  github/     │────>│  service/pipeline  │────>│   output/    │
  │  GraphQL/REST│     │  fetch data  │     │  classify + filter  │     │  table / JSON│
  └──────────────┘     └──────────────┘     └────────────────────┘     └──────────────┘
```

The pipeline is orchestrated by `service.Pipeline` which runs three
explicit stages:

```
Fetcher.Fetch(org)  →  Classifier.ClassifyAll(prs)  →  PRFilter.Apply(classified)
       │                         │                              │
 raw []PullRequest      []ClassifiedPR{PR, Source}     []ClassifiedPR (filtered)
```

`ClassifiedPR` is the intermediate type that carries a precomputed
`Source` label (`direct`, `team`, `codeowner`, or empty for passthrough)
through to the output layer. This eliminates the double-classification
that would otherwise occur if the output layer re-derived the source
independently.

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
| `classify.go` | `Source` type + constants; `Classify()` — maps a PR to `direct`/`team`/`codeowner` with `direct > team` precedence | ✅ implemented |
| `filter.go` | `Mode` type + constants; `modeToSource()`; `matchesDirect()` + `matchesTeam()` predicates | ✅ implemented |
| `pipeline.go` | `ClassifiedPR`, `Fetcher`/`Classifier`/`PRFilter` interfaces, all implementations (`FetchFunc`, `SourceClassifier`, `PassthroughClassifier`, `ModeFilter`), `Pipeline` + `NewPipeline()` + `Run()` | ✅ implemented |

`TeamService` accepts a `TeamMemberFetcher` interface, which `github.Client`
satisfies implicitly. This decouples the cache logic from the HTTP layer.

**Pipeline implementations:**

| Type | Interface | Purpose |
|------|-----------|---------|
| `FetchFunc` | `Fetcher` | Function adapter wrapping `client.FetchReviewRequestedPRs` |
| `SourceClassifier{Login, Teams}` | `Classifier` | Classifies each PR via `Classify()`; requires user context |
| `PassthroughClassifier{}` | `Classifier` | Wraps PRs with empty Source; used for `ModeAll + JSON` to skip `FetchCurrentUser` |
| `ModeFilter{Mode}` | `PRFilter` | Keeps PRs matching mode; `ModeAll` passes all through |

**Filter modes:**

| Mode | Semantics |
|------|-----------|
| `ModeAll` | No filtering — all PRs shown |
| `ModeDirect` | I'm a User reviewer AND no other User reviewer shares any of my teams |
| `ModeTeam` | My team is requested AND no individual User reviewer shares any of my teams |
| `ModeCodeowner` | Residual — PRs matching neither `direct` nor `team` |

### `output/`

Rendering layer, independent of business logic.

| File | Responsibility |
|------|----------------|
| `table.go` | `WriteTable(w, []service.ClassifiedPR)` — human-friendly table; reads precomputed `Source` from each `ClassifiedPR`; renders empty Source as `"-"` |
| `json.go` | `WriteJSON(w, any)` — machine-readable JSON; called with `[]github.PullRequest` (Source stripped) |

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
