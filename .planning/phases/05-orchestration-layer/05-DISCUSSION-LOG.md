# Phase 5: Orchestration Layer - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 05-orchestration-layer
**Areas discussed:** Orchestrator architecture, Task routing strategy, Agent isolation model, CLI & state design

---

## Orchestrator architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Sequential (like curation) | Like curation orchestrator — one agent at a time, predictable, easy to debug | |
| Event-driven (like Pool) | Subscribe to events, dispatch concurrently via event bus. Faster but harder to debug. | |
| Plan-then-dispatch (hybrid) | Plan tasks upfront (imperative), then dispatch via event bus for concurrent execution | ✓ |

**User's choice:** Plan-then-dispatch (hybrid)
**Notes:** Best of both — planned decomposition with concurrent execution. Orchestrator plans tasks upfront, then dispatches via event bus.

---

## Task routing strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Plan-time assignment (Recommended) | Route-setter assigns castes during planning. Orchestrator reads assignments. Simple. | |
| Runtime matching | Orchestrator decides caste at dispatch time based on task content analysis | ✓ |
| Default + runtime override | Plan provides default caste, orchestrator can override at runtime | |

**User's choice:** Runtime matching
**Notes:** The orchestrator decides caste assignment at dispatch time, not at plan time. More flexible.

---

## Agent isolation model

| Option | Description | Selected |
|--------|-------------|----------|
| Prompt scoping (Recommended) | Scoped context per agent — enforced by orchestrator assembling prompts | |
| Context boundaries | Separate goroutine per agent with own context, cancellation via ctx.Done() | ✓ |
| Worktree isolation | Separate file trees per agent — strongest but heaviest | |

**User's choice:** Context boundaries
**Notes:** Go-level isolation via goroutines with scoped context. Cancellation via ctx.Done().

---

## CLI & state design

| Option | Description | Selected |
|--------|-------------|----------|
| Unified (Recommended) | Integrate into COLONY_STATE.json. Autopilot migrates too. Single source of truth. | ✓ |
| Separate file | Orchestrator gets its own state file, like autopilot/state.json | |
| Stateless (recompute) | No persistent state — recomputes from COLONY_STATE.json + plan files each time | |

**User's choice:** Unified (Recommended)
**Notes:** Orchestrator state integrates into COLONY_STATE.json. Autopilot should migrate from its separate state.json.

---

## Claude's Discretion

- Exact task graph data structure
- Whether to add orchestrator-run command
- How task type hints are specified in plan files
- Whether to add orchestration event bus topics
- Exact JSON schema for orchestrator-status output
