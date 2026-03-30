# Architect Design: Phase 3 -- QUEEN.md and Wisdom Pipeline Integrity

**Architect:** Schema-13
**Phase:** 3
**Date:** 2026-03-28
**Colony Goal:** Comprehensive audit of session work
**Mode:** AUDIT (read-only verification, bug documentation)

---

## Context

Phase 3 audits two subsystems critical to colony intelligence:

1. **queen-read** -- Reads QUEEN.md (global + local) and returns structured JSON for worker priming
2. **Wisdom pipeline chain** -- The full observation-to-wisdom flow: `memory-capture` -> `learning-observe` -> `learning-promote-auto` -> `instinct-create` -> `queen-promote`

Oracle research (Mystic-36, `.aether/oracle/research-phase-3.md`) has already identified specific bugs and passes. This design defines the exact verification steps Builder must execute to confirm those findings and document the current system state.

**Key constraint:** This is an AUDIT colony. All verification is read-only unless a bug is confirmed, in which case findings are documented but code is NOT modified.

---

## Design Decisions

### Decision 1: Verify queen-read via live execution, not test mocking

**Rationale:** The Oracle found bugs in queen-read's `sources` and `priming` fields. These bugs are in the interaction between global placeholder text and local real content -- mocked tests would not catch them. Live execution against the actual QUEEN.md files is the correct verification approach.

**Approach:** Run `bash .aether/aether-utils.sh queen-read` and parse the JSON output with jq to validate each field independently.

**Alternatives considered:**
- Running existing bash tests in `tests/bash/test-queen-module.sh` -- these use isolated temp dirs with synthetic QUEEN.md files and would not reproduce the global+local interaction bugs
- Writing new tests -- violates AUDIT mode (no code modification)

**Tradeoffs:** Live execution depends on the current state of `~/.aether/QUEEN.md` and `.aether/QUEEN.md`. Results are snapshot-in-time, not repeatable CI tests.

### Decision 2: Validate wisdom pipeline chain via dispatch route tracing, not end-to-end execution

**Rationale:** Running `memory-capture` would modify `learning-observations.json`, `pheromones.json`, and potentially `COLONY_STATE.json` and `QUEEN.md` -- violating AUDIT mode. Instead, verify the chain by confirming: (a) dispatch routes exist, (b) each function's guards and dedup logic are present, (c) current data files reflect expected pipeline outputs.

**Approach:** Trace the dispatch chain in `aether-utils.sh` (lines 3614-3972) and `learning.sh`, then cross-reference against actual data state in `learning-observations.json` and `COLONY_STATE.json`.

**Alternatives considered:**
- End-to-end execution in a temp dir -- would prove the chain works but adds complexity and does not validate the REAL colony data
- Only reading the code -- insufficient; must also verify data integrity

**Tradeoffs:** Does not prove the chain works end-to-end (existing integration tests in `tests/integration/wisdom-pipeline-e2e.test.js` cover that). Does prove the chain is correctly wired and current data is consistent.

### Decision 3: Structure verification as explicit pass/fail assertions

**Rationale:** AUDIT findings must be unambiguous. Each check should produce a clear PASS or FAIL with evidence, matching the Oracle's findings format.

