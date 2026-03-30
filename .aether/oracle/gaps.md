# Knowledge Gaps

## Open Questions

All 5 tracked research questions (q1-q5) are **answered** at 88-92% confidence. The remaining gaps are operational metrics and scope items that cannot be resolved through code analysis alone.

### Operational Metrics (code analysis ceiling reached)
- Frequency of budget trimming in production usage — requires runtime monitoring
- Concurrent write collision frequency for activity.log / spawn-tree.txt — POSIX append under PIPE_BUF (4096 bytes) makes this effectively safe for typical entry sizes (<100 bytes)
- Memory-capture corruption frequency — learning.sh has backup recovery (.bak.1/.bak.2/.bak.3) mitigating the risk
- Worker fallback trigger frequency — low in Claude Code (all agents registered), only triggers in non-standard environments

### Scope Items Requiring User Input
- **E1-E6 polish items:** Referenced in research topic but no definition document found. Feature codes may come from an unpersisted conversation.
- **Secret Sentinel:** No spec or code. Aspirational name without defined scope. Related code exists (check-antipattern, Gatekeeper agent, immune scarification) but may not match intent.
- **Regression Sentinels:** No spec or code. Aspirational name without defined scope.
- **Federation (multi-colony democracy):** Too large for a single phase — single-colony assumption is deeply embedded across COLONY_STATE.json, session.json, pheromones.json, CONTEXT.md. Requires its own colony with dedicated architecture planning.
- **RAG Pipeline (C1-C6):** Entirely unstarted. No embedding, vector store, or retrieval code exists. Language choice (Python vs bash) and integration with colony-prime token budget undetermined.
- **Agent Competition (D1-D4):** Entirely unstarted. No tournament, leaderboard, or ranking code.

## Contradictions

All previously identified contradictions have been **RESOLVED** during verification:

1. ~~CLAUDE.md trim order inversion~~ — **RESOLVED.** CLAUDE.md correctly states "Rolling summary (trimmed first -- lowest retention priority)" at line 361. Code confirms at pheromone.sh:1388-1484. Prior gap entry was based on a misreading.

2. ~~midden-autopsy vs midden library~~ — **RESOLVED.** "midden-autopsy" doesn't exist in topic or codebase. Feature was always "midden library" (midden-search/midden-tag), fully implemented.

3. ~~A2 cross-colony bleed already fixed~~ — **RESOLVED.** hive.sh uses `acquire_lock_at()` with explicit lock_dir parameter and colony_tag. `LOCK_DIR` is only set once (file-lock.sh:20), never mutated. Fix is complete.

4. ~~Secret/Regression Sentinels undefined~~ — **RESOLVED as UNDEFINED.** These names exist only in oracle research files. Classified as aspirational, requiring user input to proceed.

## Discovered Unknowns

None remaining — all discovered unknowns have been either resolved or reclassified as "scope items requiring user input" above.

## Verification Corrections (Iteration 4-5)

- **learning-observe (S69):** Originally documented as "corrupted file triggers json_err non-zero exit." CORRECTED: `_learning_observe()` in learning.sh has backup recovery (attempts .bak.1, .bak.2, .bak.3 before erroring). More resilient than originally captured. Recommendation 8 (memory pipeline circuit breaker) should account for this existing recovery mechanism.

- **Hive-read type coercion (S24):** Originally documented as "String-typed confidence causes silent exclusion." CORRECTED: hive.sh line 309 already uses `tonumber` which converts string "0.8" to number 0.8. Silent exclusion from string types does NOT occur. Real remaining risk is null/undefined confidence values crashing the jq filter, caught by the fallback at line 318-321. Rec 4 (`tonumber? // 0`) is still an improvement for null safety but less critical than originally assessed.

- **Midden race mechanism (S52):** Originally documented as "both paths write to SAME temp path." CORRECTED: `_midden_try_write` (midden.sh:15) uses `${mtw_file}.tmp.$$` with PID suffix — temp paths are unique per process. The actual race is a read-modify-write race: two concurrent lockless writers each read the same base state, add their entry, and write back — last writer wins, losing one entry. Data loss conclusion remains valid, mechanism was mischaracterized.

- **Budget trimming notification (S70):** Originally documented as "records trimmed sections in log_line but NOT in prompt_section." UPDATED: `trimmed_notice` field now exists in colony-prime JSON output (pheromone.sh:1495-1542) as a separate field. However, it is NOT injected into `prompt_section` — workers still don't see it unless playbooks explicitly inject the trimmed_notice field. Rec 5 is partially addressed.

## Last Updated
Iteration 5 (VERIFY) -- 2026-03-30T09:30:00Z
