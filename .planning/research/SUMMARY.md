# Project Research Summary

**Project:** Aether v1.3 Maintenance Milestone — Pheromone Integration & System Polish
**Domain:** Multi-agent CLI orchestration system with stigmergic coordination
**Researched:** 2026-03-19
**Confidence:** HIGH

## Executive Summary

Aether is a mature multi-agent CLI orchestration system with ~10K lines of bash, 150 subcommands, 22 agents, 36 slash commands, and 490+ tests. The v1.3 maintenance milestone is not about adding new capabilities — it is about wiring together what already exists. The core insight from all four research streams is the same: the infrastructure is complete but the integration seams are broken. Signals can be written and read, but workers do not act on them. The learning pipeline exists but is polluted with test data that prevents validating real behavior. The fresh-install experience is documented but not smoke-tested and ships test artifacts.

The recommended approach is strictly sequential: clean the system before integrating it. Test data pollution in pheromones.json, COLONY_STATE.json, QUEEN.md, and constraints.json must be removed first because everything downstream depends on clean state. Only then can pheromone injection be verified end-to-end, the learning pipeline validated, and the fresh-install experience hardened. Doing integration work on a polluted system produces misleading results and wastes effort.

The key risk is the size of the blast radius. aether-utils.sh has 150 subcommands with no module boundaries. A single schema change can break tests across 47+ files simultaneously. The mitigation is strict discipline: run the full test suite after every atomic change, make schema changes additive-only, and never batch changes. The second risk is documentation drift — the docs currently claim workers "read signals" when they actually receive injected context. Updating docs before the integration is verified creates a second round of corrections.

## Key Findings

### Recommended Stack

The existing stack (bash + jq + Node.js CJS + JSON files) is the right stack for this system. No new infrastructure is needed. The one genuine technology addition is **Ajv** (`^8.18.0`) for JSON Schema validation of pheromone signals at write time — this prevents malformed signals from entering the system silently. Everything else identified as a "gap" is wiring, error handling, or prompt engineering within the existing stack.

**Core technologies:**
- **bash + jq** — Signal storage and lifecycle logic; 150 subcommands in aether-utils.sh. Stay as-is, avoid migrating signal logic to Node.js until file exceeds ~15K lines
- **Ajv `^8.18.0`** — Schema validation for pheromone signal JSON before write; only genuine new dependency. 14M ops/sec, CJS-compatible, JSON Schema standard (not Zod which requires TypeScript benefit)
- **Commander.js `^14.0.3`** — Upgrade from current `^12`; stay on v14 (v15 goes ESM-only May 2026 which would break CJS codebase)
- **AVA `^6.4.1`** — Already installed, no change. Covers JS unit tests
- **Custom bash test harness** — Keep for now; adopt bats-core only if adding 20+ new bash test cases

### Expected Features

All features for this milestone are maintenance and wiring work, not new capabilities.

**Must have (table stakes):**
- **Clean data files on install** — test artifacts in pheromones.json, COLONY_STATE.json, QUEEN.md, constraints.json make the system look broken to new users; 11 learning-observations are entirely test data
- **Pheromones that actually affect worker behavior** — the docs claim signals guide workers but no agent definition references pheromones; injection model works but must be verified end-to-end
- **Working fresh install flow** — lay-eggs -> init -> plan -> build must work without errors; no smoke test validates this today
- **Documentation that matches reality** — CLAUDE.md references eliminated features; pheromone docs describe aspirational behavior; known-issues.md has stale FIXED statuses

**Should have (competitive):**
- **Closed-loop learning pipeline** — observation -> learning -> instinct -> pheromone exists as components but is not validated end-to-end; unique differentiator among multi-agent frameworks
- **`data-clean` command** — utility to purge test artifacts and expired signals safely; no competitor needs this (they are stateless), but Aether's persistence model requires it
- **XML exchange command surface** — `/ant:export-signals`, `/ant:import-signals` using the fully-built but unactivated pheromone-xml.sh exchange module

