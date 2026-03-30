# Oracle Research: Phase 6 -- Stabilize spawn-tree parsing and JSON output

**Phase ID:** 6
**Researcher:** Mystic-70
**Date:** 2026-03-29
**Confidence:** High

---

## Context

Phase 6 addresses three known blockers in the spawn-tree system that cause test failures and data correctness issues. The colony goal is a comprehensive audit of session work; this phase specifically targets spawn-tree parsing stability.

### Signals Acknowledged

- **REDIRECT (0.9):** Minimal changes only -- smallest possible fix, no refactoring beyond needed.
- **REDIRECT (0.7):** Avoid O(n^2) parse_spawn_tree scaling that causes test timeouts.
- **REDIRECT (0.7):** Avoid repeating: spawn-tree-active test passes for wrong reason -- 5-pipe fixture format bypasses logic.
- **REDIRECT (0.7):** Avoid repeating: JSON objects stored as ant names break spawn-tree JSON output.
- **REDIRECT (0.7):** Avoid repeating: spawn-tree-load passes raw output without validation.

### Archaeologist Constraints (Preserved)

- The 7-field format (pipe_count == 6) MUST be preserved -- 3 functions check this.
- JSON escaping was added to 6 fields in commit 1b82242 -- must maintain.
- Bash 3.2 compatibility required (no associative arrays).
- Test fixtures use old 6-field format -- need updating.
- spawn-tree.txt format is read by external consumers.
- Safety limit of 5 for depth traversal is ISSUE-005 documented constraint.

---

## Key Findings

### Finding 1: O(n^2) Performance -- Root Cause Analysis

**Confidence:** High
**Source:** `/Users/callumcowie/repos/Aether/.aether/utils/spawn-tree.sh` lines 15-221

The `parse_spawn_tree()` function spawns approximately **4,000+ subprocesses** for the current 82-line spawn-tree.txt (42 spawns, 40 completions). Measured wall time: **8.5 seconds** (dangerously close to the 10-second test timeout).

The O(n^2) behavior comes from three nested patterns:

| Phase | Lines | Complexity | Subprocess Count | Description |
|-------|-------|-----------|-----------------|-------------|
| 1. Read and Parse | 39-99 | O(n) per line | ~622 | Each line: `echo\|tr\|wc\|tr` for pipe count + 7x `echo\|cut` for field extraction |
| 2. Completion Matching | 76-97 | O(n*m) | ~1,000 | Each completion event scans names_file linearly, then `sed -i` delete+insert |
| 3. Parent-Child Build | 106-141 | O(n^2) | ~1,974 | For each of 42 spawns: `sed -n` to read parent, then linear scan of names_file, then `sed -i` delete+insert |
| 4. JSON Output | 155-206 | O(n*k) | ~462 | For each of 42 spawns: 8x `sed -n` to read fields + 3x `sed` for JSON escaping |

**The critical bottleneck:** Each subprocess fork costs ~2-4ms on macOS. At 4,058 subprocesses, that is 8-16 seconds just in fork overhead.

**Why this matters now:** The spawn-tree.txt file has grown from 36 lines (when Sibyl-19 last measured at 11-29s) to 82 lines. Each new colony session adds 20-60 lines. Without a fix, the parser will become unusable.

### Finding 2: Minimal Fix for O(n^2) -- Replace with Single-Pass AWK

**Confidence:** High
**Source:** Analysis of `spawn-tree.sh` logic; awk is available on all macOS/Linux systems and is Bash 3.2 compatible.

The entire `parse_spawn_tree()` function can be replaced with a single `awk` invocation that:
1. Reads the file once (single pass)
2. Builds arrays in memory (awk has associative arrays even when bash 3.2 does not)
3. Outputs JSON directly

This eliminates ALL subprocess forks during parsing. Expected time: under 0.1 seconds for 82 lines.

**Key awk approach:**
- Use `awk -F'|'` with `NF == 7` for spawn detection
- Use `$3 ~ /^(completed|failed|blocked)$/` for completion detection (handles pipes-in-summary)
- Build arrays in awk: `names[]`, `parents[]`, `castes[]`, etc.
- Single END block outputs JSON

