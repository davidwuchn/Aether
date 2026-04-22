---
phase: 31
plan: 02
subsystem: continue-verification
tags: [verification-truth, bypass-fix, reconcile-hardening]
dependency_graph:
  requires: [31-PLAN-01]
  provides: [continue-verification-truth]
  affects: [cmd/codex_continue.go, cmd/codex_continue_test.go]
tech_stack:
  added: []
  patterns: [unified-watcher-failure, negative-outcome-enumeration, reconcile-warning]
key_files:
  created: []
  modified:
    - cmd/codex_continue.go
    - cmd/codex_continue_test.go
decisions:
  - verified_partial bypass closed — tasks with non-completed workers now return needs_redispatch instead of advancing
  - empty task list returns false instead of claimsSatisfied — no tasks means no work was recorded
  - watcher timeout is a verification failure — removed environmental exception for Codex CLI hangs
  - reconciled tasks do not bypass claim checks — manually_reconciled is now in the negative cases for advancement
  - explicit warning emitted when reconcile is used so users know verification was not bypassed
metrics:
  duration: 763s
  completed: 2026-04-22
  tasks_completed: 3
  files_modified: 2
---

# Phase 31 Plan 02: Continue Verification Truth Summary

Closed four bypass bugs that allowed `aether continue` to advance phases without real verification.

## What Changed

Three targeted fixes in `cmd/codex_continue.go` closed the bypass paths identified in the Oracle audit:

1. **verified_partial bypass (Task 02-01):** Changed the `classifyContinueTaskAssessment` return value from `"verified_partial"` (which let the phase advance) to `"needs_redispatch"` (which blocks it). Also changed `continueTasksSupportAdvancement` to return `false` for empty task lists instead of trusting `claimsSatisfied`.

2. **watcher timeout bypass (Task 02-02):** Removed the `watcher.Status == "timeout"` special case in `runCodexContinueVerification` that kept `checksPassed` true. Now all watcher failures (including timeout) correctly set `checksPassed = false` and append a blocker.

3. **reconcile bypass (Task 02-03):** Removed `|| reconciledTask` from the `taskArtifactEvidenceTrusted` logic so reconciling a task does not automatically mark its artifacts as trusted. Added `"manually_reconciled"` to the negative cases in `continueTasksSupportAdvancement`. Added an explicit warning message when reconcile is used.

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None.

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 02-01 | Close verified_partial bypass | add10b56 | cmd/codex_continue.go, cmd/codex_continue_test.go |
| 02-02 | Close watcher timeout bypass | cb4dfdb1 | cmd/codex_continue.go, cmd/codex_continue_test.go |
| 02-03 | Restore reconcile verification | e6755c44 | cmd/codex_continue.go, cmd/codex_continue_test.go |

## Verification

- `go test ./cmd/... -run "Continue"` passes (all tests)
- `go build ./cmd/aether` succeeds
- Continue with a failed worker but passing build steps does NOT advance
- Continue with a watcher timeout does NOT advance
- Continue with `--reconcile-task` on a task with missing git evidence does NOT advance

## Self-Check: PASSED

All files and commits verified present on disk.
