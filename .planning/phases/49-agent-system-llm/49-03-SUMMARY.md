---
phase: 49-agent-system-llm
plan: 03
subsystem: agent-runtime
tags: [errgroup, goroutine-pool, spawn-tree, pipe-delimited, shell-parity]

# Dependency graph
requires:
  - phase: 49-01
    provides: "Agent interface, Registry, Match method, Caste types"
  - phase: 46
    provides: "Event bus with Subscribe/Unsubscribe/Publish"
  - phase: 45
    provides: "Storage layer with AtomicWrite, ReadFile"
provides:
  - "Worker pool with errgroup.SetLimit bounded concurrency"
  - "Spawn tree tracking with 7-field pipe-delimited format matching shell"
  - "Pool subscribes to all events and dispatches to matching agents"
  - "Spawn tree parses shell-written files with completion status merging"
  - "ToJSON output matching shell parse_spawn_tree format"
affects: [49-04, agent-execution, colony-runtime]

# Tech tracking
tech-stack:
  added: [golang.org/x/sync/errgroup]
  patterns: [bounded-goroutine-pool, pipe-delimited-serialization, shell-format-parity]

key-files:
  created:
    - pkg/agent/pool.go
    - pkg/agent/pool_test.go
    - pkg/agent/spawn_tree.go
    - pkg/agent/spawn_tree_test.go
  modified:
    - pkg/agent/pool.go

key-decisions:
  - "Pool uses mutex to protect cancel/eventCh fields for concurrent Start/Stop safety"
  - "Spawn tree stores both spawn entries and completion lines, merging on parse"
  - "Completion line format uses 4 pipe-separated fields (timestamp|name|status|) matching shell awk rule"

patterns-established:
  - "errgroup.SetLimit for bounded goroutine pools with per-event agent dispatch"
  - "Mutex-protected pool fields for concurrent Start/Stop access"
  - "Pipe-delimited format with 7 fields for Go/shell spawn tree parity"

requirements-completed: [AGENT-02, AGENT-03]

# Metrics
duration: 6min
completed: 2026-04-02
---

# Phase 49 Plan 03: Worker Pool and Spawn Tree Summary

**Worker pool with errgroup.SetLimit bounded concurrency and spawn tree tracking with 7-field pipe-delimited shell format parity**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-02T03:59:32Z
- **Completed:** 2026-04-02T04:05:49Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Worker pool dispatches events to matching agents with bounded goroutines via errgroup.SetLimit
- Race condition in Pool.Start/Stop fixed -- cancel and eventCh fields protected by mutex
- Spawn tree records entries with 7 pipe-delimited fields matching shell spawn-tree.txt format
- Shell-written spawn-tree.txt parses correctly in Go with completion status merging
- All 17 tests pass with race detector enabled

## Task Commits

Each task was committed atomically:

1. **Task 1: Worker pool race fix** - `8df2a30` (fix)
2. **Task 2: Spawn tree tracking** - `cb1c090` (feat)

## Files Created/Modified
- `pkg/agent/pool.go` - Worker pool with errgroup bounded concurrency, mutex-protected Start/Stop
- `pkg/agent/pool_test.go` - 9 tests: New, concurrency, nil args, dispatch, multiple agents, bounded, stop, no-match
- `pkg/agent/spawn_tree.go` - Spawn tree with RecordSpawn, UpdateStatus, Parse, Persist, Active, ToJSON
- `pkg/agent/spawn_tree_test.go` - 8 tests: record, update, format, round-trip, shell format, active, JSON, empty

## Decisions Made
- Pool mutex protects cancel/eventCh to prevent data race between concurrent Start() and Stop() calls
- Spawn tree stores both spawn entries and completion lines separately, merging on parse for round-trip correctness
- Completion line format: `timestamp|name|status|` (4 fields) matching shell awk NF>=4 rule

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed data race in Pool.Start/Stop**
- **Found during:** Task 1 (verification -- race detector)
- **Issue:** `cancel` and `eventCh` fields written in Start() and read in Stop() without synchronization
- **Fix:** Both methods now acquire pool mutex before accessing shared fields
- **Files modified:** pkg/agent/pool.go
- **Verification:** `go test ./pkg/agent/ -race -count=1` passes clean
- **Committed in:** 8df2a30

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Essential correctness fix for concurrent access. No scope creep.

## Issues Encountered
- go.sum was missing errgroup entry; resolved with `go mod tidy`

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Pool and spawn tree ready for Plan 49-04 (LLM integration)
- Pool provides event-driven agent dispatch with bounded concurrency
- Spawn tree provides format-compatible tracking for Go/shell coexistence

## Self-Check: PASSED

- All 4 created/modified files verified present
- Both task commits (8df2a30, cb1c090) verified in git log
- All tests pass with race detector: `go test ./pkg/agent/ -race -count=1`
- `go vet ./pkg/agent/` clean

---
*Phase: 49-agent-system-llm*
*Completed: 2026-04-02*
