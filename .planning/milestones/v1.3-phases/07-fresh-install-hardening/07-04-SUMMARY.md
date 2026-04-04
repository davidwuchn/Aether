---
phase: 07-fresh-install-hardening
plan: 04
subsystem: cli
tags: [cobra, learning, instinct, swarm, spawn, storage, eternal]

# Dependency graph
requires:
  - phase: 07-01
    provides: "error-pattern-check command to alias"
  - phase: 07-02
    provides: "swarm-display-text already implemented"
provides:
  - "8 learning pipeline commands (approve/defer/display/extract/inject/promote/select/undo)"
  - "6 infrastructure utility commands (force-unlock, entropy-score, eternal-store, incident-rule-add, bootstrap-system, error-patterns-check)"
  - "6 instinct/spawn/swarm commands (instinct-read, instinct-apply, spawn-get-depth, spawn-can-spawn-swarm, swarm-display-get, swarm-activity-log)"
affects: [learning, instinct, spawn, swarm, colony-state, eternal-memory]

# Tech tracking
tech-stack:
  added: []
  patterns: [cobra-command-registration, json-output-envelope, store-loadJSON, pipe-delimited-parsing]

key-files:
  created:
    - cmd/learning_cmds.go
    - cmd/learning_cmds_test.go
    - cmd/internal_cmds.go
    - cmd/internal_cmds_test.go
  modified: []

key-decisions:
  - "swarm-display-text already existed in cmd/swarm_display.go, so only 20 new commands added instead of 21"
  - "error-patterns-check delegates RunE to existing errorPatternCheckCmd for zero-duplication"
  - "spawn-get-depth uses pipe-delimited format matching SpawnTree.Parse format"
  - "splitCSV and splitPipe helpers avoid importing strings package for simple splits"

patterns-established:
  - "Learning pipeline commands operate on learning-observations.json source_type field for proposal status tracking"
  - "Infrastructure commands resolve data paths via resolveDataDir() and resolveHubPath()"

requirements-completed: [CMD-25, CMD-29, CMD-33, CMD-34, CMD-35, CMD-36, CMD-37, CMD-38, CMD-39, CMD-40, CMD-41, CMD-42, CMD-43, CMD-44, CMD-45]

# Metrics
duration: 16min
completed: 2026-04-04
---

# Phase 07 Plan 04: Learning Pipeline & Internal Utility Commands Summary

**20 new Go commands for learning proposals, infrastructure utilities, instinct/spawn/swarm operations -- completing full command coverage**

## Performance

- **Duration:** 16 min
- **Started:** 2026-04-04T05:03:24Z
- **Completed:** 2026-04-04T05:19:24Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- 8 learning pipeline commands covering full proposal lifecycle (inject, display, select, approve, defer, promote, undo, extract-fallback)
- 6 infrastructure commands for colony maintenance (force-unlock, entropy-score, eternal-store, incident-rule-add, bootstrap-system, error-patterns-check)
- 6 instinct/spawn/swarm commands for colony state introspection (instinct-read, instinct-apply, spawn-get-depth, spawn-can-spawn-swarm, swarm-display-get, swarm-activity-log)

## Task Commits

Each task was committed atomically:

1. **Task 1: Port learning pipeline commands (8 commands)** - `61535217` (feat)
2. **Task 2: Port infrastructure and data utility commands (6 commands)** - `5d0c7179` (feat)
3. **Task 3: Port instinct, spawn, and swarm display commands (6 commands)** - `e30baab2` (feat)

## Files Created/Modified
- `cmd/learning_cmds.go` - 8 learning pipeline commands for proposal management
- `cmd/learning_cmds_test.go` - Tests for learning-inject, display-proposals, promote, approve-proposals
- `cmd/internal_cmds.go` - 12 commands: 6 infrastructure + 6 instinct/spawn/swarm utilities
- `cmd/internal_cmds_test.go` - Tests for force-unlock, entropy-score, eternal-store, bootstrap, instinct-read, spawn-get-depth, swarm-display-get, swarm-activity-log

## Decisions Made
- swarm-display-text already existed in cmd/swarm_display.go from Phase 06, so it was skipped (20 new commands instead of planned 21)
- error-patterns-check delegates RunE to existing errorPatternCheckCmd for zero duplication
- spawn-get-depth parses pipe-delimited spawn-tree.txt format (timestamp|parent|caste|name|task|depth|status)
- Custom splitCSV/splitPipe helpers avoid importing strings for simple splitting

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed duplicate resolveDataDir/resolveAetherRoot functions**
- **Found during:** Task 2 (internal_cmds.go creation)
- **Issue:** resolveDataDir already existed in build_flow_cmds.go with different implementation
- **Fix:** Removed duplicate declarations, used existing resolveDataDir() and storage.ResolveAetherRoot()
- **Files modified:** cmd/internal_cmds.go
- **Verification:** Build passes without redeclaration errors
- **Committed in:** 5d0c7179 (Task 2 commit)

**2. [Rule 3 - Blocking] Fixed spawn-get-depth to use pipe-delimited format**
- **Found during:** Task 3 (spawn-get-depth implementation)
- **Issue:** Initial implementation parsed space-delimited fields; spawn-tree.txt uses pipe-delimited format
- **Fix:** Added splitPipe helper, changed field index from 2 to 3 for agent name, used field 5 for depth
- **Files modified:** cmd/internal_cmds.go
- **Verification:** TestSpawnGetDepth passes with pipe-delimited test data
- **Committed in:** e30baab2 (Task 3 commit)

**3. [Rule 1 - Bug] Fixed swarm-display-text duplication (already existed)**
- **Found during:** Task 3 (planning phase)
- **Issue:** Plan called for creating swarm-display-text but it already existed in cmd/swarm_display.go
- **Fix:** Skipped creating duplicate command; adjusted plan from 21 to 20 new commands
- **Verification:** go run ./cmd/aether swarm-display-text --help works from existing implementation
- **Committed in:** N/A (no code change needed)

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 bug)
**Impact on plan:** All fixes necessary for correctness. swarm-display-text skip reduces total to 20 new commands.

## Issues Encountered
- Test helper naming collision (newTestStore) with existing test helpers in write_cmds_test.go -- resolved by using existing pattern from write_cmds_test.go

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Full command coverage achieved: all shell commands referenced by playbooks now have Go equivalents
- Go binary now has 212+ registered commands (was 192 before Phase 07)
- All Phase 07 plans (01-04) complete

---
*Phase: 07-fresh-install-hardening*
*Completed: 2026-04-04*

## Self-Check: PASSED

All files verified present:
- cmd/learning_cmds.go: FOUND
- cmd/learning_cmds_test.go: FOUND
- cmd/internal_cmds.go: FOUND
- cmd/internal_cmds_test.go: FOUND
- 07-04-SUMMARY.md: FOUND

All commits verified:
- 6153521 (Task 1: learning pipeline commands): FOUND
- 5d0c7179 (Task 2: infrastructure commands): FOUND
- e30baab2 (Task 3: instinct/spawn/swarm commands): FOUND
