---
phase: 07-fresh-install-hardening
plan: 02
subsystem: cli
tags: [cobra, colony-state, random, caching, progress-bar]

# Dependency graph
requires:
  - phase: 07-01
    provides: "error commands pattern established"
provides:
  - "generate-ant-name command with 20 caste prefix sets and deterministic seed"
  - "generate-commit-message command with type/scope/subject/body flags"
  - "generate-progress-bar and generate-threshold-bar commands"
  - "version-check-cached command with 24h file cache"
  - "milestone-detect command with auto-detection from phase progress"
  - "update-progress command for phase and task status mutations"
  - "print-next-up command with state-based suggestions"
  - "data-safety-stats command for data directory integrity reporting"
affects: [07-03, 07-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "resolveDataDir() helper for AETHER_ROOT-aware data directory resolution"
    - "versionCacheEntry struct for JSON-cached version checks"
    - "castePrefixes map with 20 caste entries matching shell implementation"

key-files:
  created:
    - cmd/generate_cmds.go
    - cmd/generate_cmds_test.go
    - cmd/build_flow_cmds.go
    - cmd/build_flow_cmds_test.go
  modified: []

key-decisions:
  - "Used local rand.Rand instances per command invocation to support deterministic --seed flag"
  - "Auto-detect milestone from phase completion ratio when milestone field is empty"
  - "resolveDataDir duplicates pkg/storage logic to avoid import cycles for commands that bypass store init"

patterns-established:
  - "Caste prefix map: 20 castes with 8 prefixes each, matching shell format exactly"
  - "Version caching: file-based JSON cache with 24h TTL"
  - "Progress bar: width parameterized, percentage calculated from integer division"

requirements-completed:
  - CMD-01
  - CMD-02
  - CMD-03
  - CMD-04
  - CMD-05
  - CMD-06
  - CMD-07
  - CMD-08
  - DIFF-01

# Metrics
duration: 2min
completed: 2026-04-04
---

# Phase 07 Plan 02: Build Flow Utility Commands Summary

**8 CLI commands ported from shell to Go: ant name generation with 20 castes, commit message formatting, progress/threshold bars, version caching, milestone detection, progress updates, next-step display, and data safety stats**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-04T04:27:59Z
- **Completed:** 2026-04-04T04:30:01Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- generate-ant-name produces caste-specific names matching shell format exactly (20 castes, 8 prefixes each)
- generate-commit-message validates commit types against allowed values and formats with optional scope/body
- generate-progress-bar and generate-threshold-bar render terminal bar visualizations
- version-check-cached implements 24h file-based version caching
- milestone-detect auto-detects milestone from phase completion ratio when field is empty
- update-progress supports both phase-level and task-level status updates with validation
- print-next-up provides contextual suggestions based on colony state machine
- data-safety-stats reports on data directory integrity (files, locks, pheromones, sessions)

## Task Commits

Each task was committed atomically:

1. **Task 1: Port generate-ant-name, generate-commit-message, generate-progress-bar to Go** - `009a2bb` (feat)
2. **Task 2: Port version-check-cached, milestone-detect, update-progress, print-next-up, data-safety-stats to Go** - `8477e2c` (feat)

## Files Created/Modified
- `cmd/generate_cmds.go` - 3 generate commands + bonus generate-threshold-bar (4 total)
- `cmd/generate_cmds_test.go` - 15 test functions covering all generate commands
- `cmd/build_flow_cmds.go` - 5 build flow commands with helper functions
- `cmd/build_flow_cmds_test.go` - 12 test functions covering all build flow commands

## Decisions Made
- Used local `rand.Rand` instances per invocation to support `--seed` flag for deterministic output (per DIFF-01 requirement)
- Auto-detect milestone from phase completion ratio using thresholds: 25% First Mound, 50% Open Chambers, 75% Brood Stable, 100% Sealed Chambers
- `resolveDataDir()` helper duplicates storage path resolution to avoid import cycles for commands that bypass store initialization
- Bonus `generate-threshold-bar` command included alongside the 3 required generate commands

## Deviations from Plan

None - plan executed exactly as written. Task 1 was already committed by a prior agent run.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 8 new commands registered in Go binary, ready for wiring phases
- generate-ant-name is called 11 times across slash commands for worker spawning
- update-progress is called on every build cycle
- No blockers for subsequent plans

## Self-Check: PASSED

- All 4 files verified: cmd/generate_cmds.go, cmd/generate_cmds_test.go, cmd/build_flow_cmds.go, cmd/build_flow_cmds_test.go
- Both commits verified: 009a2bb (Task 1), 8477e2c (Task 2)
- All tests passing: go test ./cmd/... -count=1

---
*Phase: 07-fresh-install-hardening*
*Completed: 2026-04-04*
