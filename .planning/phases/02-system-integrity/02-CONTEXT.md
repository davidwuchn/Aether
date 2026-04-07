# Phase 2: System Integrity - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Eliminate data-loss risks from hygiene commands and ensure all Go commands run cleanly on a fresh install. This phase covers: fixing isTestArtifact false positives, adding confirmation gates to destructive hygiene commands, creating a smoke test suite for fresh install validation, removing deprecated code, and ensuring consistent error formatting. It does NOT cover build depth, planning granularity, or orchestration — those are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Fresh install validation
- **D-01:** Create a dedicated smoke test suite (Go test file) that exercises each subcommand against a temp directory with no colony state. Fast, repeatable, part of CI.
- **D-02:** Every registered subcommand gets a test case — no panics, no silent failures, reasonable output (help text, error message, or deprecation notice). All subcommands, not just core ones.

### Deprecated cleanup scope
- **D-03:** Full removal — delete all 13 deprecated command registrations (semantic-*, survey-*, suggest-*) and the 41 deprecated shell scripts in `.aether/utils/`. Clean break, no dead code.
- **D-04:** Safety verification via grep — confirm nothing references deprecated code before deleting.

### Claude's Discretion
- Exact smoke test structure (table-driven tests vs individual test functions)
- Error message formatting standard (prefix + description + remediation hint)
- How to handle commands that require colony state vs ones that don't
- Whether isTestArtifact fix uses a source field or a different approach
- Whether destructive commands get --confirm flag or audit logging or both
- TTY check implementation for blocking agent-initiated destructive commands

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — INTG-01 through INTG-06 define the exact requirements for this phase

### Roadmap
- `.planning/ROADMAP.md` §Phase 2 — Success criteria, dependencies, and risk notes

### Phase 1 infrastructure (reusable)
- `.planning/phases/01-state-protection/01-CONTEXT.md` — Audit log infrastructure decisions, confirmation patterns established
- `pkg/storage/storage.go` — `AppendJSONL`, `ReadJSONL` methods (audit logging for hygiene commands)
- `pkg/storage/lock.go` — `FileLocker` for write safety

### isTestArtifact (needs fix)
- `cmd/suggest.go:62-84` — Current `isTestArtifact()` function with fragile substring matching
- `cmd/maintenance.go:64` — Call site in data-clean command

### Destructive commands (need safety gates)
- `cmd/maintenance.go:94-168` — `backup-prune-global` and `temp-clean` commands (no confirmation)
- `cmd/maintenance.go:17-92` — `data-clean` command (has --confirm, good pattern to follow)

### Deprecated commands (to remove)
- `cmd/deprecated_cmds.go` — 8 deprecated commands (semantic-*, survey-*)
- `cmd/suggest.go` — 5 deprecated suggest commands
- `.aether/utils/` — 41 deprecated shell scripts with DEPRECATED headers

### Error patterns
- `cmd/error_cmds.go` — Existing error command infrastructure
- `cmd/state_cmds.go` — Uses `outputOK()` / `outputError()` pattern (established convention)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `outputOK()` / `outputError()` — Consistent JSON output pattern used across all commands
- `data-clean --confirm` flag — Already implements the confirmation pattern that backup-prune and temp-clean should follow
- Phase 1 audit log (`AppendJSONL`) — Could be used to log hygiene command executions for traceability

### Established Patterns
- Deprecated commands use `newDeprecatedCmd()` helper — returns `ok:true` with deprecation notice (non-breaking)
- Commands check `if store == nil` early and return error — pattern for fresh install handling
- `storage.ResolveAetherRoot()` resolves `.aether/` location — used by commands that don't need store

### Integration Points
- `rootCmd.AddCommand()` — Where all deprecated commands are registered (cleanup target)
- `.aether/utils/` — Deprecated shell scripts directory (cleanup target)
- `cmd/maintenance.go` — All three hygiene commands in one file (safety gate target)

</code_context>

<specifics>
## Specific Ideas

- The smoke test should feel like running the real CLI — it catches "this command panics on first run" problems that unit tests miss
- Full removal of deprecated code means a smaller, cleaner codebase with no confusing dead ends
- The `data-clean --confirm` pattern is already proven — reuse it for the other hygiene commands

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---
*Phase: 02-system-integrity*
*Context gathered: 2026-04-07*
