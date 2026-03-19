---
phase: 07-fresh-install-hardening
plan: 02
subsystem: testing
tags: [e2e, bash, smoke-test, fresh-install, lifecycle]

# Dependency graph
requires:
  - phase: 06-xml-exchange-activation
    provides: Working pheromone and session subcommands tested in isolation
provides:
  - End-to-end fresh install smoke test covering install -> lay-eggs -> init -> signals -> build -> continue
  - Validation that QUEEN.md is created with clean template placeholders (no test artifacts)
  - CI-compatible test with --results-file integration
affects: [07-fresh-install-hardening, packaging, ci]

# Tech tracking
tech-stack:
  added: []
  patterns: [HOME-override isolation, hub-to-project file copy simulation, queen-init template verification]

key-files:
  created:
    - tests/e2e/test-fresh-install.sh
  modified: []

key-decisions:
  - "Used HOME override pattern from test-install.sh for true environment isolation"
  - "Handled queen-init nested JSON response (.result.created) which wraps output in {ok:true, result:{...}}"

patterns-established:
  - "Fresh install tests override HOME and run node bin/cli.js install to simulate hub creation"
  - "Lay-eggs simulation copies from $HOME/.aether/system/ to project .aether/ (mirrors real lay-eggs flow)"

requirements-completed: [INST-01]

# Metrics
duration: 3min
completed: 2026-03-19
---

# Phase 7 Plan 2: Fresh Install Smoke Test Summary

**End-to-end smoke test validating the complete install-to-build lifecycle (hub install, lay-eggs, init, pheromone signals, build, continue) in an isolated HOME environment**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-19T22:06:37Z
- **Completed:** 2026-03-19T22:09:45Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Created a 430-line fresh install smoke test that validates the full user journey in an isolated environment
- All 6 lifecycle steps pass: hub install, lay-eggs simulation, colony init, signal flow, build simulation, continue simulation
- Verified QUEEN.md is created from clean template (no test artifact contamination)
- Test integrates with master runner via --results-file flag (writes FRESH_INSTALL=PASS)
- Full test suite (537 tests) passes without regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Create fresh install smoke test script** - `e681908` (test)
2. **Task 2: Verify test integrates with existing test infrastructure** - No changes needed (verification-only task, all checks passed)

## Files Created/Modified
- `tests/e2e/test-fresh-install.sh` - End-to-end smoke test covering 6 lifecycle steps in isolated HOME

## Decisions Made
- Used HOME override pattern (from test-install.sh) rather than just temp directory (from test-lifecycle.sh) for true fresh environment isolation
- Handled queen-init's nested JSON response format -- the subcommand wraps its result in `{ok:true, result:{created:true,...}}` rather than returning `{created:true}` directly

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed queen-init JSON response assertion**
- **Found during:** Task 1 (first test run)
- **Issue:** queen-init wraps its response in `{ok:true, result:{created:true,...}}` -- the plan assumed a flat `{created:true}` response
- **Fix:** Updated jq assertion to check both `.created == true` and `.result.created == true`
- **Files modified:** tests/e2e/test-fresh-install.sh
- **Verification:** All 6 steps pass after fix
- **Committed in:** e681908 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor assertion fix to match actual subcommand response format. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Fresh install lifecycle is now validated end-to-end
- Ready for Phase 8 or further hardening (validate-package.sh content checks covered by plan 07-01)

## Self-Check: PASSED

- FOUND: tests/e2e/test-fresh-install.sh
- FOUND: commit e681908
- FOUND: 07-02-SUMMARY.md

---
*Phase: 07-fresh-install-hardening*
*Completed: 2026-03-19*
