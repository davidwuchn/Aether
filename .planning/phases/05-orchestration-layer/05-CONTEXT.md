# Phase 5: Orchestration Layer - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

A centralized Go-based coordinator that decomposes phases into tasks, assigns specialist agents at runtime, and validates outputs before marking phases complete. This phase covers: PhaseOrchestrator type, TaskRouter, agent isolation via context boundaries, output validation, orchestrator status command, and unifying orchestrator state into COLONY_STATE.json. It does NOT cover branching/worktree discipline (Phase 6) or repository hygiene (Phase 7).

</domain>

<decisions>
## Implementation Decisions

### Orchestrator architecture
- **D-01:** Hybrid model — orchestrator plans tasks upfront (imperative decomposition), then dispatches them via the event bus for concurrent execution. Combines predictable planning with concurrent execution.
- **D-02:** PhaseOrchestrator is a new type in `pkg/agent/` (alongside Pool, Registry). It implements the Agent interface so it can be registered and triggered by the event bus, like the curation orchestrator.
- **D-03:** Task decomposition happens at phase start — the orchestrator reads the phase definition, creates a task graph, then dispatches tasks as dependencies resolve.

### Task routing strategy
- **D-04:** Runtime matching — the orchestrator decides caste assignment at dispatch time, not at plan time. The route-setter provides phase structure; the orchestrator picks the best caste for each task based on task type analysis.
- **D-05:** Routing should reuse or extend the existing skill-match scoring system where possible. A task mentioning "test" routes to watcher, "research" to scout, "implement" to builder, etc.
- **D-06:** Task descriptions from the plan include type hints (e.g., `[test]`, `[implement]`, `[research]`) that guide routing. The orchestrator parses these hints to select castes.

### Agent isolation model
- **D-07:** Context boundaries — each agent runs in a goroutine with its own scoped context. Cancellation via ctx.Done(). This is Go-level isolation, not file-level or prompt-level.
- **D-08:** Agents receive only their assigned task scope (task description, relevant files, success criteria). The orchestrator assembles this scoped context before dispatch — agents don't see sibling tasks or the full phase plan.

### CLI interface & state
- **D-09:** Unified state — orchestrator state integrates into COLONY_STATE.json (single source of truth). The autopilot's separate `autopilot/state.json` should be migrated to read from COLONY_STATE.json instead, eliminating the dual-state problem.
- **D-10:** Three new cobra commands: `orchestrator-decompose` (shows task plan for a phase), `orchestrator-assign` (shows current caste assignments), `orchestrator-status` (full visibility of task assignments, progress, agent states).
- **D-11:** All commands produce JSON output via `outputOK()` following the existing pattern in `cmd/`.

### Claude's Discretion
- Exact task graph data structure (slice vs map vs custom graph)
- Whether to add a `orchestrator-run` command that triggers a full phase execution
- How task type hints are specified in plan files (struct tags, comment prefixes, separate field)
- Whether to add orchestration events to the event bus topics (e.g., `orchestrator.phase.start`)
- Exact JSON schema for orchestrator-status output

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Proven template — curation orchestrator
- `pkg/agent/curation/orchestrator.go` — Orchestrator struct, Agent interface implementation, sequential Run() with sentinel abort. The phase orchestrator follows this pattern but with hybrid dispatch instead of sequential.

### Agent system
- `pkg/agent/agent.go` — Agent interface (Name, Caste, Triggers, Execute), Caste enum, Registry, Match()
- `pkg/agent/pool.go` — Event bus dispatch with bounded concurrency, errgroup pattern
- `pkg/agent/spawn_tree.go` — SpawnEntry tracking (timestamp, parent, caste, name, task, depth, status)

### Autopilot (needs state migration)
- `cmd/autopilot.go` — Current autopilot state struct and commands (separate `autopilot/state.json`)
- `cmd/.claude/commands/ant/run.md` — Autopilot loop: Step 0 reads COLONY_STATE.json, Step 1-5 build/continue/advance cycle

### Colony state model
- `pkg/colony/colony.go` — ColonyState struct (add orchestrator fields here)
- `.planning/REQUIREMENTS.md` — ORCH-01 through ORCH-07 requirements

### Route-setter (provides phase structure)
- `.claude/agents/ant/aether-route-setter.md` — Phase decomposition, task formatting, `{granularity_min}-{granularity_max}` bounds

### Skills system (routing reference)
- `.aether/skills/` — skill-match scoring that routing logic can extend

### Phase 4 context (granularity patterns)
- `.planning/phases/04-planning-granularity-controls/04-CONTEXT.md` — Enum pattern, persistence, autopilot integration patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Curation orchestrator pattern (`pkg/agent/curation/orchestrator.go`) — Agent interface, Run(), step/result structs, sentinel abort logic
- Agent Registry + Pool (`pkg/agent/agent.go`, `pkg/agent/pool.go`) — event dispatch, goroutine management, Match() for trigger-based routing
- Caste enum (`pkg/agent/agent.go:17-27`) — all 9 castes already defined
- SpawnTree (`pkg/agent/spawn_tree.go`) — tracks agent lifecycle, can be extended for orchestrator task tracking
- `outputOK()`/`outputError()` — standard JSON output pattern for all cobra commands

### Established Patterns
- Agent interface: Name(), Caste(), Triggers(), Execute()
- Commands registered in `init()` with `rootCmd.AddCommand()`
- State stored as JSON via `store.SaveJSON()` / `store.LoadJSON()`
- Event bus topics use dot notation: `consolidation.*`, `phase.end`
- Streaming support via StreamManager for agents that implement StreamingAgent

### Integration Points
- `pkg/agent/` — new orchestrator.go file alongside pool.go, spawn_tree.go
- `pkg/colony/colony.go` — add orchestrator state fields to ColonyState struct
- `cmd/` — three new cobra commands for decompose/assign/status
- Event bus — orchestrator subscribes to `phase.start` or similar topic
- Autopilot — migrate from separate state.json to COLONY_STATE.json

</code_context>

<specifics>
## Specific Ideas

- The curation orchestrator's `StepResult` / `CurationResult` structs are a good template for task results
- Task type hints could be as simple as parsing keywords from task descriptions ("implement X" → builder, "test Y" → watcher, "research Z" → scout) rather than requiring explicit struct tags
- The `orchestrator-status` command should show a table similar to spawn tree output: task name, assigned caste, status, duration

</specifics>

<deferred>
## Deferred Ideas

- Branch/worktree integration with orchestrator — Phase 6 scope
- Dynamic agent spawning based on task complexity (ORCH-08, v2 requirement)
- Graph-based task routing with dependency resolution (ORCH-09, v2 requirement)

</deferred>

---

*Phase: 05-orchestration-layer*
*Context gathered: 2026-04-07*
