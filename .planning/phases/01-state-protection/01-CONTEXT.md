# Phase 1: State Protection - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Make every colony state mutation traceable, recoverable, and safe from corruption. This phase covers the audit log, state-history command, corruption detection, checkpoint snapshots, and BoundaryGuard for sensitive paths. It does NOT cover build depth, planning granularity, or orchestration — those are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Audit scope
- **D-01:** Only significant mutations are logged — state-mutate, state-write, phase-insert, phase advance, and plan changes. High-frequency build internals (context-update, worker-spawn) are excluded to avoid log noise and performance overhead.
- **D-02:** Each audit entry must contain: before/after diffs, timestamp, source command, and SHA-256 checksum (per STATE-02).

### state-history UX
- **D-03:** Default output is compact — one line per mutation showing timestamp, command, field changed, and summary (like `git log --oneline`).
- **D-04:** A `--diff` flag shows full before/after JSON diffs for a specific entry or the last N entries.
- **D-05:** A `--tail N` flag limits output to the last N entries (default: 20).

### state-write safety
- **D-06:** `state-write` requires a `--force` flag to execute. Without `--force`, it prints an error suggesting `state-mutate` instead.
- **D-07:** When `state-write --force` is used, the mutation IS recorded in the audit log (it's a significant mutation).

### Checkpoint policy
- **D-08:** Auto-checkpoint snapshots are created before destructive operations only: phase advance, plan overwrite, and `state-write --force`.
- **D-09:** Non-destructive mutations (goal update, phase status change, field set) are recorded in the audit log but do NOT trigger checkpoints.
- **D-10:** Checkpoints are stored in `.aether/data/checkpoints/` with timestamp-based naming.

### Claude's Discretion
- Exact audit log JSONL schema design (field names, nesting)
- Checkpoint rotation/retention policy (how many to keep, when to prune)
- `state-history` output formatting details (column widths, color if TTY)
- BoundaryGuard implementation approach (file watcher, hook-based, or explicit check)
- Corruption detection heuristics beyond the known jq-expression bug
- Whether the audit log itself should be checksummed or signed

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — STATE-01 through STATE-07 define the exact requirements for this phase

### Roadmap
- `.planning/ROADMAP.md` §Phase 1 — Success criteria, dependencies, and risk notes

### State mutation entry points (existing code)
- `cmd/state_cmds.go` — `state-mutate` command (primary mutation path, ~760 lines of expression parsing)
- `cmd/state_extra.go` — `state-write`, `state-checkpoint`, `phase-insert` commands
- `cmd/context_update.go` — `context-update` command (high-frequency, excluded from audit)

### Storage layer (existing infrastructure)
- `pkg/storage/storage.go` — `AtomicWrite`, `AppendJSONL`, `ReadJSONL` methods (audit log ready)
- `pkg/storage/lock.go` — `FileLocker` with syscall.Flock (cross-process write safety)

### State model
- `pkg/colony/colony.go` — `ColonyState` struct definition (lines 45-64), `Plan` struct (lines 71-75)

### Known issues
- Memory note: "LLM reconstructs full JSON causing Frankenstein state" — the corruption bug where jq expressions are stored literally in the events field. STATE-05 must detect and reject this pattern.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `Store.AppendJSONL()` — Already handles append-only JSONL writes with locking. Perfect for the audit log.
- `Store.ReadJSONL()` — Already reads JSONL with malformed line handling. Perfect for state-history.
- `Store.AtomicWrite()` — Already does temp-file + rename with JSON validation. Used by all state writes.
- `FileLocker` — Cross-process locking via syscall.Flock. Serializes concurrent writes.
- `state-checkpoint` command — Already saves named snapshots. Needs auto-trigger wiring.

### Established Patterns
- State mutations go through `store.SaveJSON()` or `store.AtomicWrite()` — both use the FileLocker.
- Commands use `outputOK()` / `outputError()` for consistent JSON output.
- Expression parsing in `state_cmds.go` uses regex-based pattern matching (not actual jq).

### Integration Points
- `state-mutate` (cmd/state_cmds.go) — Primary target for audit instrumentation
- `state-write` (cmd/state_extra.go) — Needs --force guard + audit logging
- `phase-insert` (cmd/state_extra.go) — Needs audit logging
- Phase advance logic (wherever current_phase is incremented) — Needs checkpoint + audit
- Plan overwrite (wherever plan is replaced) — Needs checkpoint + audit
- `context-update` (cmd/context_update.go) — Explicitly EXCLUDED from audit

</code_context>

<specifics>
## Specific Ideas

- The audit log should feel like `git log` — familiar, scannable, useful for answering "what happened?"
- state-history default output should answer "what changed recently?" in under a second
- Checkpoints are insurance, not a backup strategy — they exist for quick rollback, not disaster recovery

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---
*Phase: 01-state-protection*
*Context gathered: 2026-04-07*
