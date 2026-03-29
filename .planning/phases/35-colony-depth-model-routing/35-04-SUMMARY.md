---
phase: 35-colony-depth-model-routing
plan: 04
subsystem: infra
tags: [colony-depth, build-playbooks, agent-gating, status-dashboard]

requires:
  - phase: 35-01
    provides: "colony-depth get/set API in queen.sh and aether-utils.sh"
  - phase: 35-03
    provides: "--depth flag parsing in build-prep.md and build-full.md"
provides:
  - "Depth-gated agent spawns in build playbooks (Oracle, Architect, Scout, Chaos, Archaeologist)"
  - "Colony depth display in /ant:status dashboard"
  - "colony_depth cross-stage state variable in build pipeline"
affects: [build-wave, build-verify, build-context, build-prep, status]

tech-stack:
  added: []
  patterns: ["DEPTH CHECK gate pattern at top of agent spawn steps", "colony-depth get with graceful fallback to standard"]

key-files:
  created: []
  modified:
    - ".aether/docs/command-playbooks/build-wave.md"
    - ".aether/docs/command-playbooks/build-verify.md"
    - ".aether/docs/command-playbooks/build-context.md"
    - ".aether/docs/command-playbooks/build-prep.md"
    - ".aether/docs/command-playbooks/build-full.md"
    - ".claude/commands/ant/status.md"
    - ".claude/commands/ant/build.md"

key-decisions:
  - "Used DEPTH CHECK pattern as a guard clause at the top of each gated spawn step for consistency"
  - "Inserted depth display in status.md as Step 2.5.5 to avoid renumbering existing non-sequential steps"
  - "Depth read uses graceful fallback to standard when colony-depth get fails"

patterns-established:
  - "DEPTH CHECK: Guard clause pattern at top of agent spawn sections -- check colony_depth before proceeding"
  - "Depth label mapping: light/standard/deep/full with human-readable descriptions"

requirements-completed: [INFRA-01]

duration: 6min
completed: 2026-03-29
---

# Phase 35 Plan 04: Depth Gating in Build Playbooks Summary

**Colony depth enforcement wired into all build playbooks -- Oracle/Architect gated to deep+, Scout blocked at light, Chaos gated to full, Archaeologist skipped at light, depth displayed in /ant:status**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-29T09:46:22Z
- **Completed:** 2026-03-29T09:53:03Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Oracle and Architect spawns gated to deep/full depth only via DEPTH CHECK guards
- Scout caste assignment blocked at light depth (reassigns to Builder or skips)
- Chaos spawn gated to full depth only
- Archaeologist pre-build scan skipped at light depth
- Colony depth read and displayed during build prep with human-readable labels
- Spawn plan display in build-wave.md dynamically reflects depth-conditional agents
- /ant:status dashboard shows current depth with label and "(default)" indicator
- colony_depth added as cross-stage state variable in build.md orchestrator

## Task Commits

Each task was committed atomically:

1. **Task 1: Add depth gating to build playbooks** - `34c5464` (feat)
2. **Task 2: Add depth display to /ant:status dashboard** - `6d5ea41` (feat)

## Files Created/Modified
- `.aether/docs/command-playbooks/build-wave.md` - DEPTH CHECK gates on Oracle (Step 5.0.1), Architect (Step 5.0.2), Scout caste assignment, spawn plan display
- `.aether/docs/command-playbooks/build-verify.md` - DEPTH CHECK gate on Chaos (Step 5.6)
- `.aether/docs/command-playbooks/build-context.md` - DEPTH CHECK gate on Archaeologist (Step 4.1)
- `.aether/docs/command-playbooks/build-prep.md` - Depth read step with display and cross-stage state
- `.aether/docs/command-playbooks/build-full.md` - Same depth read step (mirror of build-prep.md)
- `.claude/commands/ant/status.md` - Step 2.5.5 Colony Depth with dashboard display line
- `.claude/commands/ant/build.md` - colony_depth added to cross-stage state list

## Decisions Made
- Used DEPTH CHECK as a consistent guard clause pattern at the top of each gated spawn step, making it easy to find and audit
- Inserted depth display as Step 2.5.5 in status.md to avoid renumbering the existing non-sequential step numbering (2.4, 2.5, 2.6, 2.8, 2.7)
- Depth read uses graceful fallback defaulting to "standard" when colony-depth get fails, ensuring builds never break due to missing depth configuration

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Colony depth is now fully functional: get/set API (Plan 01), --depth flag (Plan 03), enforcement in build playbooks (this plan), status display (this plan)
- Depth enforcement is additive -- existing builds continue working at "standard" depth with no changes required

## Self-Check: PASSED

All files exist, all commits verified.

---
*Phase: 35-colony-depth-model-routing*
*Completed: 2026-03-29*
