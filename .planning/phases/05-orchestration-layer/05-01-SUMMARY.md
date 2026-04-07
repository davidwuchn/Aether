---
phase: 05-orchestration-layer
plan: 01
name: "Orchestration Foundation Types"
subsystem: agent
tags: [orchestration, task-graph, routing, go]
dependency_graph:
  requires: [colony.Task, agent.Caste]
  provides: [TaskGraph, TaskRouter, TaskContract]
  affects: [phase-orchestrator-plan-03]
tech_stack:
  added: []
  patterns: [adjacency-list, kahns-algorithm, two-pass-routing]
key_files:
  created:
    - pkg/agent/task_graph.go
    - pkg/agent/task_graph_test.go
    - pkg/agent/task_router.go
    - pkg/agent/task_router_test.go
  modified: []
decisions:
  - "Reordered keyword matching: scout before watcher to avoid substring collision (investigate contains test)"
  - "Used int64 for TaskResult.Duration (milliseconds) instead of time.Duration for simpler JSON serialization"
  - "taskID() falls back to Goal text when colony.Task.ID is nil, avoiding empty node IDs"
metrics:
  duration: ~10 minutes
  completed: 2026-04-07
  tasks: 2
  files: 4
  tests: 20
---

# Phase 05 Plan 01: Orchestration Foundation Types Summary

Core data structures for the orchestration layer: TaskGraph (dependency-aware scheduling), TaskRouter (runtime caste assignment), and TaskContract (versioned agent-role contracts). These are the foundation types that the PhaseOrchestrator (Plan 03) will consume.

## What Was Built

### TaskGraph (`pkg/agent/task_graph.go`)
- **TaskNode** struct: ID, Goal, Caste, Status, DependsOn, Criteria, TypeHint
- **TaskResult** struct: outcome of task execution with JSON tags
- **TaskContract** struct: versioned agent-role contract (ORCH-06)
- **TaskGraph** struct: map-based adjacency list with in-degree tracking
- **BuildTaskGraph()**: converts `[]colony.Task` to a graph, assigns castes, detects cycles via Kahn's algorithm
- **Ready()**: returns tasks with in-degree 0 (ready to execute)
- **Complete()**: marks task done, decrements dependents, returns newly-ready tasks
- **Nodes()/Node()**: accessors for graph inspection

### TaskRouter (`pkg/agent/task_router.go`)
- **ParseTypeHint()**: extracts `[bracket]` notation from task goals via regex
- **RouteTask()**: two-pass routing -- type hints first, keyword matching second
- **hintToCaste()**: maps hint strings to castes (builder/watcher/scout/architect)
- Keyword groups: scout (research, investigate, ...), architect (design, architect, ...), watcher (test, verify, ...), builder (implement, create, ...)
- Default caste: CasteBuilder

### Tests
- 8 tests for TaskGraph: empty, single, deps, cycle detection, ready, complete, chain, type hint assignment
- 12 tests for TaskRouter: type hint parsing (4), route task (6), hintToCaste (2)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Keyword ordering substring collision**
- **Found during:** Task 2 test run
- **Issue:** "investigate" contains "test" as a substring. With watcher keywords checked before scout keywords, `RouteTask("investigate why tests fail")` returned CasteWatcher instead of CasteScout.
- **Fix:** Reordered keyword matching: scout check runs before watcher check. This is safe because watcher-specific words (test, verify, assert, check, validate) won't appear as substrings in scout tasks.
- **Files modified:** pkg/agent/task_router.go
- **Commit:** 846b9528

## Verification

- `go test ./pkg/agent/ -run "TestBuildTaskGraph|TestTaskGraph|TestParseTypeHint|TestRouteTask|TestHintToCaste" -v -count=1` -- all 20 tests pass
- `go vet ./pkg/agent/` -- no issues
- `go build ./pkg/agent/` -- compiles cleanly

## Commits

| Task | Hash | Message |
|------|------|---------|
| 1 | d0fae193 | feat(05-01): add TaskGraph with dependency scheduling and cycle detection |
| 2 | 846b9528 | feat(05-01): add TaskRouter with type hint and keyword caste routing |

## Known Stubs

None -- all types are fully implemented with working logic.

## Threat Flags

None -- type hint regex is limited to `[a-z]` bracket content (T-05-01-01 mitigated). Cycle detection prevents infinite loops (T-05-01-02 mitigated).

## Self-Check: PASSED
