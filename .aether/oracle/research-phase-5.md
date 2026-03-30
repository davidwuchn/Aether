# Oracle Research: Phase 5 -- Full Regression Test Suite

**Phase ID:** 5
**Researcher:** Seer-17
**Date:** 2026-03-28
**Confidence:** High

---

## Context

Phase 5 requires running the full regression test suite to verify all session work is functioning correctly. Two specific tasks:
- Task 5.1: Run `npm test` (AVA unit tests) -- expect 616+ tests, 0 failures
- Task 5.2: Run `test-aether-utils.sh` (bash integration tests) -- expect test 42 passes, 6 pre-existing failures unchanged

---

## Key Findings

### 1. Test Infrastructure Overview

| Layer | Framework | Location | Test Count |
|-------|-----------|----------|------------|
| Unit (JS) | AVA v6 | `tests/unit/**/*.test.js` | 616 tests across 45 files |
| Integration (JS) | AVA v6 | `tests/integration/**/*.test.js` | 191 tests across 20 files |
| Bash Integration | Custom (test-helpers.sh) | `tests/bash/*.sh` | 611 tests across 69 files |
| E2E | Custom shell | `tests/e2e/*.sh` | 26 files |

**Source:** `package.json`, `tests/` directory, grep counts of `test(` and `run_test` calls.

### 2. npm test Scope

`npm test` runs `npm run test:unit` which invokes `ava`. AVA is configured to run only `tests/unit/**/*.test.js` (45 files, 616 tests). Integration tests are NOT included in `npm test` -- they require `npm run test:all`.

**Source:** `package.json` scripts section and AVA config.

### 3. Test 42 Identification

Test 42 is `test_pheromone_expire_promotes_eternal` ("pheromone-expire promotes high-strength signals to eternal memory"). It is the 42nd `run_test` call in `test-aether-utils.sh` (line 1772).

This test was fixed in commit `26e7345` which replaced hardcoded 2024-01-01 fixture dates with dynamic date computation. The old dates caused strength decay over 817 days, dropping effective_strength to 0. The fix uses cross-platform dynamic dates (macOS `-v`, Linux `-d`, static fallback) keeping strength at 0.87.

**Source:** `tests/bash/test-aether-utils.sh` lines 1605-1671, commit `26e7345`.

### 4. The 6 Pre-existing Failures in test-aether-utils.sh

Per COLONY_STATE.json hints (line 47), the 6 pre-existing failures are:

1. **test_version** (Test 2) -- Expects version "1.0.0" but actual is "2.5.0". Hardcoded expected value never updated.
2. **test_validate_state_colony** (Test 3) -- validate-state for COLONY_STATE.json -- likely schema evolution mismatch.
3. **test_fallback_json_err** (Test 16) -- Fallback json_err test -- related to error-handler changes.
4. **test_spawn_tree_rotation_exists** (Test 28, ARCH-03) -- Checks for `_rotate_spawn_tree` function existence.
5. **test_queen_read_validates_metadata** (Test 29, ARCH-06) -- queen-read JSON validation gates.
6. **test_validate_state_has_schema_migration** (Test 30, ARCH-02) -- Schema migration presence check.

**Source:** COLONY_STATE.json Phase 1 task 1.2 hints: "Pre-existing: version, validate-state, json_err, ARCH-03, ARCH-06, ARCH-02"

### 5. Recent Changes Affecting Tests

The most recent commit (`a72e46e`) fixed 3 HIGH quality issues in queen.sh:
- Trap composition (composing with `_aether_exit_cleanup` instead of overriding)
- JSON escaping (using jq for safe construction)
- Local variable declarations

This commit also added 5 new test files with 41 total tests:
- `test-queen-charter.test.sh` (13 tests)
- `test-colony-version-template.sh` (7 tests)
- `test-emoji-audit.sh` (8 tests)
- `test-midden-bridge.sh` (4 tests)
- `test-seal-version-increment.sh` (9 tests)

**Source:** Commit `a72e46e` stat output.

### 6. No Uncommitted Source Changes

Only `spawn-tree.txt` has unstaged changes (runtime artifact). All source code (aether-utils.sh, queen.sh, midden.sh) and test files are clean against HEAD.

**Source:** `git status --short` and `git diff HEAD` checks.

### 7. Dependencies

- AVA v6.0.0 (devDependency)
- proxyquire v2.1.3 (devDependency, for mocking require)
- sinon v19.0.5 (devDependency, for stubs/spies)
- jq (system dependency, required by bash tests)

**Source:** `package.json` devDependencies.

---

## Recommendations

### For Task 5.1: npm test

1. **Run `npm test` from project root.** This invokes AVA on all 45 unit test files (616 tests). Expect all 616 to pass with 0 failures.
2. **Timeout:** AVA is configured with 30s timeout per test. Total run should complete in under 2 minutes.
3. **Parallel execution:** AVA runs tests in parallel across separate processes. No special ordering needed.
4. **If failures occur:** Check for stale node_modules (`npm ci` to clean reinstall) or date-sensitive tests.

### For Task 5.2: test-aether-utils.sh

1. **Run `bash tests/bash/test-aether-utils.sh` from project root.** This file contains 43 tests.
2. **Test 42 should PASS.** It was fixed in commit `26e7345` with dynamic dates. Verify "PASS: pheromone-expire promotes high-strength signals to eternal memory".
3. **Expect exactly 6 failures** from pre-existing issues (version mismatch, schema changes, architecture checks). These are known and documented.
4. **If more than 6 failures appear**, the queen.sh changes from commit `a72e46e` may have introduced regressions. Cross-reference new failures against the 6 known ones.
5. **The 5 new bash test files are separate** -- they are NOT run by `test-aether-utils.sh`. They would need individual execution to verify.

### Risk Factors

1. **Date sensitivity:** Test 42 uses dynamic dates. If the system clock is significantly wrong, the fallback static dates (`2026-03-26`) could age and eventually cause the same issue. Low risk for now.
2. **jq dependency:** All bash tests require jq. Verify it is installed (`which jq`).
3. **Temp directory cleanup:** Bash tests create temp directories via `mktemp`. If previous runs left stale temp dirs with locks, tests could behave unexpectedly (unlikely with clean shell state).
4. **The QUEEN.md.tmp files** in the repo (untracked) are harmless artifacts from atomic-write operations. They do not affect tests.

---

## Sources

- `/Users/callumcowie/repos/Aether/package.json` -- Test scripts, AVA config, dependencies
- `/Users/callumcowie/repos/Aether/tests/bash/test-aether-utils.sh` -- 43 bash tests, test 42 at line 1605
- `/Users/callumcowie/repos/Aether/tests/bash/test-helpers.sh` -- Bash test framework
- `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json` -- Phase plan, pre-existing failure list
- `/Users/callumcowie/repos/Aether/.aether/docs/known-issues.md` -- Known bugs and architecture issues
- `/Users/callumcowie/repos/Aether/.aether/oracle/analysis-TESTS.md` -- Prior Oracle test analysis
- Git commit `26e7345` -- Test 42 fix (dynamic dates)
- Git commit `a72e46e` -- Queen.sh quality fixes + 5 new test files

---

## Open Questions

1. The "6 pre-existing failures" count comes from the colony plan, but the exact list (version, validate-state, json_err, ARCH-03, ARCH-06, ARCH-02) should be verified during the actual test run. Some may have been inadvertently fixed by recent commits.
2. Integration tests (191 tests) and bash tests beyond test-aether-utils.sh (611 total across all bash files) are not covered by the Phase 5 plan. Consider whether a broader regression sweep is warranted.
