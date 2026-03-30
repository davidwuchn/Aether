# Architect Design: Phase 6 -- Stabilize spawn-tree parsing and JSON output

**Phase ID:** 6
**Architect:** Draft-80
**Date:** 2026-03-29
**Oracle Research:** `.aether/oracle/research-phase-6.md` (Mystic-70)

---

## Context

Three blockers prevent the spawn-tree subsystem from producing correct JSON output:
1. O(n^2) subprocess forking in `parse_spawn_tree()` causes test timeouts (8.5s for 82 lines)
2. Ant names and task summaries containing special characters can break JSON output
3. `_spawn_tree_active()` and `_spawn_tree_depth()` pass raw output to `json_ok` without structural validation

Additionally, test fixtures use stale 6-field format (5 pipes) while production uses 7-field format (6 pipes), causing tests to pass for wrong reasons.

### Signals Acknowledged

- **REDIRECT (0.9):** Minimal changes only -- every change below is the smallest fix for its blocker.
- **REDIRECT (0.7):** O(n^2) -- replaced with single-pass awk (O(n)).
- **REDIRECT (0.7):** JSON objects as ant names -- escaped via awk `gsub()` on all string fields.
- **REDIRECT (0.7):** spawn-tree-load raw output -- `jq` validation added to all three wrappers.

### Archaeologist Constraints (Preserved)

- 7-field format (pipe_count == 6): Preserved. Awk uses `NF == 7 && $7 == "spawned"`.
- JSON escaping (commit 1b82242): Maintained via awk `gsub()` on name, parent, task fields.
- Bash 3.2 compatible: Awk runs as external process; no bash associative arrays used.
- Test fixtures: Updated from 6-field to 7-field format.
- spawn-tree.txt format: Unchanged. Only the parser changes.
- Safety limit of 5 for depth: `get_spawn_depth` and `get_spawn_lineage` are NOT modified.

---

## Design Decisions

### Decision 1: Replace `parse_spawn_tree()` body with single awk invocation

**What changes:** The function body of `parse_spawn_tree()` in `.aether/utils/spawn-tree.sh` (lines 15-221).

**What stays the same:**
- Function signature: `parse_spawn_tree [file_path]`
- Missing-file guard at top (lines 19-22) -- keep as-is
- Output JSON schema (spawns array with name/parent/caste/task/status/spawned_at/completed_at/children, metadata object with total_count/active_count/completed_count/file_exists)
- The tmpdir cleanup at line 220 is removed (no longer needed -- awk uses no temp files)

**Implementation approach:**
- Single `awk -F'|'` command replaces lines 24-217
- Spawn detection: `NF == 7 && $7 == "spawned"`
- Completion detection: `$3 ~ /^(completed|failed|blocked)$/ && !(NF == 7 && $7 == "spawned")`
  - The `!(NF == 7 && $7 == "spawned")` guard prevents a theoretical spawn line from matching the completion rule (in practice castes never equal "completed", but defense-in-depth costs nothing)
- All data stored in awk arrays indexed by integer `n`
- `name_to_idx[$4]` maps ant names to array indices for O(1) completion and parent lookups
- JSON output via `printf` in END block
- JSON escaping: `gsub(/\\/, "\\\\", var); gsub(/"/, "\\\"", var); gsub(/\t/, "\\t", var)` on name, parent, task, and child-name fields

**Rationale:** Eliminates ~4,000 subprocess forks. Expected time: < 0.1s for 82 lines vs current 8.5s. This is the smallest change that fixes the O(n^2) blocker -- a single function body replacement.

**Tradeoffs:** Awk code is less readable than bash for developers unfamiliar with awk. The function becomes a single awk invocation with embedded logic rather than line-by-line bash. This is acceptable because the function has a single clear purpose (parse file to JSON) and the awk is well-structured with comments.

### Decision 2: Replace `get_active_spawns()` body with single awk invocation

**What changes:** The function body of `get_active_spawns()` in `.aether/utils/spawn-tree.sh` (lines 272-331).

