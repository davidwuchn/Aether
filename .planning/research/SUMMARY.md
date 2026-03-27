# Project Research Summary

**Project:** Aether v2.4 Living Wisdom
**Domain:** Multi-agent colony orchestration -- wisdom accumulation, agent definition gaps, model routing
**Researched:** 2026-03-27
**Confidence:** HIGH

## Executive Summary

Aether's wisdom system is fully coded (~3,400 lines across queen.sh, hive.sh, learning.sh, pheromone.sh) with 580+ passing tests, but it is effectively dead: QUEEN.md sections show placeholder text, the hive brain stays empty, and instincts never accumulate beyond 0-1 per colony. The root cause is not a bug but a chain of soft dependencies -- builder agents silently skip learning extraction, hive promotion only fires at seal (which users rarely run), and there are no deterministic fallbacks. The fix is surgical: wire existing functions into the build/continue playbooks at specific points, add fallback extraction logic, and make hive promotion continuous rather than end-of-life-only.

Two agent castes (Oracle and Architect) are referenced in docs and the Queen's workflow patterns but have no dedicated agent definition files. Oracle runs only as a slash command wizard, and Architect was supposedly merged into Keeper but the merge is incomplete -- neither Keeper nor Route-Setter covers the Architect's original design-decision role. Both need proper agent `.md` files with opus model routing, plus mirror files for OpenCode and packaging.

The PITFALLS.md research covers a tangential but adjacent concern: per-caste model routing. While model routing is out of scope for v2.4 per PROJECT.md ("Per-worker model routing via env vars" is explicitly out of scope), the pitfalls are relevant because the new Oracle and Architect agents need opus model slots, and the existing model infrastructure (model-profiles.yaml, dual parsers, 184 hardcoded test assertions) creates friction. The recommended approach: use Claude Code's native `"opus"` and `"sonnet"` model slot names in Task tool calls, let the LiteLLM proxy handle GLM mapping, and do NOT attempt env var passing (proven not to work in v1).

## Key Findings

### Recommended Stack (from STACK.md)

No new npm packages or languages. All changes are bash playbook modifications and markdown agent definition files.

**Core changes:**
- **6 new agent `.md` files** (Oracle + Architect, each with Claude, OpenCode, and packaging mirrors) -- fills the two missing caste definitions with proper model routing
- **continue-advance.md modifications** -- add Step 2.6 (queen-write-learnings) and Step 3d (hive-promote) to wire the wisdom pipeline into the continue flow
- **build-complete.md fallback** -- add deterministic pattern extraction from git diff + test results when builders skip learning output
- **colony-name subcommand** -- trivial 10-line addition to aether-utils.sh to DRY up colony name extraction (currently repeated 6+ times via inline jq)
- **Do NOT modify** queen.sh, hive.sh, learning.sh, pheromone.sh, or existing agent definitions -- all existing code works correctly; the problem is in the calling code

### Expected Features (from FEATURES.md)

**Must have (table stakes):**
- T1: Builders report learnings during builds -- the linchpin; without this the entire pipeline starves
- T2: QUEEN.md Build Learnings auto-populates -- code exists, just needs to be called
- T3: Instincts accumulate reliably from phase patterns -- needs stronger enforcement in continue-advance
- T6: Oracle agent definition file -- fills critical gap in agent roster
- T7: Architect agent definition file -- fills gap; distinct from Keeper (curation) and Route-Setter (planning)

**Should have (competitive differentiators):**
- D1: Wisdom growth visible in build output -- makes the learning loop tangible
- D2: Cross-colony wisdom flows into new colonies -- emergent once hive is populated
- D5: `/ant:wisdom` status command -- observability for accumulated wisdom
- D6: Phase completion auto-promotes learnings -- every phase should produce at least one QUEEN.md entry

**Defer (v2.5+):**
- D3: Oracle spawnable during builds (requires build-wave Deep Research pattern changes)
- D4: Architect in planning (requires route-setter integration)
- Automatic wisdom pruning/decay (removes user knowledge without consent)
- Real-time wisdom injection mid-build (inconsistent context within a phase)
- Wisdom sharing between users (privacy violation -- hive is machine-local by design)

### Architecture Approach (from ARCHITECTURE.md)

The wisdom system has three layers, all with working infrastructure but disconnected from the build/continue flow:

1. **QUEEN.md layer** -- queen-write-learnings and queen-promote-instinct work correctly but are never called from continue-advance (except queen-promote-instinct for confidence >= 0.8 instincts). Fix: add Step 2.6 to call queen-write-learnings after learning extraction.
2. **Hive brain layer** -- hive-promote works correctly but only fires during seal. Fix: add Step 3d to continue-advance to promote high-confidence instincts to hive continuously during colony work.
3. **Instinct layer** -- instinct-create works but instincts are trivialized or skipped by the LLM executing continue. Fix: stronger prompting or structural enforcement in continue-advance.