**Approach:** Builder runs a sequence of verification commands, each producing one of: `PASS` (expected result confirmed), `CONFIRMED-BUG` (Oracle's bug finding verified), or `UNEXPECTED` (new finding not in Oracle report).

---

## Component Structure

### Task 3.1: Verify queen-read

This task validates `queen-read` output against the actual QUEEN.md files.

#### Check 3.1.1: Valid JSON structure

**Command:**
```bash
bash .aether/aether-utils.sh queen-read 2>/dev/null | jq -e '.ok == true' >/dev/null && echo "PASS" || echo "FAIL"
```

**Expected:** PASS (Oracle Finding 5 confirmed)

**Verify keys exist:**
```bash
bash .aether/aether-utils.sh queen-read 2>/dev/null | jq -e '.result | has("metadata", "wisdom", "priming", "sources")' >/dev/null && echo "PASS" || echo "FAIL"
```

#### Check 3.1.2: Metadata matches QUEEN.md

**Commands:**
```bash
# Get metadata from queen-read
QR=$(bash .aether/aether-utils.sh queen-read 2>/dev/null)

# Verify version
echo "$QR" | jq -r '.result.metadata.version' # Expected: "2.0.0"

# Verify stats match actual counts
echo "$QR" | jq '.result.metadata.stats'
# Expected: {"total_user_prefs":2,"total_codebase_patterns":12,"total_build_learnings":1,"total_instincts":3}
```

Cross-reference against QUEEN.md:
- Count `## User Preferences` entries: should be 2 (the charter Intent and Vision lines)
- Count `## Codebase Patterns` entries: should be 12 (6 repo + 5 hive + 1 charter)
- Count `## Build Learnings` entries: should be 1 (Phase 0 migration-test)
- Count `## Instincts` entries: should be 3

**Expected:** PASS (Oracle Finding 6 confirmed)

#### Check 3.1.3: Sources field (KNOWN BUG)

**Command:**
```bash
QR=$(bash .aether/aether-utils.sh queen-read 2>/dev/null)
echo "$QR" | jq '.result.sources'
# Expected: {"has_global": false, "has_local": false}
```

**Expected:** CONFIRMED-BUG -- Both return false despite both files existing.

**Root cause (from Oracle Finding 1):** Lines 286-289 of `queen.sh` check `$meta.source` which does not exist in the METADATA JSON. The METADATA has no `source` field. The `has_global` and `has_local` bash variables (lines 190-201) are correctly set but never passed to the jq expression.

**Evidence to capture:** Show that `~/.aether/QUEEN.md` exists AND `.aether/QUEEN.md` exists, while sources reports both false.

#### Check 3.1.4: Priming flags (KNOWN BUG)

**Command:**
```bash
QR=$(bash .aether/aether-utils.sh queen-read 2>/dev/null)
echo "$QR" | jq '.result.priming'
# Expected: {"has_user_prefs": true, "has_codebase_patterns": false, "has_build_learnings": false, "has_instincts": false}
```

**Expected:** CONFIRMED-BUG -- `has_codebase_patterns`, `has_build_learnings`, and `has_instincts` return false despite real content existing.

**Root cause (from Oracle Finding 2):** The global `~/.aether/QUEEN.md` contains placeholder text like `*No codebase patterns recorded yet.*`. When global + local wisdom is concatenated (lines 222-238), the combined string includes both the placeholder AND real content. The regex test at lines 281-284 matches the placeholder text and returns false for the whole combined string.

**Evidence to capture:** Show that local QUEEN.md has 12 codebase patterns while priming says false. Show the global QUEEN.md placeholder text that causes the poisoning.

#### Check 3.1.5: Metadata evolution_log drift (KNOWN DATA DRIFT)

**Command:**
```bash
QR=$(bash .aether/aether-utils.sh queen-read 2>/dev/null)
echo "$QR" | jq '.result.metadata.evolution_log | length'
# Expected: 2

# Count markdown evolution log data rows (exclude header rows)
grep -c "^| 20" .aether/QUEEN.md
# Expected: ~20 (data rows starting with date)
```

**Expected:** CONFIRMED-DRIFT -- JSON has 2 entries, markdown has ~20 data rows.

**Root cause (from Oracle Finding 3):** The awk-based append logic in `queen-promote` does not reliably update the JSON `evolution_log` array inside the METADATA comment block.

#### Check 3.1.6: Single-line METADATA extraction (LATENT BUG)

**Command:**
```bash
# Show global QUEEN.md metadata format (single-line)
grep "METADATA" ~/.aether/QUEEN.md

# Show local QUEEN.md metadata format (multi-line)
grep -c "METADATA" .aether/QUEEN.md
```

**Expected:** CONFIRMED-BUG (latent) -- Global uses single-line `<!-- METADATA {...} -->` format. The `sed '1d;$d'` extraction on line 243 of queen.sh would delete the entire content for single-line format. Currently mitigated because local QUEEN.md (multi-line) takes priority for metadata extraction.

### Task 3.2: Verify wisdom pipeline chain

This task validates the pipeline: `memory-capture` -> `learning-observe` -> `learning-promote-auto` -> `instinct-create` -> `queen-promote`.

#### Check 3.2.1: Dispatch routes exist

**Commands:**
```bash
# Verify all dispatch routes in aether-utils.sh
grep -n "memory-capture)" .aether/aether-utils.sh
grep -n "learning-observe)" .aether/aether-utils.sh
grep -n "learning-promote-auto)" .aether/aether-utils.sh
grep -n "instinct-create)" .aether/aether-utils.sh
grep -n "queen-promote)" .aether/aether-utils.sh
```

**Expected:** PASS -- All 5 dispatch routes present (Oracle Finding 7 confirmed)

#### Check 3.2.2: memory-capture chains correctly

Verify that `memory-capture` (lines 3627-3741) calls:
1. `learning-observe` (line 3658)
2. `pheromone-write` (line 3713)
3. `learning-promote-auto` (line 3724)

**Commands:**
```bash
sed -n '3627,3741p' .aether/aether-utils.sh | grep -c "learning-observe"
sed -n '3627,3741p' .aether/aether-utils.sh | grep -c "pheromone-write"
sed -n '3627,3741p' .aether/aether-utils.sh | grep -c "learning-promote-auto"
```

**Expected:** PASS -- All three subprocess calls present

#### Check 3.2.3: learning-promote-auto chains correctly

Verify that `learning-promote-auto` (learning.sh lines 366-448) calls:
1. `queen-promote` (line 423)
2. `instinct-create` (lines 426-434)

**Commands:**
```bash
sed -n '366,448p' .aether/utils/learning.sh | grep -c "queen-promote"
sed -n '366,448p' .aether/utils/learning.sh | grep -c "instinct-create"
```

**Expected:** PASS

#### Check 3.2.4: Auto-promotion threshold policy

**Command:**
```bash
bash .aether/aether-utils.sh queen-thresholds 2>/dev/null | jq '.result'
```

**Expected output (from Oracle Finding 8):**

| Type | Propose | Auto |
|------|:---:|:---:|
| pattern | 1 | 1 |
| philosophy | 1 | 3 |
| redirect | 1 | 2 |
| stack | 1 | 2 |
| decree | 0 | 0 |
| failure | 1 | 2 |
| build_learning | 0 | 0 |
| instinct | 0 | 0 |

Note: Documentation says "2 observations" but actual thresholds vary by type. This is a documentation accuracy issue, not a code bug.

#### Check 3.2.5: learning-observe dedup and safety

Verify these guards exist in learning.sh:
1. Content hash dedup via SHA-256 (line 118)
2. Backup rotation before writes (lines 187-191)
3. Corruption recovery with backup fallback (lines 135-170)
4. Lock acquisition (lines 172-176)

**Commands:**
```bash
grep -c "sha256" .aether/utils/learning.sh
grep -c "bak\." .aether/utils/learning.sh
grep -c "acquire_lock" .aether/utils/learning.sh
```

**Expected:** PASS (Oracle Finding 11 confirmed) -- sha256 >= 2, bak >= 6, acquire_lock >= 1

#### Check 3.2.6: instinct-create dedup and cap

Verify in learning.sh:
1. Exact dedup: trigger+action match (line 1525)
2. Fuzzy dedup: Jaccard similarity >= 0.80 (line 1571)
3. Cap enforcement: max 30, sorted by confidence (lines 1653-1654)

**Commands:**
```bash
grep -c "jaccard" .aether/utils/learning.sh
grep "sort_by" .aether/utils/learning.sh
grep "\\[:30\\]" .aether/utils/learning.sh
```

**Expected:** PASS (Oracle Finding 10 confirmed) -- jaccard >= 2, sort_by present, [:30] cap present

#### Check 3.2.7: Current data integrity

Verify the current state of pipeline data files:

```bash
# Observations file: count entries
jq '.observations | length' .aether/data/learning-observations.json
# Expected: 42

# Instincts in COLONY_STATE: count entries
jq '.memory.instincts | length' .aether/data/COLONY_STATE.json
# Expected: 6

# QUEEN.md instincts section: count instinct entries
grep -c "\[instinct\]" .aether/QUEEN.md
# Expected: 3

# Cross-reference: instinct actions in COLONY_STATE should appear in QUEEN.md
# (promoted instincts with confidence >= 0.8)
jq -r '.memory.instincts[] | select(.confidence >= 0.8) | .action' .aether/data/COLONY_STATE.json
```

**Expected:** PASS -- Data files are consistent with expected pipeline outputs.

---

## Data Flow

```
memory-capture("learning", "content", "pattern")
  |
  +-> learning-observe("content", "pattern", "colony_name")
  |     |-> SHA-256 content hash
  |     |-> Check existing observations (dedup)
  |     |-> Increment count or create new observation
  |     |-> Return: {observation_count, threshold, threshold_met}
  |
  +-> pheromone-write(FEEDBACK/REDIRECT, derived_content, ...)
  |     |-> Auto-emit pheromone signal based on event_type
  |
  +-> learning-promote-auto("pattern", "content", "colony_name")
        |-> Check threshold (pattern=1, philosophy=3, etc.)
        |-> Check if already in QUEEN.md (content dedup)
        |-> If threshold met AND not duplicate:
              |
              +-> queen-promote("pattern", "content", "colony_name")
              |     |-> Write to QUEEN.md Codebase Patterns section
              |     |-> Update METADATA stats
              |     |-> Append to evolution_log table
              |
              +-> instinct-create(--trigger, --action, --confidence, ...)
                    |-> Exact dedup check
                    |-> Fuzzy dedup (Jaccard >= 0.80)
                    |-> Create/merge/update in COLONY_STATE.json
                    |-> Enforce 30-instinct cap
```

---

## Interfaces

No new interfaces are being created. This is an audit of existing interfaces:

| Interface | Location | Consumer |
|-----------|----------|----------|
| `queen-read` return JSON | `queen.sh:183-298` | `colony-prime` (worker priming) |
| `memory-capture` return JSON | `aether-utils.sh:3627-3741` | Build/continue commands |
| `learning-observe` return JSON | `learning.sh:97-309` | `memory-capture` |
| `learning-promote-auto` return JSON | `learning.sh:366-448` | `memory-capture` |
| `instinct-create` return JSON | `learning.sh:1492-1661` | `learning-promote-auto`, direct calls |
| `queen-promote` return JSON | `queen.sh:317+` | `learning-promote-auto`, `/ant:seal` |

---

## Implementation Notes for Builder

### Execution Order

Execute Task 3.1 checks first (queen-read is simpler, provides baseline). Then execute Task 3.2 checks.

### Output Format

For each check, Builder should produce:

```
CHECK 3.1.1: Valid JSON structure
STATUS: PASS
EVIDENCE: queen-read returns {"ok": true, ...} with all required keys
```

or

```
CHECK 3.1.3: Sources field
STATUS: CONFIRMED-BUG (Oracle Finding 1)
EVIDENCE: has_global=false, has_local=false despite both files existing
ROOT CAUSE: $meta.source field does not exist in METADATA JSON
```

### Key Files to Read (Not Modify)

| File | Purpose |
|------|---------|
| `.aether/utils/queen.sh` lines 183-298 | `_queen_read` function |
| `.aether/aether-utils.sh` lines 3627-3741 | `memory-capture` dispatcher |
| `.aether/utils/learning.sh` lines 97-309 | `_learning_observe` |
| `.aether/utils/learning.sh` lines 366-448 | `_learning_promote_auto` |
| `.aether/utils/learning.sh` lines 1492-1661 | `_instinct_create` |
| `.aether/QUEEN.md` | Local wisdom file (has real content) |
| `~/.aether/QUEEN.md` | Global wisdom file (has placeholder text) |
| `.aether/data/learning-observations.json` | Observation records |
| `.aether/data/COLONY_STATE.json` | Instincts storage |

### Commands That Are Safe to Run

All verification commands are read-only:
- `bash .aether/aether-utils.sh queen-read` -- reads QUEEN.md, returns JSON, no side effects
- `bash .aether/aether-utils.sh queen-thresholds` -- returns threshold config, no side effects
- `jq` queries against data files -- read-only
- `grep` / `sed -n` for code inspection -- read-only

### Commands That Must NOT Be Run

- `memory-capture` -- writes to learning-observations.json, pheromones.json, potentially QUEEN.md
- `learning-observe` -- writes to learning-observations.json
- `queen-promote` -- writes to QUEEN.md
- `instinct-create` -- writes to COLONY_STATE.json
- Any `--write` or mutation command

### Expected Bug Summary

| ID | Bug | Severity | Status |
|----|-----|----------|--------|
| BUG-1 | queen-read `sources` always false/false | Medium | Confirm via live execution |
| BUG-2 | queen-read `priming` poisoned by global placeholders | High | Confirm via live execution |
| BUG-3 | Single-line METADATA extraction fails | Low (latent) | Confirm format difference |
| DRIFT-1 | evolution_log JSON vs markdown drift | Low | Confirm count mismatch |

---

## Tradeoffs

1. **Live execution vs isolated tests:** We verify against real data, which gives us true system state but is not repeatable in CI. The existing test suite (`tests/bash/test-queen-module.sh`, `tests/integration/wisdom-pipeline-e2e.test.js`) provides the repeatable coverage.

2. **Read-only pipeline verification vs end-to-end execution:** We trace code paths and verify data consistency rather than exercising the full pipeline. This respects AUDIT mode but does not prove the chain executes correctly from scratch. The existing integration tests cover that.

3. **Documenting bugs vs fixing them:** AUDIT mode requires documentation only. Fixes would be a separate phase if the colony decides to act on the findings.

---

## Signals Acknowledged

- **FOCUS (AUDIT colony):** All checks designed as read-only verification. No commands that modify data are included. Builder must not run mutation commands.
- **REDIRECT (generate-ant-name piping):** Not applicable to this phase, acknowledged. No ant naming occurs in queen-read or wisdom pipeline verification.