**What stays the same:**
- Function signature: `get_active_spawns [file_path]`
- Missing-file guard (lines 275-278)
- Output JSON schema: array of objects with name/caste/parent/task/spawned_at

**Implementation approach:**
- Single `awk -F'|'` command
- First rule collects completions into `done_set[$2]` (fires on completion lines)
- Second rule collects spawn data into arrays (fires on spawn lines)
- END block iterates spawns, emitting only those NOT in `done_set`
- Same JSON escaping as Decision 1

**Rationale:** Same O(n^2) issue as parse_spawn_tree -- per-spawn `grep` for completion status is eliminated.

### Decision 3: Add `jq` validation to `_spawn_tree_active()` and `_spawn_tree_depth()` wrappers

**What changes:** Two functions in `.aether/utils/spawn.sh` (lines 208-226).

**_spawn_tree_active -- before:**
```bash
_spawn_tree_active() {
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || { ... }
    active=$(get_active_spawns)
    json_ok "$active"
}
```

**_spawn_tree_active -- after:**
```bash
_spawn_tree_active() {
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    active=$(get_active_spawns)
    if echo "$active" | jq -e . >/dev/null 2>&1; then
      json_ok "$active"
    else
      json_err "$E_VALIDATION_FAILED" "spawn-tree active produced invalid JSON"
      return 1
    fi
}
```

**_spawn_tree_depth -- before:**
```bash
_spawn_tree_depth() {
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-tree-depth <ant_name>"
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || { ... }
    depth=$(get_spawn_depth "$ant_name")
    json_ok "$depth"
}
```

**_spawn_tree_depth -- after:**
```bash
_spawn_tree_depth() {
    ant_name="${1:-}"
    [[ -z "$ant_name" ]] && json_err "$E_VALIDATION_FAILED" "Usage: spawn-tree-depth <ant_name>"
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || {
      json_err "$E_FILE_NOT_FOUND" "spawn-tree.sh not found"
      exit 1
    }
    depth=$(get_spawn_depth "$ant_name")
    if echo "$depth" | jq -e . >/dev/null 2>&1; then
      json_ok "$depth"
    else
      json_err "$E_VALIDATION_FAILED" "spawn-tree depth produced invalid JSON"
      return 1
    fi
}
```

**Rationale:** Matches the existing validation pattern already in `_spawn_tree_load()` (line 200). Ensures invalid JSON never reaches callers wrapped in `{"ok":true,"result":...}`.

### Decision 4: Fix test fixtures to use 7-field format

**What changes:** Two test data blocks in `tests/unit/spawn-tree.test.js`.

**Lines 126-129 (deep chain test):** Add `|default` model field before `|spawned`:
```javascript
const testData = [
  '2026-02-13T10:00:00Z|Queen|builder|Level1|Task 1|default|spawned',
  '2026-02-13T10:01:00Z|Level1|builder|Level2|Task 2|default|spawned',
  '2026-02-13T10:02:00Z|Level2|builder|Level3|Task 3|default|spawned'
].join('\n');
```

**Lines 197-203 (empty active spawns test):** Add `|default` model field before `|spawned`:
```javascript
const testData = [
  '2026-02-13T10:00:00Z|Queen|builder|Done1|Task 1|default|spawned',
  '2026-02-13T10:01:00Z|Done1|completed|Completed task',
  '2026-02-13T10:02:00Z|Queen|builder|Done2|Task 2|default|spawned',
  '2026-02-13T10:03:00Z|Done2|completed|Completed task'
].join('\n');
```

**Rationale:** Fixtures must match production format for tests to exercise real code paths. Completion lines keep 4 fields (they are format-correct already).

### Decision 5: Do NOT change `get_spawn_depth()`, `get_spawn_lineage()`, or `get_spawn_children()`

These three functions use `grep` directly on spawn-tree.txt. They are already O(1) per call (single grep) and are not in the failing test path. Changing them would violate the REDIRECT for minimal changes.

