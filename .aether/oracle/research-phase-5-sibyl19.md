# Oracle Research: Phase 5 -- Full Regression Test Suite (Updated)

**Phase ID:** 5
**Researcher:** Sibyl-19
**Date:** 2026-03-29
**Confidence:** High
**Supersedes:** research-phase-5.md (Seer-17, 2026-03-28)

---

## Context

Phase 5 requires running the full regression test suite to verify all session work is functioning correctly. Two specific tasks:
- Task 5.1: Run `npm test` (AVA unit tests) -- expect 616+ tests, 0 failures
- Task 5.2: Run `test-aether-utils.sh` (bash integration tests) -- expect test 42 passes, 6 pre-existing failures unchanged

Known blockers mention "JSON objects stored as ant names break spawn-tree JSON output (4 AVA test failures)."

---

## Key Findings

### 1. AVA Test Count: 616 Total, Spawn-Tree Tests Flaky Due to Timeout

The AVA test suite contains 616 tests across the `tests/unit/` directory (46 files). In the first run during this research session, all 616 passed. In subsequent runs, 1-4 spawn-tree tests fail intermittently due to timeout issues (see Finding 3).

The `npm test` command runs only unit tests (`ava` on `tests/unit/**/*.test.js`). Integration tests require `npm run test:all`.

**Source:** `package.json` lines 26-28, AVA config lines 42-47, multiple test runs during this research.

### 2. The JSON-Objects-As-Names Blocker Is RESOLVED

The original blocker stated "JSON objects stored as ant names break spawn-tree JSON output." The current spawn-tree.txt (68 lines, 36 spawns) produces valid JSON when parsed by `spawn-tree-load`. All completion events with JSON summaries (e.g., `Verify-73|completed|{"findings":...}`) parse correctly because the pipe-count detection differentiates spawn events (6 pipes) from completion events (3 pipes), and JSON payloads do not contain pipe characters.

**Source:** `/Users/callumcowie/repos/Aether/.aether/data/spawn-tree.txt`, manual `jq` validation of full `spawn-tree-load` output.

### 3. CRITICAL: Spawn-Tree Tests Fail Due to 10-Second execSync Timeout

The spawn-tree parser (`spawn-tree.sh`) uses pure-bash with temporary files, `sed`, and `wc`. With 68 lines, `spawn-tree-load` takes **11-29 seconds** depending on system load.

The test file (`tests/unit/spawn-tree.test.js` line 18) uses `execSync` with `timeout: 10000` (10 seconds). Any test calling `spawn-tree-load` against the real spawn-tree.txt will intermittently fail when parsing exceeds 10 seconds.

**Affected tests (use real spawn-tree.txt):**
- `spawn-tree-load returns valid tree JSON`
- `spawn-tree-depth returns correct depth for known spawn`
- `spawn-tree-load includes parent-child relationships`
- `spawn-tree-active returns only active spawns`

**Unaffected tests (use temp fixture files):**
- `spawn-tree-load handles missing file gracefully`
- `spawn-tree-depth handles deep chains correctly`
- `spawn-tree-active returns empty array when no active spawns`
- `spawn-tree-depth returns 0 for Queen`
- `spawn-tree-depth returns depth 1 for unknown ant`

**Fix:** Increase `timeout: 10000` to `timeout: 30000` on line 18 of `tests/unit/spawn-tree.test.js`. This is a one-line change.

**Source:** `tests/unit/spawn-tree.test.js` line 18, `time` measurement of `spawn-tree-load` (11.28s), AVA runs with `--timeout=60s` showing individual test times of 11-29s.

### 4. Test 42 Passes Consistently

Test 42 (`test_pheromone_expire_promotes_eternal`) passes on every run. The dynamic date fix from commit `26e7345` is working correctly.

**Source:** Multiple `bash tests/bash/test-aether-utils.sh` runs.

### 5. Exactly 6 Pre-existing Bash Failures (Unchanged)

| # | Test Name | Failure Reason |
|---|-----------|---------------|
| 2 | `test_version` | Expects "1.0.0" but version is "2.5.0" |
| 5 | `test_validate_state_missing` | validate-state missing file handling changed |
| 16 | `test_fallback_json_err` | Fallback json_err behavior changed |
| 28 | `test_spawn_tree_rotation_exists` (ARCH-03) | `_rotate_spawn_tree` function not implemented |
| 29 | `test_queen_read_validates_metadata` (ARCH-06) | queen-read metadata validation gates missing |
| 30 | `test_validate_state_has_schema_migration` (ARCH-02) | `_migrate_colony_state` function not implemented |

**Correction from Seer-17:** Previous research listed test 3 (`test_validate_state_colony`) as a failure. It actually passes. Test 5 (`test_validate_state_missing`) is the failing one.

**Source:** Multiple bash test runs, `grep '✗ FAIL' | sort -u`.

---

## Recommendations

### For Task 5.1: npm test

1. **The builder MUST increase the execSync timeout** from 10000 to 30000 in `tests/unit/spawn-tree.test.js` line 18 before running `npm test`. Without this fix, spawn-tree tests will fail intermittently.
2. After the timeout fix, run `npm test` and expect 616 tests, 0 failures.
3. If any non-spawn-tree test fails, it is a genuine regression and should be investigated.

### For Task 5.2: test-aether-utils.sh

1. Run `bash tests/bash/test-aether-utils.sh`. Expect 37 PASS, 6 FAIL, test 42 PASS.
2. Verify the 6 failures match tests 2, 5, 16, 28, 29, 30. Any other failure is a regression.

### Risk Factors

1. **Spawn-tree performance degradation:** As spawn-tree.txt grows, parse time increases linearly. The `_rotate_spawn_tree` function (ARCH-03) was designed but never implemented. Each new colony session adds 20-60 lines.
2. **Date sensitivity:** Test 42 uses dynamic dates. Static fallback dates will age but only affect systems without a working `date -v` or `date -d` command.

---

## Sources

- `/Users/callumcowie/repos/Aether/package.json` -- Test scripts, AVA config
- `/Users/callumcowie/repos/Aether/tests/unit/spawn-tree.test.js` -- 9 tests, 10s timeout at line 18
- `/Users/callumcowie/repos/Aether/tests/bash/test-aether-utils.sh` -- 43 tests
- `/Users/callumcowie/repos/Aether/.aether/utils/spawn-tree.sh` -- Parser (434 lines)
- `/Users/callumcowie/repos/Aether/.aether/utils/spawn.sh` -- `_spawn_tree_load` wrapper
- `/Users/callumcowie/repos/Aether/.aether/data/spawn-tree.txt` -- 68 lines, 36 spawns
- `/Users/callumcowie/repos/Aether/.aether/docs/known-issues.md` -- Known issues
- Multiple `npm test` runs (616 total, 0-4 flaky spawn-tree failures)
- Multiple `bash test-aether-utils.sh` runs (37 pass / 6 stable failures)