The proposed flow adds two new steps to continue-advance.md: Step 2.6 (queen-write-learnings after extraction) and Step 3d (hive-promote after instinct promotion). Both are non-blocking. Seal becomes a catch-up pass rather than the sole promotion point.

Agent files follow the established 22-file pattern (frontmatter + role + execution_flow + pheromone_protocol + return_format). Oracle gets Write tool (can persist findings) and WebSearch/WebFetch; Architect is read-only like Scout (designs via return JSON, not files).

### Critical Pitfalls (from PITFALLS.md)

While PITFALLS.md focuses on per-caste model routing (a broader v2.3 concern), several pitfalls are directly relevant to v2.4:

1. **Oracle/Architect agents need proper model slots** -- The Task tool's `model` parameter must be used (not env vars, proven not to work in v1 archive). Reasoning castes get `"opus"`, execution castes get `"sonnet"`. The new agents should use slot names, not GLM-specific names.
2. **184 hardcoded model names in tests** -- Any change to model-profiles.yaml will break 184 test assertions across 6 files. Must centralize test mocks before any YAML changes. For v2.4, if agent files reference model slots, this matters less (agents don't change the YAML) but remains a ticking time bomb.
3. **GLM-5 looping on reasoning castes** -- Oracle (opus slot = GLM-5) is a reasoning caste susceptible to looping. The proxy constraints (temperature 0.4, max_tokens 2500) help but can be escaped. The Oracle agent definition should include explicit termination conditions in its prompt.
4. **Dual YAML parsers (bash awk vs Node.js yaml.load)** -- Can return different results for the same config. Standardize on Node.js for any model resolution.
5. **Config swap fragility** -- Aether should always pass model slot names (`"opus"`/`"sonnet"`) and let the runtime (Claude API or LiteLLM proxy) handle mapping. Do NOT detect active mode.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Agent Definitions (Oracle + Architect)

**Rationale:** These are independent of the wisdom pipeline changes and can be built first. They fill documented gaps in the agent roster and establish the pattern for the two new opus-slot agents. No existing code needs modification -- purely additive work.

**Delivers:** 6 new agent `.md` files (Oracle and Architect for Claude, OpenCode, and packaging mirrors), updated workers.md and CLAUDE.md counts (22 -> 24 agents).

**Addresses:** T6 (Oracle agent file), T7 (Architect agent file)

**Avoids:** Pitfall 9 (archive confusion -- do not reference archived routing implementation), Pitfall 11 (use `"opus"` not `"inherit"`)

**Research flags:** Standard patterns -- all 22 existing agents follow an identical structure. No research needed.

### Phase 2: Wisdom Pipeline Wiring

**Rationale:** The core value of the milestone. This phase connects existing functions (queen-write-learnings, hive-promote) into the continue-advance flow. Two surgical additions to continue-advance.md: Step 2.6 (persist learnings to QUEEN.md) and Step 3d (promote to hive brain). Both are non-blocking.

**Delivers:** QUEEN.md Build Learnings section populates after each phase, hive brain receives entries during colony work (not just at seal), visible wisdom growth in continue output.

**Addresses:** T2 (QUEEN.md learnings), T4 (instinct promotion), T5 (hive brain on seal), D1 (visible wisdom growth), D6 (auto-promote learnings)

**Uses:** Existing queen.sh, hive.sh, learning.sh functions -- no code changes to these files

**Implements:** The "living" part of "Living Wisdom" -- wisdom that accumulates as colonies work

**Avoids:** Anti-pattern 1 (don't call queen-write-learnings before extraction), Anti-pattern 2 (keep hive-promote non-blocking)

**Research flags:** Moderate research needed -- the continue-advance.md playbook is 434 lines and the new steps must integrate cleanly with the existing Step 2/3 flow. Should verify the exact insertion points and ensure no step numbering conflicts.

### Phase 3: Builder Learning Extraction (Deterministic Fallback)

**Rationale:** This is the linchpin (T1) but depends on Phase 2 being in place -- there is no point extracting learnings if the pipeline to persist them is not wired. This phase adds deterministic fallback extraction to build-complete.md so that even when builders skip learning output, wisdom still accumulates from git diff analysis.

**Delivers:** Fallback pattern extraction in build-complete.md, colony-name subcommand in aether-utils.sh, guaranteed learning data flowing into the pipeline.

**Addresses:** T1 (builders report learnings), T3 (instincts accumulate)

**Avoids:** Root cause #1 (builder synthesis JSON missing learning.patterns_observed), Root cause #4 (colony name extraction failure)

**Research flags:** Needs research -- the fallback extraction logic (what to extract from git diff, how to structure it) is not well-defined. The quality of deterministic extraction vs AI extraction is unknown and needs validation.

### Phase 4: Wisdom Observability + Validation

**Rationale:** After Phases 1-3, the wisdom system should be working. This phase adds the `/ant:wisdom` status command for observability, runs end-to-end integration tests, and validates the full flow (colony work -> observations -> instincts -> QUEEN.md -> hive -> future colonies).

**Delivers:** `/ant:wisdom` command, integration test suite, documented verification that wisdom flows correctly.

**Addresses:** D5 (wisdom status command), D2 (cross-colony wisdom -- emergent, needs verification)

**Research flags:** Standard patterns -- `/ant:wisdom` assembles data from existing subcommands. Integration test follows existing colony lifecycle test patterns.

### Phase Ordering Rationale

- Phase 1 (agents) is independent and establishes the 24-agent roster before docs reference it
- Phase 2 (pipeline wiring) is the highest-value change and must come before Phase 3 (no point extracting learnings without a pipeline to persist them)
- Phase 3 (builder fallback) depends on Phase 2 and is the most uncertain work (quality of deterministic extraction is unvalidated)
- Phase 4 (observability) is low-risk cleanup that validates everything works end-to-end
- This ordering avoids the anti-pattern of building extraction before persistence, and ensures the pipeline is wired before we try to push data through it

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** Fallback learning extraction quality is unvalidated. How good is git-diff-based extraction compared to AI extraction? Needs a spike or prototype.
- **Phase 2:** continue-advance.md integration points need precise mapping. The playbook is complex and step numbering must be verified.

Phases with standard patterns (skip research-phase):
- **Phase 1:** Agent definition files follow an identical 22-file pattern. Well-documented, low risk.
- **Phase 4:** Slash command and integration test patterns are well-established in the codebase.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All findings from direct codebase analysis. No new dependencies. Existing functions verified working. |
| Features | HIGH | Feature research grounded in codebase analysis plus user testing feedback. Dependency graph is clear. |
| Architecture | HIGH | All utility functions traced end-to-end. Data flow diagram verified against actual code paths. |
| Pitfalls | HIGH | Direct codebase inspection of 184 test assertions, archived routing system, dual parsers, proxy config. |

**Overall confidence:** HIGH

### Gaps to Address

- **Builder learning extraction quality:** We know builders skip learning output, and we know the fix (deterministic fallback), but we do not know how good git-diff-based extraction will be at producing meaningful learnings. This is the biggest uncertainty. Mitigation: Phase 3 should include a prototype/evaluation step before full integration.
- **Instinct quality from LLM extraction:** The research notes that LLMs executing `/ant:continue` often "trivialize" instincts. Stronger prompting in continue-advance.md may help, but this is a behavioral problem (LLM quality) not a code problem. Mitigation: add structural enforcement (e.g., minimum instinct length, required format) and validate in Phase 4.
- **Hive emptiness is a chicken-and-egg problem:** Until colonies complete with hive promotion enabled, the hive stays empty, which means colony-prime has nothing to inject. The first colony to use the new pipeline will not benefit from cross-colony wisdom. Mitigation: accept this as expected; document that the system "warms up" after the first colony seals.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis: queen.sh (1,242 lines), hive.sh (562 lines), learning.sh (1,553 lines), pheromone.sh colony-prime (~700 lines)
- Playbook analysis: build-complete.md (350 lines), continue-advance.md (434 lines), seal.md (lines 290-337), build-wave.md (598 lines)
- Agent definition analysis: all 22 existing agents in `.claude/agents/ant/`
- workers.md caste table and model slot assignments
- model-profiles.yaml, model-profiles.js (446 lines), model-verify.js (289 lines)
- `.aether/archive/model-routing/README.md` -- explains why v1 routing failed
- `.claude/get-shit-done/references/model-profile-resolution.md` -- GSD working pattern for model slots
- `.planning/PROJECT.md` -- milestone v2.4 scope and constraints

### Secondary (MEDIUM confidence)
- User testing feedback documented in PROJECT.md: "QUEEN.md and hive brain are template-only -- never populated with real data"
- Existing QUEEN.md in Aether repo: 1 instinct, 6 patterns, 1 build learning from ~20 phases of work (evidence that current system barely works)

### Tertiary (LOW confidence)
- Whether builders can reliably produce high-quality learning observations (needs validation in Phase 3)
- Quality of deterministic git-diff-based extraction as a fallback (untested hypothesis)

---
*Research completed: 2026-03-27*
*Ready for roadmap: yes*
