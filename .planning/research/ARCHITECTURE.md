# Architecture Research: Living Wisdom Pipeline and Dedicated Agent Files

**Domain:** Aether colony wisdom system integration (QUEEN.md, hive brain, instincts, Oracle/Architect agents)
**Researched:** 2026-03-27
**Confidence:** HIGH (based on direct codebase analysis of all utility modules, playbooks, and agent definitions)

---

## Executive Summary

The Aether wisdom system has three layers that already have working infrastructure but are disconnected from the actual build/continue flow:

1. **QUEEN.md wisdom** -- `_queen_promote()`, `_queen_write_learnings()`, and `_queen_promote_instinct()` all work correctly in isolation. The problem: nothing in the build or continue playbooks calls `queen-write-learnings` to persist build learnings, and `queen-promote-instinct` is only called in continue-advance.md Step 3c for high-confidence instincts.

2. **Hive brain** -- `_hive_promote()` (abstract + store) works correctly. The problem: it is ONLY called during `/ant:seal` Step 3.7. No wisdom flows to the hive during normal colony work -- only at the very end of a colony's life.

3. **Instincts** -- `_instinct_create()` and `_instinct_apply()` work correctly. The problem: `continue-advance.md` Step 3 extracts instincts from phase patterns, but these instincts only live in `COLONY_STATE.json`. They do not flow to QUEEN.md (except confidence >= 0.8 via Step 3c) and do not flow to hive (except at seal).

The core gap: **build learnings are extracted but never written to QUEEN.md**, and **hive only gets populated at seal time rather than continuously during colony work**.

Additionally, two castes referenced in the system have no dedicated agent files:
- **Oracle** -- referenced in Queen's workflow patterns ("Deep Research" pattern), `/ant:oracle` command exists but spawns an ad-hoc RALF loop, not a proper agent definition
- **Architect** -- referenced in CLAUDE.md as a caste, workers.md lists it with personality traits, but no agent file exists

---

## Part 1: Oracle and Architect Agent Files

### Current State

**Oracle:**
- `/ant:oracle` command exists at `.claude/commands/ant/oracle.md` -- a 200+ line command handler that configures and launches a "RALF iterative loop" (Research-Analyze-Learn-Focus)
- The command writes to `.aether/oracle/` and never touches colony state (non-invasive guarantee)
- Queen agent references "Oracle-led" in the "Deep Research" pattern
- NO agent definition file exists at `.claude/agents/ant/aether-oracle.md`
- Oracle is listed in workers.md caste table and model routing table (slot: opus)

**Architect:**
- Referenced in CLAUDE.md as a caste with personality traits ("Systematic", "Pattern-focused")
- Referenced in workers.md with personality "Systematic", "Pattern-focused"
- Listed in model routing table (slot: opus)
- NO agent definition file exists at `.claude/agents/ant/aether-architect.md`
- No `/ant:architect` command exists
- The Route-Setter agent (`aether-route-setter.md`) partially covers planning/decomposition duties

### What Agent Files Need

Based on analysis of the 22 existing agent definitions, each follows a consistent structure:

```
---
name: aether-{caste}
description: "{when to use this agent}"
tools: {tool list}
color: {color}
model: {opus|sonnet|haiku}
---

<role> -- who the agent is
<glm_safety> -- GLM-5 loop risk warning (for opus slot only)
<execution_flow> -- step-by-step workflow
<critical_rules> -- non-negotiable rules
<pheromone_protocol> -- how to respond to injected signals
<return_format> -- structured JSON output schema
<success_criteria> -- self-check before completion
<failure_modes> -- tiered failure handling
<escalation> -- when to escalate
<boundaries> -- what the agent can/cannot do
```

### Oracle Agent Design

**File:** `.claude/agents/ant/aether-oracle.md`

Based on the existing oracle.md command handler and the Scout agent pattern:

- **Model:** opus (deep research needs full reasoning)
- **Tools:** Read, Write, Edit, Bash, Grep, Glob, WebSearch, WebFetch
- **Key difference from Scout:** Oracle can write findings to disk (to `.aether/oracle/`), has access to Bash for running research tools, and does multi-round iterative research (the RALF loop). Scout is read-only, single-pass, quick lookup.
- **Spawning context:** Queen's "Deep Research" pattern spawns Oracle. The oracle.md command handler may also be updated to use the agent definition for consistency.

