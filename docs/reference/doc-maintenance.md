# Documentation Maintenance Rules

## Which Docs to Update for Which Changes

### Adding/Changing a Package

| Document | Update Required? |
|----------|-----------------|
| `DESIGN.md` | Yes — update package map table |
| `docs/adrs/<research-slug>/` | Yes — update relevant design doc |
| `README.md` | Only if it affects CLI usage or quick start |

### Changing a Core Feature or API

| Document | Update Required? |
|----------|-----------------|
| `DESIGN.md` | Yes — if architectural decisions changed |
| `README.md` | Yes — if user-visible behavior changed |
| `docs/adrs/<research-slug>/` | Yes — update active design doc status |

### Adding a New ADR

| Document | Update Required? |
|----------|-----------------|
| `DESIGN.md` | Yes — add row to ADR summary table |
| New file | Create `docs/adrs/<research-slug>/NNN-title.md` |

### Changing Beads Workflow or Conventions

| Document | Update Required? |
|----------|-----------------|
| `docs/reference/beads-workflow.md` | Yes |
| `AGENTS.md` | Only if core rules change |

### Changing CLI Flags or Interface

| Document | Update Required? |
|----------|-----------------|
| `README.md` | Yes — update CLI flags or usage section |
| Active design doc | Yes — note interface change |

## General Rules

1. **Every PR should include doc updates** for any user-visible or
   architecture-level change.
2. **DESIGN.md is the architecture index** — if you add a new doc file, link
   it from DESIGN.md.
3. **AGENTS.md is the agent entry point** — keep it thin (TOC only); put
   details in docs/.
4. **Don't duplicate** — cross-link between docs instead of copying content.
5. **docs/ is the system of record** — organize by content type, not audience.
