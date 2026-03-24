# gh-inbox

A `gh` CLI extension that surfaces GitHub items needing your attention with smart CODEOWNERS filtering.

## Install

```bash
gh extension install radiohead/gh-inbox
```

## Usage

```bash
gh inbox prs review --org grafana              # PRs awaiting my review
gh inbox prs review --org grafana --filter focus       # highest-priority: my team's PRs, direct reviews
gh inbox prs review --org grafana --filter nearby      # expand to sibling teams
gh inbox prs review --org grafana --filter-type direct # only directly-assigned reviews
gh inbox prs review --org grafana --filter-status open # only PRs with no reviews yet
gh inbox prs review --org grafana --output json        # JSON output for scripting
```

### Filter Presets

| Preset | What It Shows |
|--------|---------------|
| `all` (default) | All PRs requesting your review |
| `focus` | Direct + codeowner reviews from your team (open only) |
| `nearby` | Expand focus to sibling teams |
| `org` | All org-internal PRs (excludes external contributors) |

### Granular Flags

Combine `--filter-type`, `--filter-source`, and `--filter-status` for precise control:

- `--filter-type`: `direct`, `team`, `codeowner`
- `--filter-source`: `TEAM`, `GROUP`, `ORG`, `OTHER`
- `--filter-status`: `open`, `in_review`, `approved`, `all`

## How It Works

```
GitHub GraphQL API → github/ → service/ → output/
```

| Stage | Package | Responsibility |
|-------|---------|----------------|
| Fetch | `internal/github/` | GraphQL queries, REST calls, auth via `gh auth token` |
| Classify | `internal/service/` | Three-axis PR classification (ReviewType, AuthorSource, ReviewStatus) |
| Filter | `internal/service/` | Composable filter presets and granular criteria |
| Output | `internal/output/` | Table renderer, JSON serializer |

### CODEOWNERS-Aware Filtering

GitHub's `asCodeOwner` field is unreliable when CODEOWNERS team auto-assign expands a team into individual users. Instead, `gh-inbox` uses **teammate detection** via `SharesTeamWith` to reliably distinguish direct review requests from CODEOWNERS fan-out.

## Development

```bash
make build   # Build
make test    # Run all tests
make lint    # Lint (golangci-lint)
```

## Project Status

**V0.1** (`prs review`) is implemented. See [VISION.md](VISION.md) for the full roadmap and [DESIGN.md](DESIGN.md) for architecture details.
