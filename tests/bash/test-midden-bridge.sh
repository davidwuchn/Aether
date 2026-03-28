#!/usr/bin/env bash
# Tests for midden-ingest-errors (midden-ERROR_LOG bridge)
# Tasks 3.1 + 3.2 + 3.3

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# ============================================================================
# Helper: Create isolated test environment with midden support
# ============================================================================
setup_bridge_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data/midden"

    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    chmod +x "$tmpdir/.aether/aether-utils.sh"

    local utils_source
    utils_source="$(dirname "$AETHER_UTILS")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmpdir/.aether/"
    fi

    local exchange_source
    exchange_source="$(dirname "$AETHER_UTILS")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmpdir/.aether/"
    fi

    local schemas_source
    schemas_source="$(dirname "$AETHER_UTILS")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmpdir/.aether/"
    fi

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test midden-bridge",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {"instincts": []},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

    cat > "$tmpdir/.aether/data/midden/midden.json" << 'EOF'
{"version":"1.0.0","entries":[],"entry_count":0}
EOF

    echo "$tmpdir"
}

# Helper: run aether-utils against a test env
run_cmd() {
    local tmpdir="$1"
    shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>&1
}

# ============================================================================
# Test 1: No errors.log → json_ok with ingested:0
# ============================================================================
test_no_errors_log() {
    local tmpdir
    tmpdir=$(setup_bridge_env)

    # Ensure no errors.log exists
    rm -f "$tmpdir/.aether/data/errors.log"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-ingest-errors) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_json_valid "$result"; then
        test_fail "valid JSON" "invalid JSON: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local ingested
    ingested=$(echo "$result" | jq -r '.result.ingested')
    if [[ "$ingested" != "0" ]]; then
        test_fail "ingested=0" "ingested=$ingested"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 2: Empty errors.log → json_ok with ingested:0
# ============================================================================
test_empty_errors_log() {
    local tmpdir
    tmpdir=$(setup_bridge_env)

    # Create empty errors.log
    touch "$tmpdir/.aether/data/errors.log"

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-ingest-errors) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local ingested
    ingested=$(echo "$result" | jq -r '.result.ingested')
    if [[ "$ingested" != "0" ]]; then
        test_fail "ingested=0 for empty file" "ingested=$ingested"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 3: errors.log with 3 lines → json_ok with ingested:3
# ============================================================================
test_three_error_lines() {
    local tmpdir
    tmpdir=$(setup_bridge_env)

    # Write 3 log lines in the format written by _aether_log_error
    cat > "$tmpdir/.aether/data/errors.log" << 'EOF'
[2026-03-27T10:00:00Z] First error message
[2026-03-27T10:01:00Z] Second error message
[2026-03-27T10:02:00Z] Third error message
EOF

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-ingest-errors) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local ingested
    ingested=$(echo "$result" | jq -r '.result.ingested')
    if [[ "$ingested" != "3" ]]; then
        test_fail "ingested=3" "ingested=$ingested"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify 3 entries exist in midden.json with correct category and source
    local midden_count
    midden_count=$(jq '.entries | length' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$midden_count" != "3" ]]; then
        test_fail "3 entries in midden.json" "found $midden_count"
        rm -rf "$tmpdir"
        return 1
    fi

    local category
    category=$(jq -r '.entries[0].category' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$category" != "error_log" ]]; then
        test_fail "category=error_log" "category=$category"
        rm -rf "$tmpdir"
        return 1
    fi

    local source
    source=$(jq -r '.entries[0].source' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$source" != "error-handler" ]]; then
        test_fail "source=error-handler" "source=$source"
        rm -rf "$tmpdir"
        return 1
    fi

    # Verify errors.log was moved to errors.log.ingested
    if [[ -f "$tmpdir/.aether/data/errors.log" ]]; then
        test_fail "errors.log moved away" "errors.log still exists"
        rm -rf "$tmpdir"
        return 1
    fi

    if [[ ! -f "$tmpdir/.aether/data/errors.log.ingested" ]]; then
        test_fail "errors.log.ingested exists" "errors.log.ingested not found"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Test 4: --dry-run doesn't modify midden or move file
# ============================================================================
test_dry_run_no_modify() {
    local tmpdir
    tmpdir=$(setup_bridge_env)

    # Write 2 log lines
    cat > "$tmpdir/.aether/data/errors.log" << 'EOF'
[2026-03-27T10:00:00Z] Error one
[2026-03-27T10:01:00Z] Error two
EOF

    local result exit_code=0
    result=$(run_cmd "$tmpdir" midden-ingest-errors --dry-run) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "exit code 0 for dry-run" "exit code $exit_code, output: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    if ! assert_ok_true "$result"; then
        test_fail "ok=true" "ok was not true: $result"
        rm -rf "$tmpdir"
        return 1
    fi

    local ingested
    ingested=$(echo "$result" | jq -r '.result.ingested')
    if [[ "$ingested" != "2" ]]; then
        test_fail "ingested=2 (dry-run count)" "ingested=$ingested"
        rm -rf "$tmpdir"
        return 1
    fi

    # Midden should still have 0 entries (dry-run doesn't write)
    local midden_count
    midden_count=$(jq '.entries | length' "$tmpdir/.aether/data/midden/midden.json")
    if [[ "$midden_count" != "0" ]]; then
        test_fail "midden unchanged (0 entries)" "found $midden_count entries"
        rm -rf "$tmpdir"
        return 1
    fi

    # errors.log should still exist (dry-run doesn't move it)
    if [[ ! -f "$tmpdir/.aether/data/errors.log" ]]; then
        test_fail "errors.log still exists after dry-run" "errors.log was moved"
        rm -rf "$tmpdir"
        return 1
    fi

    rm -rf "$tmpdir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================

run_test test_no_errors_log "midden-ingest-errors: no errors.log returns ingested:0"
run_test test_empty_errors_log "midden-ingest-errors: empty errors.log returns ingested:0"
run_test test_three_error_lines "midden-ingest-errors: 3 log lines ingests 3 entries"
run_test test_dry_run_no_modify "midden-ingest-errors: --dry-run counts without modifying"

test_summary
