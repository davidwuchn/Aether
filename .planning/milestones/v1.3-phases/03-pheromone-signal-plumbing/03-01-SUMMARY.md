---
phase: 03-pheromone-signal-plumbing
plan: 01
subsystem: pheromone
tags: [jq, epoch-conversion, decay-math, pheromone-read, pheromone-expire, ava]

requires:
  - phase: 02-command-audit-data-tooling
    provides: "Clean data files and audited command references"
provides:
  - "Unified to_epoch function across all pheromone subcommands"
  - "10 decay math edge case tests covering signal lifecycle"
  - "Bug fix for active:false signals being re-activated"
affects: [03-02, 03-03, pheromone-expire, pheromone-read]

tech-stack:
  added: []
  patterns:
    - "Consistent to_epoch jq function (365 days/year, 30*86400s/month) across all pheromone subcommands"
    - "Integration test pattern for decay math using temp directory isolation and direct pheromones.json manipulation"

key-files:
  created:
    - tests/integration/pheromone-decay-math.test.js
  modified:
    - .aether/aether-utils.sh

key-decisions:
  - "Replaced approx_epoch (365.25 days/year) with to_epoch (365 days/year) for consistency over precision"
  - "Fixed jq // operator treating active:false as null -- used explicit if/elif chain instead"

patterns-established:
  - "createTestSignal helper for writing controlled pheromone signals directly to pheromones.json"
  - "readActiveSignals helper for calling pheromone-read and parsing JSON result"

requirements-completed: [PHER-06, PHER-02]

duration: 4min
completed: 2026-03-19
---

# Phase 03 Plan 01: Epoch Unification & Decay Math Tests Summary

**Unified dual epoch conversion functions in aether-utils.sh and added 10 decay math edge case tests covering signal lifecycle from fresh to fully decayed**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-19T17:36:35Z
- **Completed:** 2026-03-19T17:41:09Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Eliminated the `approx_epoch` function (365.25 days/year constants) from pheromone-expire, replacing it with the standard `to_epoch` function (365 days/year constants) used by all other pheromone subcommands
- Created 10 comprehensive decay math tests covering zero elapsed, half-life, full decay, past-decay, missing strength fallback, phase_end expiry, past/future ISO expiry, pre-deactivated signals, and REDIRECT type-specific decay
- Fixed a bug where signals with `active: false` were incorrectly re-activated by pheromone-read due to jq's `//` operator treating `false` as null

## Task Commits

Each task was committed atomically:

1. **Task 1: Unify epoch conversion in pheromone-expire to use to_epoch** - `1c4bcd2` (fix)
2. **Task 2: Write decay math edge case tests** - `5f888c0` (test)

## Files Created/Modified
- `tests/integration/pheromone-decay-math.test.js` - 10 edge case tests for pheromone-read decay math (407 lines)
- `.aether/aether-utils.sh` - Replaced approx_epoch with to_epoch in pheromone-expire; fixed active:false re-activation bug in pheromone-read

## Decisions Made
- Replaced `approx_epoch` (using 31557600s/year = 365.25 days, 2629800s/month) with `to_epoch` (using 365*86400s/year, 30*86400s/month) -- consistency across subcommands matters more than astronomical precision for signal TTLs
- Fixed the `(.active // true)` jq expression to `if .active == false then false else true end` because jq's `//` (alternative operator) treats `false` as "no value" and falls through to `true`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed active:false signals being re-activated by pheromone-read**
- **Found during:** Task 2 (decay math tests, test case 9)
- **Issue:** jq's `//` operator in `(.active // true)` treats `false` as null/empty, so signals explicitly set to `active: false` were returned as active by pheromone-read
- **Fix:** Changed to explicit `if .active == false then false else true end` chain
- **Files modified:** `.aether/aether-utils.sh` (pheromone-read jq pipeline, line 7172)
- **Verification:** Test 9 ("Signal with active: false") now passes
- **Committed in:** `5f888c0` (part of Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Bug fix was necessary for correctness -- without it, deactivated signals would reappear in worker context. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Epoch conversion is now unified -- all pheromone subcommands use identical `to_epoch` with same constants
- Decay math is tested for all edge cases identified in research
- Ready for 03-02 (pheromone-read integration into build/continue playbooks)

## Self-Check: PASSED

- FOUND: .aether/aether-utils.sh
- FOUND: tests/integration/pheromone-decay-math.test.js
- FOUND: 03-01-SUMMARY.md
- FOUND: commit 1c4bcd2
- FOUND: commit 5f888c0

---
*Phase: 03-pheromone-signal-plumbing*
*Completed: 2026-03-19*
