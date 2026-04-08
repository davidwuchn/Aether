# Phase 6: Branching & Worktree Discipline - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Enforce branch naming conventions, track worktree lifecycles from creation through cleanup, prevent orphaned branches, and ensure safe merge protocols. This gives the orchestrator (Phase 5) a disciplined worktree allocation system. It does NOT cover repository hygiene (Phase 7) or final validation (Phase 8).

</domain>

<decisions>
## Implementation Decisions

### Branch naming convention
- **D-01:** Two-track naming — agents get `phase-N/caste-name` format (e.g., `phase-2/builder-1`), humans get prefix-based naming with `feature/`, `fix/`, `experiment/`, `colony/` prefixes.
- **D-02:** `worktree-create` validates branch names against both naming tracks. Invalid names are handled at Claude's discretion (enforcement level not specified by user).
- **D-03:** The naming convention is enforced at worktree creation time, not at git branch creation time (Aether controls worktrees, not general git operations).

### Merge protocol gates
- **D-04:** Merge to main requires two gates: `go test ./...` passes AND `clash-check` finds no file conflicts. Both must pass before merge proceeds.
- **D-05:** On merge failure (tests fail or clash detected), the merge is blocked and a flag/blocker is created to report the failure. No auto-fix or auto-retry — the user decides what to do.
- **D-06:** After a successful merge: worktree directory is removed, branch is deleted, status is updated to `merged` in COLONY_STATE.json. Full auto-cleanup.

### Worktree lifecycle tracking
- **D-07:** Worktree lifecycle state is tracked in COLONY_STATE.json (not a separate file). This matches Phase 5's pattern of keeping all state in one place.
- **D-08:** Worktree statuses: `allocated`, `in-progress`, `merged`, `orphaned`.
- **D-09:** A worktree with no commits in 48 hours is flagged as potentially orphaned by `worktree-orphan-scan`.
- **D-10:** Worktree tracking is audited via Phase 1's append-only audit log (creation, merge, cleanup are significant mutations).

### Orchestrator integration
- **D-11:** The orchestrator creates worktrees by calling `worktree-allocate` — the single entry point for all worktree creation. The orchestrator does not call git directly for worktree management.
- **D-12:** `worktree-allocate` creates the branch with enforced naming convention (`phase-N/caste-name`), creates the git worktree, and registers it in COLONY_STATE.json with `allocated` status.
- **D-13:** After agent completes work, orchestrator calls the merge protocol (D-04 gates), and on success triggers full auto-cleanup (D-06).

### Claude's Discretion
- Whether invalid branch names are rejected outright or warned but allowed
- Exact WorktreeEntry struct fields in ColonyState
- How `worktree-orphan-scan` reports results (JSON output vs table)
- Whether to add a `worktree-status` command for individual worktree inspection
- Pre-commit hook integration for go vet/fmt (not required for merge, but optional)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — BRAN-01 through BRAN-06 define the exact requirements for this phase

### Roadmap
- `.planning/ROADMAP.md` §Phase 6 — Success criteria, dependencies, risk notes

### Existing worktree code (must be extended, not replaced)
- `cmd/clash.go` — `clash-check`, `clash-setup`, `worktree-create`, `worktree-cleanup` commands. `worktree-create` currently accepts arbitrary branch names (no validation). Must add naming validation and state registration.
- `cmd/worktree_merge.go` — Deprecated `worktree-merge` command. Can be removed or ignored.

### Colony state model
- `pkg/colony/colony.go` — ColonyState struct. Add worktree registry fields here (per D-07).
- `pkg/storage/storage.go` — `AtomicWrite`, `AppendJSONL` for audit logging (Phase 1 infrastructure).

### Phase 1 context (audit integration)
- `.planning/phases/01-state-protection/01-CONTEXT.md` — Audit log patterns, significant mutations list. Worktree creation/merge/cleanup should be added as audited operations.

### Phase 5 context (orchestrator integration)
- `.planning/phases/05-orchestration-layer/05-CONTEXT.md` — Orchestrator architecture, task routing, agent isolation. The orchestrator's worktree allocation call must go through `worktree-allocate`.
- `pkg/agent/curation/orchestrator.go` — Proven orchestrator pattern.
- `cmd/orchestrator.go` — Orchestrator commands (if implemented by Phase 5).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `worktree-create` (cmd/clash.go:116-147) — Already creates worktrees via git. Needs naming validation wrapper and state registration. The core git operations are solid.
- `worktree-cleanup` (cmd/clash.go:149-185) — Already removes worktrees and prunes stale references. Needs state deregistration.
- `clash-check` (cmd/clash.go:14-66) — Already checks file conflicts across worktrees. Ready to use as a merge gate.
- `clash-setup` (cmd/clash.go:82-113) — Installs git merge driver for clash detection.
- `outputOK()`/`outputError()` — Standard JSON output pattern for all commands.
- Phase 1 audit log — `AppendJSONL` ready for worktree mutation logging.

### Established Patterns
- Commands use cobra with flags registered in `init()` and `rootCmd.AddCommand()`
- State stored as JSON via `store.SaveJSON()` / `store.LoadJSON()`
- Agent system uses Caste enum for agent types (builder, watcher, scout, etc.)
- Git operations use `context.WithTimeout` with `GitTimeout` constant

### Integration Points
- `cmd/clash.go` — Extend `worktree-create` with naming validation and state registration, or create new `worktree-allocate` command alongside
- `pkg/colony/colony.go` — Add `Worktrees []WorktreeEntry` field to ColonyState
- Phase 5 orchestrator — Will call `worktree-allocate` to create isolated worktrees per agent task
- Phase 1 audit log — Worktree mutations (allocate, merge, cleanup) are audited as significant mutations

</code_context>

<specifics>
## Specific Ideas

- The `phase-N/caste-name` naming convention makes it immediately obvious which worktrees belong to which phase and which agent
- The 48-hour stale threshold balances catching real orphans with not flagging active multi-day work
- Full auto-cleanup after merge means no manual worktree housekeeping

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---
*Phase: 06-branching-worktree-discipline*
*Context gathered: 2026-04-07*
