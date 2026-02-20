# gh-inbox

CLI tool that surfaces GitHub items needing your attention with smart CODEOWNERS filtering

## Quick Start

```bash
make build
gh inbox                    # show all action items (table)
gh inbox prs --review       # PRs awaiting my review
gh inbox prs --authored     # my PRs needing attention
gh inbox issues             # issues needing action
gh inbox discussions        # discussions needing response
gh inbox --json             # machine-readable output
```

## Features

- **CODEOWNERS-aware review filtering** — skip PRs where you're only a CODEOWNERS reviewer and others are assigned
- **Unresolved thread detection** — surface your PRs where reviewers are waiting for a response
- **Issue and discussion tracking** — catch mentions and assignments needing action
- **JSON output** — compose with `jq` for automation and scripting

## Pipeline

```
GitHub GraphQL API → fetch + filter → table / JSON output
```

| Stage | Package | Responsibility |
|-------|---------|----------------|
| Fetch | `github/` | GraphQL queries, auth via `gh auth token` |
| Filter | `github/prs.go`, `github/issues.go` | CODEOWNERS logic, response detection |
| Output | `output/` | Table renderer, JSON serializer |

## Development

```bash
make build   # Build
make test    # Run all tests
make lint    # Lint
```

## Project Status

This project follows a staged implementation plan. See [DESIGN.md](DESIGN.md) for
the full design and roadmap.
