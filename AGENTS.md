# Agent Instructions: gh-inbox

## Quick Reference

| Resource | Description |
|----------|-------------|
| [DESIGN.md](DESIGN.md) | Full design doc: architecture, GraphQL queries, implementation phases |
| [docs/](docs/) | Detailed specs: architecture and other reference docs |
| [agent-docs/](agent-docs/) | Agent workflow: beads, doc maintenance, checklists |

## Core Rules

1. **Claim a beads task before starting work.** Set it to `in_progress` with
   your name as assignee. See [beads-workflow.md](agent-docs/beads-workflow.md).

2. **Close your beads task when done.** Use `bd close inbox-xxx` after all
   acceptance criteria are met.

3. **Update docs with every change.** Follow the rules in
   [doc-maintenance.md](agent-docs/doc-maintenance.md) — every PR should
   include relevant documentation updates.

4. **Follow stage order.** Stages are sequential (V0.1 before V0.2, etc.).
   Check that the previous stage's beads task is closed before starting.

5. **One PR per feature/stage.** Keep PRs focused. Use the
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

1. **File issues for remaining work** — create beads issues for follow-ups
2. **Run quality gates** (if code changed) — `make test`, linters, builds
3. **Update issue status** — close finished work, update in-progress items
4. **Push to remote:**
   ```bash
   git pull --rebase
   bd sync
   git push
   ```
5. **Verify** — all changes committed and pushed

## Agent Documentation

- [Beads Workflow](agent-docs/beads-workflow.md) — inbox prefix, epic/feature
  structure, claim/close workflow
- [Doc Maintenance](agent-docs/doc-maintenance.md) — rules for which docs to
  update for which changes
- [Stage Checklist](agent-docs/stage-checklist.md) — per-stage completion
  template