**Defer (v2+):**
- **Agent Teams migration** — Claude Code Agent Teams (Feb 2026) enables inter-worker communication; significant architecture change, current system works adequately
- **Cross-colony wisdom sync** — requires XML exchange to be activated and tested first
- **Dashboard web UI** — out of scope for a CLI tool; ASCII dashboards already work

### Architecture Approach

Aether implements textbook stigmergic coordination: workers never communicate directly, all coordination flows through shared environment files (pheromones.json, COLONY_STATE.json). The Queen orchestrator reads signals, compiles them into `prompt_section` via `colony-prime`, and injects that context into each worker's spawn prompt. This injection model is architecturally sound — the gap is not the model, it is missing verification that the injection is happening correctly and that workers are acting on the injected context.

**Major components and their signal roles:**
1. **pheromones.json** — Single source of truth for active signals; constraints.json is a deprecated backward-compat write that should eventually be eliminated
2. **colony-prime** — The single aggregation point: wisdom + signals + learnings + context capsule = prompt_section. All new signal types must flow through this, never directly to workers
3. **memory-capture** — Unified pipeline entry point: observe + emit pheromone + check promotion threshold. All new event recording must go through this, not parallel paths
4. **build-wave.md + continue-advance.md** — The integration points where signals enter and exit the build lifecycle; these are prompt-engineering files, not code, and are where most wiring work happens

### Critical Pitfalls

1. **Test data cleanup removes canonical seed data** — QUEEN.md has 5 legitimate seed entries mixed with 25+ test entries. Aggressive cleanup removes seeds that new colonies need; insufficient cleanup ships test artifacts. Prevention: document canonical entries before deleting, create a golden-state snapshot as baseline, add CI validation
2. **Write-only signal system** — The pheromone plumbing was built bottom-up without verifying workers act on signals. Fix must be top-down: start from agent definitions (aether-builder.md, aether-watcher.md), not from plumbing. Start with REDIRECT (simplest behavior) before FOCUS or FEEDBACK
3. **Test cascade from schema changes** — aether-utils.sh has no module boundaries; one JSON schema change can fail 50+ tests simultaneously. Prevention: additive-only schema changes, run full test suite after every atomic change, write tests first (TDD)
4. **XML archival leaves dead references** — The XML exchange system has 6 files, tests, and aether-utils.sh help entries but zero active wiring. "Archive" must mean a specific, complete operation: move files, disable tests, remove from help listing, update validate-package.sh
5. **Lock contention during parallel builds** — File-based locking has no jitter; parallel workers racing on pheromones.json can cause timeouts. Prevention: batch signal writes per wave, add jitter to retry, test with 3+ concurrent writes

## Implications for Roadmap

Based on combined research, the dependency chain is clear: clean before integrating, integrate before documenting, document before polishing.

### Phase 1: Data Cleanup and State Reset

**Rationale:** Everything downstream depends on clean state. The learning pipeline cannot be validated with test-only observations. The fresh install cannot be verified if templates ship test artifacts. Signal injection cannot be audited when signals are fake. This phase has no upstream dependencies and is the prerequisite for all other phases.

**Delivers:** Clean, verifiable baseline state. Canonical QUEEN.md with only seed entries. Empty or minimal pheromones.json. Reset COLONY_STATE.json. Archived XML system with no dead references. Stale constraints.json entries removed.

**Addresses:** Test data purge (P1), broken command audit (P1), stale state cleanup

**Avoids:** Pitfall 1 (seed data removal), Pitfall 3 (test cascade), Pitfall 4 (XML dead references), Pitfall 8 (stale cross-project COLONY_STATE), Pitfall 12 (XML-era constraints)

**Research flag:** Standard patterns — cleanup work is well-understood, no deep research needed

### Phase 2: Pheromone Integration Verification

**Rationale:** With clean data, the pheromone injection chain can be audited and verified end-to-end. The architectural approach (injection model via colony-prime) is correct; the gaps are in verification, error handling, and signal lifecycle management. This phase makes signals actually do what the docs claim.

