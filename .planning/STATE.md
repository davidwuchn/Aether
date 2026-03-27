# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-27)

**Core value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.
**Current focus:** Phase 21 -- Test Infrastructure Refactor (v2.3 Per-Caste Model Routing)

## Current Position

Phase: 21 of 24 (Test Infrastructure Refactor)
Plan: 02 of 2
Status: Ready to execute
Last activity: 2026-03-27 -- 21-01 complete (centralized mock profile helper)

Progress: [===       ] 50%

## Performance Metrics

**Velocity (from v2.1):**
- Total plans completed: 43
- Average duration: 5min
- Total execution time: 3.8 hours

**Recent Trend:**
- v2.1 completed 8 phases, 39 plans in ~3 hours
- v2.2 completed 4 phases, 5 plans
- v2.3: 4 phases planned, Phase 21 plan 1 complete (2 plans remaining)

*Updated after each plan completion*

## Accumulated Context

### Decisions

- [v2.3]: Phase 1 must complete before any model-profiles.yaml changes -- 184 hardcoded model names in tests will break otherwise
- [v2.3]: Use Approach A (agent frontmatter) for MVP routing -- simpler than Task tool model param, zero playbook changes needed
- [v2.3]: Aether routes by slot name (opus/sonnet), never by actual model name -- keeps dual-mode support clean
- [v2.3]: Specialist castes stay on `inherit` -- Tracker, Auditor, Gatekeeper, etc. don't need explicit routing
- [v2.2]: Focus exclusively on wisdom systems -- ceremony/verification improvements deferred to v2.3
- [v2.3/21-01]: No caching in mock-profiles helper -- each function call reads fresh YAML so tests break intentionally if YAML changes
- [v2.3/21-01]: buildMockProfiles uses spread merge for workerModels/modelMetadata, full replacement for taskRouting

### Pending Todos

None yet.

### Blockers/Concerns

- GLM-5 constraint passing through subagent spawning is unverified -- if Claude Code does not forward temperature/top_p/max_tokens, reasoning castes may loop on GLM-5
- Task tool `model` parameter vs frontmatter precedence untested -- if Task param does not override frontmatter, Approach B cannot coexist with Approach A
- Oracle and Architect castes lack dedicated agent files -- their work runs through Queen or direct CLI, which may use the wrong model

## Session Continuity

Last session: 2026-03-27
Stopped at: Completed 21-01, ready for 21-02
Resume file: None