`get_spawn_children()` has the same subprocess-per-line pattern as the other two, but it has no failing tests and is not a measured performance bottleneck. Defer.

### Decision 6: Do NOT change the test timeout (10s)

The awk rewrite reduces parse time from 8.5s to < 0.1s. The 10-second timeout becomes a generous regression guard rather than a tight constraint.

---

## Component Structure

### Files Modified (3 files)

| File | Change | Size of Change |
|------|--------|---------------|
| `.aether/utils/spawn-tree.sh` | Replace `parse_spawn_tree()` body (lines 24-220) and `get_active_spawns()` body (lines 280-330) with awk | ~200 lines removed, ~100 lines added |
| `.aether/utils/spawn.sh` | Add `jq` validation to `_spawn_tree_active()` and `_spawn_tree_depth()` | ~8 lines added |
| `tests/unit/spawn-tree.test.js` | Fix 5 fixture lines from 6-field to 7-field format | 5 lines changed |

### Files NOT Modified

| File | Why |
|------|-----|
| `.aether/aether-utils.sh` | Dispatch case statements unchanged |
| `.aether/data/spawn-tree.txt` | Live data; format unchanged |
| Any other test file | Only spawn-tree tests affected |

---

## Data Flow

### Before (current -- O(n^2))

```
spawn-tree.txt
  -> parse_spawn_tree() [bash while-read loop]
    -> per line: echo|tr|wc|tr (pipe count) + 7x echo|cut (field extract)
    -> per completion: while-read names_file + 2x sed -i (status update)
    -> per spawn: sed -n (parent read) + while-read names_file + sed -i (children update)
    -> per spawn: 8x sed -n (field read) + 3x sed (JSON escape)
  -> JSON string assembled via bash concatenation
  -> _spawn_tree_load validates with jq
  -> json_ok wraps in {"ok":true,"result":...}
```

### After (fixed -- O(n))

```
spawn-tree.txt
  -> parse_spawn_tree() [single awk invocation]
    -> awk reads file once, builds arrays in memory
    -> awk END block outputs JSON via printf
  -> _spawn_tree_load validates with jq
  -> json_ok wraps in {"ok":true,"result":...}
```

### Unchanged paths

- `get_spawn_depth()`: grep-based, already O(1) per call
- `get_spawn_lineage()`: grep-based, already O(chain_length) per call
- `get_spawn_children()`: while-read loop, unchanged (no failing tests)

---

## Implementation Notes for Builder

### Task 6.1: Fix the 3 blockers

**Order of implementation:**

1. **First:** Replace `parse_spawn_tree()` body in `.aether/utils/spawn-tree.sh`
   - Keep lines 15-22 (function signature + missing-file guard)
   - Replace lines 24-220 with awk invocation (see Oracle sketch in `research-phase-6.md` lines 254-338)
   - Remove the tmpdir creation, file touches, while-read loop, all sed operations, and the tmpdir cleanup
   - The awk END block must output JSON with no trailing newline issue -- use `printf` not `print` for the final line

2. **Second:** Replace `get_active_spawns()` body in `.aether/utils/spawn-tree.sh`
   - Keep lines 272-278 (function signature + missing-file guard)
   - Replace lines 280-330 with awk invocation (see Oracle sketch in `research-phase-6.md` lines 343-383)

3. **Third:** Add `jq` validation to `.aether/utils/spawn.sh`
   - Modify `_spawn_tree_active()` (line 208): wrap `json_ok "$active"` in jq validation
   - Modify `_spawn_tree_depth()` (line 217): wrap `json_ok "$depth"` in jq validation

### Critical details for the awk implementation

1. **Completion detection must check field 3 content, NOT field count.** Use `$3 ~ /^(completed|failed|blocked)$/` because completion summaries can contain pipe characters (see Anvil-57 line in live data).

2. **Guard against spawn lines matching completion rule.** Add `&& !(NF == 7 && $7 == "spawned")` to the completion detection rule. This is defense-in-depth.

