---
phase: 33-input-escaping-atomic-write-safety
plan: 03
status: complete
started: 2026-03-29T05:08:00Z
completed: 2026-03-29T05:35:00Z
---

## Summary

Audited all acquire_lock callers and added trap-based lock cleanup. Hardened atomic_write documentation and added safety stats tracking for stale lock cleanup.

## What was built

- Trap-based lock cleanup (`trap 'release_lock 2>/dev/null || true' EXIT`) added to all lock-acquiring functions in pheromone.sh
- Safety stats tracking (`_safety_stats_increment`) wired into file-lock.sh stale lock auto-cleanup
- atomic_write.sh documented: does NOT interact with locks, caller responsibility
- JSON validation reject tracking added to atomic_write

## Key files

### Created
(none)

### Modified
- `.aether/utils/pheromone.sh` — trap-based lock cleanup on all acquire_lock paths
- `.aether/utils/file-lock.sh` — safety stats increment on stale lock cleanup
- `.aether/utils/atomic-write.sh` — safety stats increment on JSON validation reject, documentation

## Commits
- `4623cb4` — add trap-based lock cleanup to pheromone.sh functions
- `cfa4c55` — lock safety hardening in atomic-write.sh and file-lock.sh

## Deviations
- Lock audit for learning.sh, midden.sh, hive.sh, flag.sh: these files already had correct release_lock patterns from prior work. No changes needed.
- Safety stats file (safety-stats.json) creation deferred to Plan 04 which wires it into /ant:status.

## Self-Check: PASSED
- All acquire_lock callers in pheromone.sh have trap-based cleanup
- Stale lock auto-cleanup tracked in safety stats
- atomic_write rejects invalid JSON (existing behavior preserved)
- 616 tests pass
