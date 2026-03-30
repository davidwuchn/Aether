# Oracle Research: Phase 3 -- QUEEN.md and Wisdom Pipeline Integrity

**Researcher:** Mystic-36
**Date:** 2026-03-28
**Phase:** Audit QUEEN.md and wisdom pipeline integrity
**Colony Goal:** Comprehensive audit of session work

---

## Context

This research audits two critical subsystems:
1. **queen-read** -- Reads QUEEN.md and returns structured JSON for worker priming
2. **Wisdom pipeline chain** -- memory-capture -> learning-observe -> learning-promote-auto -> instinct-create -> queen-promote

The audit focuses on correctness, data integrity, and identifying bugs that could cause silent failures or incorrect worker priming.

---

## Key Findings

### Finding 1: queen-read `sources` field always returns false/false (BUG)

**Source:** `.aether/utils/queen.sh`, lines 286-289

The `sources` object in queen-read output uses `$meta.source` to determine `has_global` and `has_local`:
```jq
sources: {
  has_global: ($meta.source == "global" or $meta.source == "local"),
  has_local: ($meta.source == "local")
}
```

However, the QUEEN.md METADATA block does NOT contain a `source` field. The actual metadata has `version`, `wisdom_version`, `last_evolved`, `colonies_contributed`, `stats`, and `evolution_log` -- but no `source`. As a result, `$meta.source` evaluates to `null`, and both booleans are always `false`.

**Live verification:** `queen-read` returns `{"has_global": false, "has_local": false}` despite both `~/.aether/QUEEN.md` and `.aether/QUEEN.md` existing.

**Confidence:** HIGH
**Impact:** Any downstream consumer checking whether global/local QUEEN.md files exist via the `sources` field gets incorrect data.

---

### Finding 2: queen-read `priming` flags poisoned by global placeholders (BUG)

**Source:** `.aether/utils/queen.sh`, lines 281-284

The priming booleans use regex tests like:
```jq
has_codebase_patterns: ($codebase_patterns | length) > 0 and ($codebase_patterns | test("No codebase patterns recorded yet") | not)
```

Since queen-read concatenates global + local wisdom (line 222-238), and the global `~/.aether/QUEEN.md` still has placeholder text like `*No codebase patterns recorded yet.*`, the combined string matches the regex even though the local file has 12 real codebase patterns.

**Live verification:**
- `has_user_prefs`: true (only correct one -- global has real user prefs description text)
- `has_codebase_patterns`: FALSE (wrong -- 12 real patterns exist locally)
- `has_build_learnings`: FALSE (wrong -- 1 real learning exists locally)
- `has_instincts`: FALSE (wrong -- 3 real instincts exist locally)

**Confidence:** HIGH
**Impact:** Workers may skip wisdom injection or behave as if no colony wisdom exists, degrading colony intelligence.

---

### Finding 3: Metadata `evolution_log` drifts from markdown Evolution Log table (DATA DRIFT)

**Source:** `.aether/QUEEN.md`, lines 85-98

The METADATA JSON `evolution_log` array has only 2 entries while the markdown Evolution Log table has 22 data rows. The awk-based append logic in `queen-promote` (lines 510-568) is supposed to maintain the JSON evolution_log, but it only handles specific structural patterns and likely fails silently when the JSON structure doesn't match expected patterns.

**Live verification:** `queen-read` returns metadata with 2 evolution_log entries; the markdown table shows 22 events.

**Confidence:** HIGH
**Impact:** Low -- the JSON evolution_log is not actively consumed by any critical path. The markdown table is the visual record. But this drift signals fragility in the awk-based JSON manipulation approach.

---

### Finding 4: Single-line METADATA format breaks extraction (LATENT BUG)

**Source:** `.aether/utils/queen.sh`, lines 243-246

The metadata extraction uses:
```bash
metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_file" | sed '1d;$d' | tr -d '\n')
```

This works for multi-line METADATA blocks (local QUEEN.md) but fails for single-line blocks (global `~/.aether/QUEEN.md` format: `<!-- METADATA {...} -->`). The `sed '1d;$d'` deletes both the first and last line -- when it's all one line, this deletes the entire content.

