---
phase: 03-build-depth-controls
plan: 02
subsystem: cli-validation
tags: [go, cobra, colony-depth, validation, tdd]

# Dependency graph
requires:
  - phase: 03-01
    provides: ColonyDepth enum with Valid() method and ErrInvalidDepth
provides:
  - Depth validation in state-mutate field mode (ColonyDepth.Valid() check)
  - Depth validation in state-mutate expression mode (post-mutation unmarshal + Valid())
  - --depth flag on init command with validation and explicit "standard" default
affects: [03-03, any future plan that reads/writes colony_depth]

# Tech tracking
tech-stack:
  added: []
  patterns: [typed-field-post-mutation-validation]

key-files:
  created: []
  modified:
    - cmd/state_cmds.go
    - cmd/state_cmds_test.go
    - cmd/init_cmd.go
    - cmd/init_cmd_test.go

key-decisions:
  - "Expression mode validates by unmarshaling mutated data before AtomicWrite, preventing invalid values from ever reaching disk"
  - "Init --depth defaults explicitly to DepthStandard (not empty) per D-06"
  - "Validation errors use outputError with clear message listing all valid values"

patterns-established:
  - "Post-mutation typed field validation: after raw byte manipulation, unmarshal into typed struct and validate before persisting"

requirements-completed: [DEPTH-01, DEPTH-08]

# Metrics
duration: 4min
completed: 2026-04-07
---

# Phase 03 Plan 02: Depth Validation + Init Flag Summary

**ColonyDepth validation on state-mutate (field + expression modes) and --depth flag on init with explicit "standard" default**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-07T19:06:32Z
- **Completed:** 2026-04-07T19:10:54Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- state-mutate field mode rejects invalid depth values (e.g. "banana") with ColonyDepth.Valid()
- state-mutate expression mode validates after mutation but before disk write, preventing persistence
- init --depth flag accepts light|standard|deep|full with validation before state creation
- init defaults to "standard" depth when --depth flag is omitted (explicit default per D-06)
- 11 new tests covering all depth validation paths

## Task Commits

Each task was committed atomically:

1. **Task 1: Add depth validation to state-mutate (field mode and expression mode)** - `f11f6c05` (test RED), `30b2616c` (feat GREEN)
2. **Task 2: Add --depth flag to init command** - `63053346` (test RED), `946d3106` (feat GREEN)

_Note: TDD tasks each have test commit followed by implementation commit._

## Files Created/Modified
- `cmd/state_cmds.go` - Added ColonyDepth.Valid() check in field mode switch case; added post-mutation typed field validation in expression mode before AtomicWrite
- `cmd/state_cmds_test.go` - 5 new tests: field valid (light, standard), field invalid (banana), expression valid (deep), expression invalid
- `cmd/init_cmd.go` - Added --depth flag with StringVar, depth validation before state creation, ColonyDepth field in state struct, explicit DepthStandard default
- `cmd/init_cmd_test.go` - 6 new tests: all four valid depths, default behavior, invalid depth rejection

## Decisions Made
- Expression mode validates by unmarshaling the mutated raw bytes into a ColonyState struct before calling AtomicWrite. This catches invalid typed fields without needing to parse the expression itself.
- Init defaults to DepthStandard explicitly (not empty string) so downstream code never sees an unset depth.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing test failure in `pkg/exchange` (TestImportPheromonesFromRealShellXML) confirmed unrelated to this plan's changes. No action taken per scope boundary rules.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Depth validation is fully in place for both state-mutate and init paths
- Plan 03-03 can rely on depth always being valid when present in COLONY_STATE.json
- The post-mutation validation pattern established here can be extended to other typed fields if needed

---
*Phase: 03-build-depth-controls*
*Completed: 2026-04-07*

## Self-Check: PASSED

All 5 files found. All 4 commits verified.
