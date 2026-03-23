# Agent Instructions: gh-inbox

## Quick Reference

| Resource | Description |
|----------|-------------|
| [CONSTITUTION.md](CONSTITUTION.md) | Project invariants, taste rules, architecture constraints |
| [DESIGN.md](DESIGN.md) | Architecture index: pipeline, ADRs, package map |
| [docs/](docs/) | Full documentation: adrs, specs, research, reference |
| [docs/reference/](docs/reference/) | Workflow docs: beads, doc maintenance, checklists |

## Core Rules

1. **Read CONSTITUTION.md first.** It defines invariants you must not violate
   without explicit human approval.

2. **Claim a beads task before starting work.** Set it to `in_progress` with
   your name as assignee. See [docs/reference/beads-workflow.md](docs/reference/beads-workflow.md).

3. **Close your beads task when done.** Use `bd close inbox-xxx` after all
   acceptance criteria are met.

4. **Update docs with every change.** Follow the rules in
   [docs/reference/doc-maintenance.md](docs/reference/doc-maintenance.md) — every PR should
   include relevant documentation updates.

5. **One PR per feature/stage.** Keep PRs focused. Use
   [docs/reference/stage-checklist.md](docs/reference/stage-checklist.md)
   to verify completeness.

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

## Codebase Architecture Insights

*Last Updated: 2026-03-23 | Analysis Confidence: 92%*

**Stack**: Go | cobra CLI | go-gh/v2 | shurcooL-graphql

### Key Insights

- Strict layered architecture with codified invariants (CONSTITUTION.md) enforcing unidirectional dependency flow across four layers
- Three-axis PR classification (ReviewType, AuthorSource, ReviewStatus) with composable filter presets and priority scoring
- SAML-aware error severity framework enabling graceful degradation for multi-org users
- Minimal dependency philosophy: four well-justified direct dependencies, zero external test dependencies
- Pipeline-based data flow: Fetch → Classify → Filter → Output, with architectural invariants preventing stage cross-calls

### Detailed Documentation

- **Architecture invariants**: [CONSTITUTION.md](CONSTITUTION.md)
- **Component map & design decisions**: [DESIGN.md](DESIGN.md)
- **Per-domain analysis**: [docs/](docs/) (infrastructure, testing, error handling, API design)

## Agent Documentation

- [Beads Workflow](docs/reference/beads-workflow.md) — inbox prefix, epic/feature
  structure, claim/close workflow
- [Doc Maintenance](docs/reference/doc-maintenance.md) — rules for which docs to
  update for which changes
- [Stage Checklist](docs/reference/stage-checklist.md) — per-stage completion
  template
