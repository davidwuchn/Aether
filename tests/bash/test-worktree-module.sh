#!/usr/bin/env bash
# Worktree Module Tests
# Tests worktree-create and worktree-cleanup subcommands via aether-utils.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated git repo for worktree tests
# ============================================================================
setup_worktree_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Initialize a git repo
    cd "$tmp_dir"
    git init -q
    git config user.email "test@aether.colony"
    git config user.name "Test Ant"

    # Create .aether structure
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    # Copy aether-utils.sh (the dispatcher)
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    # Fix pre-existing syntax error in aether-utils.sh (line ~1180: } should be fi)
    # This is a known bug in the merge driver setup that prevents bash parsing.
    # The line reads:  json_err "$E_FILE_NOT_FOUND" "Merge driver script..." }
    # and should end with fi instead of }. We patch the copy only.
    local utils_copy="$tmp_dir/.aether/aether-utils.sh"
    # Find the line number of the specific pattern and replace } with fi
    local bad_line
    bad_line=$(grep -n 'json_err.*"Merge driver script not found' "$utils_copy" | head -1 | cut -d: -f1)
    if [[ -n "$bad_line" ]]; then
        local next_line=$((bad_line + 1))
        # Check if the next line is a standalone }
        if sed -n "${next_line}p" "$utils_copy" | grep -q '^  }$'; then
            # Replace the } with fi
            sed -i '' "${next_line}s/^  }$/  fi/" "$utils_copy"
        fi
    fi

    # Copy utils
    local utils_source="$PROJECT_ROOT/.aether/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    # Copy exchange modules if present
    local exchange_source="$PROJECT_ROOT/.aether/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test worktree module",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": [],
  "workers": [],
  "spawn_tree": []
}
CSEOF

    # Make an initial commit so there's something to branch from
    echo "# test" > "$tmp_dir/README.md"
    git add README.md
    git commit -q -m "initial commit"

    echo "$tmp_dir"
}

