---
phase: 08-slash-command-wiring
plan: 01
subsystem: cli
tags: [cobra, json, jq, flags, slash-commands]

# Dependency graph
requires:
  - phase: 07-fresh-install-hardening
    provides: Go binary with flag-list, history, phase commands
provides:
  - "--json flag on flag-list, history, phase for jq pipelines"
  - "normalize-args command for YAML wiring preamble replacement"
affects: [08-02, slash-command-generation]

# Tech tracking
tech-stack:
  added: []
  patterns: [json-envelope-flag-pattern, normalize-args-env-precedence]

key-files:
  created: [cmd/normalize_args.go, cmd/normalize_args_test.go]
  modified: [cmd/flags.go, cmd/history.go, cmd/phase.go, cmd/flags_test.go, cmd/history_test.go, cmd/phase_test.go, cmd/testing_main_test.go]

key-decisions:
  - "normalize-args uses outputOK JSON envelope for consistency with all Go commands; YAML generator extracts with jq -r .result"
  - "--json flag pattern: check bool flag before table rendering, output envelope with structured data"

patterns-established:
  - "JSON output flag: add --json bool var, check before table render, use outputOK envelope"
  - "Test globals: all new flag vars must be added to saveGlobals/TestMain in testing_main_test.go"

requirements-completed: [WIRE-01, WIRE-02, WIRE-03]

# Metrics
duration: 8min
completed: 2026-04-04
---

# Phase 08: Slash Command Wiring Summary

**--json flags on flag-list/history/phase for jq pipelines plus normalize-args command replacing shell preamble**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-04T06:29:37Z
- **Completed:** 2026-04-04T06:37:18Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- flag-list --json produces {"ok":true,"result":{"flags":[...]}} parseable by jq
- history --json produces {"ok":true,"result":{"events":[...]}} with parsed pipe-delimited fields
- phase --json produces {"ok":true,"result":{"name","status","description","tasks"}} for any phase number
- normalize-args reads ARGUMENTS env var (Claude Code) with fallback to positional args (OpenCode), collapses whitespace

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Failing tests for --json flags** - `50534a4` (test)
2. **Task 1 GREEN: --json flag implementation** - `830f069` (feat)
3. **Task 2 RED: Failing tests for normalize-args** - `c6c14b4` (test)
4. **Task 2 GREEN: normalize-args implementation** - `b5268f7` (feat)

_Note: TDD tasks have multiple commits (test -> feat)_

## Files Created/Modified
- `cmd/flags.go` - Added flagListJSON var, --json flag registration, JSON output branch before table render
- `cmd/history.go` - Added historyJSON var, --json flag, historyEntry struct for parsed events, JSON output branch
- `cmd/phase.go` - Added phaseJSON var, --json flag, taskEntry struct for tasks, JSON output branch
- `cmd/normalize_args.go` - New command: ARGUMENTS env var precedence, positional fallback, whitespace collapse
- `cmd/flags_test.go` - TestFlagsListJSON, TestFlagsListJSONEmpty tests added
- `cmd/history_test.go` - TestHistoryJSON, TestHistoryJSONEmpty tests added
- `cmd/phase_test.go` - TestPhaseJSON, TestPhaseJSONNotFound, TestPhaseJSONSpecificNumber tests added
- `cmd/normalize_args_test.go` - 5 tests: positional, env var, precedence, empty, whitespace collapse
- `cmd/testing_main_test.go` - Added flagListJSON, historyJSON, phaseJSON, historyLimit, historyFilter, phaseNumber to saveGlobals/TestMain

## Decisions Made
- normalize-args uses outputOK JSON envelope for consistency with all Go commands; YAML generator will extract with `jq -r .result`
- Empty arrays in JSON output use typed nil-slices ([]historyEntry{}, []taskEntry{}) to avoid null serialization
- history --json parses pipe-delimited events into structured fields (timestamp, type, source, message) rather than returning raw strings

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## Next Phase Readiness
- All 4 commands ready for YAML wiring in Plan 02
- flag-list --json, history --json, phase --json produce structured data for jq pipelines
- normalize-args provides consistent argument normalization for both Claude Code and OpenCode command generation

---
*Phase: 08-slash-command-wiring*
*Completed: 2026-04-04*

## Self-Check: PASSED

All 9 files verified present. All 4 commits verified in git log.
