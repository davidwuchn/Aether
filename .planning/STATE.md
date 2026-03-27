# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-27)

**Core value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.
**Current focus:** v2.4 Living Wisdom

## Current Position

Phase: 25 — Agent Definitions (Oracle + Architect)
Plan: 2/2 complete (both executed)
Status: Phase 25 complete — ready for Phase 26 (Build Pipeline Wiring)
Last activity: 2026-03-27 — 25-02 complete (Oracle + Architect wired into build flow, docs updated to 24 agents)

## Performance Metrics

**Velocity (from v2.1):**
- Total plans completed: 53
- Average duration: 5min
- Total execution time: 4.3 hours

**Recent Trend:**
- v2.1 completed 8 phases, 39 plans in ~3 hours
- v2.2 completed 4 phases, 5 plans
- v2.3: 4 phases planned, Phase 21 complete, Phase 22 complete (3/3 plans), Phase 23 complete (2/2 plans), Phase 24 complete (2/2 plans: safety warnings + spawn-tree resolution + caste table + config swap)
- v2.4: Roadmap created, 4 phases planned (25-28), 11 requirements mapped, Phase 25 complete (2/2 plans: agent defs + build wiring)

*Updated after each plan completion*

## Accumulated Context

### Decisions

- [v2.4]: 4-phase structure: agents first (independent), pipeline wiring (highest value), fallback+dedup (depends on pipeline), integration validation (depends on all)
- [v2.4]: Phase numbering continues from v2.3 end (phase 24) — v2.4 starts at phase 25
- [v2.4/25-01]: Oracle output convention: .aether/data/research/oracle-{phase_id}.md -- shared research directory with Architect
- [v2.4/25-01]: Architect output convention: .aether/data/research/architect-{phase_id}.md -- write-to-research-dir, must-not-modify-source-code boundary
- [v2.4/25-01]: Oracle distinguished from Scout by Write capability + deeper research + actionable recommendations; Architect distinguished from Keeper by creating new designs vs synthesizing knowledge
- [v2.4/25-02]: Oracle spawns before Architect, both before workers — non-blocking failures (log warning, continue build)
- [v2.4/25-02]: Architect in Orchestration tier, Oracle in Niche tier in CLAUDE.md agent table
- [v2.4/25-02]: Pre-worker specialist spawn pattern: Oracle (research) -> Architect (design) -> Workers (implementation)
- [v2.3]: Phase 1 must complete before any model-profiles.yaml changes -- 184 hardcoded model names in tests will break otherwise
- [v2.3]: Use Approach A (agent frontmatter) for MVP routing -- simpler than Task tool model param, zero playbook changes needed
- [v2.3]: Aether routes by slot name (opus/sonnet), never by actual model name -- keeps dual-mode support clean
- [v2.3]: Specialist analysis castes on opus (Tracker, Auditor, Gatekeeper, Measurer) -- reasoning depth improves accuracy on bounded analysis tasks
- [v2.2]: Focus exclusively on wisdom systems -- ceremony/verification improvements deferred to v2.3
- [v2.3/21-01]: No caching in mock-profiles helper -- each function call reads fresh YAML so tests break intentionally if YAML changes
- [v2.3/21-01]: buildMockProfiles uses spread merge for workerModels/modelMetadata, full replacement for taskRouting
- [v2.3/21-02]: Soft-gate regression test logs violations without failing -- avoids false positives from legitimate test inputs
- [v2.3/21-02]: Module-level YAML-derived constants (BUILDER_MODEL, ALT_MODEL) at file top avoid repeated helper calls
- [v2.3/21-03]: Loop-based provider verification in integration tests -- automatically covers new models added to YAML
- [v2.3/22-01]: keeper placed on inherit tier (3 inherit castes: chronicler, includer, keeper) -- CONTEXT.md listed only 2, needs update
- [v2.3/22-01]: Slot-based worker_models: castes store slot names (opus/sonnet/inherit), model_slots section provides resolution table
- [v2.3/22-02]: Agent frontmatter model: field activates Claude Code native routing -- 10 opus, 11 sonnet, 3 inherit across 24 agents
- [v2.3/22-03]: OpenCode verify-castes mirror updated alongside Claude Code version for sync policy parity
- [v2.3/23-01]: getModelSlotForCaste returns DEFAULT_SLOT ('inherit') for missing castes -- silent fallback, no console warnings
- [v2.3/23-01]: validateSlot uses {valid, error} return pattern for centralized slot-name validation
- [v2.3/24-01]: Spawn-tree auto-resolution uses bash $0 subprocess to model-slot get -- avoids tight coupling between spawn.sh and model-slot modules
- [v2.3/24-01]: GLM-5 safety warnings in XML-style glm_safety blocks for machine-parseability
- [v2.3/24-02]: Static caste table in command files -- avoids dynamic resolution failure modes
- [v2.3/24-02]: Claude API as explicit default in all config swap docs, GLM proxy opt-in

### Pending Todos

None yet.

### Blockers/Concerns

- Builder learning extraction quality is unvalidated — git-diff-based fallback is the biggest uncertainty in v2.4
- Hive brain is a chicken-and-egg problem — first colony to use the pipeline will not benefit from cross-colony wisdom (expected behavior)
- GLM-5 constraint passing through subagent spawning is unverified — deferred (not blocking v2.4)
- Task tool `model` parameter vs frontmatter precedence untested — deferred (not blocking v2.4)

## Session Continuity

Last session: 2026-03-27
Stopped at: Completed 25-02 (Oracle + Architect wired into build flow)
Resume file: None
