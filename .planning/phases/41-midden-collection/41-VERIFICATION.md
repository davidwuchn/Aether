---
phase: 41-midden-collection
verified: 2026-03-31T01:46:00Z
status: passed
score: 4/4 must-haves verified
gaps: []
---

# Phase 41: Midden Collection Verification Report

**Phase Goal:** Failure records from merged branches are collected into main's midden with idempotency and cross-PR pattern detection
**Verified:** 2026-03-31T01:46:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `midden-collect --branch <branch> --merge-sha <sha>` ingests failure records from the branch into main's midden | VERIFIED | `_midden_collect()` function at midden.sh:526-708. Reads branch worktree midden.json, enriches entries with provenance fields (collected_from, collected_at, merge_commit, original_entry_id), appends to main midden. Test `test_collect_basic` passes. |
| 2 | Running midden-collect twice with the same merge SHA produces no duplicates (idempotent) | VERIFIED | Dual-layer idempotency: Layer 1 (merge fingerprint in collected-merges.json, lines 613-622) returns `already_collected` on second call. Layer 2 (per-entry ID dedup, lines 631-647) skips entries with existing IDs. Test `test_collect_idempotent` and `test_collect_layer2_dedup` both pass. |
| 3 | `midden-handle-revert --sha <sha>` tags affected entries rather than deleting them | VERIFIED | `_midden_handle_revert()` function at midden.sh:711-828. Adds `reverted:<sha>` tag to entries (line 783) and sets `reviewed: false`. Returns `entries_deleted: 0` (line 826). Test `test_revert_basic` verifies tagging; `test_revert_preserves_entries` verifies entries still exist after revert. |
| 4 | `midden-cross-pr-analysis` returns failure patterns detected across 2+ PRs | VERIFIED | `_midden_cross_pr_analysis()` function at midden.sh:831-943. Filters cross-branch entries, groups by category, computes scoring formula `(unique_prs/5)*0.6 + (total_entries/10)*0.4`, classifies as `cross-pr-systemic` (2+ PRs, 3+ entries) or `cross-pr-critical` (3+ PRs, 5+ entries). Returns `systemic_categories` array. Test `test_cross_pr_detect_systemic` passes. |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.aether/utils/midden.sh` | Four new functions: _midden_collect, _midden_handle_revert, _midden_cross_pr_analysis, _midden_prune | VERIFIED | All four functions present at lines 526-1054 (528 lines). Uses acquire_lock/atomic_write patterns. No TODOs, no placeholders. |
| `tests/bash/test-midden-collection.sh` | Comprehensive tests for all four subcommands | VERIFIED | 682 lines, 13 test cases covering collect (5 tests), revert (3 tests), cross-pr-analysis (3 tests), prune (2 tests). All 13 pass. |
| `.aether/aether-utils.sh` | Dispatch entries and help JSON for 4 new subcommands | VERIFIED | Dispatch at lines 4769-4772. Help JSON at lines 1278-1281. All four subcommands reachable via `aether midden-collect`, etc. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| aether-utils.sh case statement | _midden_collect | `midden-collect) _midden_collect "$@"` | WIRED | Line 4769 |
| aether-utils.sh case statement | _midden_handle_revert | `midden-handle-revert) _midden_handle_revert "$@"` | WIRED | Line 4770 |
| aether-utils.sh case statement | _midden_cross_pr_analysis | `midden-cross-pr-analysis) _midden_cross_pr_analysis "$@"` | WIRED | Line 4771 |
| aether-utils.sh case statement | _midden_prune | `midden-prune) _midden_prune "$@"` | WIRED | Line 4772 |
| midden.sh _midden_collect | midden.sh _midden_write patterns | Reuses acquire_lock, atomic_write, json_ok/json_err | WIRED | Locking at lines 654, 677; atomic_write at lines 668, 699; json helpers throughout |
| midden.sh _midden_collect | worktree path resolution | `$AETHER_ROOT/.aether/worktrees/$mc_branch/.aether/data/midden/midden.json` + git worktree list fallback | WIRED | Lines 558-568 |
| continue-verify.md | _midden_collect | `bash .aether/aether-utils.sh midden-collect` | WIRED | Step 2.0.6, line 425 |
| continue-advance.md | _midden_collect | `bash .aether/aether-utils.sh midden-collect` | WIRED | Step 2.0.6, line 360 |
| continue-advance.md | _midden_cross_pr_analysis | `bash .aether/aether-utils.sh midden-cross-pr-analysis --window 14` | WIRED | Step 2.0.7, line 384 |
| build-verify.md | _midden_collect | `bash .aether/aether-utils.sh midden-collect` | WIRED | Step 5.9, line 417 |
| build-verify.md | _midden_cross_pr_analysis | `bash .aether/aether-utils.sh midden-cross-pr-analysis --window 14` | WIRED | Step 5.9, line 430 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| _midden_collect | branch midden entries | `$AETHER_ROOT/.aether/worktrees/$branch/.aether/data/midden/midden.json` | FLOWING | Reads real worktree midden, filters with jq, enriches, appends to main midden |
| _midden_cross_pr_analysis | main midden entries with collected_from | `$COLONY_DATA_DIR/midden/midden.json` | FLOWING | Filters cross-branch entries, groups by category, computes scores |
| continue-verify Step 2.0.6 | $last_merged_branch, $last_merge_sha | COLONY_STATE context / git log | CONDITIONAL | Correctly guards with `if [[ -n "$last_merge_branch" && -n "$last_merge_sha" ]]` -- runs only when merge context available |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 13 midden-collection tests pass | `bash tests/bash/test-midden-collection.sh` | 13 passed, 0 failed | PASS |
| Existing midden-library tests pass (no regression) | `bash tests/bash/test-midden-library.sh` | 14 passed, 0 failed | PASS |
| Full npm test suite passes (no regression) | `npm test` | 509 passed | PASS |
| Dispatch entries exist in aether-utils.sh | `grep -c 'midden-collect)' .aether/aether-utils.sh` | Returns 1 | PASS |
| All four dispatch entries present | `grep -cE 'midden-(collect|handle-revert|cross-pr-analysis|prune)\)' .aether/aether-utils.sh` | Returns 4 | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| MIDD-01 | 41-01, 41-02 | midden-collect ingests failure records from merged branch into main's midden with idempotency via merge fingerprints | SATISFIED | _midden_collect function implements collection with dual-layer idempotency (merge fingerprint + entry ID dedup). Tests pass. |
| MIDD-02 | 41-01, 41-02 | midden-handle-revert tags reverted entries rather than deleting them, preserving failure history | SATISFIED | _midden_handle_revert adds `reverted:<sha>` tag, returns `entries_deleted: 0`. Tests verify entries preserved. |
| MIDD-03 | 41-01, 41-03 | midden-cross-pr-analysis detects systemic failure patterns across multiple PRs | SATISFIED | _midden_cross_pr_analysis classifies cross-pr-systemic/cross-pr-critical, auto-emits REDIRECT. Tests pass. |

**Orphaned requirements:** None. REQUIREMENTS.md (v2.6) does not define MIDD-01/02/03 (those are in v2.7-REQUIREMENTS.md). All MIDD IDs from v2.7-REQUIREMENTS.md are covered by plans 41-01, 41-02, and 41-03.

### Anti-Patterns Found

No anti-patterns detected. Zero TODO/FIXME/PLACEHOLDER comments in midden.sh. No empty return statements. No hardcoded empty data flowing to output.

### Human Verification Required

### 1. Worktree Path Resolution End-to-End

**Test:** Create a real git worktree branch, add midden entries to the branch, merge the branch, then run `midden-collect --branch <branch> --merge-sha <sha>`
**Expected:** Failure records from the branch appear in main's midden with enrichment fields
**Why human:** Requires real git worktree operations that cannot be simulated in a programmatic spot-check

### 2. Cross-PR REDIRECT Emission Verification

**Test:** After collecting failures from 2+ branches in the same category, run `midden-cross-pr-analysis` and check that a REDIRECT pheromone was actually written to `~/.aether/pheromones.json`
**Expected:** A REDIRECT signal with source `auto:cross-pr` appears in the pheromone store
**Why human:** The test mocks the pheromone-write call (redirects to /dev/null), so real emission needs manual verification

### Gaps Summary

No gaps found. All four ROADMAP success criteria are satisfied:

1. `midden-collect` ingests failure records with provenance enrichment -- implemented and tested
2. Dual-layer idempotency prevents duplicates -- implemented and tested
3. `midden-handle-revert` tags entries (no deletion) -- implemented and tested
4. `midden-cross-pr-analysis` detects cross-PR patterns -- implemented and tested

Additionally, all workflow wiring is complete (continue-verify, continue-advance, build-verify), dispatch entries exist in aether-utils.sh, help JSON is present, and no regressions were introduced (509 npm tests + 14 existing midden tests all pass).

---

_Verified: 2026-03-31T01:46:00Z_
_Verifier: Claude (gsd-verifier)_
