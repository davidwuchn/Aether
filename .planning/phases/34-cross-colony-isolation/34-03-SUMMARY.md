---
phase: 34-cross-colony-isolation
plan: 03
subsystem: infra
tags: [colony-isolation, per-colony-data, directory-resolution, migration]

# Dependency graph
requires:
  - phase: 34-01
    provides: colony-name subcommand for colony identification
  - phase: 34-02
    provides: hub-level lock primitives (acquire_lock_at, release_lock_at)
provides:
  - COLONY_DATA_DIR infrastructure for per-colony data isolation
  - Auto-migration from flat DATA_DIR to colonies/{name}/ subdirectories
  - Updated file references in aether-utils.sh (62 COLONY_DATA_DIR usages)
affects: [34-04, 34-05, future multi-colony work]

# Tech tracking
tech-stack:
  added: []
  patterns: [per-colony data directories, automatic migration on first access, colony_name-based path resolution]

key-files:
  created: []
  modified: [.aether/aether-utils.sh]

key-decisions:
  - "COLONY_STATE.json remains at DATA_DIR root as the anchor for colony identification"
  - "Per-colony files use COLONY_DATA_DIR, shared files use DATA_DIR"
  - "Migration uses presence-based detection (no version field per locked decision)"
  - "Migration function intentionally uses DATA_DIR for source paths"

patterns-established:
  - "Per-colony data isolation: colonies/{sanitized-name}/ for all colony-specific files"
  - "Graceful fallback: pre-init (no COLONY_STATE.json) uses DATA_DIR directly"
  - "Auto-migration: flat files moved on first access with lock-based concurrency safety"

requirements-completed: [SAFE-02]

# Metrics
duration: 4min
completed: 2026-03-29
---

# Phase 34: Cross-Colony Isolation - Plan 03 Summary

**Per-colony data directory infrastructure with automatic migration from flat DATA_DIR to colonies/{sanitized-name}/ subdirectories**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-29T07:25:30Z
- **Completed:** 2026-03-29T07:29:45Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- Task 1 was already complete from previous agents: `_resolve_colony_data_dir()` and `_maybe_migrate_colony_data()` infrastructure functions with startup wiring
- Task 2: Updated ~52 per-colony file references from DATA_DIR to COLONY_DATA_DIR across aether-utils.sh
- COLONY_DATA_DIR reference count increased from 12 to 62
- All per-colony files now use COLONY_DATA_DIR (activity.log, pheromones.json, learning-observations.json, etc.)
- Shared files remain at DATA_DIR (COLONY_STATE.json, backups/, survey/)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add COLONY_DATA_DIR resolution and auto-migration infrastructure** - Already complete from previous agents (functions existed at lines 147-292)
2. **Task 2: Update per-colony file references in aether-utils.sh from DATA_DIR to COLONY_DATA_DIR** - `6c5566f` (feat)

**Plan metadata:** `6c5566f` (feat: update per-colony DATA_DIR references to COLONY_DATA_DIR)

## Files Created/Modified

- `.aether/aether-utils.sh` - Updated ~52 per-colony file references to use COLONY_DATA_DIR:
  - constraints.json, activity.log, activity-phase-*.log, watch-progress.txt
  - error-patterns.json, signatures.json, view-state.json
  - checkpoint-allowlist.json, rolling-summary.log
  - spawn-tree.txt, midden/midden.json, pheromones.json
  - learning-observations.json, learning-deferred.json
  - flags.json, run-state.json

## Decisions Made

- COLONY_STATE.json stays at DATA_DIR root as the colony identification anchor
- Migration function (`_maybe_migrate_colony_data`) intentionally references DATA_DIR for source paths (correct behavior)
- Colony name sanitization: lowercase, non-alphanumeric to hyphens, trimmed
- Empty sanitized names fail loudly per locked decision (no silent fallback)

## Deviations from Plan

None - plan executed exactly as written. Task 1 infrastructure was already in place from previous parallel agents. Task 2 completed as specified with all per-colony references updated.

## Issues Encountered

- Initial grep showed 52 per-colony DATA_DIR references needing updates
- All references were systematically updated to COLONY_DATA_DIR
- 8 remaining DATA_DIR references are all in `_maybe_migrate_colony_data` function (intentional)
- Pre-existing js-yaml module test failure (unrelated to our changes)

## User Setup Required

None - no external service configuration required. Auto-migration runs transparently on first access.

## Next Phase Readiness

- COLONY_DATA_DIR infrastructure complete and functional
- Auto-migration tested and working (files moved from DATA_DIR to colonies/aether-colony/)
- Ready for 34-04 (per-colony utils modules update) and 34-05 (OpenCode command updates)
- No blockers or concerns

---
*Phase: 34-cross-colony-isolation*
*Completed: 2026-03-29*