3. **JSON escaping must be applied to name, parent, task, and child-name fields.** Use awk's `gsub()`:
   ```awk
   gsub(/\\/, "\\\\", var)
   gsub(/"/, "\\\"", var)
   gsub(/\t/, "\\t", var)
   ```

4. **The children array uses integer indices stored as space-separated string.** In END block, `split()` the string and look up names by index. Use `cidxs[j]+0` to force numeric coercion.

5. **printf must NOT add a trailing newline to the final output.** The `json_ok` wrapper adds its own newline. Use `printf "}"` not `print "}"` at the end.

6. **The awk script must be single-quoted** to prevent shell expansion of `$1`, `$2`, etc.

### Task 6.2: Validate end-to-end

**Test fixture changes in `tests/unit/spawn-tree.test.js`:**

- Line 128: `'2026-02-13T10:00:00Z|Queen|builder|Level1|Task 1|spawned'` -> `'2026-02-13T10:00:00Z|Queen|builder|Level1|Task 1|default|spawned'`
- Line 129: `'2026-02-13T10:01:00Z|Level1|builder|Level2|Task 2|spawned'` -> `'2026-02-13T10:01:00Z|Level1|builder|Level2|Task 2|default|spawned'`
- Line 130: `'2026-02-13T10:02:00Z|Level2|builder|Level3|Task 3|spawned'` -> `'2026-02-13T10:02:00Z|Level2|builder|Level3|Task 3|default|spawned'`
- Line 199: `'2026-02-13T10:00:00Z|Queen|builder|Done1|Task 1|spawned'` -> `'2026-02-13T10:00:00Z|Queen|builder|Done1|Task 1|default|spawned'`
- Line 202: `'2026-02-13T10:02:00Z|Queen|builder|Done2|Task 2|spawned'` -> `'2026-02-13T10:02:00Z|Queen|builder|Done2|Task 2|default|spawned'`

**Verification steps:**
1. Run `npm test -- tests/unit/spawn-tree.test.js` -- all 9 tests should pass
2. Run `bash .aether/aether-utils.sh spawn-tree-load` -- should return valid JSON with `"ok":true`
3. Run `bash .aether/aether-utils.sh spawn-tree-active` -- should return valid JSON
4. Verify parse time is under 1 second for live 85-line spawn-tree.txt

---

## Tradeoffs

| Tradeoff | Chose | Over | Rationale |
|----------|-------|------|-----------|
| Awk for parsing | Single awk invocation | Bash optimization (read+IFS splitting) | Awk eliminates ALL subprocess forks; bash optimization would still need sed for parent-child building |
| Defense-in-depth completion guard | Add `!(NF == 7 && $7 == "spawned")` | Trust that castes never equal "completed" | Zero cost, prevents future format changes from introducing bugs |
| Leave get_spawn_children unchanged | Defer awk rewrite | Rewrite now | No failing tests, REDIRECT says minimal changes only |
| Leave get_spawn_depth/lineage unchanged | Keep grep-based | Consolidate into awk | Already O(1), no performance issue, changing adds risk |
| 10s test timeout | Keep as-is | Increase to 30s (Sibyl-19 recommendation) | Awk fix removes root cause; 10s becomes regression guard |

---

## Risks

1. **macOS awk dialect:** macOS ships `nawk` (not `gawk`). The awk features used (associative arrays, `gsub`, `NF`, `split`, `printf`) are all POSIX-compliant and work on `nawk`. No risk.
2. **Empty spawn-tree.txt:** The missing-file guard handles this. An empty file produces `{"spawns":[],"metadata":{...,"file_exists":true}}` because awk's END block still runs with `n=0`.
3. **Very long lines (JSON in completion summaries):** Awk handles arbitrarily long lines by default. No buffer limit concern.
4. **Concurrent writes to spawn-tree.txt during parse:** This is a pre-existing race condition not introduced by this change. Awk reads the file once, same as the current bash approach.