**Live verification:** Tested with both formats. Multi-line produces valid JSON; single-line produces empty string.

**Confidence:** HIGH
**Impact:** Currently mitigated because the local QUEEN.md (multi-line) takes priority for metadata. But if a repo has no local QUEEN.md, metadata extraction from global fails silently, falling back to the default placeholder metadata.

---

### Finding 5: queen-read returns valid JSON structure overall (PASS)

**Source:** Live execution of `bash .aether/aether-utils.sh queen-read`

Despite the bugs above, queen-read:
- Returns valid JSON wrapped in `{"ok": true, "result": {...}}`
- Contains all required v2 keys: `metadata`, `wisdom`, `priming`, `sources`
- Wisdom sections correctly combine global + local content with newline separator
- Metadata version, last_evolved, and stats fields are all present and valid

**Confidence:** HIGH

---

### Finding 6: Metadata stats match actual QUEEN.md content (PASS)

**Source:** `.aether/QUEEN.md` manual count vs metadata stats

| Section | Actual Count | Metadata Stat | Match? |
|---------|:---:|:---:|:---:|
| User Preferences | 2 | 2 | YES |
| Codebase Patterns | 12 | 12 | YES |
| Build Learnings | 1 | 1 | YES |
| Instincts | 3 | 3 | YES |

**Confidence:** HIGH

---

### Finding 7: Wisdom pipeline chain is correctly wired (PASS)

**Source:** `.aether/aether-utils.sh` lines 3627-3741, `.aether/utils/learning.sh`

The memory-capture orchestrator correctly chains:
1. `memory-capture` (dispatcher, lines 3627-3741) receives event_type + content
2. Calls `learning-observe` via subprocess -> records/increments observation count
3. Calls `learning-promote-auto` via subprocess -> checks auto-promotion threshold
4. If threshold met: calls `queen-promote` to write to QUEEN.md
5. Then calls `instinct-create` to create an instinct from the promoted learning
6. Also emits appropriate pheromone signal based on event_type

All dispatch routes confirmed present in the case statement (lines 3614, 3623, 3625, 3972).

**Confidence:** HIGH

---

### Finding 8: Auto-promotion thresholds are type-dependent, not always 2 (CLARIFICATION)

**Source:** `.aether/aether-utils.sh` lines 961-987

The documentation says "auto-promotion triggers after 2 observations" but the actual thresholds vary by type:

| Wisdom Type | Propose Threshold | Auto Threshold |
|-------------|:---:|:---:|
| pattern | 1 | 1 |
| philosophy | 1 | 3 |
| redirect | 1 | 2 |
| stack | 1 | 2 |
| decree | 0 | 0 |
| failure | 1 | 2 |
| build_learning | 0 | 0 |
| instinct | 0 | 0 |

The "2 observations" claim is only accurate for redirect, stack, and failure types. Pattern auto-promotes after just 1 observation. Philosophy requires 3.

**Confidence:** HIGH

---

### Finding 9: learning-promote-auto has robust deduplication and guard clauses (PASS)

**Source:** `.aether/utils/learning.sh` lines 366-448

The auto-promotion function includes:
- Threshold check against policy (exits early if not met)
- Missing QUEEN.md check (exits early)
- Content dedup: `grep -Fq "$content" "$queen_file"` prevents re-promoting already-present wisdom
- Recurrence-calibrated confidence: `min(0.7 + (obs_count - 1) * 0.05, 0.9)`
- Creates instinct after successful queen-promote

**Confidence:** HIGH

---

### Finding 10: instinct-create has fuzzy dedup via Jaccard similarity (PASS)

**Source:** `.aether/utils/learning.sh` lines 1492-1661

The instinct creation includes:
- Exact dedup: checks trigger+action match, boosts confidence +0.1 if found
- Fuzzy dedup: Jaccard similarity >= 0.80 on both trigger AND action triggers merge
- Cap enforcement: max 30 instincts, sorted by confidence, lowest evicted
- Uses _state_mutate for atomic writes to COLONY_STATE.json

**Confidence:** HIGH

---

### Finding 11: learning-observe has backup rotation and corruption recovery (PASS)

**Source:** `.aether/utils/learning.sh` lines 92-309

