# Aether: Oracle Deep Research Engine

## What This Is

Aether is a self-managing development assistant that uses ant colony metaphor to orchestrate AI workers across coding sessions. It has 36 commands, 22 agents, and ~10,000 lines of shell infrastructure. The colony is fully self-improving with complete learning pipelines (v1.0). The oracle (`/ant:oracle`) is the colony's deep research capability — users invoke it to do thorough, verified, actionable research on any topic for projects they're developing with Aether.

## Core Value

The oracle produces research you can act on — verified against official sources, iteratively deepened, and structured for the specific topic. It runs autonomously for as long as needed, drilling deeper each iteration rather than just covering more surface area.

## Current Milestone: v1.1 Oracle Deep Research

**Goal:** Rebuild the oracle as a proper Ralph-loop-based deep research engine that produces thorough, source-verified, actionable research — rebranded for the Aether colony.

**Target features:**
- Proper iterative loop where each iteration builds meaningfully on the last (not just appending text)
- Source verification — check official docs, cite where info came from
- Configurable research strategy — breadth-first vs depth-first, focus areas
- Flexible output structure suited to the specific topic
- Autonomous operation that decides when it's truly done (confidence-based)
- User can steer mid-session when desired, or fire-and-forget
- Research findings can feed into colony knowledge (QUEEN.md, instincts, pheromones)
- Works on any repo where Aether is installed, not just Aether itself

## Requirements

### Validated

- Colony command infrastructure (36 commands, all functional)
- Pheromone signal system (FOCUS/REDIRECT/FEEDBACK emit and display)
- colony-prime injection (pheromones reach builders via prompt_section)
- Midden failure tracking (recent failures shown to builders)
- Graveyard file cautions (unstable files flagged to builders)
- Survey territory intelligence (codebase patterns fed to builders)
- State persistence across sessions (COLONY_STATE.json, CONTEXT.md)
- Memory-capture pipeline (learning-observe, observation counting)
- Instinct infrastructure (instinct-create, instinct-read exist)
- QUEEN.md infrastructure (queen-init, queen-read, queen-promote exist)
- Suggest-analyze/approve pipeline (pheromone suggestions exist)
- Phase learnings auto-inject into future builder prompts -- v1.0
- Key decisions auto-convert to FEEDBACK pheromones -- v1.0
- Recurring error patterns auto-emit REDIRECT pheromones -- v1.0
- Learning observations auto-promote to QUEEN.md when thresholds met -- v1.0
- Escalated flags inject as warnings into next phase builders -- v1.0
- colony-prime reads CONTEXT.md decisions for builder injection -- v1.0
- instinct-create called during continue flow with confidence >= 0.7 -- v1.0
- instinct-read results included in colony-prime output (domain-grouped) -- v1.0
- queen-promote called during seal and continue flows -- v1.0
- Success criteria patterns create instincts on recurrence -- v1.0

### Active

- [ ] Oracle uses proper Ralph-loop pattern with meaningful iteration-over-iteration depth
- [ ] Source verification — findings cite official docs, claims are checked
- [ ] Configurable iteration strategy (breadth-first vs depth-first)
- [ ] Configurable focus areas to prioritize certain aspects
- [ ] Output structure adapts to the specific research topic
- [ ] Mid-session steering — user can redirect research without restarting
- [ ] Research findings can integrate into colony knowledge
- [ ] Works as standalone tool on any Aether-equipped repo

### Out of Scope

- Cross-colony wisdom sharing -- solve single-colony learning first
- Model routing verification -- separate concern
- XML migration -- do gradually as files are touched

## Context

Aether v1.1.11 with v1.0 colony wiring shipped. 535+ tests passing. The oracle currently exists as a basic Ralph-loop adaptation: `oracle.sh` runs a bash for-loop spawning fresh Claude instances that append to `progress.md`. The loop checks for `<oracle>COMPLETE</oracle>` to terminate. However, the research quality is shallow — iterations don't meaningfully build on each other, sources aren't verified, and the output is a flat append log with no structure.

The original Ralph pattern (github.com/snarktank/ralph) by Geoffrey Huntley uses prd.json for structured task tracking, a "Codebase Patterns" section that every iteration reads first, and quality gates between iterations. The oracle needs these disciplines adapted for research rather than implementation.

Current oracle files: `.aether/oracle/oracle.sh` (134 lines, the loop runner), `.aether/oracle/oracle.md` (36 lines, the per-iteration prompt), `.claude/commands/ant/oracle.md` (378 lines, the wizard/launcher command).

## Constraints

- **Modify existing oracle** -- upgrade oracle.sh, oracle.md, and the oracle command; don't create a separate system
- **Backward compatible** -- existing colonies and other commands must not break
- **Must work in Claude Code** -- all output via unicode/emoji, no ANSI
- **Bash 3.2 compatible** -- macOS ships bash 3.2
- **Test coverage** -- new behavior needs tests in existing test framework
- **Works on any repo** -- oracle research is for user projects, not just Aether development
- **Colony branding** -- rebrand Ralph metaphor to fit the oracle/ant colony theme

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Connect, don't add | System has all pieces, just disconnected | ✓ Good -- v1.0 |
| colony-prime is the integration point | Single function that assembles all context | ✓ Good -- v1.0 |
| Upgrade existing oracle, not new system | Oracle infrastructure exists, just needs depth | -- Pending |
| Ralph loop rebranded as oracle | Colony theme, not Simpsons reference | -- Pending |
| Research quality over research speed | Each iteration must add real depth | -- Pending |

---
*Last updated: 2026-03-13 after v1.1 milestone start*
