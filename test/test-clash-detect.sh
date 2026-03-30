#!/usr/bin/env bash
# Test: Clash detection utility
# Tests the clash-detect.sh module for multi-worktree file conflict detection.
#
# These tests create temporary git repos with worktrees to verify
# that clash-detect correctly identifies file conflicts across worktrees.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLASH_DETECT="$REPO_ROOT/.aether/utils/clash-detect.sh"
AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

PASS=0
FAIL=0
SKIP=0

pass() { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }

# ============================================================================
# Setup: Create isolated git repo with worktrees
# ============================================================================
TMP_DIR=""
REPO_DIR=""
WT1_DIR=""
WT2_DIR=""
WT1_BRANCH="test-clash-branch-1"
WT2_BRANCH="test-clash-branch-2"

setup_test_env() {
    TMP_DIR=$(mktemp -d)
    REPO_DIR="$TMP_DIR/test-repo"
    mkdir -p "$REPO_DIR"
    git -C "$REPO_DIR" init -b main >/dev/null 2>&1
    git -C "$REPO_DIR" config user.email "test@test.com"
    git -C "$REPO_DIR" config user.name "Test"

    # Create an initial commit with a shared file
    echo "original content" > "$REPO_DIR/shared.txt"
    git -C "$REPO_DIR" add shared.txt
    git -C "$REPO_DIR" commit -m "initial" >/dev/null 2>&1

    # Create worktree base dir
    mkdir -p "$REPO_DIR/.aether/worktrees"

    # Create two worktrees
    git -C "$REPO_DIR" worktree add "$REPO_DIR/.aether/worktrees/$WT1_BRANCH" -b "$WT1_BRANCH" main >/dev/null 2>&1
    git -C "$REPO_DIR" worktree add "$REPO_DIR/.aether/worktrees/$WT2_BRANCH" -b "$WT2_BRANCH" main >/dev/null 2>&1
    WT1_DIR="$REPO_DIR/.aether/worktrees/$WT1_BRANCH"
    WT2_DIR="$REPO_DIR/.aether/worktrees/$WT2_BRANCH"
}

cleanup_test_env() {
    if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
        # Remove worktrees first
        git -C "$REPO_DIR" worktree remove "$WT1_DIR" --force 2>/dev/null || true
        git -C "$REPO_DIR" worktree remove "$WT2_DIR" --force 2>/dev/null || true
        git -C "$REPO_DIR" worktree prune 2>/dev/null || true
        rm -rf "$TMP_DIR"
    fi
}

# Run clash-detect with AETHER_ROOT set to the test repo
run_clash_detect() {
    AETHER_ROOT="$REPO_DIR" bash "$CLASH_DETECT" "$@"
}

echo "=== Clash Detection Tests ==="
echo ""

# --- Test 1: No conflict when no other worktree has changes ---
echo "1. No conflict when no worktree has changes"
setup_test_env
result=$(run_clash_detect --file "shared.txt" 2>&1)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass "Returns no conflict when no worktree has changes"
else
    fail "Expected {ok:true, result:{conflict:false}}, got: $result"
fi
cleanup_test_env

# --- Test 2: Detect conflict when another worktree has changes ---
echo "2. Detect conflict when another worktree has uncommitted changes"
setup_test_env
# Modify shared.txt in worktree 1
echo "modified in wt1" > "$WT1_DIR/shared.txt"
# Check from worktree 2's perspective
result=$(AETHER_ROOT="$REPO_DIR" WORKTREE_DIR="$WT2_DIR" bash "$CLASH_DETECT" --file "shared.txt" 2>&1)
if echo "$result" | jq -e '.ok == true and .result.conflict == true' >/dev/null 2>&1; then
    pass "Detects conflict when another worktree has changes to the same file"
else
    fail "Expected conflict detected, got: $result"
fi
cleanup_test_env

# --- Test 3: No conflict for .aether/data/ files (branch-local, allowlisted) ---
echo "3. No conflict for .aether/data/ files (branch-local allowlist)"
setup_test_env
mkdir -p "$WT1_DIR/.aether/data"
echo "modified" > "$WT1_DIR/.aether/data/COLONY_STATE.json"
result=$(AETHER_ROOT="$REPO_DIR" WORKTREE_DIR="$WT2_DIR" bash "$CLASH_DETECT" --file ".aether/data/COLONY_STATE.json" 2>&1)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass ".aether/data/ files are allowlisted and bypass clash detection"
else
    fail "Expected .aether/data/ to be allowlisted, got: $result"
fi
cleanup_test_env

# --- Test 4: Returns list of conflicting worktrees ---
echo "4. Returns list of conflicting worktrees"
setup_test_env
echo "modified in wt1" > "$WT1_DIR/shared.txt"
echo "modified in wt2" > "$WT2_DIR/shared.txt"
result=$(run_clash_detect --file "shared.txt" 2>&1)
conflicting=$(echo "$result" | jq -r '.result.conflicting_worktrees // [] | length' 2>/dev/null)
if [[ "$conflicting" -ge 1 ]]; then
    pass "Returns non-empty conflicting_worktrees list (found $conflicting)"
else
    fail "Expected non-empty conflicting_worktrees, got: $result"
fi
cleanup_test_env

# --- Test 5: No conflict for file not in any worktree ---
echo "5. No conflict for file not modified in any worktree"
setup_test_env
echo "modified in wt1" > "$WT1_DIR/other-file.txt"
result=$(run_clash_detect --file "shared.txt" 2>&1)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass "No conflict for unrelated files"
else
    fail "Expected no conflict for unrelated file, got: $result"
fi
cleanup_test_env

# --- Test 6: Usage error on missing --file argument ---
echo "6. Usage error on missing --file argument"
setup_test_env
result=$(run_clash_detect 2>&1)
if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
    pass "Returns error when --file is missing"
else
    fail "Expected error for missing --file, got: $result"
fi
cleanup_test_env

# --- Test 7: No conflict when only the current worktree has changes ---
echo "7. No conflict when only the current worktree has changes"
setup_test_env
echo "modified in wt1" > "$WT1_DIR/shared.txt"
result=$(AETHER_ROOT="$REPO_DIR" WORKTREE_DIR="$WT1_DIR" bash "$CLASH_DETECT" --file "shared.txt" 2>&1)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass "No conflict when only the checking worktree has changes"
else
    fail "Expected no conflict for self-changes, got: $result"
fi
cleanup_test_env

echo ""
echo "=== RESULTS ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo "Skipped: $SKIP"
echo ""

if [[ "$FAIL" -gt 0 ]]; then
    echo "STATUS: SOME TESTS FAILED"
    exit 1
else
    echo "STATUS: ALL TESTS PASSED"
    exit 0
fi