**Return format should include:**
```json
{
  "ant_name": "{name}",
  "caste": "oracle",
  "status": "completed",
  "summary": "Research synthesis",
  "findings": [...],
  "confidence": "high|medium|low",
  "sources": [...],
  "gaps": [...],
  "recommendations": [...]
}
```

### Architect Agent Design

**File:** `.claude/agents/ant/aether-architect.md`

The Architect caste is distinct from Route-Setter:
- **Route-Setter** decomposes goals into phases and tasks (tactical planning)
- **Architect** designs system structure, component boundaries, data flow, and integration points (strategic design)

Based on the existing Route-Setter agent:

- **Model:** opus (system design needs deep reasoning)
- **Tools:** Read, Grep, Glob, Bash
- **No Write/Edit** -- Architect produces design documents as structured return JSON, not files. If a design needs to be persisted, route to Keeper or Chronicler.
- **Spawning context:** Not currently spawned by any command. This is a new integration point. Architect could be spawned during `/ant:plan` for complex projects, or during `/ant:build` for phases requiring system design.

**Return format should include:**
```json
{
  "ant_name": "{name}",
  "caste": "architect",
  "status": "completed",
  "summary": "Architecture analysis",
  "component_boundaries": [...],
  "data_flow": "...",
  "integration_points": [...],
  "tradeoffs": [...],
  "recommendations": [...]
}
```

### Mirror Files Required

Each new agent file needs mirrors in:
- `.aether/agents-claude/aether-oracle.md` (packaging mirror, byte-identical)
- `.aether/agents-claude/aether-architect.md` (packaging mirror, byte-identical)
- `.opencode/agents/aether-oracle.md` (structural parity)
- `.opencode/agents/aether-architect.md` (structural parity)

---

## Part 2: Wisdom Pipeline Integration Analysis

### Current Wisdom Flow (What Exists)

```
BUILD PHASE:
  Worker (Builder) executes task
       |
       v
  build-complete.md Step 5.9: Synthesize results
    - Collects learning.patterns_observed from worker output
    - Calls memory-capture for up to 2 patterns (line 58)
    - memory-capture calls learning-observe (records observation)
    - memory-capture calls learning-promote-auto (if threshold met)
    - memory-capture calls pheromone-write (auto-emits FEEDBACK)
       |
       v
  learning-observations.json -- observations accumulate with content_hash

CONTINUE PHASE:
  continue-advance.md Step 2: Extract learnings
    - Appends to memory.phase_learnings in COLONY_STATE.json
    - Step 2.5: Calls memory-capture for each learning (re-records observation)
    - Step 3: Creates instincts from patterns (instinct-create)
    - Step 3c: Promotes confidence >= 0.8 instincts to QUEEN.md (queen-promote-instinct)
    - Step 2.1.5: Checks for promotion proposals (learning-check-promotion)

SEAL PHASE:
  seal.md Step 3.7: Hive promotion
    - Reads high-confidence instincts (>= 0.8)
    - Calls hive-promote for each (abstract + store to ~/.aether/hive/wisdom.json)
```

### What's Missing (The Gaps)

#### Gap 1: Build learnings never reach QUEEN.md

`queen-write-learnings()` exists and works. It takes phase_id, phase_name, and a JSON array of learnings. It writes directly to the QUEEN.md Build Learnings section, bypassing observation thresholds. **But nothing calls it.**

The continue-advance.md Step 2 extracts learnings into COLONY_STATE.json (line 24-39) and calls memory-capture (Step 2.5, line 52-73), but does NOT call `queen-write-learnings`. The learnings only exist in COLONY_STATE.json and learning-observations.json.

**Fix:** After extracting learnings in continue-advance.md Step 2, call `queen-write-learnings` to persist them to QUEEN.md.

#### Gap 2: Hive brain only populated at seal time

`hive-promote()` works correctly. It abstracts repo-specific instincts into generalized wisdom and stores them in `~/.aether/hive/wisdom.json`. But it is ONLY called in seal.md Step 3.7.

