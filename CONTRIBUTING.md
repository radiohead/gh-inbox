# Contributing to gh-inbox

## Development Setup

```bash
# Build
make build

# Test
make test

# Lint
make lint
```

## Workflow

1. **Find work:** `bd ready` (or `bd list --status=open`)
2. **Claim a task:** `bd update inbox-N --claim`
3. **Create a branch:** `git checkout -b feat/short-description`
4. **Implement:** follow [AGENTS.md](AGENTS.md) rules
5. **Verify:** run tests and linters (see commands above)
6. **Close the task:** `bd close inbox-N`
7. **Open a PR:** one PR per feature/stage

## PR Guidelines

- **Title format:** `type: short description` (e.g., `feat: add X`, `fix: Y`)
- **Commit format:** Title (one-liner) / What (description) / Why (rationale)
- **Scope:** one PR per feature/stage; keep PRs focused
- **Checklist:** see [docs/reference/stage-checklist.md](docs/reference/stage-checklist.md)

## Code Review

- All PRs require review before merging
- Address all comments or explain why not
- Keep discussions in PR; decisions that affect architecture go in
  [docs/adrs/](docs/adrs/)

## Agent-Specific Workflow

AI agents working on this project: read [AGENTS.md](AGENTS.md) for the
agent-specific workflow, beads conventions, and session completion checklist.