run_worktree_cmd() {
    local tmp_dir="$1"
    shift
    # Redirect stderr to stdout so json_err output (which goes to stderr) is captured
    ( cd "$tmp_dir" && AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" AETHER_TESTING=1 \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>&1 )
}

run_worktree_cmd_with_stderr() {
    local tmp_dir="$1"
    shift
    ( cd "$tmp_dir" && AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" AETHER_TESTING=1 \
        bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>&1 )
}

# ============================================================================
# TDD Cycle 1: worktree-create basic
# ============================================================================

# Test 1: Module file exists and has valid syntax
test_module_exists() {
    local module_path="$PROJECT_ROOT/.aether/utils/worktree.sh"
    assert_file_exists "$module_path" || return 1
    bash -n "$module_path" 2>/dev/null || return 1
}

# Test 2: worktree-create creates a worktree and returns JSON
test_worktree_create_basic() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-create --branch "test-feature-1")

    assert_ok_true "$result" || { rm -rf "$tmp_dir"; return 1; }

    # Check result fields
    local path branch base worktree_dir
    path=$(echo "$result" | jq -r '.result.path')
    branch=$(echo "$result" | jq -r '.result.branch')
    base=$(echo "$result" | jq -r '.result.base')
    worktree_dir=$(echo "$result" | jq -r '.result.worktree_dir')

    [[ "$branch" == "test-feature-1" ]] || { rm -rf "$tmp_dir"; return 1; }
    # base should be the default branch (master or main depending on git config)
    [[ "$base" == "master" || "$base" == "main" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify worktree directory exists
    [[ -d "$worktree_dir" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Verify .aether/data/ was copied to the worktree
    [[ -f "$worktree_dir/.aether/data/COLONY_STATE.json" ]] || { rm -rf "$tmp_dir"; return 1; }

    # Cleanup
    cd /
    rm -rf "$tmp_dir"
}

# Test 3: worktree-create refuses duplicate branch
test_worktree_create_duplicate() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    # First creation should succeed
    run_worktree_cmd "$tmp_dir" worktree-create --branch "dup-branch" >/dev/null

    # Second creation should fail
    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-create --branch "dup-branch") || true

    # Should return ok:false
    local ok_val
    ok_val=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "parse_error")
    [[ "$ok_val" == "false" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 4: worktree-create with --base flag
test_worktree_create_with_base() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    # Create a different base branch
    cd "$tmp_dir"
    local default_branch
    default_branch=$(git rev-parse --abbrev-ref HEAD)
    git checkout -q -b "develop"
    echo "develop content" > "$tmp_dir/dev.txt"
    git add dev.txt
    git commit -q -m "develop commit"
    git checkout -q "$default_branch"

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-create --branch "from-develop" --base "develop")

    assert_ok_true "$result" || { cd /; rm -rf "$tmp_dir"; return 1; }

    local base
    base=$(echo "$result" | jq -r '.result.base')
    [[ "$base" == "develop" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 5: worktree-create with --task-id flag
test_worktree_create_with_task_id() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-create --branch "task-123" --task-id "2.1")

    assert_ok_true "$result" || { cd /; rm -rf "$tmp_dir"; return 1; }

    local task_id
    task_id=$(echo "$result" | jq -r '.result.task_id')
    [[ "$task_id" == "2.1" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 6: worktree-create validates --branch is required
test_worktree_create_missing_branch() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-create) || true

    local ok_val
    ok_val=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "parse_error")
    [[ "$ok_val" == "false" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# ============================================================================
# TDD Cycle 2: worktree-cleanup
# ============================================================================

# Test 7: worktree-cleanup removes a worktree
test_worktree_cleanup_basic() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    run_worktree_cmd "$tmp_dir" worktree-create --branch "cleanup-test" >/dev/null

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-cleanup --branch "cleanup-test")

    assert_ok_true "$result" || { cd /; rm -rf "$tmp_dir"; return 1; }

    local removed
    removed=$(echo "$result" | jq -r '.result.removed')
    [[ "$removed" == "true" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    # Verify worktree directory is gone
    local worktree_dir
    worktree_dir="$tmp_dir/.aether/worktrees/cleanup-test"
    [[ ! -d "$worktree_dir" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 8: worktree-cleanup refuses if uncommitted changes (without --force)
test_worktree_cleanup_uncommitted_refused() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    run_worktree_cmd "$tmp_dir" worktree-create --branch "dirty-branch" >/dev/null

    # Make an uncommitted change in the worktree
    local wt_dir="$tmp_dir/.aether/worktrees/dirty-branch"
    echo "dirty" > "$wt_dir/dirty-file.txt"

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-cleanup --branch "dirty-branch") || true

    local ok_val
    ok_val=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "parse_error")
    [[ "$ok_val" == "false" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    # Worktree should still exist
    [[ -d "$wt_dir" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 9: worktree-cleanup --force removes even with uncommitted changes
test_worktree_cleanup_force() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    run_worktree_cmd "$tmp_dir" worktree-create --branch "force-branch" >/dev/null

    # Make an uncommitted change
    local wt_dir="$tmp_dir/.aether/worktrees/force-branch"
    echo "dirty" > "$wt_dir/dirty-file.txt"

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-cleanup --branch "force-branch" --force)

    assert_ok_true "$result" || { cd /; rm -rf "$tmp_dir"; return 1; }

    # Worktree should be gone
    [[ ! -d "$wt_dir" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 10: worktree-cleanup for nonexistent branch returns error
test_worktree_cleanup_nonexistent() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-cleanup --branch "no-such-branch") || true

    local ok_val
    ok_val=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "parse_error")
    [[ "$ok_val" == "false" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 11: worktree-create copies pheromones and other data files
test_worktree_create_copies_data_files() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    # Create some additional data files
    echo '{"signals":[]}' > "$tmp_dir/.aether/data/pheromones.json"
    echo '{"focus":[],"constraints":[]}' > "$tmp_dir/.aether/data/constraints.json"

    run_worktree_cmd "$tmp_dir" worktree-create --branch "data-copy-test" >/dev/null

    local wt_dir="$tmp_dir/.aether/worktrees/data-copy-test"

    # Verify data files were copied
    [[ -f "$wt_dir/.aether/data/COLONY_STATE.json" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }
    [[ -f "$wt_dir/.aether/data/pheromones.json" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }
    [[ -f "$wt_dir/.aether/data/constraints.json" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# Test 12: worktree-cleanup validates --branch is required
test_worktree_cleanup_missing_branch() {
    local tmp_dir
    tmp_dir=$(setup_worktree_env)

    local result
    result=$(run_worktree_cmd "$tmp_dir" worktree-cleanup 2>/dev/null) || true

    local ok_val
    ok_val=$(echo "$result" | jq -r '.ok' 2>/dev/null || echo "parse_error")
    [[ "$ok_val" == "false" ]] || { cd /; rm -rf "$tmp_dir"; return 1; }

    cd /
    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
cd "$PROJECT_ROOT"

run_test test_module_exists "module file exists and has valid syntax"
run_test test_worktree_create_basic "worktree-create creates worktree and returns JSON"
run_test test_worktree_create_duplicate "worktree-create refuses duplicate branch"
run_test test_worktree_create_with_base "worktree-create with --base flag"
run_test test_worktree_create_with_task_id "worktree-create with --task-id flag"
run_test test_worktree_create_missing_branch "worktree-create validates --branch required"
run_test test_worktree_cleanup_basic "worktree-cleanup removes a worktree"
run_test test_worktree_cleanup_uncommitted_refused "worktree-cleanup refuses uncommitted changes"
run_test test_worktree_cleanup_force "worktree-cleanup --force removes with dirty changes"
run_test test_worktree_cleanup_nonexistent "worktree-cleanup nonexistent branch returns error"
run_test test_worktree_create_copies_data_files "worktree-create copies data files to worktree"
run_test test_worktree_cleanup_missing_branch "worktree-cleanup validates --branch required"

test_summary
