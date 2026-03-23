# Beads Workflow

## Project Setup

- **Prefix:** `inbox`
- **Workspace root:** `/Users/igor/Code/radiohead/gh-inbox`
- **Epic:** `INBOX-1` — gh-inbox: CLI tool that surfaces GitHub items needing your attention with smart CODEOWNERS filtering
- **Feature tasks:** One per implementation stage (see structure below)

## Issue Types

| Type    | When to Use                                        |
|---------|----------------------------------------------------|
| epic    | Top-level project tracking (one per project)       |
| feature | Major stage or capability (one per DESIGN.md phase)|
| task    | Implementation unit within a feature               |
| bug     | Defect found during or after implementation        |
| chore   | Maintenance, docs, CI, tooling                     |

## Epic/Feature Structure

```
inbox-62x (epic) gh-inbox: CLI tool that surfaces GitHub items needing your attention with smart CODEOWNERS filtering
  ├── inbox-cul (feature) V0.1: gh inbox prs --review with CODEOWNERS filtering
  ├── inbox-6zf (feature) V0.2: gh inbox prs --authored with unresolved thread detection
  ├── inbox-ab1 (feature) V0.3: gh inbox issues (assigned + mentioned)
  ├── inbox-lbq (feature) V0.4: gh inbox discussions needing response
  ├── inbox-4a6 (feature) V0.5: --json output mode
  └── inbox-tyq (feature) V1.0: Config file, multiple orgs, polish
```

Each feature may have child tasks broken out as needed during implementation.

## Claim/Close Workflow

### Starting Work

1. Check ready tasks: `bd ready` or `bd list --status open`
2. Assign yourself: `bd update inbox-xxx --assignee <agent-name>`
3. Set in-progress: `bd update inbox-xxx --status in_progress`

### Completing Work

1. Verify all acceptance criteria are met
2. Ensure relevant docs are updated (see [doc-maintenance.md](doc-maintenance.md))
3. Close the task: `bd close inbox-xxx`
4. Check if closing this task unblocks the next stage

### Dependencies

Stages are sequential — each stage blocks the next. A stage cannot start
until the previous stage is closed.

## Conventions

- Always call `set_context` with the workspace root before any beads operations
- Use descriptive titles that include the stage number
- Add comments to issues for notable decisions or blockers
- Keep the epic updated with overall progress
