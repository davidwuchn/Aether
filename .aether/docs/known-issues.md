# Known Issues and Workarounds

Documented issues from Oracle research findings. These are known limitations and bugs in the Aether system.

---

## Resolved Issues

### BUG-004: Missing error code in flag-acknowledge
**Location:** `flag-acknowledge` subcommand in `cmd/flag_cmds.go`
**Severity:** MEDIUM
**Status:** RESOLVED -- Go CLI returns proper errors via `flag-acknowledge` command.
Previously the bash CLI had missing error codes; now fully ported to Go with proper error handling.

### BUG-006: No lock release on JSON validation failure
**Location:** `AtomicWrite` in `pkg/storage/storage.go`
**Severity:** MEDIUM
**Status:** RESOLVED -- Go implementation uses `defer s.locker.Unlock(path)` (line 52),
ensuring the lock is always released even on JSON validation failure. The old bash
lock-ownership contract issue no longer applies.

### BUG-007: 17+ instances of missing error codes
**Location:** Various subcommands across the CLI and domain modules
**Severity:** MEDIUM
**Status:** RESOLVED -- Go CLI uses typed errors throughout. All commands return proper
error values; no silent `2>/dev/null` suppression patterns exist in Go code.

### BUG-008: Missing error code in flag-add jq failure
**Location:** `flag-add` subcommand in `cmd/flag_cmds.go`
**Severity:** HIGH
**Status:** RESOLVED -- Go CLI validates flags and returns structured errors. No jq dependency.

### BUG-009: Missing error codes in file checks
**Location:** `flag-acknowledge` and related subcommands in `cmd/flag_cmds.go`
**Severity:** MEDIUM
**Status:** RESOLVED -- Go CLI uses proper file-not-found and validation error handling.

### BUG-010: Missing error codes in context-update
**Location:** `context-update` subcommand (now in Go CLI)
**Severity:** MEDIUM
**Status:** RESOLVED -- Fully ported to Go with proper error paths.

### BUG-012: Missing error code in unknown command
**Location:** Default case in CLI dispatch
**Severity:** LOW
**Status:** RESOLVED -- Go CLI returns clear error for unknown subcommands.

### ISSUE-001: Inconsistent error code usage
**Location:** Multiple locations
**Severity:** MEDIUM
**Status:** RESOLVED -- Go CLI uses typed errors consistently across all commands.

### Historical: ~120 broken positional CLI calls in markdown
**Description:** Markdown playbooks and commands used positional arguments for CLI calls
(e.g., `aether pheromone-write FOCUS "content"`) but the Go CLI expected named flags
(e.g., `aether pheromone-write --type FOCUS --content "content"`). This caused all
pheromone, learning, midden, spawn tracking, activity log, flag, and registry calls
to silently fail.
**Status:** RESOLVED (2026-04-12) -- All ~120 broken CLI calls across markdown files
converted to use correct flag-based syntax. Commands affected:
- `spawn-log` / `spawn-complete` now use `--parent/--caste/--name/--task/--summary` flags
- `activity-log` now uses `--command/--details` flags
- `midden-write` now uses `--category/--message/--source` flags
- `pheromone-write` now uses `--type/--content/--source/--reason/--ttl` flags
- `memory-capture` now uses `--type/--content` flags
- `flag-add` now uses `--severity/--title/--source` flags (plus `--description/--phase`)
- `flag-resolve` / `flag-acknowledge` now use `--id` flag
- `registry-add` now uses correct flag aliases
- `context-update` now uses correct sub-actions
- `session-update` now uses correct flags
- `instinct-create` verified correct

---

## Open Issues

### ISSUE-005: Potential infinite loop in spawn-tree
**Location:** `RecordSpawn` in `pkg/agent/spawn_tree.go`
**Severity:** LOW
**Description:** Edge case with circular parent chain could cause issues
**Mitigation:** Safety limit of 5 exists
**Status:** Open -- low risk due to safety limit

### ISSUE-006: Fallback json_err incompatible
**Location:** Previously in bash CLI fallback error handler
**Severity:** LOW
**Status:** RESOLVED -- No longer applicable; Go CLI does not have a fallback json_err pattern.

---

## Architecture Gaps

### GAP-007: No error code standards documentation
**Description:** Error codes exist but aren't documented for external consumers
**Impact:** Developers don't know which codes to use
**Status:** Partially addressed -- Go CLI uses typed errors, but no standalone error code reference doc exists yet

### GAP-008: Missing error path test coverage
**Description:** Error handling paths not fully tested
**Impact:** Bugs in error handling go undetected
**Status:** Partially addressed -- 2900+ tests pass across 13 packages, but error-specific path coverage remains incomplete

---

*Updated 2026-04-12 during v1.0.0 CLI flag migration colony*
