---
phase: 06-branching-worktree-discipline
plan: 01
subsystem: infra
tags: [git, worktree, branch-naming, lifecycle, cli]

requires: []
provides:
  - WorktreeEntry struct with lifecycle tracking in COLONY_STATE.json
  - validateBranchName with two-track naming (agent + human)
  - worktree-allocate command with git worktree creation and state registration
  - worktree-list command with cross-reference to git worktree list
  - worktree-orphan-scan command with configurable staleness threshold
  - sanitizeBranchPath for filesystem-safe directory names
affects: [06-02, merge-protocol]

tech-stack:
  added: []
  patterns: [two-track-branch-naming, lifecycle-state-machine, audit-logging]

key-files:
  created:
    - cmd/worktree.go
    - cmd/worktree_test.go
  modified:
    - pkg/colony/colony.go

key-decisions:
  - "Two-track naming: agent (phase-N/caste-name) and human (feature|fix|experiment|colony/name) branches"
  - "WorktreeEntry tracked in COLONY_STATE.json with allocated/in-progress/merged/orphaned lifecycle"
  - "48-hour default threshold for orphan detection"

patterns-established:
  - "Two-track branch naming: agent (phase-N/caste-name) vs human (prefix/name) with strict validation"
  - "Worktree lifecycle state machine: allocated -> in-progress -> merged (or orphaned)"
  - "Audit logging via AppendJSONL for all worktree mutations"

requirements-completed: [BRAN-01, BRAN-02, BRAN-03, BRAN-06]

duration: 8min
completed: 2026-04-08
---

# Plan 06-01: Worktree Allocation & Lifecycle Summary

**Worktree allocation, listing, and orphan detection with two-track branch naming enforcement and COLONY_STATE.json lifecycle tracking**

## Performance

- **Duration:** 8 min
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- WorktreeEntry struct with ID, Branch, Path, Status, Phase, Agent, timestamps in pkg/colony
- Three CLI commands: worktree-allocate (with branch validation + git creation), worktree-list (with on-disk cross-reference), worktree-orphan-scan (48h threshold)
- Comprehensive test coverage: 33 tests including validation, JSON round-trips, edge cases, ID uniqueness

## Task Commits

1. **Task 1: WorktreeEntry type and allocate/list/orphan-scan** - `f0eb77da` (test), `f89b8fbd` (feat), `31850ef8` (fix)
2. **Task 2: Edge case tests and full coverage** - `d2761610` (test)

## Files Created/Modified
- `cmd/worktree.go` - Three new CLI commands with branch naming validation, git worktree management, state tracking, and audit logging (489 lines)
- `cmd/worktree_test.go` - 33 tests covering validation, JSON round-trips, command behavior, edge cases, and ID uniqueness (1127 lines)
- `pkg/colony/colony.go` - WorktreeEntry struct, WorktreeStatus constants, Worktrees field on ColonyState

## Decisions Made
- Used constant worktreeBaseDir (".aether/worktrees/") with sanitized branch paths for safety
- validateBranchName rejects ".." (path traversal) and enforces strict format matching
- generateWorktreeID uses unix timestamp + 4 random bytes for uniqueness

## Deviations from Plan
None - plan executed as specified

## Issues Encountered
- Helper function collision with existing cmd/helpers.go required fix commit
- Store init behavior needed adjustment in tests

## Next Phase Readiness
- WorktreeEntry type and lifecycle states available for merge-back command
- State tracking infrastructure ready for Plan 02 (merge protocol)
- Branch validation can be reused by merge-back

---
*Plan: 06-01*
*Completed: 2026-04-08*
