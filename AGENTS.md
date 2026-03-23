# Agent Instructions: gh-inbox

> Read [CONSTITUTION.md](CONSTITUTION.md) before starting any non-trivial work.

## Quick Reference

| Resource | Description |
|----------|-------------|
| [CONSTITUTION.md](CONSTITUTION.md) | Project invariants, taste rules, architecture constraints |
| [DESIGN.md](DESIGN.md) | Architecture index: pipeline, ADRs, package map |
| [docs/](docs/) | Full documentation: adrs, specs, research, reference |
| [docs/reference/](docs/reference/) | Workflow docs: beads, doc maintenance, checklists |

For full docs directory layout, see [docs/README.md](docs/README.md).

## Core Rules

1. **Read CONSTITUTION.md first.** It defines invariants you must not violate
   without explicit human approval.

2. **Claim a beads task before starting work.** Set it to `in_progress`.
   See [docs/reference/beads-workflow.md](docs/reference/beads-workflow.md)
   for the full workflow.

3. **Close your beads task when done.** Use `bd close inbox-N`
   after all acceptance criteria are met.

4. **Update docs with every change.** Follow the rules in
   [docs/reference/doc-maintenance.md](docs/reference/doc-maintenance.md) —
   every PR should include relevant documentation updates.

5. **One PR per feature/stage.** Keep PRs focused. Use
   [docs/reference/stage-checklist.md](docs/reference/stage-checklist.md)
   to verify completeness before closing work.

## Code Conventions

- **Go project layout:** `cmd/` for CLI entry points, `internal/` for library
  packages. No `pkg/` directory.
- **Tests:** Table-driven tests following
  [Go wiki conventions](https://go.dev/wiki/TableDrivenTests).
- **Dependencies:** Minimal — prefer standard library; justify each dependency.
- **Makefile targets:** `build`, `test`, `lint`, `run`. Always verify with
  `make test` before committing.

- **Commit format:** Title (one-liner) / What (description) / Why (rationale).

## Session Completion

**When ending a work session**, complete ALL steps:

1. File issues for remaining work — `bd create` for follow-ups
2. Run quality gates (if code changed) — `make test`, linters, builds
3. Close finished beads tasks — `bd close inbox-N`
4. Sync beads to remote — `bd dolt push`
5. Commit and push — `git add <files> && git commit -m "..." && git push`

## Deeper Context

- [docs/README.md](docs/README.md) — full docs directory conventions
- [DESIGN.md](DESIGN.md) — architecture decisions and package map
- [CONTRIBUTING.md](CONTRIBUTING.md) — development setup and PR process
