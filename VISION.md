# Vision: gh-inbox

A single command to answer: **"What needs my attention on GitHub right now?"**

`gh-inbox` is a `gh` CLI extension that aggregates PRs, issues, and discussions
across orgs — with smart filtering that understands CODEOWNERS, team
relationships, and review state.

## Roadmap

```
V0.1  prs review          ✅ Implemented
V0.2  prs authored         ○ Next
V0.3  issues               ○ Planned
V0.4  discussions           ○ Planned
V0.5  --json output mode    ○ Planned
V1.0  config + multi-org    ○ Planned
```

### V0.1: `gh inbox prs review` (Implemented)

Show PRs awaiting my review with CODEOWNERS-aware filtering.

- Three-axis PR classification: ReviewType (direct/team/codeowner), AuthorSource (TEAM/GROUP/ORG/OTHER), ReviewStatus (open/in_review/approved)
- Filter presets (all/focus/nearby/org) and granular flags (--filter-type, --filter-source, --filter-status)
- Teammate detection via `SharesTeamWith` (replaces unreliable `asCodeOwner`)
- Disk-based team member caching with separate TTLs for team data and PR data
- SAML-aware error handling with graceful degradation for multi-org users
- Table and JSON output formats

### V0.2: `gh inbox prs authored`

Show my PRs that need attention — unresolved review threads where a reviewer is waiting for my response.

- Query `author:@me` PRs via GraphQL
- Detect unresolved `ReviewThread` where last comment is not by me
- Surface PRs with `CHANGES_REQUESTED` review state and no subsequent push
- Same table/JSON output as `prs review`

### V0.3: `gh inbox issues`

Surface issues needing my action — assigned in-progress items and unresponded mentions.

- Query Projects v2 API for items with Status="In Progress" assigned to me
- Search `mentions:@me` issues where I haven't commented after the mention
- Staleness detection (no update in X days) as a V2 enhancement

### V0.4: `gh inbox discussions`

Show discussions needing my response.

- Search `mentions:@me type:discussion` via GraphQL
- Detect unanswered replies on discussions I authored
- Leverage `Discussion.isAnswered` for resolved/unresolved state

### V0.5: `--json` output mode

Structured JSON output across all subcommands for automation and scripting.

- Consistent JSON schema across prs/issues/discussions
- Composable with `jq` for custom pipelines

### V1.0: Config file, multiple orgs, polish

Production-ready release with persistent configuration.

- Config file (`~/.config/gh-inbox/config.yml`) for default orgs, username, preferences
- Multi-org support: query across configured orgs in a single invocation
- Unified `gh inbox` dashboard summarizing all item types
- Polished error messages, help text, and documentation

## Activity History (Epic)

Orthogonal to the main roadmap — snapshot-based state tracking for all item types.

- Each `gh inbox` invocation persists a snapshot to a local store
- State transitions computed across consecutive snapshots (e.g., REVIEW_REQUESTED → CHANGES_REQUESTED → APPROVED)
- Append-friendly store format (newline-delimited JSON) in platform-appropriate cache dir
- Accessible via Go API; CLI surface (`gh inbox prs history #1234`) deferred to follow-up

## Design Principles

- **Minimal dependencies**: stdlib + `go-gh` + `shurcooL-graphql` + `cobra`
- **Pipeline architecture**: Fetch → Classify → Filter → Output with strict unidirectional deps
- **Fail-safe**: fail-open for membership checks (don't hide PRs), fail-closed for classification (don't promote unknowns)
- **Zero auth setup**: delegates to `gh auth token`
- **Composable**: table output for humans, JSON for automation
