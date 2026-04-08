---
phase: 03-build-depth-controls
plan: 03
subsystem: build-orchestration
tags: [depth, context-budget, playbooks, measurer, ambassador]

# Dependency graph
requires:
  - phase: 03-01
    provides: "context-budget subcommand producing JSON with depth-scaled budgets"
provides:
  - "Depth-aware context budgets in build-context.md and build-wave.md"
  - "Measurer depth gating (deep/full only) in build-verify.md"
  - "Ambassador depth gating (deep/full only) in build-wave.md"
  - "Builder count limits by depth level in build-wave.md"
  - "Corrected depth label descriptions in build-prep.md"
affects: [build-playbooks, depth-controls, specialist-gating]

# Tech tracking
tech-stack:
  added: []
  patterns: ["depth-based budget scaling via aether context-budget subcommand"]

key-files:
  created: []
  modified:
    - .aether/docs/command-playbooks/build-context.md
    - .aether/docs/command-playbooks/build-prep.md
    - .aether/docs/command-playbooks/build-wave.md
    - .aether/docs/command-playbooks/build-verify.md

key-decisions:
  - "Archaeology cap scales to half of context budget (depth-proportional)"
  - "Midden and graveyard caps remain fixed at 2000 chars (not depth-scaled)"
  - "Builder count limits: light=1, standard=2, deep/full=unlimited"

patterns-established:
  - "context-budget call pattern: command with jq parse, fallback to 8000 default"

requirements-completed: [DEPTH-02, DEPTH-04, DEPTH-05]

# Metrics
duration: 1min
completed: 2026-04-07
---

# Phase 03 Plan 03: Wire context-budget and specialist gating Summary

**Depth-aware token budgets via context-budget subcommand with Measurer/Ambassador gating and builder count limits**

## Performance

- **Duration:** 1 min
- **Started:** 2026-04-07T19:13:59Z
- **Completed:** 2026-04-07T19:15:46Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Replaced all hardcoded budget values (8000, 4000) in playbooks with `aether context-budget --depth "$colony_depth"` calls
- Added Measurer depth gating: skips at light/standard, runs at deep/full
- Added Ambassador depth gating: skips at light/standard, runs at deep/full
- Added builder count limits: light=1, standard=2, deep/full=unlimited
- Fixed depth label descriptions in build-prep.md to match corrected status.go values

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire context-budget into build-context.md and build-prep.md** - `09ef07c1` (feat)
2. **Task 2: Wire context-budget into build-wave.md and add measurer/ambassador depth gating** - `c1733f55` (feat)

## Files Created/Modified
- `.aether/docs/command-playbooks/build-context.md` - Replaced hardcoded 8000 research budget with context-budget call
- `.aether/docs/command-playbooks/build-prep.md` - Fixed depth label descriptions (standard includes Watcher, deep excludes Chaos)
- `.aether/docs/command-playbooks/build-wave.md` - Added depth-based archaeology cap, builder count limits, Ambassador depth gating
- `.aether/docs/command-playbooks/build-verify.md` - Added Measurer depth gating (deep/full only)

## Decisions Made
- Archaeology cap scales to half of context budget rather than a fixed value -- this keeps the proportional relationship as budgets grow
- Midden (2000 chars) and graveyard (2000 chars per worker) caps remain fixed -- they are not context budgets and don't need depth scaling
- Builder count limits use 3 tiers: light=1 (fastest), standard=2 (balanced), deep/full=unlimited (thorough)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 3 plans in Phase 03 (build-depth-controls) are now complete
- Build playbooks fully depth-aware: budgets scale, specialists gate correctly, builder counts limit
- Ready for Phase 04 transition

---
*Phase: 03-build-depth-controls*
*Completed: 2026-04-07*
