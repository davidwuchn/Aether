# Feature Research: Aether v2.4 Living Wisdom

**Domain:** Multi-agent colony orchestration -- making wisdom systems actually learn from colony work
**Researched:** 2026-03-27
**Confidence:** HIGH (all research from codebase analysis -- queen.sh, hive.sh, learning.sh, pheromone.sh, build/continue playbooks, seal playbook, agent definitions)

---

## Context

Aether's wisdom system exists as substantial, well-tested code (~2,400 lines across queen.sh, hive.sh, learning.sh, and pheromone.sh) but user testing revealed these are dead features -- the code paths are never triggered during actual colony work. The pipeline exists: observation -> instinct -> QUEEN.md -> hive brain -> future colonies. The plumbing is built. But the spigots are closed.

The milestone must connect three things:
1. **QUEEN.md population** -- build learnings, instincts, and codebase patterns must flow in automatically during builds
2. **Hive brain accumulation** -- cross-colony patterns must actually be promoted during seal
3. **Dedicated Oracle/Architect agents** -- missing agent .md files mean these castes have no proper model routing, no defined behavior, and no consistent invocation pattern

### Current State of the Wisdom Pipeline

**What exists (code complete):**
- `queen-write-learnings` -- writes phase learnings to QUEEN.md Build Learnings section (bypasses thresholds)
- `queen-promote-instinct` -- promotes confidence >= 0.8 instincts to QUEEN.md Instincts section
- `queen-promote` -- promotes threshold-meeting observations to QUEEN.md Codebase Patterns
- `queen-seed-from-hive` -- seeds QUEEN.md from cross-colony hive wisdom (called in build-context.md)
- `instinct-create` -- creates instincts in COLONY_STATE.json with dedup, confidence tracking, 30-cap
- `instinct-apply` -- records when instinct was used in practice (success/failure feedback)
- `learning-observe` -- records observation of a learning across colonies (threshold tracking)
- `learning-promote-auto` -- auto-promotes high-confidence learnings via recurrence policy
- `hive-promote` -- orchestrates abstract + store pipeline for cross-colony wisdom
- `hive-read` -- reads wisdom with domain filtering, confidence thresholds
- `colony-prime` -- assembles unified worker context from QUEEN.md + hive + signals + instincts

**What is broken (never triggered in practice):**
- Build learnings are only written if `phase_learnings` data exists in COLONY_STATE.json -- but builders don't write to this field
- Instincts are created in continue-advance.md playbooks, but the LLM executing `/ant:continue` often skips these steps or writes trivial instincts
- Hive promotion only runs during `/ant:seal`, which users rarely run (they run `/ant:entomb` directly)
- Queen-seed-from-hive is called in build-context.md but produces empty results when hive is empty (chicken-and-egg)