Robust data safety features:
- 3-tier backup rotation (.bak.1, .bak.2, .bak.3) before every write
- Corruption detection with backup recovery attempt
- Lock acquisition for concurrent access (if lock system available)
- Content hash dedup via SHA-256

**Confidence:** HIGH

---

## Recommendations

### Rec 1: Fix queen-read `sources` field (BUG FIX)

Replace the metadata-derived sources with the actual filesystem check results. Pass `has_global` and `has_local` as jq variables:

```bash
result=$(jq -n \
  --argjson meta "$metadata" \
  --argjson actual_has_global "$has_global" \
  --argjson actual_has_local "$has_local" \
  ...
  '{
    ...
    sources: {
      has_global: $actual_has_global,
      has_local: $actual_has_local
    }
  }')
```

**Rationale:** The current logic reads a nonexistent `source` field from metadata, producing always-false results.
**Based on:** Finding 1

### Rec 2: Fix queen-read `priming` detection to check per-source (BUG FIX)

Instead of testing the combined string for placeholder text, test only the LOCAL wisdom (or test each source independently). If either global or local has real content, the priming flag should be true.

**Rationale:** Concatenation with placeholder text from the global file poisons the regex check.
**Based on:** Finding 2

### Rec 3: Handle single-line METADATA extraction (BUG FIX)

Replace the `sed '1d;$d'` approach with something that handles both formats:
```bash
metadata=$(sed -n '/<!-- METADATA/,/-->/p' "$queen_file" | sed 's/<!-- METADATA//;s/-->//' | tr -d '\n' | sed 's/^[[:space:]]*//')
```

**Rationale:** The current approach silently fails on the global QUEEN.md format. While mitigated by local-first priority, this will bite any repo that relies solely on global wisdom.
**Based on:** Finding 4

### Rec 4: Accept evolution_log drift as non-critical (NO ACTION)

The JSON evolution_log in METADATA drifts from the markdown table because the awk-based JSON manipulation in queen-promote is fragile. Since no critical path reads the JSON evolution_log, this is low priority. If it becomes important, switch to jq-based manipulation for the metadata block.

**Rationale:** Fixing awk-based JSON editing is high effort and error-prone. The markdown table is the authoritative visual record.
**Based on:** Finding 3

### Rec 5: Update documentation on auto-promotion thresholds (DOCS)

CLAUDE.md and other docs state "auto-promotion triggers after 2 observations" which is only accurate for redirect/stack/failure types. Pattern auto-promotes after 1 observation. Philosophy requires 3.

**Rationale:** Incorrect threshold documentation may lead to confusion in audit and testing.
**Based on:** Finding 8

---

## Open Questions

1. **Is the `sources` field in queen-read actively consumed by any downstream worker or command?** If not consumed, the fix is lower priority. If colony-prime uses it to decide whether to inject global vs local wisdom, it's critical.

2. **Should the global QUEEN.md placeholders be cleaned up?** The global file at `~/.aether/QUEEN.md` still has placeholder text in all sections. Since it's combined with local, this pollutes the priming detection AND the actual wisdom text sent to workers (description paragraphs appear twice).

3. **Should the auto-promotion threshold for `pattern` type (currently 1) be raised to 2?** Auto-promoting after a single observation seems aggressive and may produce low-quality wisdom entries.

---

## Sources

| Source | Path/URL |
|--------|----------|
| queen.sh | `.aether/utils/queen.sh` |
| learning.sh | `.aether/utils/learning.sh` |
| aether-utils.sh (dispatcher) | `.aether/aether-utils.sh`, lines 3627-3741 |
| aether-utils.sh (thresholds) | `.aether/aether-utils.sh`, lines 961-1003 |
| Local QUEEN.md | `.aether/QUEEN.md` |
| Global QUEEN.md | `~/.aether/QUEEN.md` |
| learning-observations.json | `.aether/data/learning-observations.json` |
| Queen module tests | `tests/bash/test-queen-module.sh` |

---

## Signals Acknowledged

- **FOCUS (AUDIT colony):** All investigation was read-only. No modifications made to any file. Bugs identified and documented for downstream workers to fix.
- **REDIRECT (generate-ant-name):** Not applicable to this research task, acknowledged.
