# Agent Instructions: gh-inbox

## Quick Reference

| Resource | Description |
|----------|-------------|
| [DESIGN.md](DESIGN.md) | Full design doc: architecture, GraphQL queries, implementation phases |
| [docs/](docs/) | Detailed specs: architecture and other reference docs |
| [agent-docs/](agent-docs/) | Agent workflow: doc maintenance, checklists |

## Core Rules

1. **Update docs with every change.** Follow the rules in
   [doc-maintenance.md](agent-docs/doc-maintenance.md) — every PR should
   include relevant documentation updates.

2. **Follow stage order.** Stages are sequential (V0.1 before V0.2, etc.).
   Check that the previous stage's beads task is closed before starting.

3. **One PR per feature/stage.** Keep PRs focused. Use the
   [stage-checklist.md](agent-docs/stage-checklist.md) to verify completeness.

## Code Conventions

- **Go project layout:** `cmd/` for CLI entry points, `internal/` for library
  packages. No `pkg/` directory.
- **Tests:** Table-driven tests following
  [Go wiki conventions](https://go.dev/wiki/TableDrivenTests).
- **Dependencies:** Minimal — prefer standard library; justify each dependency.
- **Makefile targets:** `build`, `test`, `lint`, `run`. Always verify with
  `make test` before committing.

- **Commit format:** Title (one-liner) / What (description) / Why (rationale).

## Landing the Plane (Session Completion)

**When ending a work session**, complete ALL steps:

1. **File issues for remaining work** — create issues for follow-ups
2. **Run quality gates** (if code changed) — `make test`, linters, builds
3. **Update issue status** — close finished work, update in-progress items
4. **Push to remote:**
   ```bash
   git pull --rebase
   git push
   ```
5. **Verify** — all changes committed and pushed

## Agent Documentation

- [Doc Maintenance](agent-docs/doc-maintenance.md) — rules for which docs to
  update for which changes
- [Stage Checklist](agent-docs/stage-checklist.md) — per-stage completion
  template
