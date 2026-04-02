---
phase: 49-agent-system-llm
plan: 04
subsystem: agent
tags: [curation, memory-maintenance, orchestrator, sentinel-abort, event-bus]

# Dependency graph
requires:
  - phase: 49-01
    provides: Agent interface, Caste types, Trigger struct, Registry
  - phase: 46
    provides: Event bus with Subscribe, Cleanup, JSONL persistence
  - phase: 45
    provides: Storage layer with LoadJSON, SaveJSON, ReadJSONL, AtomicWrite

provides:
  - 8 curation ants implementing Agent interface (sentinel, nurse, critic, herald, janitor, archivist, librarian, scribe)
  - Sequential orchestrator with sentinel abort matching shell orchestrator.sh behavior
  - CurationAnt interface extending Agent with Run(ctx, dryRun) method
  - StepResult and CurationResult structs for curation run reporting

affects: [agent-system-llm, memory-pipeline, seal-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns: [CurationAnt interface extending Agent, sentinel-first abort pattern, sequential step orchestration]

key-files:
  created:
    - pkg/agent/curation/orchestrator.go
    - pkg/agent/curation/orchestrator_test.go
    - pkg/agent/curation/sentinel.go
    - pkg/agent/curation/nurse.go
    - pkg/agent/curation/critic.go
    - pkg/agent/curation/herald.go
    - pkg/agent/curation/janitor.go
    - pkg/agent/curation/archivist.go
    - pkg/agent/curation/librarian.go
    - pkg/agent/curation/scribe.go
  modified: []

key-decisions:
  - "Sentinel skips .jsonl files during corruption check (line-delimited, not single JSON object)"
  - "Each ant's Triggers() returns nil since the orchestrator handles event subscription"
  - "Nurse recalculates trust scores using simple captures-based formula (0.25 per capture, max 1.0)"
  - "Critic detects contradictions by grouping instincts by topic and counting distinct conclusions"
  - "Herald promotes instincts with confidence >= 0.80 to QUEEN.md Patterns section"
  - "Librarian counts events from JSONL via ReadJSONL, observations/instincts/pheromones from JSON"

patterns-established:
  - "CurationAnt interface: Agent + Run(ctx, dryRun) for orchestrator step execution"
  - "Sentinel abort: orchestrator breaks loop after sentinel error, marks remaining steps as skipped"
  - "Lightweight ant pattern: each Run reads data, does simple operation, returns StepResult with summary map"

requirements-completed: [AGENT-04]

# Metrics
duration: 1min
completed: 2026-04-02
---

# Phase 49 Plan 04: Curation Ants Summary

**8 curation ants with sequential orchestrator, sentinel abort, and event bus integration matching shell orchestrator.sh order**

## Performance

- **Duration:** 1 min
- **Started:** 2026-04-02T03:58:58Z
- **Completed:** 2026-04-02T03:59:49Z
- **Tasks:** 1
- **Files modified:** 10

## Accomplishments
- All 8 curation ants implement Agent interface with correct Name, Caste, Triggers, Execute methods
- Orchestrator runs steps in exact shell-matching order: sentinel, nurse, critic, herald, janitor, archivist, librarian, scribe
- Sentinel abort prevents remaining steps when colony data stores contain corrupt JSON
- 12 tests passing covering interface compliance, step order, sentinel abort, dry-run, and individual ant behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: Curation orchestrator and 8 ant stubs** - `72ec4c0` (feat)

## Files Created/Modified
- `pkg/agent/curation/orchestrator.go` - Sequential 8-step curation runner with sentinel abort, subscribes to consolidation.* events
- `pkg/agent/curation/orchestrator_test.go` - 12 tests: interface compliance, step order, sentinel abort, dry-run, ant behavior
- `pkg/agent/curation/sentinel.go` - Health check ant that validates 6 colony data stores for JSON corruption
- `pkg/agent/curation/nurse.go` - Trust score recalculation ant for instincts with out-of-date scores
- `pkg/agent/curation/critic.go` - Contradiction detection ant grouping instincts by topic
- `pkg/agent/curation/herald.go` - High-confidence (>= 0.80) instinct promotion to QUEEN.md
- `pkg/agent/curation/janitor.go` - Expired event cleanup via bus.Cleanup
- `pkg/agent/curation/archivist.go` - Low-confidence (< 0.30) instinct archival flagging
- `pkg/agent/curation/librarian.go` - Inventory statistics across observations, instincts, events, pheromones
- `pkg/agent/curation/scribe.go` - Report generation ant producing text summaries

## Decisions Made
- Sentinel skips .jsonl files during corruption check since they are line-delimited, not single JSON objects
- Each ant returns nil from Triggers() since the orchestrator handles all event subscriptions
- Nurse uses a lightweight captures-based formula (0.25 per capture, max 1.0) for trust recalculation
- Critic detects contradictions by grouping instincts by topic and checking for multiple distinct conclusions
- Herald reads existing QUEEN.md and appends promoted patterns rather than overwriting

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing go.sum entry missing for pool.go (from another plan's untracked file) -- not related to this plan's changes and not fixed per scope boundary rules

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Curation ants ready for integration with agent lifecycle (trigger on consolidation.* and phase.end events)
- Detailed business logic for nurse, critic, herald can evolve when full memory pipeline is wired
- Scribe report generation can be enhanced to consume prior StepResults for richer output

## Self-Check: PASSED

All 10 source files and SUMMARY.md verified present. Commit 72ec4c0 verified in git log.

---
*Phase: 49-agent-system-llm*
*Completed: 2026-04-02*
