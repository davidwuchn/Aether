# Research Synthesis

## Topic
Implementation plan for unimplemented Aether features: bug fixes (A1-A4), TODO clearance (B1-B4), RAG pipeline (C1-C6), agent competition (D1-D4), GSD-inspired colony features (headless mode, immune response, vital signs, federation, council expansion, midden library, secret sentinel, quick mode, crash recovery), and polish items (E1-E6)

## Findings by Question

### Q1: Feature implementation status audit (answered, 92%)

**GSD-Inspired Features — 6 of ~10 COMPLETED** [S1, S2, S3, S4, S5]

The sealed colony at `2026-03-30` (6 phases, all completed) implemented:

| Feature | Status | Key Files | Tests |
|---------|--------|-----------|-------|
| Midden Library | DONE | `midden-search`, `midden-tag` subcommands | `tests/bash/test-midden-library.sh` |
| Colony Vital Signs | DONE | `colony-vital-signs` subcommand → `state-api.sh` | `tests/bash/test-vital-signs.sh` |
| Headless Autopilot + Pending Decisions | DONE | `autopilot-set-headless`, `autopilot-headless-check`, `pending-decision-*` subcommands, `--headless` in run.yaml | `tests/bash/test-headless-autopilot.sh`, `tests/unit/pending-decisions.test.js` |
| Immune Response (trophallaxis + scarification) | DONE | `.aether/utils/immune.sh` module | `tests/bash/test-immune-module.sh` |
| /ant:quick Scout Missions | DONE | `.claude/commands/ant/quick.md` | N/A |
| Council Expansion | DONE | `.aether/utils/council.sh` module | `tests/bash/test-council-module.sh` |

**GSD-Inspired Features — NOT Implemented:**

| Feature | Status | Notes |
|---------|--------|-------|
| Federation (multi-colony democracy) | UNSTARTED | Concept only in FUTURE-IDEAS.md |
| Secret Sentinel | UNSTARTED | No code found |
| Regression Sentinels | UNSTARTED | No code found |
| Crash Recovery / Orphan Detection | UNSTARTED | Session recovery exists but no crash/orphan detection |

**RAG Pipeline (C1-C6): ENTIRELY UNSTARTED** [S5]
- No embedding, vector store, or retrieval-augmented generation code exists
- Zero relevant hits for 'rag', 'embedding', 'vector store' in aether-utils.sh

**Agent Competition Model (D1-D4): ENTIRELY UNSTARTED** [S5]
- No tournament, leaderboard, ranking, or competition code
- Only plan.json references mention this feature set

