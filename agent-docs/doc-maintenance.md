# Documentation Maintenance Rules

## Which Docs to Update for Which Changes

### Adding/Changing a Package

| Document               | Update Required?                              |
|------------------------|-----------------------------------------------|
| `docs/architecture.md` | Yes — update package responsibility section   |
| `DESIGN.md`            | Yes — update architecture/package map section |
| `README.md`            | Only if it affects CLI usage or quick start   |

### Changing a Core Feature or API

| Document               | Update Required?                              |
|------------------------|-----------------------------------------------|
| `docs/architecture.md` | Yes — update relevant section                 |
| `README.md`            | Yes — if user-visible behavior changed        |
| `DESIGN.md`            | Yes — if architectural decisions changed      |

### Adding a New ADR

| Document   | Update Required?                                    |
|------------|-----------------------------------------------------|
| `DESIGN.md`| Yes — add row to ADR summary table if one exists    |
| New file   | Create `docs/adr/NNN-title.md`                      |

### Changing Beads Workflow or Conventions

| Document                       | Update Required?                    |
|--------------------------------|-------------------------------------|
| `agent-docs/beads-workflow.md` | Yes                                 |
| `AGENTS.md`                    | Only if core rules change           |

### Changing CLI Flags or Interface

| Document   | Update Required?                                    |
|------------|-----------------------------------------------------|
| `README.md`| Yes — update CLI flags or usage section             |
| `DESIGN.md`| Only if it changes the planned interface            |

## General Rules

1. **Every PR should include doc updates** for any user-visible or
   architecture-level change.
2. **DESIGN.md is the design index** — if you add a new doc file, link it from
   DESIGN.md.
3. **AGENTS.md is the agent index** — if you add a new agent-docs file, link
   it from AGENTS.md.
4. **Don't duplicate** — cross-link between docs instead of copying content.
5. **Keep DESIGN.md authoritative** — update it to reflect implementation
   reality as the project evolves.
