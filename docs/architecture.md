# System Architecture

## Pipeline Overview

gh-inbox fetches GitHub data via GraphQL and REST APIs, filters it with
client-side logic, and renders results as a human-friendly table or JSON.

```
  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
  │  GitHub API  │────>│  github/     │────>│   output/    │
  │  GraphQL/REST│     │  fetch+filter│     │  table / JSON│
  └──────────────┘     └──────────────┘     └──────────────┘
```

## Package Responsibilities

### `cmd/`

CLI entry point. Parses subcommands (`prs`, `issues`, `discussions`) and
flags (`--review`, `--authored`, `--json`), then wires fetch + output.

### `github/`

All API interaction. Key files:

| File | Responsibility |
|------|----------------|
| `client.go` | GraphQL + REST client, delegates auth to `gh auth token` |
| `queries.go` | All GraphQL query strings |
| `prs.go` | PR fetching + CODEOWNERS filtering logic |
| `issues.go` | Issue fetching + mention-response detection |
| `discussions.go` | Discussion fetching + unanswered-reply detection |

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
- REST only for Notifications (`/notifications` endpoint, REST-only)
- Single batched query per subcommand where possible to minimize API calls
- GraphQL rate limit: 5000 points/hr — well within budget for typical usage
