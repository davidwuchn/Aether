#!/usr/bin/env bash
# Merge Driver Lockfile Tests
# Tests npm lockfile merge driver and setup-merge-driver subcommand

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"
MERGE_DRIVER_SOURCE="$PROJECT_ROOT/.aether/utils/merge-driver-lockfile.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with git repo
# ============================================================================
setup_merge_driver_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'CSEOF'
{
  "version": "3.0",
  "goal": "Test merge driver",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
CSEOF

    echo "$tmp_dir"
}

run_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

# ============================================================================
# Test 1: Merge driver script exists and is executable
# ============================================================================
test_merge_driver_script_exists() {
    test_start "merge driver script exists and is valid bash"
    assert_file_exists "$MERGE_DRIVER_SOURCE" || return 1
    bash -n "$MERGE_DRIVER_SOURCE" 2>/dev/null || return 1
}

# ============================================================================
# Test 2: Merge driver keeps "ours" on conflict
# ============================================================================
test_merge_driver_keeps_ours() {
    test_start "merge driver keeps ours and exits 0"
    local tmp_dir ancestor ours theirs result exit_code

    tmp_dir=$(mktemp -d)

    ancestor="$tmp_dir/ancestor.json"
    ours="$tmp_dir/ours.json"
    theirs="$tmp_dir/theirs.json"

    echo '{"version": "ancestor", "lockfileVersion": 3}' > "$ancestor"
    echo '{"version": "ours", "lockfileVersion": 3}' > "$ours"
    echo '{"version": "theirs", "lockfileVersion": 3}' > "$theirs"

    bash "$MERGE_DRIVER_SOURCE" "$ancestor" "$ours" "$theirs" 2>/dev/null
    exit_code=$?

    # Should exit 0 (conflict resolved)
    assert_exit_code "$exit_code" 0 || return 1

    # "ours" file should still have "ours" content (driver keeps ours)
    local ours_content
    ours_content=$(cat "$ours")
    assert_contains "$ours_content" '"version": "ours"' || return 1

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 3: Merge driver handles missing files gracefully
# ============================================================================
test_merge_driver_missing_files() {
    test_start "merge driver handles missing ancestor gracefully"
    local tmp_dir ours theirs exit_code

    tmp_dir=$(mktemp -d)

    ours="$tmp_dir/ours.json"
    theirs="$tmp_dir/theirs.json"

    echo '{"version": "ours"}' > "$ours"
    echo '{"version": "theirs"}' > "$theirs"

    # Pass a nonexistent ancestor - driver should still resolve
    bash "$MERGE_DRIVER_SOURCE" "/nonexistent/ancestor" "$ours" "$theirs" 2>/dev/null
    exit_code=$?

    assert_exit_code "$exit_code" 0 || return 1

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 4: setup-merge-driver subcommand returns valid JSON
# ============================================================================
test_setup_merge_driver_json() {
    test_start "setup-merge-driver returns valid JSON with ok:true"
    local tmp_dir output

    tmp_dir=$(setup_merge_driver_env)

    # Init a git repo in tmp_dir so git config works
    cd "$tmp_dir"
    git init -q 2>/dev/null

    output=$(run_cmd "$tmp_dir" setup-merge-driver)

    assert_json_valid "$output" || return 1
    assert_ok_true "$output" || return 1
    assert_json_field_equals "$output" ".result.driver" "lockfile" || return 1
    assert_json_field_equals "$output" ".result.configured" "true" || return 1

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 5: setup-merge-driver configures git merge driver
# ============================================================================
test_setup_merge_driver_configures_git() {
    test_start "setup-merge-driver configures git merge.lockfile.name and driver"
    local tmp_dir output driver_name driver_cmd

    tmp_dir=$(setup_merge_driver_env)

    cd "$tmp_dir"
    git init -q 2>/dev/null

    output=$(run_cmd "$tmp_dir" setup-merge-driver)

    driver_name=$(git config --get merge.lockfile.name)
    driver_cmd=$(git config --get merge.lockfile.driver)

    [[ "$driver_name" == "npm lockfile auto-merge" ]] || { test_fail "Expected driver name 'npm lockfile auto-merge', got '$driver_name'"; return 1; }
    [[ "$driver_cmd" == *"merge-driver-lockfile.sh"* ]] || { test_fail "Driver command should reference merge-driver-lockfile.sh, got '$driver_cmd'"; return 1; }
    [[ "$driver_cmd" == *"%O %A %B"* ]] || { test_fail "Driver command should use %O %A %B placeholders, got '$driver_cmd'"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 6: setup-merge-driver is idempotent
# ============================================================================
test_setup_merge_driver_idempotent() {
    test_start "setup-merge-driver is idempotent (safe to run multiple times)"
    local tmp_dir output1 output2

    tmp_dir=$(setup_merge_driver_env)

    cd "$tmp_dir"
    git init -q 2>/dev/null

    output1=$(run_cmd "$tmp_dir" setup-merge-driver)
    output2=$(run_cmd "$tmp_dir" setup-merge-driver)

    assert_ok_true "$output1" || return 1
    assert_ok_true "$output2" || return 1

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 7: .gitattributes gets merge=lockfile for package-lock.json
# ============================================================================
test_gitattributes_created() {
    test_start "setup-merge-driver creates .gitattributes with lockfile merge"
    local tmp_dir output

    tmp_dir=$(setup_merge_driver_env)

    cd "$tmp_dir"
    git init -q 2>/dev/null

    output=$(run_cmd "$tmp_dir" setup-merge-driver)

    [[ -f "$tmp_dir/.gitattributes" ]] || { test_fail ".gitattributes file not created"; return 1; }

    local content
    content=$(cat "$tmp_dir/.gitattributes")
    assert_contains "$content" "package-lock.json merge=lockfile" || return 1

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 8: .gitattributes is not overwritten if it already has the rule
# ============================================================================
test_gitattributes_idempotent() {
    test_start "setup-merge-driver does not duplicate .gitattributes entries"
    local tmp_dir output line_count_1 line_count_2

    tmp_dir=$(setup_merge_driver_env)

    cd "$tmp_dir"
    git init -q 2>/dev/null

    output=$(run_cmd "$tmp_dir" setup-merge-driver)
    line_count_1=$(grep -c "package-lock.json merge=lockfile" "$tmp_dir/.gitattributes")

    output=$(run_cmd "$tmp_dir" setup-merge-driver)
    line_count_2=$(grep -c "package-lock.json merge=lockfile" "$tmp_dir/.gitattributes")

    [[ "$line_count_1" -eq 1 ]] || { test_fail "Expected 1 entry after first run, got $line_count_1"; return 1; }
    [[ "$line_count_2" -eq 1 ]] || { test_fail "Expected 1 entry after second run, got $line_count_2"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 9: Merge driver preserves existing .gitattributes content
# ============================================================================
test_gitattributes_preserves_existing() {
    test_start "setup-merge-driver preserves existing .gitattributes content"
    local tmp_dir output

    tmp_dir=$(setup_merge_driver_env)

    cd "$tmp_dir"
    git init -q 2>/dev/null

    # Create .gitattributes with existing content
    echo "*.txt text" > "$tmp_dir/.gitattributes"

    output=$(run_cmd "$tmp_dir" setup-merge-driver)

    local content
    content=$(cat "$tmp_dir/.gitattributes")
    assert_contains "$content" "*.txt text" || return 1
    assert_contains "$content" "package-lock.json merge=lockfile" || return 1

    rm -rf "$tmp_dir"
}

# ============================================================================
# Test 10: Merge driver does not modify theirs file
# ============================================================================
test_merge_driver_does_not_modify_theirs() {
    test_start "merge driver does not modify theirs file"
    local tmp_dir ancestor ours theirs exit_code theirs_content

    tmp_dir=$(mktemp -d)

    ancestor="$tmp_dir/ancestor.json"
    ours="$tmp_dir/ours.json"
    theirs="$tmp_dir/theirs.json"

    echo '{"version": "ancestor"}' > "$ancestor"
    echo '{"version": "ours"}' > "$ours"
    echo '{"version": "theirs"}' > "$theirs"

    # Record theirs checksum before
    local theirs_before
    theirs_before=$(md5sum "$theirs" | cut -d' ' -f1)

    bash "$MERGE_DRIVER_SOURCE" "$ancestor" "$ours" "$theirs" 2>/dev/null
    exit_code=$?

    local theirs_after
    theirs_after=$(md5sum "$theirs" | cut -d' ' -f1)

    assert_exit_code "$exit_code" 0 || return 1
    [[ "$theirs_before" == "$theirs_after" ]] || { test_fail "theirs file was modified"; return 1; }

    rm -rf "$tmp_dir"
}

# ============================================================================
# Run all tests
# ============================================================================
test_merge_driver_script_exists
test_merge_driver_keeps_ours
test_merge_driver_missing_files
test_setup_merge_driver_json
test_setup_merge_driver_configures_git
test_setup_merge_driver_idempotent
test_gitattributes_created
test_gitattributes_idempotent
test_gitattributes_preserves_existing
test_merge_driver_does_not_modify_theirs

test_summary