During normal colony work, instincts accumulate in COLONY_STATE.json but never flow to the hive until the colony is sealed. If a colony is never sealed (common in development), wisdom is lost.

**Fix:** Add hive-promote calls during continue-advance.md Step 3c (when high-confidence instincts are promoted to QUEEN.md, also promote to hive). This makes hive population continuous rather than end-of-life only.

#### Gap 3: QUEEN.md Build Learnings section stays empty

The `_queen_write_learnings()` function (queen.sh lines 645-817) is fully implemented with:
- Phase subsection headers (### Phase N: name)
- Deduplication (skips if claim already in QUEEN.md)
- Metadata updates (total_build_learnings count, last_evolved timestamp)
- Evolution log entries

But nothing in the build or continue flow calls it. The Build Learnings section always shows the placeholder: "*No build learnings recorded yet.*"

**Fix:** Call `queen-write-learnings` in continue-advance.md after learning extraction.

#### Gap 4: Hive wisdom retrieval in colony-prime works but hive is empty

`colony-prime()` (pheromone.sh lines 1074-1133) correctly reads hive wisdom via `hive-read` and injects it into worker prompts. The code works, but because hive-promote only fires at seal, the hive is usually empty for active colonies.

**Fix:** Making hive-promote continuous (Gap 2 fix) automatically fixes this. Future colonies will benefit from accumulated wisdom.

### Proposed Complete Wisdom Flow

```
BUILD PHASE (existing, no changes needed):
  Worker (Builder) executes task
       |
       v
  build-complete.md Step 5.9: Synthesize results
    - Collects learning.patterns_observed from worker output
    - Calls memory-capture for up to 2 patterns
    - memory-capture -> learning-observe -> learning-observations.json
    - memory-capture -> learning-promote-auto (if threshold met -> queen-promote)
    - memory-capture -> pheromone-write (auto-emits FEEDBACK)

CONTINUE PHASE (CHANGES HERE):
  continue-advance.md Step 2: Extract learnings
    - Append to memory.phase_learnings in COLONY_STATE.json     [EXISTING]
    - Step 2.5: Call memory-capture for each learning             [EXISTING]
       |
       v
    *** NEW: Step 2.6: Persist learnings to QUEEN.md ***
    - Call queen-write-learnings with extracted learnings         [NEW]
    - This writes to Build Learnings section of .aether/QUEEN.md
    - Deduplication handled internally
       |
       v
  continue-advance.md Step 3: Create instincts
    - Step 3/3a/3b: Create instincts from patterns               [EXISTING]
       |
       v
    Step 3c: Promote high-confidence instincts to QUEEN.md
    - queen-promote-instinct for confidence >= 0.8                [EXISTING]
       |
       v
    *** NEW: Step 3d: Promote to hive brain ***
    - For each newly promoted instinct (confidence >= 0.8):
      Call hive-promote --text "When {trigger}: {action}"       [NEW]
        --source-repo "$PWD" --confidence {conf}
        --domain "$domain_tags"
    - This abstracts repo-specific text and stores in hive
    - NON-BLOCKING (failure logged, continue proceeds)

SEAL PHASE (existing, no changes needed):
  seal.md Step 3.7: Hive promotion
    - Reads high-confidence instincts
    - Calls hive-promote for each
    - NOW: most instincts already in hive from Step 3d above
    - Seal becomes a catch-up pass for any that slipped through

FUTURE COLONIES (automatic, no changes needed):
  colony-prime Step 4 (build-context.md)
    - Reads hive wisdom via hive-read
    - Injects into worker prompts as "HIVE WISDOM (Domain: ...)"
    - Workers receive accumulated cross-colony patterns
```

### Data Flow Diagram

```
                         BUILD
                           |
                    Worker returns
                    patterns_observed
                           |
                    memory-capture
                    /      |       \
                   v       v        v
         learning-    pheromone-  learning-
         observe      write      promote-auto
           |          (FEEDBACK)  (if threshold)
           v
    learning-observations.json
   (observations accumulate)


                         CONTINUE
                           |
              Extract learnings (Step 2)
              Append to COLONY_STATE.json
                           |
                    memory-capture
                    (Step 2.5 - re-records)
                           |
              +---------- NEW ----------+
              |  queen-write-learnings   |  --> .aether/QUEEN.md
              |  (Step 2.6)              |      (Build Learnings section)
              +--------------------------+
                           |
              Extract instincts (Step 3)
              instinct-create
                           |
              queen-promote-instinct (Step 3c)
              (confidence >= 0.8)
                           |
              +---------- NEW ----------+
              |  hive-promote            |  --> ~/.aether/hive/wisdom.json
              |  (Step 3d)               |      (cross-colony patterns)
              +--------------------------+
                           |
              learning-check-promotion (Step 2.1.5)
              (tick-to-approve UX for user review)


                         SEAL
                           |
              hive-promote (Step 3.7)
              (catch-up pass)
              +--------+
              |  hive  | --> ~/.aether/hive/wisdom.json
              +--------+


                         FUTURE BUILD
                           |
              colony-prime (Step 4)
              hive-read (domain-scoped)
                           |
              Inject "HIVE WISDOM" into worker prompts
              Workers receive accumulated patterns
```

---

## Part 3: Integration Points

### New Components

| Component | File | Purpose |
|-----------|------|---------|
| Oracle agent definition | `.claude/agents/ant/aether-oracle.md` | Deep research worker |
| Architect agent definition | `.claude/agents/ant/aether-architect.md` | System design worker |
| Oracle mirror | `.aether/agents-claude/aether-oracle.md` | Packaging mirror |
| Architect mirror | `.aether/agents-claude/aether-architect.md` | Packaging mirror |
| Oracle OpenCode | `.opencode/agents/aether-oracle.md` | OpenCode structural parity |
| Architect OpenCode | `.opencode/agents/aether-architect.md` | OpenCode structural parity |

### Modified Components

| Component | File | Change |
|-----------|------|--------|
| continue-advance.md | `.aether/docs/command-playbooks/continue-advance.md` | Add Step 2.6 (queen-write-learnings) and Step 3d (hive-promote) |
| workers.md | `.aether/workers.md` | Add Oracle and Architect to caste table, update count |
| CLAUDE.md | `CLAUDE.md` | Update agent count (22 -> 24), add Oracle/Architect to agent table |
| queen.sh | `.aether/utils/queen.sh` | No code changes needed (functions already exist) |
| hive.sh | `.aether/utils/hive.sh` | No code changes needed (functions already exist) |
| learning.sh | `.aether/utils/learning.sh` | No code changes needed (functions already exist) |
| pheromone.sh | `.aether/utils/pheromone.sh` | No code changes needed (colony-prime already reads hive) |

### NOT Modified

| Component | Why |
|-----------|-----|
| aether-utils.sh (dispatcher) | No new subcommands needed -- existing queen-write-learnings, hive-promote, etc. are sufficient |
| build-complete.md | Already calls memory-capture correctly; learnings flow to continue phase |
| build-wave.md | No changes to worker spawning |
| build-context.md | No changes to colony-prime context loading |
| seal.md | Already calls hive-promote; the new Step 3d in continue just makes it a catch-up |

---

## Part 4: Build Order

Dependencies determine the order:

### Phase A: Agent Files (no dependencies on each other, can be parallel)

1. **Create `aether-oracle.md`** -- agent definition following the 22-file pattern
2. **Create `aether-architect.md`** -- agent definition following the 22-file pattern
3. **Create mirror files** -- 4 mirrors total (2 Claude, 2 OpenCode)
4. **Update workers.md** -- add to caste table, update count to 24
5. **Update CLAUDE.md** -- update agent count and table

### Phase B: Wisdom Pipeline (depends on Phase A being complete, since docs reference agent count)

1. **Add Step 2.6 to continue-advance.md** -- call `queen-write-learnings` after learning extraction
2. **Add Step 3d to continue-advance.md** -- call `hive-promote` for high-confidence instincts
3. **Test end-to-end flow**: run a colony, complete phases, verify QUEEN.md gets populated, verify hive gets entries

### Phase C: Validation

1. **Write tests** for the new continue-advance steps (queen-write-learnings called with correct args, hive-promote called with correct args)
2. **Integration test**: simulate a colony lifecycle (init -> plan -> build -> continue -> seal) and verify:
   - QUEEN.md Build Learnings section has entries
   - QUEEN.md Instincts section has entries (existing behavior)
   - `~/.aether/hive/wisdom.json` has entries before seal (new behavior)
   - colony-prime injects hive wisdom into future build worker prompts

### Dependency Graph

```
Phase A (Agent Files)     Phase B (Pipeline)
  oracle.md                   |
  architect.md                |
  mirrors                    |
  workers.md update          |
  CLAUDE.md update           |
        |                    |
        v                    v
    [done]  ----------->  continue-advance.md
                             Step 2.6 + 3d
                                  |
                                  v
                           Phase C (Tests)
                             integration test
```

Phase A and Phase B can be partially parallelized if the docs update (workers.md, CLAUDE.md) is deferred to the end. The core agent files and pipeline changes are independent.

---

## Part 5: Anti-Patterns

### Anti-Pattern 1: Calling queen-write-learnings BEFORE learning extraction completes

The continue-advance.md Step 2 extracts learnings first, then Step 2.6 should write them. Do NOT reverse this order -- learnings must be extracted and validated before writing to QUEEN.md.

### Anti-Pattern 2: Making hive-promote blocking

The seal.md already establishes the pattern: hive-promote is NON-BLOCKING. If it fails, log and continue. The new Step 3d in continue-advance must follow this same pattern. Phase advancement must never fail due to hive promotion errors.

### Anti-Pattern 3: Duplicating existing pheromone_protocol in new agent files

Copy the `<pheromone_protocol>` section verbatim from an existing agent (e.g., Builder or Scout). Do NOT rewrite it. The protocol is standardized and must be identical across all agents.

### Anti-Pattern 4: Giving Architect Write/Edit tools

Architect is a design role, not an implementation role. Like Scout, it should produce structured return JSON, not files. If a design needs to be persisted, the caller (Queen or Route-Setter) should spawn a Keeper or Chronicler.

### Anti-Pattern 5: Over-engineering the Oracle agent

The existing `/ant:oracle` command handler is 200+ lines of RALF loop logic. The agent definition should NOT duplicate this logic. The agent should focus on the research behavior and return format. The RALF loop orchestration stays in the command handler.

---

## Part 6: Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| queen-write-learnings corrupts QUEEN.md | Low | High | Function has dedup, atomic write, and backup. Tested. |
| hive-promote fails during continue | Medium | Low | NON-BLOCKING pattern. Failure logged, continue proceeds. |
| New agent files break packaging | Medium | Medium | Run `bin/validate-package.sh` after creating mirrors. |
| Token budget exceeded by wisdom injection | Low | Medium | colony-prime has 8K/4K budget with trim order. Hive wisdom already included. |
| Duplicate hive entries from Step 3d + seal | Low | Low | hive-store deduplicates by content hash. Same entry = skip. |
| OpenCode agent parity broken | Low | Low | Structural parity, not byte-identical. Review manually. |

---

## Sources

- HIGH confidence: Direct analysis of `.aether/utils/queen.sh` (1242 lines) -- all functions fully traced
- HIGH confidence: Direct analysis of `.aether/utils/hive.sh` (562 lines) -- all functions fully traced
- HIGH confidence: Direct analysis of `.aether/utils/learning.sh` (1553 lines) -- all functions fully traced
- HIGH confidence: Direct analysis of `.aether/utils/pheromone.sh` `_colony_prime()` function (lines 735-1284) -- full context assembly traced
- HIGH confidence: Direct analysis of `.aether/docs/command-playbooks/build-complete.md` (350 lines) -- learning extraction traced
- HIGH confidence: Direct analysis of `.aether/docs/command-playbooks/continue-advance.md` (434 lines) -- instinct promotion and pheromone emission traced
- HIGH confidence: Direct analysis of `.claude/commands/ant/seal.md` (lines 290-337) -- hive promotion at seal traced
- HIGH confidence: Direct analysis of `.claude/commands/ant/oracle.md` (first 50 lines) -- RALF loop command handler
- HIGH confidence: Pattern analysis of 22 existing agent definitions -- consistent structure confirmed
- HIGH confidence: `.planning/PROJECT.md` -- milestone v2.4 scope and requirements

---

*Architecture research for: Aether v2.4 Living Wisdom*
*Researched: 2026-03-27*