**Compatibility note:** `awk` (or `gawk`/`mawk`) is POSIX standard. The `-F'|'` flag and associative arrays are available in all awk implementations, including macOS's default `/usr/bin/awk` (which is `nawk`). This is fully Bash 3.2 compatible because awk runs as a separate process -- no bash associative arrays needed.

### Finding 3: Test Fixture Format Mismatch -- 6-Field vs 7-Field

**Confidence:** High
**Source:** `/Users/callumcowie/repos/Aether/tests/unit/spawn-tree.test.js` lines 126-129, 197-203

Test fixtures use the **old 6-field format** (5 pipes):
```
2026-02-13T10:00:00Z|Queen|builder|Level1|Task 1|spawned
```

The actual spawn-tree.txt uses the **current 7-field format** (6 pipes):
```
2026-03-27T22:57:52Z|Queen|oracle|Seer-84|Phase 1 research|opus|spawned
```

The code checks `pipe_count -eq 6` (7 fields). Test fixtures with 5 pipes are **silently ignored** by both `parse_spawn_tree()` and `get_active_spawns()`.

**Impact:**
- `spawn-tree-depth handles deep chains` test: PASSES, but only because `get_spawn_depth()` uses `grep "|$current|"` pattern matching which is format-independent.
- `spawn-tree-active returns empty array` test: **FALSE PASS** -- the fixture's spawn lines are silently skipped (not parsed), so the result is empty. The test expects empty, so it passes for the wrong reason.

**Fix:** Update test fixtures to 7-field format by adding the model field:
```
2026-02-13T10:00:00Z|Queen|builder|Level1|Task 1|default|spawned
```

### Finding 4: Pipe-In-Summary Bug -- Completion Lines Misclassified

**Confidence:** High
**Source:** `/Users/callumcowie/repos/Aether/.aether/data/spawn-tree.txt` line 40

One actual completion line contains `||` in its summary text:
```
2026-03-28T03:08:09Z|Anvil-57|completed|5 PASS checks, 2 CONFIRMED-BUGs (queen-promote pipefail without || true at grep calls)
```

This line has **5 pipes** instead of the expected 3, so the pipe-count detection (`pipe_count -eq 3`) skips it entirely. **Anvil-57's completion is never recorded** by `parse_spawn_tree`.

**Root cause:** The pipe-count approach is fundamentally fragile for lines where the summary field can contain arbitrary text (including pipe characters).

**Minimal fix for awk rewrite:** Use `$3 ~ /^(completed|failed|blocked)$/` for completion detection instead of counting fields. This checks the content of field 3 regardless of how many total fields the line has.

### Finding 5: JSON Escaping Status -- Mostly Complete

**Confidence:** High
**Source:** `/Users/callumcowie/repos/Aether/.aether/utils/spawn-tree.sh` lines 184, 191-193, 314-316; commit 1b82242 diff

In commit 1b82242, JSON escaping was added to 6 fields across two functions:
- `parse_spawn_tree()`: name, parent, task, children (4 fields escaped)
- `get_active_spawns()`: child_name, parent, task (3 fields escaped)

**Fields NOT escaped but interpolated into JSON strings:**
- `caste` -- always a simple identifier (oracle, builder, etc.), low risk
- `status` / `spawn_status` -- always a simple identifier, low risk
- `timestamp` / `completed` -- ISO 8601 format, no special chars, low risk

**Assessment:** The current escaping is adequate for actual data patterns. The awk rewrite should maintain equivalent escaping for name, parent, and task fields.

### Finding 6: spawn-tree-load Has jq Validation; spawn-tree-active and spawn-tree-depth Do Not

**Confidence:** High
**Source:** `/Users/callumcowie/repos/Aether/.aether/utils/spawn.sh` lines 194-226

The `_spawn_tree_load()` wrapper validates JSON with `jq -e .` before passing to `json_ok` (line 200). If invalid, it returns `json_err`.

The `_spawn_tree_active()` wrapper passes output **directly** to `json_ok` without any validation (line 214).

Similarly, `_spawn_tree_depth()` passes output directly to `json_ok` without validation (line 225).

