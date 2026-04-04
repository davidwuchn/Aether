---
phase: 03-pheromone-signal-plumbing
plan: 02
subsystem: commands
tags: [pheromone, session-persistence, resume, decay, yaml-generation]

# Dependency graph
requires:
  - phase: 03-01
    provides: pheromone-read subcommand with decay calculation
provides:
  - resume.md Step 3 reads pheromones.json via pheromone-read
  - Signal rendering with decay-aware effective_strength percentage
  - Unified YAML source for both Claude and OpenCode providers
affects: [resume, session-persistence, pheromone-display]

# Tech tracking
tech-stack:
  added: []
  patterns: [pheromone-read for signal retrieval, TOOL_PREFIX macro for provider-neutral bash calls]

key-files:
  created: []
  modified:
    - .aether/commands/resume.yaml
    - .claude/commands/ant/resume.md
    - .opencode/commands/ant/resume.md

key-decisions:
  - "Unified Step 3 across both providers (removed {{#claude}}/{{#opencode}} blocks) since pheromone-read is provider-agnostic"

patterns-established:
  - "Commands reading pheromone signals should use pheromone-read subcommand, not direct file access"
  - "YAML source changes require regeneration via npm run generate"

requirements-completed: [PHER-07]

# Metrics
duration: 4min
completed: 2026-04-03
---

# Phase 3 Plan 2: Resume Pheromone Fix Summary

**Fixed /ant:resume to read pheromones.json via pheromone-read with decay-aware display, replacing broken constraints.json read and false no-decay claim**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-03T20:45:26Z
- **Completed:** 2026-04-03T20:49:26Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Resume Step 3 now calls pheromone-read all instead of reading constraints.json
- Signal display shows type, content, and decay-aware effective_strength percentage
- Removed incorrect "Pheromones persist until explicitly cleared -- no decay" claim
- Unified YAML source across both Claude and OpenCode providers (removed provider-specific blocks)

## Task Commits

Each task was committed atomically:

1. **Task 1: Update resume.md Step 3 to read pheromones.json via pheromone-read** - `d8127be` (feat)
2. **Task 2: Sync resume.md changes to YAML source and verify lint** - `d740b52` (feat)

## Files Created/Modified
- `.aether/commands/resume.yaml` - Updated YAML source: unified Step 3, signal rendering, error handling, key constraints
- `.claude/commands/ant/resume.md` - Regenerated Claude command with pheromone-read Step 3
- `.opencode/commands/ant/resume.md` - Regenerated OpenCode command (was already correct, now matches unified source)

## Decisions Made
- Unified the YAML source across both providers rather than keeping separate {{#claude}}/{{#opencode}} blocks. Both providers can call pheromone-read the same way, so provider-specific blocks were unnecessary complexity.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] YAML source required regeneration after direct .md edit**
- **Found during:** Task 2 (sync verification)
- **Issue:** Task 1 edited .claude/commands/ant/resume.md directly, but this file is generated from .aether/commands/resume.yaml. The lint:sync check caught the desync.
- **Fix:** Updated the YAML source (.aether/commands/resume.yaml) with the same changes, then ran `npm run generate` to regenerate both Claude and OpenCode files
- **Files modified:** .aether/commands/resume.yaml, .claude/commands/ant/resume.md, .opencode/commands/ant/resume.md
- **Verification:** npm run lint:sync passes, npm test passes (524 tests)
- **Committed in:** d740b52 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary fix -- commands are generated from YAML sources, so the YAML must be updated. This is the correct project workflow per CLAUDE.md.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- /ant:resume now correctly reads pheromones.json and displays signals with decay-aware state
- PHER-07 (session persistence) requirement addressed
- Ready for plan 03 (or other plans in phase 03)

## Self-Check: PASSED

All files verified present:
- .aether/commands/resume.yaml
- .claude/commands/ant/resume.md
- .opencode/commands/ant/resume.md

All commits verified:
- d8127be (Task 1)
- d740b52 (Task 2)

---
*Phase: 03-pheromone-signal-plumbing*
*Completed: 2026-04-03*