**What's missing entirely:**
- No `aether-oracle.md` agent file -- Oracle runs via `/ant:oracle` command only, no Task-tool spawnable agent
- No `aether-architect.md` agent file -- Architect caste referenced in workers.md but no agent definition
- No mechanism for builders to report learnings back during builds (the write side is missing)
- No `/ant:wisdom` or similar command to view accumulated wisdom status
- No visible feedback when wisdom is actually written (user can't see the system learning)

---

## Table Stakes

Missing any of these = the wisdom system still feels dead.

| # | Feature | Why Expected | Complexity | Dependencies | Notes |
|---|---------|--------------|------------|--------------|-------|
| T1 | **Builders report learnings during builds** | The core gap. Currently builders produce code but never write observations. Without the write side, the entire pipeline starves. | MEDIUM | build-wave.md playbook, memory-capture subcommand | Builders must call `memory-capture` or `learning-observe` after each plan. This is the single most important feature. |
| T2 | **QUEEN.md Build Learnings section populates automatically** | User opens QUEEN.md after a build and sees real phase learnings. Currently shows placeholders forever. | LOW | T1, queen-write-learnings | Code exists. Needs build-wave.md to actually call it with phase_learnings data. |
| T3 | **Instincts accumulate from phase patterns** | After 2-3 phases, `/ant:status` shows multiple instincts. Currently 0-1 instincts even after full colony lifecycle. | MEDIUM | continue-advance.md, instinct-create | Playbooks already describe this (Step 3/3a/3b/3c). Problem is the LLM skips or trivializes it. Needs stronger prompting or structural enforcement. |
| T4 | **High-confidence instincts promote to QUEEN.md** | Instincts with confidence >= 0.8 appear in QUEEN.md Instincts section. Currently only 1 instinct ever promoted (the migration test one). | LOW | T3, queen-promote-instinct | Code exists in continue-advance.md Step 3c. Fires only if T3 produces instincts. |
| T5 | **Hive brain receives promoted instincts on seal** | After `/ant:seal`, high-confidence instincts appear in `~/.aether/hive/wisdom.json`. Currently hive stays empty unless manually populated. | LOW | seal.md, hive-promote | Code exists in seal.md Step 3.7. Problem: users skip seal and go straight to entomb. |
| T6 | **Oracle agent definition file** | `aether-oracle.md` exists in `.claude/agents/ant/` with proper model routing (opus slot for reasoning-heavy work). Currently Oracle only runs as a slash command, not a spawnable agent. | LOW | workers.md caste table | ~200 lines. Follows same pattern as aether-scout.md. Must reference opus slot. |
| T7 | **Architect agent definition file** | `aether-architect.md` exists in `.claude/agents/ant/` with proper model routing (opus slot). Currently Architect caste has no agent definition -- referenced in workers.md personality table but never defined. | LOW | workers.md caste table | ~200 lines. Fills a gap in the 22-agent roster. Route-setter currently does architectural planning but has no dedicated agent. |

---

## Differentiators

Features that make Aether's wisdom system feel alive and valuable.

| # | Feature | Value Proposition | Complexity | Dependencies | Notes |
|---|---------|-------------------|------------|--------------|-------|
| D1 | **Wisdom growth visible in build output** | During `/ant:continue`, the user sees "QUEEN.md: 2 new instincts promoted" or "Hive: 1 pattern shared across colonies" as part of the standard output. Makes the learning loop tangible. | LOW | T1-T5 | Add summary lines to continue-advance.md output. The data is already computed; just needs to be displayed. |
| D2 | **Cross-colony wisdom flows into new colonies** | User starts a colony in Repo B and sees relevant patterns from Repo A in the worker context. The "Aether gets smarter over time" promise, actually delivered. | MEDIUM | T5, colony-prime, hive-read | colony-prime already calls hive-read. Problem: hive is empty. Once T5 feeds the hive, this works automatically. |
| D3 | **Oracle as a spawnable research agent** | Queen can spawn Oracle during Deep Research pattern (build-wave.md Step 5.0.5) instead of only running it as a standalone command. Oracle gets opus model routing for deep reasoning. | MEDIUM | T6, build-wave.md | Currently the "Deep Research" pattern in build-wave.md announces itself but doesn't spawn Oracle. With T6, it can use Task tool to spawn aether-oracle.md. |
| D4 | **Architect as planning agent** | Route-setter or Queen can spawn Architect for architectural decisions during planning. Architect gets opus model for structural reasoning. | MEDIUM | T7, route-setter agent, plan command | Currently architectural decisions are made ad-hoc by Queen or route-setter. Architect provides a dedicated reasoning caste. |
| D5 | **`/ant:wisdom` status command** | User runs `/ant:wisdom` and sees: QUEEN.md entries count, hive entries count, instincts count, last promotion date, cross-colony stats. One command to verify the system is learning. | LOW | queen-read, hive-read, instinct-read | New slash command. Assembles data from existing subcommands into a readable summary. |
| D6 | **Phase completion auto-promotes learnings** | When `/ant:continue` extracts learnings, it automatically writes them to QUEEN.md Build Learnings (no threshold). This is different from instinct promotion which requires confidence >= 0.8. Every phase completion should produce at least one learning entry. | LOW | T1, queen-write-learnings, continue-advance.md | Add to continue-advance.md Step 2. queen-write-learnings bypasses thresholds -- every build writes learnings. |

---

## Anti-Features

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Automatic wisdom pruning/decay | Prevent QUEEN.md from growing unbounded | Removes the user's accumulated knowledge without consent. Wisdom should persist unless the user explicitly manages it. | Cap enforcement (already exists: max 30 instincts, 200 hive entries) + user-facing `/ant:wisdom --prune` for manual cleanup |
| Wisdom sharing between users | Cross-user colony learning | Privacy violation. User A's codebase patterns exposed to User B via hive. Hive is machine-local by design. | Keep hive at `~/.aether/` (single user). Future: opt-in export/import of anonymized patterns |
| Real-time wisdom injection during builds | Workers get updated wisdom mid-phase | Creates inconsistent context within a single phase. Workers spawned in Wave 1 see different wisdom than Wave 2. | Wisdom is loaded once at build start via colony-prime. Updates appear in next build. |
| Wisdom-based task assignment | Automatically assign tasks based on accumulated patterns | Premature optimization. Aether's task assignment is already caste-based. Adding wisdom-based routing adds complexity without clear benefit. | Keep caste-based assignment. Wisdom influences behavior (via prompt injection), not assignment. |
| Generic "AI memory" features | Store arbitrary user notes as wisdom | Pollutes the wisdom system with non-validated, non-structured content. Wisdom should be earned through colony work, not manually entered. | Use `/ant:dream` for free-form notes. Wisdom is colony-earned only. |

---

## Feature Dependencies

```
[T1: Builders report learnings]
    └──requires──> [build-wave.md playbook changes]
    └──enables───> [T2: QUEEN.md learnings populate]
                    └──enables───> [D1: Visible wisdom growth]

[T3: Instincts accumulate]
    └──requires──> [continue-advance.md enforcement]
    └──enables───> [T4: Instincts promote to QUEEN.md]
                    └──enables───> [T5: Hive brain receives instincts]
                                    └──enables───> [D2: Cross-colony wisdom flows]

[T6: Oracle agent file]
    └──enables───> [D3: Oracle spawnable during builds]

[T7: Architect agent file]
    └──enables───> [D4: Architect in planning]

[D5: /ant:wisdom command]
    └──requires──> [T2, T4, T5] (data to display)

[D6: Auto-promote learnings on phase completion]
    └──requires──> [T1, T2]
```

### Dependency Notes

- **T1 is the linchpin.** Without builders reporting learnings, the entire pipeline starves. Every other feature depends on data flowing in.
- **T5 requires seal, not entomb.** Hive promotion only fires during `/ant:seal`. If users skip seal, the hive stays empty. Consider adding hive promotion to entomb as well (non-blocking).
- **T6 and T7 are independent.** They don't depend on the wisdom pipeline. They fill gaps in the agent roster. Can be built in parallel.
- **D2 is emergent, not built.** Once T1-T5 work, D2 happens automatically because colony-prime already reads hive wisdom. No code changes needed for D2 itself.

---

## MVP Definition

### Launch With (Phase 1 of milestone)

Minimum to make the wisdom system feel alive.

- [ ] **T1** -- Builders report learnings during builds (the write side)
- [ ] **T6** -- Oracle agent definition file (fill the gap)
- [ ] **T7** -- Architect agent definition file (fill the gap)
- [ ] **D1** -- Wisdom growth visible in build output

Rationale: T1 feeds the pipeline, T6/T7 fill the agent roster gap, D1 makes it visible. After Phase 1, a user running a 3-phase colony should see real learnings in QUEEN.md and real instincts in COLONY_STATE.json.

### Add After Validation (Phase 2 of milestone)

- [ ] **T2** -- QUEEN.md Build Learnings auto-populate (depends on T1 data)
- [ ] **T3** -- Instincts accumulate reliably (stronger enforcement in continue-advance)
- [ ] **T4** -- Instincts promote to QUEEN.md (depends on T3 data)
- [ ] **D6** -- Phase completion auto-promotes learnings
- [ ] **D5** -- `/ant:wisdom` status command

### Future Consideration (Phase 3 or defer)

- [ ] **T5** -- Hive brain receives instincts on seal (needs seal flow validation)
- [ ] **D2** -- Cross-colony wisdom flows into new colonies (emergent once T5 works)
- [ ] **D3** -- Oracle spawnable during builds (requires build-wave.md Deep Research pattern changes)
- [ ] **D4** -- Architect in planning (requires route-setter integration)

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| T1: Builders report learnings | HIGH -- makes everything else work | MEDIUM -- playbook changes + builder prompt | P1 |
| T6: Oracle agent file | HIGH -- fills critical gap | LOW -- ~200 line .md file | P1 |
| T7: Architect agent file | MEDIUM -- fills gap | LOW -- ~200 line .md file | P1 |
| D1: Visible wisdom growth | HIGH -- makes learning tangible | LOW -- display existing data | P1 |
| T3: Instincts accumulate | HIGH -- core promise | MEDIUM -- enforcement problem | P2 |
| T2: QUEEN.md learnings populate | HIGH -- visible proof | LOW -- wire existing code | P2 |
| D6: Auto-promote learnings | MEDIUM -- reduces friction | LOW -- add to continue-advance | P2 |
| D5: /ant:wisdom command | MEDIUM -- observability | LOW -- new slash command | P2 |
| T4: Instincts promote to QUEEN.md | MEDIUM -- visible proof | LOW -- already coded | P2 |
| T5: Hive brain on seal | MEDIUM -- cross-colony value | LOW -- already coded | P3 |
| D2: Cross-colony wisdom | MEDIUM -- emergent | NONE -- automatic | P3 |
| D3: Oracle spawnable | LOW -- nice to have | MEDIUM -- build-wave changes | P3 |
| D4: Architect in planning | LOW -- nice to have | MEDIUM -- route-setter changes | P3 |

---

## What the Living Wisdom Experience Should Look Like

### During a Build

**Before (current):**
```
Phase 3: Implement auth module
[Builder Hammer-42] Completed plan 1 of 2
[Builder Hammer-42] Completed plan 2 of 2
Phase 3 complete. Run /ant:continue to verify and advance.
```

**After (with T1 + D1):**
```
Phase 3: Implement auth module
[Builder Hammer-42] Completed plan 1 of 2
[Builder Hammer-42] Completed plan 2 of 2
[Builder Hammer-42] Learning: JWT token validation must check expiry before verifying signature
Phase 3 complete. Run /ant:continue to verify and advance.
```

### During Continue

**Before (current):**
```
Verifying Phase 3 work...
Tests: PASS (47/47)
Extracting learnings...
Recorded observations for threshold tracking
Advancing to Phase 4.
```

**After (with T1-T4 + D1 + D6):**
```
Verifying Phase 3 work...
Tests: PASS (47/47)
Extracting learnings...
  - 3 learnings recorded from Phase 3
  - 1 new instinct created (auth-patterns, confidence: 0.75)
  - 1 instinct promoted to QUEEN.md (testing: 0.85)
  - 2 learnings written to QUEEN.md Build Learnings
Advancing to Phase 4.
```

### During Seal

**Before (current):**
```
Colony sealed.
Wisdom review: no proposals to review.
```

**After (with T5 + D1):**
```
Colony sealed.
Wisdom review: 3 proposals met thresholds, 2 promoted.
Hive brain: 2 instincts promoted to cross-colony wisdom.
Cross-colony stats: 8 total patterns from 3 repos.
```

### Checking Wisdom Status

**New command `/ant:wisdom`:**
```
COLONY WISDOM STATUS
=====================
QUEEN.md (.aether/QUEEN.md):
  User Preferences:     0
  Codebase Patterns:    8 (5 hive, 3 repo)
  Build Learnings:      12 (across 4 phases)
  Instincts:            3 (highest: 0.92)

Hive Brain (~/.aether/hive/):
  Total entries:        15
  Contributing repos:   3
  Highest confidence:   0.95

Colony Memory (.aether/data/COLONY_STATE.json):
  Active instincts:     7
  Phase learnings:      12
  Decisions:            8

Last promotion: 2026-03-27T14:30:00Z
```

### When Starting a New Colony

**Before (current):**
Workers get context from QUEEN.md (mostly placeholder text) and empty hive.

**After (with D2):**
Workers get context from QUEEN.md (populated with real learnings from previous colonies) AND cross-colony hive wisdom. A builder in Repo B sees:
```
HIVE WISDOM (Cross-Colony Patterns):
- [0.95] When creating testimonials: Use clearly labeled placeholders instead of fabricating content
- [0.90] When implementing auth: JWT token validation must check expiry before verifying signature
- [0.85] When building bash utilities: use process substitution not pipes to while loops
```

---

## Competitor / Analog Feature Analysis

| Feature | Cursor Rules | Windsurf | Aider | Aether (planned) |
|---------|-------------|----------|-------|------------------|
| Project-specific rules | CLAUDE.md | .windsurfrules | .aider.conf.yml | QUEEN.md (validated, not static) |
| Auto-learn from work | No | No | No | Yes -- learnings from builds |
| Cross-project memory | No | No | No | Yes -- hive brain |
| Confidence-based promotion | No | No | No | Yes -- observation thresholds |
| Instinct decay/evolution | No | No | No | Yes -- success/failure feedback |

Aether's wisdom system is genuinely differentiated. No other AI coding tool has a learning loop that accumulates validated patterns across projects. The gap is not the design -- it's the execution (wiring the triggers).

---

## Sources

- HIGH confidence: Codebase analysis of queen.sh (1,242 lines), hive.sh (562 lines), learning.sh (1,553 lines), pheromone.sh colony-prime function (~700 lines)
- HIGH confidence: build-wave.md, build-context.md, continue-advance.md, seal.md playbook analysis
- HIGH confidence: Existing QUEEN.md in Aether repo showing actual data (1 instinct, 6 patterns, 1 build learning)
- HIGH confidence: workers.md caste table and model slot assignments
- MEDIUM confidence: User testing feedback (documented in PROJECT.md: "QUEEN.md and hive brain are template-only -- never populated with real data")
- LOW confidence: Whether builders can reliably produce high-quality learning observations (needs validation in Phase 1)

---
*Feature research for: Aether v2.4 Living Wisdom*
*Researched: 2026-03-27*