**Fix:** Add the same `jq -e .` validation to both `_spawn_tree_active()` and `_spawn_tree_depth()`.

### Finding 7: NF-Based Detection Is Better Than Pipe-Counting

**Confidence:** High
**Source:** POSIX awk specification

Using `NF` (number of fields) in awk with `-F'|'` is equivalent to pipe counting but more robust because:
1. No subprocess spawning (no `echo | tr | wc`)
2. Handles edge cases better (empty fields, trailing pipes)
3. Awk splits the line once; all fields are accessible as `$1`, `$2`, etc.

For spawn events: `NF == 7` (7 fields = 6 pipes)
For completion events: Check `$3 ~ /^(completed|failed|blocked)$/` instead of counting pipes, because the summary field (field 4+) may contain pipes.

---

## Recommendations

### Recommendation 1: Replace parse_spawn_tree with Single-Pass AWK (Task 6.1, Blocker 1)

**Rationale:** This is the minimal change that fixes the O(n^2) performance. A single awk invocation replaces ~4,000 subprocess forks with in-memory array processing.

**Approach:**
1. Replace the body of `parse_spawn_tree()` with a single `awk -F'|'` command
2. Use awk's `NF == 7` for spawn detection, `$3 ~ /^(completed|failed|blocked)$/` for completion detection
3. Build all arrays in awk's main block, output JSON in END block
4. Keep the function signature identical: `parse_spawn_tree [file_path]`
5. Keep the empty-file/missing-file edge case at the top

**JSON escaping in awk:** Use `gsub(/\\/, "\\\\"); gsub(/"/, "\\\""); gsub(/\t/, "\\t")` on name, parent, and task fields before interpolation.

**Based on:** Findings 1, 2, 7

### Recommendation 2: Similarly Replace get_active_spawns with AWK (Task 6.1, Blocker 1)

**Rationale:** `get_active_spawns()` has the same subprocess-per-line pattern plus a `grep` per spawn event to check completion status. Replace with single-pass awk.

**Approach:**
1. Single awk pass: first collect completion ant names, then output non-completed spawns
2. Use the same `$3 ~ /^(completed|failed|blocked)$/` pattern for completion detection

**Based on:** Findings 1, 4, 7

### Recommendation 3: JSON-Escape Ant Names in AWK Output (Task 6.1, Blocker 2)

**Rationale:** The existing escaping logic must be preserved in the awk rewrite. Escape name ($4), parent ($2), task ($5) using awk's `gsub()`.

**Based on:** Finding 5

### Recommendation 4: Add jq Validation to _spawn_tree_active and _spawn_tree_depth (Task 6.1, Blocker 3)

**Rationale:** Only `_spawn_tree_load` validates JSON before returning. The other two wrappers pass raw output through.

**Fix in `/Users/callumcowie/repos/Aether/.aether/utils/spawn.sh`:**
```bash
_spawn_tree_active() {
    source "$SCRIPT_DIR/utils/spawn-tree.sh" 2>/dev/null || { ... }
    active=$(get_active_spawns)
    if echo "$active" | jq -e . >/dev/null 2>&1; then
      json_ok "$active"
    else
      json_err "$E_VALIDATION_FAILED" "spawn-tree active produced invalid JSON"
      return 1
    fi
}
```
Same pattern for `_spawn_tree_depth`.

**Based on:** Finding 6

### Recommendation 5: Fix Test Fixtures to Use 7-Field Format (Task 6.2)

**Rationale:** Test fixtures use old 6-field format that is silently ignored by the parser. Tests pass for wrong reasons.

**Fix in `/Users/callumcowie/repos/Aether/tests/unit/spawn-tree.test.js`:**
- Line 128: Add `|default` before `|spawned` -> `'2026-02-13T10:00:00Z|Queen|builder|Level1|Task 1|default|spawned'`
- Line 129: Same -> `'2026-02-13T10:01:00Z|Level1|builder|Level2|Task 2|default|spawned'`
- Line 130: Same -> `'2026-02-13T10:02:00Z|Level2|builder|Level3|Task 3|default|spawned'`
- Line 199: Same -> `'2026-02-13T10:00:00Z|Queen|builder|Done1|Task 1|default|spawned'`
- Line 202: Same -> `'2026-02-13T10:02:00Z|Queen|builder|Done2|Task 2|default|spawned'`

