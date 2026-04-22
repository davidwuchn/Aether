---
phase: 32-continue-unblock
plan: 02
subsystem: runtime

tags: [go, tdd, stale-cleanup, continue-pipeline, e2e-recovery]

requires:
  - phase: 32-01
    provides: detectAbandonedBuild, abandoned result branch with recovery commands

provides:
  - cleanupStaleContinueReports function that removes stale report files before verification
  - Integration test proving full recovery pipeline: abandoned detection -> re-dispatch -> verify -> advance
  - Best-effort cleanup that does not block continue on errors

affects:
  - 32-03 (continue gate hardening)
  - 33-dispatch-fixes

tech-stack:
  added: []
  patterns:
    - "Pre-verification cleanup: remove stale artifacts before running expensive checks"
    - "Best-effort cleanup: errors logged but not blocking"
    - "TDD RED/GREEN for runtime behavior changes"

key-files:
  created: []
  modified:
    - cmd/codex_continue.go - cleanupStaleContinueReports function, wired before verification
    - cmd/codex_continue_test.go - TestContinueClearsStaleReports, TestContinueEndToEndAfterAbandonedRecovery

key-decisions:
  - "Use os.Remove with store.BasePath() for cleanup instead of store.AtomicWrite, since deletion is the intent"
  - "Cleanup runs after abandoned-build check but before verification, so abandoned results still get their continue.json written"
  - "Test design: verify stale review.json is removed when continue is blocked at gates (the only path where review.json is not overwritten)"

patterns-established:
  - "Stale artifact cleanup before verification: prevents confusing users with leftover files from previous runs"
  - "E2E recovery test pattern: abandoned -> re-seed -> advance proves the full pipeline"

requirements-completed: [REQ-3, REQ-5, REQ-6]

# Metrics
duration: 18m
completed: 2026-04-22
---

# Phase 32 Plan 02: Stale Report Cleanup and E2E Recovery Summary

**Continue pipeline now cleans stale report artifacts before verification, and the full recovery path from abandoned build detection through re-dispatch to phase advancement is proven end-to-end**

## Performance

- **Duration:** 18 min
- **Started:** 2026-04-22T23:30:00Z
- **Completed:** 2026-04-22T23:48:00Z
- **Tasks:** 2 (TDD RED + GREEN)
- **Files modified:** 2

## Accomplishments

- `cleanupStaleContinueReports` removes verification.json, gates.json, continue.json, and review.json from the current phase's build directory before verification runs
- Cleanup is best-effort: `os.Remove` errors are silently ignored, continue proceeds regardless
- Cleanup runs after abandoned-build detection but before verification, preserving the abandoned result path
- `TestContinueClearsStaleReports` proves stale review.json is removed when continue is blocked at gates (the only path where review.json would not be overwritten by the current run)
- `TestContinueEndToEndAfterAbandonedRecovery` proves the full pipeline: abandoned detection -> re-dispatch (seed completed dispatches) -> verify -> advance

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing tests for stale cleanup and E2E recovery** - `27401db1` (test)
2. **Task 2: Implement stale report cleanup and verify full pipeline** - `e96184da` (feat)

## Files Created/Modified

- `cmd/codex_continue.go` - Added `cleanupStaleContinueReports` function; wired call before `runCodexContinueVerification`
- `cmd/codex_continue_test.go` - Added `TestContinueClearsStaleReports` (stale removal when blocked at gates) and `TestContinueEndToEndAfterAbandonedRecovery` (full pipeline)

## Decisions Made

- Used `os.Remove` with `store.BasePath()` for file deletion rather than `store.AtomicWrite` with empty content, since the intent is removal not zeroing
- Cleanup is placed after the abandoned-build early-exit check so abandoned results still write their `continue.json` for audit
- Test design targets the specific case where stale data persists: `review.json` from a previous run that reached review, when the current run gets blocked at gates (review stage never runs)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test passed unexpectedly during RED phase because continue naturally overwrites report files**
- **Found during:** Task 1 (test writing)
- **Issue:** Initial test design checked that stale markers were gone from all 4 report files, but continue already overwrites verification.json, gates.json, continue.json, and review.json during normal execution, so the test passed without any cleanup function existing
- **Fix:** Redesigned `TestContinueClearsStaleReports` to target the specific leak case: create stale `review.json`, make verification fail (so continue is blocked at gates and never reaches review), assert the stale `review.json` is removed. This is the only path where a stale file from a previous run would persist without cleanup
- **Files modified:** `cmd/codex_continue_test.go`
- **Verification:** Test fails without `cleanupStaleContinueReports`, passes with it
- **Committed in:** `27401db1` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Test redesign necessary for correct TDD RED phase. No scope creep.

## Issues Encountered

- Initial test design did not properly isolate the cleanup behavior because continue already overwrites its report files. Fixed by targeting the review.json leak case (blocked at gates -> review never runs -> stale review.json persists).

## Known Stubs

None — all behavior is fully wired and tested.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: tampering | cmd/codex_continue.go:cleanupStaleContinueReports | Only removes report files in the current phase's build directory; reports are regenerated during the continue run. Disposition: accept (per threat model T-32-03). |
| threat_flag: denial-of-service | cmd/codex_continue.go:cleanupStaleContinueReports | Cleanup errors are silently ignored; continue proceeds regardless. Disposition: accept (per threat model T-32-04). |

## Next Phase Readiness

- Continue pipeline now has clean pre-verification state: stale reports are removed before fresh verification runs
- Full recovery pipeline is proven: abandoned detection -> re-dispatch -> verify -> advance
- Ready for next continue unblock plan (32-03)

## Self-Check: PASSED

- [x] SUMMARY.md created at `.planning/phases/32-continue-unblock/32-02-SUMMARY.md`
- [x] RED commit `27401db1` exists (test: add failing tests for stale report cleanup and E2E recovery)
- [x] GREEN commit `e96184da` exists (feat: implement stale report cleanup before continue verification)
- [x] All continue tests pass (`go test ./cmd/... -run TestContinue -count=1 -timeout 120s`)
- [x] Full cmd test suite passes (`go test ./cmd/... -count=1 -timeout 180s`)
- [x] No bypass paths introduced (cleanup is best-effort, does not affect gating logic)

---
*Phase: 32-continue-unblock*
*Completed: 2026-04-22*