**Delivers:** Verified signal flow from `/ant:focus` through to worker prompt context. Pheromone-expire wired into build lifecycle (not just continue). Decay math tested with known timestamps. Signal garbage collection preventing accumulation. Agent definitions updated to reference and acknowledge signals. All three agent definition locations kept in sync (claude/agents, aether/agents-claude, opencode/agents).

**Addresses:** Pheromone injection audit (P1), signal lifecycle management, decay math validation

**Avoids:** Pitfall 2 (write-only system), Pitfall 5 (decay math untested), Pitfall 7 (signal accumulation), Pitfall 9 (lock contention), Pitfall 10 (agent definition drift)

**Research flag:** Likely needs research-phase — signal injection across the prompt_section boundary involves multiple interacting components; validate the audit approach before executing

### Phase 3: Learning Pipeline End-to-End Validation

**Rationale:** Once signals work correctly and state is clean, the learning pipeline (observation -> learning -> instinct -> pheromone) can be validated with real data. This is Aether's primary differentiator over stateless competitors and should be verified before documentation is updated to describe it.

**Delivers:** End-to-end integration test following one observation through full pipeline. Real observation data in learning-observations.json. Verified that instincts appear in colony-prime output. Closed-loop confirmation that the learning system improves colony behavior.

**Addresses:** Learning pipeline e2e test (P2), closed-loop verification, real-world instinct validation

**Avoids:** Pitfall 1 (test data skewing pipeline results — must be clean from Phase 1 first)

**Research flag:** Standard patterns — pipeline components are individually tested; integration test follows established test patterns already in codebase

### Phase 4: Fresh Install Polish and Documentation Update

**Rationale:** Documentation must be updated last, after all verified behavior is confirmed. The fresh install flow can only be smoke-tested once cleanup (Phase 1) and pheromone integration (Phase 2) are complete. This phase also validates that the distribution pipeline does not ship polluted state.

**Delivers:** Automated smoke test for lay-eggs -> init -> plan -> build -> continue on a clean repo. Pre-publish validation step that rejects QUEEN.md test artifacts. Documentation updated to match verified behavior (not aspirational). `/ant:pheromones` behavior documented accurately (injection model, not independent worker reads).

**Addresses:** Fresh install smoke test (P1), documentation accuracy pass (P1), distribution pipeline validation

**Avoids:** Pitfall 6 (polluted state ships in npm), Pitfall 11 (docs promise features not working)

**Research flag:** Standard patterns — smoke testing and documentation are well-understood; pre-publish validation follows existing validate-package.sh patterns

### Phase 5: Secondary Features (P2 Items)

**Rationale:** After the core system is clean, verified, and documented, the P2 enhancements add real value without risk of obscuring underlying issues.

**Delivers:** `data-clean` command for safe artifact removal. XML exchange command surface (`/ant:export-signals`, `/ant:import-signals`). Error code standardization (BUG-004, -007, -008, -009, -010, -012). Commander.js upgrade to v14.

**Addresses:** All P2 features from FEATURES.md

**Research flag:** Standard patterns for data-clean and Commander upgrade; XML command surface may need brief research-phase to validate pheromone-xml.sh API surface

### Phase Ordering Rationale

- **Cleanup before integration:** Clean state is a prerequisite; integrating on polluted data produces misleading validation results
- **Integration before documentation:** Docs must describe verified behavior; updating docs mid-integration creates a second correction pass
- **Documentation before polish:** P2 features should be documented in the same pass as P1 behavior, not retrofitted separately
- **XML deferred from cleanup to secondary:** XML archival in Phase 1 removes the dead code; XML command surface in Phase 5 activates the exchange functionality if and only if the core system is solid

### Research Flags

Phases likely needing `/gsd:research-phase` during planning:
- **Phase 2 (Pheromone Integration):** Multiple interacting components across bash, playbooks, and agent definitions; the audit approach should be planned carefully before executing to avoid missing integration points or creating documentation that describes half-fixed behavior