**Based on:** Finding 3

### Recommendation 6: Do NOT Change the execSync Timeout

**Rationale:** The previous research (Sibyl-19) recommended increasing the test timeout from 10s to 30s. This is a band-aid. With the awk rewrite, parse time will drop from 8.5s to under 0.1s. The 10s timeout is generous and should remain as a regression guard.

**Based on:** Findings 1, 2

---

## Risks

1. **AWK dialect differences:** macOS uses `nawk`, Linux typically uses `gawk` or `mawk`. All support associative arrays and `gsub()`. Test on macOS specifically.
2. **Completion summary with pipes:** The `$3 ~ /^(completed|failed|blocked)$/` approach handles the `||` edge case (Finding 4) because it checks field 3 content rather than field count.
3. **External consumers:** The spawn-tree.txt FILE FORMAT is unchanged. Only the PARSER changes. External consumers reading the file directly are unaffected.
4. **get_spawn_depth and get_spawn_lineage:** These use `grep` directly on the file and do NOT go through the awk parser. They are already fast (single grep per call) and do not need changes.

---

## Open Questions

1. Should `get_spawn_children()` (lines 336-373) also be rewritten to awk? It has the same subprocess-per-line pattern but is not tested in the failing tests. Suggest: defer to minimize changes per REDIRECT signal.
2. The `Anvil-57` completion event is permanently misrecorded in the current spawn-tree.txt due to pipes in its summary. The awk rewrite will handle it correctly going forward, but the existing data line remains malformed. Suggest: no manual data correction needed.

---

## Implementation Sketch (AWK parse_spawn_tree)

```bash
parse_spawn_tree() {
  local file_path="${1:-$SPAWN_TREE_FILE}"

  if [[ ! -f "$file_path" ]]; then
    echo '{"spawns":[],"metadata":{"total_count":0,"active_count":0,"completed_count":0,"file_exists":false}}'
    return 0
  fi

  awk -F'|' '
  BEGIN { n=0; active=0; completed_n=0 }

  # Spawn event: 7 fields, last field is "spawned"
  NF == 7 && $7 == "spawned" {
    names[n] = $4
    parents[n] = $2
    castes[n] = $3
    tasks[n] = $5
    statuses[n] = "spawned"
    timestamps[n] = $1
    models[n] = $6
    completed_at[n] = ""
    children_str[n] = ""
    name_to_idx[$4] = n
    n++
  }

  # Completion event: field 3 is completed/failed/blocked
  $3 ~ /^(completed|failed|blocked)$/ && NF >= 4 {
    ant = $2
    if (ant in name_to_idx) {
      idx = name_to_idx[ant]
      statuses[idx] = $3
      completed_at[idx] = $1
    }
  }

  END {
    # Build parent-child relationships
    for (i = 0; i < n; i++) {
      p = parents[i]
      if (p in name_to_idx) {
        pidx = name_to_idx[p]
        if (children_str[pidx] == "") children_str[pidx] = i
        else children_str[pidx] = children_str[pidx] " " i
      }
    }

    # Count statuses
    for (i = 0; i < n; i++) {
      if (statuses[i] == "spawned" || statuses[i] == "active") active++
      else if (statuses[i] ~ /^(completed|failed|blocked)$/) completed_n++
    }

    # Output JSON
    printf "{"
    printf "\"spawns\":["
    for (i = 0; i < n; i++) {
      if (i > 0) printf ","
      # Escape name, parent, task
      nm = names[i]; gsub(/\\/, "\\\\", nm); gsub(/"/, "\\\"", nm); gsub(/\t/, "\\t", nm)
      pr = parents[i]; gsub(/\\/, "\\\\", pr); gsub(/"/, "\\\"", pr); gsub(/\t/, "\\t", pr)
      tk = tasks[i]; gsub(/\\/, "\\\\", tk); gsub(/"/, "\\\"", tk); gsub(/\t/, "\\t", tk)

      printf "{\"name\":\"%s\",\"parent\":\"%s\",\"caste\":\"%s\",", nm, pr, castes[i]
      printf "\"task\":\"%s\",\"status\":\"%s\",", tk, statuses[i]
      printf "\"spawned_at\":\"%s\",\"completed_at\":\"%s\",", timestamps[i], completed_at[i]
      printf "\"children\":["
      if (children_str[i] != "") {
        split(children_str[i], cidxs, " ")
        for (j = 1; j <= length(cidxs); j++) {
          if (j > 1) printf ","
          cn = names[cidxs[j]+0]
          gsub(/\\/, "\\\\", cn); gsub(/"/, "\\\"", cn); gsub(/\t/, "\\t", cn)
          printf "\"%s\"", cn
        }
      }
      printf "]}"
    }
    printf "],"
    printf "\"metadata\":{\"total_count\":%d,\"active_count\":%d,\"completed_count\":%d,\"file_exists\":true}", n, active, completed_n
    printf "}"
  }
  ' "$file_path"
}
```