**TODO Clearance (B1-B4):** [S14]
- B1 (npm deprecation): Status unclear — npm packaging still active
- B2 (Model Routing Verification): UNSTARTED — remains a FUTURE-IDEAS.md concept
- B3 (XML Integration): PARTIALLY DONE — XML utilities exist (.aether/utils/xml-*.sh, .aether/exchange/)
- B4 (YAML Command Generator): UNSTARTED — .aether/commands/*.yaml are definitions, not a generator

**Bug Fixes (A1-A4):** [S5, S15]
- A1 (unescaped ant_name): Needs deeper code investigation
- A2 (cross-colony bleed): Needs deeper code investigation
- A3 (colony depth selector): Needs deeper code investigation
- A4 (JSON escaping): Partially addressed — queen.sh got fixes, spawn-tree got awk-based escaping

**Polish Items (E1-E6):** Not yet investigated — need to locate the original brief defining these items.

**CLI vs. bash boundary** [S71, S2, S1]: CLI (bin/cli.js + 18 bin/lib/ modules) handles plumbing: installation/hub distribution, model profile CRUD, telemetry, and state-sync. CLI NEVER invokes aether-utils.sh directly. Bash layer (178 subcommands) handles all domain logic. Slash commands bridge the two. The only cross-link is CLI copying aether-utils.sh to hub during npm install.

**Colony-prime orchestration** [S72, S29, S1]: Read-only aggregator assembling 9 sections from 8 data sources. QUEEN.md wisdom → Pheromone-prime → Hive-read → Context-capsule → Phase-learnings → Key-decisions → Blockers → Rolling-summary. Output: JSON with prompt_section + log_line + metadata. Per-wave refresh during builds.

**Subcommand criticality** [S73, S1, S3]: Of 178 total subcommands, only 43 (24%) are critical path. Top 7 account for ~50% of invocations. 76 subcommands (43%) are dead code — case-statement dispatch (no dynamic invocation) confirms these are unreachable from .md files.

**Agent coupling** [S4, S15, S73]: One-way coupling: playbooks → Task tool → agent → results back. Agents warned not to modify .aether/data/. Healthiest coupling pattern in the system.

### Q2: Dependency relationships and coupling risks (answered, 90%)

**COLONY_STATE.json as coupling nexus** [S14, S1]: 219 direct references across 38 of 43 slash commands, plus 153 references in aether-utils.sh. Dual access path: slash commands via direct jq AND via aether-utils.sh subcommands.

**Module sourcing** [S1, S16]: 9 utility modules sourced at startup with conditional guards (`[[ -f ]] && source`). Missing module degrades silently.

**CLI/bash parallelism** [S2, S1, S71]: Nearly parallel systems with minimal cross-calling. Independent file-lock implementations (bash noclobber vs Node.js 'wx' flag). Temporally disjoint: CLI writes hub only during install, bash during colony operations.

**Build/continue data flow** [S44, S45, S15, S46, S47, S48, S49, S50]: 9-playbook state mutation sequence traced. State set to EXECUTING with lock → context read (no lock) → parallel builders with mixed locking → verification with load-state lock → 7 mandatory gates (read-only) → COLONY_STATE.json advance via LLM Write (no bash lock) → finalize.

**Concurrent write protection** [S54, S52, S53, S55, S51]: Inconsistent across 5 shared files:
- pheromones.json: FULLY PROTECTED (acquire_lock)
- midden.json: PARTIALLY PROTECTED (lock with lockless fallback — read-modify-write race under concurrent failures)
- activity.log: UNPROTECTED (POSIX append, safe under PIPE_BUF for short entries)
- spawn-tree.txt: UNPROTECTED (same)
- COLONY_STATE.json: MIXED (bash commands use lock, LLM Write in continue-advance does not)

**Midden race** [S52]: `_midden_try_write` uses PID-specific temp files (`${mtw_file}.tmp.$$`). Race is read-modify-write (last writer wins), not temp file collision. Impact: silent loss of one failure record during high-concurrency lockless writes.

**Continue-advance race** [S51, S55, S49]: Latent window between continue-verify lock release and continue-advance LLM Write. Slow builder's spawn-complete could be overwritten. Mitigated by sequential flow.

### Q3: Risk areas (answered, 88%)

**Silent error suppression** [S1, S28]: ~338 instances of error-swallowing patterns. ERR trap disabled for suggest-analyze (~200-line untrapped window during builds).

**Worker response validation** [S17, S20]: Schema-only, not semantic. Cannot detect Builder claiming 'completed' when tests failed. Primary hallucination vector.

**Verification chain** [S18, S19, S26, S27]: Prompt-enforced, not programmatically enforced. 'Iron Laws' in agent markdown. Watcher requires 4 execution verification steps and quality score ceiling rule.

**Agent fallback** [S25, S4]: CONFIRMED across 8 FALLBACK comments in 4 playbooks (build-wave.md, build-context.md, build-verify.md, build-full.md). Specialized 200+ line agent → 1-sentence role description. Dramatic degradation.

**Anti-pattern detection** [S21]: CONFIRMED exactly 6 patterns at aether-utils.sh:1925-1997: (1) didSet recursion (Swift), (2) TypeScript `any` type, (3) console.log (TS/JS non-test), (4) Python bare except, (5) exposed secrets (all languages), (6) TODO/FIXME/XXX/HACK (all languages). No OWASP top 10 beyond hardcoded secrets.

**Dual file-lock** [S22, S23]: Different atomic-create mechanisms. Mitigated by temporal separation — CLI during install only, bash during colony operations.

**Hive-read type handling** [S24]: CORRECTED — hive.sh:309 uses `tonumber` which already converts string "0.8" to number 0.8. String-typed confidence does NOT cause silent exclusion. Real remaining risk: null/undefined values crash the jq filter, caught by fallback at line 318-321 returning empty results for entire query.

**Memory pipeline** [S56, S57, S58, S59, S68, S69]: Sequential kill-switch at step 1 (learning-observe failure kills all 5 downstream steps). However, learning-observe has 3-deep backup recovery (learning.sh:137-191), so permanent failure requires ALL backups corrupted.

**Autopilot** [S38, S60, S61]: All 10 pause conditions are binary. No detection for subtly bad work. run-state.json and COLONY_STATE.json can desync with no reconciliation.

**Historical evidence** [S62]: 17 midden entries confirm 5+ risks materialized: cross-repo lock isolation, string-typed confidence, unbound variables, unsanitized json_ok, fix ratio rising.

### Q4: Memory and context management (answered, 90%)

**Dual-file state persistence** [S32, S33]: session.json (lightweight) and COLONY_STATE.json (authoritative). Missing COLONY_STATE.json blocks resume.

**Three-tier session resume** [S33, S34]: Brief (1-line), Extended (+ timestamp), Full (complete state + HANDOFF.md + pheromones + survey).

**Colony-prime budget** [S29]: 8K normal / 4K compact. Trim order (first removed): rolling-summary → phase-learnings → key-decisions → hive-wisdom → context-capsule → user-prefs → queen-wisdom → pheromone-signals. Blockers NEVER trimmed. REDIRECT signals preserved.

**Budget trimming notification** [S70, S29]: UPDATED — `trimmed_notice` field now computed (pheromone.sh:1495-1509) and included in colony-prime JSON output (line 1541) as a separate field. Also emits stderr warning. However, NOT injected into `prompt_section` — workers still don't see it unless playbooks explicitly inject the trimmed_notice field. Rec 5 partially addressed.

**Rolling summary** [S30, S29]: Bounded 15-entry log, 180-char cap. Colony-prime injects last 5. First section trimmed under budget pressure.

**Context capsule** [S31, S29, S67]: CONFIRMED resilient — 220-word max snapshot. Missing COLONY_STATE.json → clean early exit (`json_ok '{"exists":false}'` at aether-utils.sh:4202-4204). Corrupted → cascading jq defaults. One of the most defensive functions.

**Hive brain** [S29, S36, S35]: Domain-scoped retrieval via registry.json tags. LRU tracking. Falls back to eternal memory. Content hash dedup. Cross-repo confidence boosting (2=0.70, 4+=0.95). 200-entry LRU cap.

**Staleness** [S32, S33]: Detected (>24h) but never blocks restoration. Drift detection is informational. Availability over caution.

**CONTEXT.md** [S63, S64, S65, S66]: 11 action types with lock protection (LOCK-04). Decision-to-pheromone bridge: continue-advance extracts decisions, auto-emits FEEDBACK pheromones (up to 3). Dedup catches double-emit.

**Memory-capture pipeline** [S68, S69, S56]: Sequential kill-switch at step 1. Corrupted learning-observations.json → tries .bak.1/.bak.2/.bak.3 recovery first (learning.sh:137-191) → json_err only if ALL backups corrupted → kills all 5 downstream steps. Callers' `2>/dev/null || true` hides total pipeline death. More resilient than originally captured.

### Q5: Expert architectural recommendations (answered, 88%)

**RECOMMENDATION 1 — Per-phase COLONY_STATE.json checkpointing** [S38, S39, S43, S42]: Only entomb creates backups. Fix: cp COLONY_STATE.json COLONY_STATE.json.phase-N.bak before each build-wave. Near-zero cost. Feasible: build-prep already holds file lock [S44].

**RECOMMENDATION 2 — Semantic verification evidence trail** [S41, S17, S20, S38]: 8 programmatic gates exist but Builder/Watcher claims aren't verified. Fix: capture test runner exit code and cross-reference; verify Builder's files_created against filesystem.

**RECOMMENDATION 3 — Error suppression triage** [S1, S28]: ~338 instances → correct/lazy/dangerous categories. Priority: suggest-analyze ERR trap gap, atomic_write suppressions, conditional module sourcing.

**RECOMMENDATION 4 — Hive jq null safety** [S24, S36, S35]: REVISED — `tonumber` already handles string→number conversion. Real fix: `(tonumber? // 0)` for null/undefined safety. Less critical than originally assessed (~5-line fix, defensive improvement).

**RECOMMENDATION 5 — Context trimming notification routing** [S29, S30]: PARTIALLY ADDRESSED — `trimmed_notice` field exists in colony-prime output but not in prompt_section. Remaining fix: prepend trimmed_notice to prompt_section when non-empty. Also fix CLAUDE.md documentation if inversion exists.

**RECOMMENDATION 6 — Agent fallback degradation logging** [S25, S4]: Silent degradation → midden entry + WARNING in build synthesis + optional FEEDBACK pheromone.

**RECOMMENDATION 7 — Git checkpoint branches for autopilot** [S38, S42, S1]: Lightweight git tag (aether/pre-phase-N) before each build. Combined with Rec 1 for full state + code rollback. autofix-checkpoint already implements git stashing.

**RECOMMENDATION 8 — Memory pipeline circuit breaker** [S56, S68, S69]: REFINED — learning-observe already has .bak.1/.bak.2/.bak.3 recovery. Circuit breaker should only trigger when all backups exhausted. Fix: after backup recovery fails, reset file to template + log midden entry + retry once.

**RECOMMENDATION 9 — Autopilot state reconciliation** [S60, S42, S38]: run-state.json ↔ COLONY_STATE.json phase comparison at loop start, pause on disagreement.

**PRIORITY MATRIX** [S62, S24, S38, S43, S56, S68]:
- **Tier 1 (immediate):** Rec 1 (3 lines, prevents total loss), Rec 8 (refined: circuit breaker after backup exhaustion), Rec 4 (5 lines, null safety)
- **Tier 2 (medium):** Rec 5 (route trimmed_notice to prompt_section), Rec 2 (20 lines/check), Rec 9 (10 lines, depends Rec 1)
- **Tier 3 (larger):** Rec 6 (10 lines), Rec 7 (5 lines, depends Rec 1), Rec 3 (audit 338 instances)

**FEASIBILITY** [S44, S41, S24, S1, S4, S67, S66, S71]: All top-priority fixes implementable. Architecture sound at macro level — risks concentrated in implementation details.

**COVERAGE** [S22, S23, S21, S71, S1]: 9 recommendations cover 8 of 10 Q3 risks. Unaddressed: dual file-lock (mitigated by temporal separation) and check-antipattern scope (documentation issue, recommend renaming).

---

## Verification Pass (Iteration 5)

### New Corrections (3 additional)

1. **Hive-read type coercion (S24):** `tonumber` at hive.sh:309 already converts string→number. Original claim of "silent exclusion from string types" overstated. Real risk: null/undefined crashes filter, caught by fallback. Rec 4 downgraded from "confirmed bug" to "defensive improvement."

2. **Midden race mechanism (S52):** `_midden_try_write` uses `${mtw_file}.tmp.$$` (PID-specific temp files). Race is read-modify-write, not shared temp path collision. Data loss conclusion unchanged.

3. **Budget trimming (S70):** `trimmed_notice` field now computed and in colony-prime JSON output, but NOT in prompt_section. Rec 5 partially addressed — remaining work is routing notice into the prompt.

### Confidence Adjustments

| Question | Before | After | Reason |
|----------|--------|-------|--------|
| q1 | 88% | 92% | All claims verified multi-source, architecture fully traced |
| q2 | 85% | 90% | All coupling risks confirmed, midden mechanism corrected |
| q3 | 84% | 88% | Single-source claims verified, hive-read correction applied |
| q4 | 85% | 90% | Memory pipeline confirmed, budget trimming updated |
| q5 | 83% | 88% | Recommendations validated, feasibility confirmed |

**New overall confidence: 90%**

## Sources

All 73 sources (S1-S73) are codebase references. Key sources by verification status:

| Source | File | Verification |
|--------|------|-------------|
| S1 | .aether/aether-utils.sh | Direct read, multi-iteration |
| S21 | aether-utils.sh:1925-1997 | Direct read iteration 5 — 6 patterns confirmed |
| S24 | .aether/utils/hive.sh:309 | Direct read iteration 5 — tonumber handles strings |
| S25 | .aether/docs/command-playbooks/ | Direct read iteration 5 — 8 FALLBACK comments confirmed |
| S52 | .aether/utils/midden.sh:9-15 | Direct read iteration 5 — PID-specific temp files |
| S67 | aether-utils.sh:4202-4210 | Direct read iteration 5 — graceful degradation confirmed |
| S69 | .aether/utils/learning.sh:137-191 | Direct read iteration 5 — backup recovery confirmed |
| S70 | .aether/utils/pheromone.sh:1495-1542 | Direct read iteration 5 — trimmed_notice exists |

## Last Updated
Iteration 5 (VERIFY) -- 2026-03-30T09:30:00Z