Phases with standard patterns (skip research-phase):
- **Phase 1 (Cleanup):** File cleanup and state reset are well-understood; the risk is carefulness (see Pitfall 1), not knowledge gaps
- **Phase 3 (Learning Pipeline):** Components are individually tested; integration test follows existing test patterns in codebase
- **Phase 4 (Fresh Install + Docs):** Smoke testing and documentation update are well-understood; pre-publish validation extends an existing script
- **Phase 5 (Secondary Features):** Each item has a clear implementation path; may need brief API check for XML commands

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Existing stack is verified working; Ajv recommendation based on benchmarks and CJS compatibility; Commander v14/v15 ESM boundary verified |
| Features | HIGH | Based on deep codebase analysis; test pollution confirmed by file inspection; pheromone flow gaps confirmed by grep analysis of agent definitions |
| Architecture | HIGH | Primary source is the codebase itself; stigmergic pattern validated against multi-agent literature; component boundaries traced through actual code paths |
| Pitfalls | HIGH | Grounded in 390-line CONCERNS.md audit plus direct file inspection; moderate-confidence items sourced from multi-agent failure research that aligns with observed codebase issues |

**Overall confidence:** HIGH

### Gaps to Address

- **Worker signal acknowledgment mechanism:** Research identifies that workers should acknowledge which signals influenced their decisions, but the exact implementation (structured output field? prose reference in worker output?) is not specified. Resolve during Phase 2 planning.
- **constraints.json deprecation timeline:** The file is written by pheromone-write for backward compatibility and may be read by undiscovered consumers. A full audit of constraints.json readers is needed before deprecating. Handle during Phase 2 audit.
- **XML command surface scope:** pheromone-xml.sh exists and is tested but the public API surface for `/ant:export-signals` and `/ant:import-signals` is undefined. Needs brief design pass during Phase 5 planning.
- **Lock contention under real parallel load:** Pitfall 9 identifies the theoretical problem; the actual severity depends on how many workers emit pheromones simultaneously in practice. Validate during Phase 2 with a multi-worker test before deciding whether batching is required.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis: `aether-utils.sh`, `pheromones.json`, `COLONY_STATE.json`, `QUEEN.md`, `constraints.json`, `learning-observations.json` — test pollution confirmed by inspection
- Direct codebase analysis: `.claude/agents/ant/*.md` — zero pheromone references in agent definitions confirmed
- Direct codebase analysis: `build-context.md`, `build-wave.md`, `continue-advance.md` — signal injection chain traced
- Direct codebase analysis: `.aether/exchange/*.sh` — XML system confirmed built and untested from commands
- `.planning/codebase/CONCERNS.md` — 390-line codebase audit identifying all technical debt

### Secondary (MEDIUM confidence)
- [Commander.js npm](https://www.npmjs.com/package/commander) — v14 latest stable, v15 ESM-only May 2026
- [Ajv npm](https://www.npmjs.com/package/ajv) + [Ajv docs](https://ajv.js.org/) — v8.18.0 latest, CJS-compatible, 14M ops/sec benchmark
- [bats-core GitHub](https://github.com/bats-core/bats-core) — v1.13.0, Bash 3.2+ compatible
- [Multi-Agent Orchestration Patterns 2026](https://www.ai-agentsplus.com/blog/multi-agent-orchestration-patterns-2026) — hierarchical decomposition, generator-critic patterns
- [Stigmergy (Wikipedia)](https://en.wikipedia.org/wiki/Stigmergy) — foundational pattern validation
- [Multi-Agent System Reliability Failure Patterns](https://www.getmaxim.ai/articles/multi-agent-system-reliability-failure-patterns-root-causes-and-production-validation-strategies/) — 79% of failures from specification issues, validates top-down integration approach

### Tertiary (LOW confidence — verify before acting)
- [Event-Driven Architecture pitfalls](https://medium.com/wix-engineering/event-driven-architecture-5-pitfalls-to-avoid-b3ebf885bdb1) — context propagation patterns; full content not verified
- [Why Multi-Agent Systems Fail](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) — error amplification without coordination topology; general pattern, not Aether-specific

---
*Research completed: 2026-03-19*
*Ready for roadmap: yes*