### AWK get_active_spawns sketch

```bash
get_active_spawns() {
  local file_path="${1:-$SPAWN_TREE_FILE}"

  if [[ ! -f "$file_path" ]]; then
    echo "[]"
    return 0
  fi

  awk -F'|' '
  # Two-pass in single read: first collect completions, then output active
  $3 ~ /^(completed|failed|blocked)$/ && NF >= 4 {
    done_set[$2] = 1
  }
  NF == 7 && $7 == "spawned" {
    spawn_lines[spawn_n] = $0
    spawn_names[spawn_n] = $4
    spawn_parents[spawn_n] = $2
    spawn_castes[spawn_n] = $3
    spawn_tasks[spawn_n] = $5
    spawn_ts[spawn_n] = $1
    spawn_n++
  }
  END {
    printf "["
    first = 1
    for (i = 0; i < spawn_n; i++) {
      if (!(spawn_names[i] in done_set)) {
        if (!first) printf ","
        first = 0
        nm = spawn_names[i]; gsub(/\\/, "\\\\", nm); gsub(/"/, "\\\"", nm); gsub(/\t/, "\\t", nm)
        pr = spawn_parents[i]; gsub(/\\/, "\\\\", pr); gsub(/"/, "\\\"", pr); gsub(/\t/, "\\t", pr)
        tk = spawn_tasks[i]; gsub(/\\/, "\\\\", tk); gsub(/"/, "\\\"", tk); gsub(/\t/, "\\t", tk)
        printf "{\"name\":\"%s\",\"caste\":\"%s\",\"parent\":\"%s\",\"task\":\"%s\",\"spawned_at\":\"%s\"}", nm, spawn_castes[i], pr, tk, spawn_ts[i]
      }
    }
    printf "]"
  }
  ' "$file_path"
}
```

**Note:** This awk approach processes completions AND spawns in a single pass because awk processes all rules for each line. Both the completion rule and spawn rule fire on their respective lines, and the END block has access to all collected data.

---

## Sources

- `/Users/callumcowie/repos/Aether/.aether/utils/spawn-tree.sh` -- Parser implementation (434 lines)
- `/Users/callumcowie/repos/Aether/.aether/utils/spawn.sh` -- Wrapper functions `_spawn_tree_load`, `_spawn_tree_active`, `_spawn_tree_depth` (lines 194-226)
- `/Users/callumcowie/repos/Aether/tests/unit/spawn-tree.test.js` -- 9 AVA tests (220 lines)
- `/Users/callumcowie/repos/Aether/.aether/data/spawn-tree.txt` -- Live data (82 lines, 42 spawns, 40 completions)
- `/Users/callumcowie/repos/Aether/.aether/data/midden/midden.json` -- Midden entries documenting spawn-tree failures
- `/Users/callumcowie/repos/Aether/.aether/docs/known-issues.md` -- ISSUE-005: safety limit of 5 for depth
- `/Users/callumcowie/repos/Aether/.aether/oracle/research-phase-5-sibyl19.md` -- Previous research (Sibyl-19) on spawn-tree timeouts
- Git commit 1b82242 -- JSON escaping additions to spawn-tree.sh
- POSIX awk specification -- Associative arrays, gsub, NF
