# Phase 2: System Integrity - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 02-system-integrity
**Areas discussed:** Fresh install validation, Deprecated cleanup scope

---

## Fresh install validation

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated smoke test suite | Go test file exercising each subcommand against temp dir with no colony state. Fast, repeatable, part of CI. | ✓ |
| Extend existing tests | Add fresh install scenarios to existing test files. No new files, scattered across codebase. | |
| Manual validation script | Script run manually that checks each command. Simple but not automated. | |

**User's choice:** Dedicated smoke test suite
**Notes:** All subcommands should be covered, not just core ones.

| Option | Description | Selected |
|--------|-------------|----------|
| All subcommands | Every registered subcommand gets a test case — no panics, no silent failures, reasonable output. | ✓ |
| Core commands only | Only commonly-used commands. Skip niche ones. | |
| All + output schema checks | Everything plus verify output schema matches docs. | |

**User's choice:** All subcommands
**Notes:** Coverage = done when every subcommand has a test case.

---

## Deprecated cleanup scope

| Option | Description | Selected |
|--------|-------------|----------|
| Full removal | Delete all 13 deprecated commands and 41 shell scripts. Clean break. | ✓ |
| Remove scripts, keep commands | Remove shell scripts but keep deprecated commands returning notices. | |
| Status quo | Leave as-is. Deprecated commands already return notices, scripts not in active paths. | |

**User's choice:** Full removal
**Notes:** Clean break — no dead code anywhere.

| Option | Description | Selected |
|--------|-------------|----------|
| Grep + delete | Grep codebase to confirm nothing references them, then delete. | ✓ |
| Two-phase: manifest then delete | Mark in manifest first, delete in later phase for rollback window. | |

**User's choice:** Grep + delete
**Notes:** Simple verification before removal.

---

## Claude's Discretion

- isTestArtifact fix approach (source field vs other)
- Destructive command safety pattern (--confirm vs audit logging vs both)
- Error message formatting standard details
- Smoke test structure (table-driven vs individual)
- TTY check for blocking agent-initiated destructive commands

## Deferred Ideas

None — discussion stayed within phase scope.
