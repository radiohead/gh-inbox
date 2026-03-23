# docs/

Documentation index for gh-inbox.

## Directory Structure

```
docs/
├── architecture/     # Per-domain codebase analysis — output of /map-codebase
├── adrs/             # Architecture Decision Records — grouped by research slug
│   └── {research-slug}/  # e.g., adrs/my-research/001-title.md
├── designs/          # Design documents for specific features
├── specs/            # SDD spec packages (spec.md + plan.md + tasks.md)
├── research/         # Findings and analysis — point-in-time snapshots
├── reviews/          # Code review records
├── investigations/   # Investigation reports and debugging post-mortems
├── reference/        # Evergreen tool/API docs — updated in place
│   ├── beads-workflow.md     # Beads issue tracking conventions
│   ├── doc-maintenance.md    # Rules for which docs to update when
│   └── stage-checklist.md    # Per-stage completion template
├── methodologies/    # Architectural philosophy — evergreen
├── _templates/       # One template per document type
│   ├── adr.md
│   └── research.md
└── _archive/         # Raw output, code dumps, PoC artifacts
```

## Document Types

### `architecture/` — Architecture Analysis

For: per-domain codebase analysis produced by `/map-codebase` and similar
skills. Content describes the current state of the codebase, not decisions.
Updated in place as the codebase evolves.

### `adrs/` — Architecture Decision Records

For: recording an architectural decision — what was decided, why, and what
the consequences are.

**Subdirectory convention**: ADRs are grouped in subdirectories named after the
research report that spawned them. Derive the slug by stripping the date prefix
and `.md` extension from the research filename. ADRs with no research origin go
under `adrs/legacy/`.

**Numbering**: local to each subdirectory (`NNN-title.md`). No global sequence.

**Lifecycle**: `proposed` → `accepted` → `deprecated` | `superseded`

**Required header fields** (all 4 must be present):
```
**Created**: YYYY-MM-DD
**Status**: proposed | accepted | deprecated | superseded
**Bead**: inbox-xxx (or "none")
**Supersedes**: path/to/old.md (or "none")
```

Template: [`_templates/adr.md`](_templates/adr.md)

### `designs/` — Design Documents

For: detailed design docs for specific features or components. May include
diagrams, API contracts, and implementation details.

### `specs/` — Spec Packages

For: structured spec-driven development (SDD). Each spec lives in its own
subdirectory: `spec.md` + `plan.md` + `tasks.md`.

Use `/plan-spec` to generate and `/build-spec` to implement.

### `research/` — Research Reports

For: investigated a topic, evaluated options, gathered findings. No lifecycle —
point-in-time snapshots. Filename must include date prefix.

**Required header fields**:
```
**Created**: YYYY-MM-DD
**Confidence**: X% (Low|Medium|High)
**Sources**: N
```

Template: [`_templates/research.md`](_templates/research.md)

### `reviews/` — Code Reviews

For: code review records and post-review analysis.

### `investigations/` — Investigation Reports

For: debugging post-mortems, root cause analyses. Point-in-time snapshots.
Filename must include date prefix: `YYYY-MM-DD-short-name.md`.

### `reference/` — Reference Docs

For: documenting how to use a shipped tool or workflow. Evergreen, updated
in place. No template — tool docs have their own natural structure.

### `methodologies/` — Methodologies

Evergreen philosophical and architectural docs. No format changes — content
determines structure.

### `_archive/` — Archive

Raw output, code dumps, PoC artifacts. No format requirements.

## Naming Conventions

| Scope | Convention | Example |
|---|---|---|
| Feature subdirs | Lowercase hyphenated | `my-feature/`, `auth-refactor/` |
| Point-in-time docs | `YYYY-MM-DD-short-name.md` | `2025-11-14-implementation-plan.md` |
| Evergreen docs | Descriptive name only (no date) | `permissions-philosophy.md` |
| Special dirs | Underscore prefix | `_templates/`, `_superseded/`, `_archive/` |

Short names drop the feature prefix — the directory provides context:
`implementation-plan.md` not `my-feature-implementation-plan.md`.
