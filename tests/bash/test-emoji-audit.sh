#!/usr/bin/env bash
# Tests for emoji-audit.sh utility
# Verifies:
# 1. The script runs without error
# 2. It finds emojis in command files
# 3. It produces valid JSON output
# 4. The reference map has at least 20 entries

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

EMOJI_AUDIT_SCRIPT="$REPO_ROOT/.aether/utils/emoji-audit.sh"
COMMANDS_DIR="$REPO_ROOT/.claude/commands/ant"

# ============================================================================
# Test 1: emoji-audit.sh script exists
# ============================================================================
test_emoji_audit_script_exists() {
    if [[ ! -f "$EMOJI_AUDIT_SCRIPT" ]]; then
        test_fail "emoji-audit.sh not found at $EMOJI_AUDIT_SCRIPT" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 2: Script runs without error and exits 0
# ============================================================================
test_emoji_audit_runs_without_error() {
    local output exit_code
    exit_code=0
    output=$(bash "$EMOJI_AUDIT_SCRIPT" "$REPO_ROOT" 2>&1) || exit_code=$?

    if [[ "$exit_code" -ne 0 ]]; then
        test_fail "emoji-audit.sh exited with non-zero code" "exit_code=$exit_code, output=$output"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 3: Output is valid JSON
# ============================================================================
test_emoji_audit_produces_valid_json() {
    local output
    output=$(bash "$EMOJI_AUDIT_SCRIPT" "$REPO_ROOT" 2>/dev/null)

    if ! assert_json_valid "$output"; then
        test_fail "emoji-audit.sh output is not valid JSON" "got: $output"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 4: Output has ok:true field
# ============================================================================
test_emoji_audit_ok_true() {
    local output
    output=$(bash "$EMOJI_AUDIT_SCRIPT" "$REPO_ROOT" 2>/dev/null)

    if ! assert_ok_true "$output"; then
        test_fail "emoji-audit.sh output does not have ok:true" "got: $output"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 5: Output has result field with required subfields
# ============================================================================
test_emoji_audit_result_fields() {
    local output
    output=$(bash "$EMOJI_AUDIT_SCRIPT" "$REPO_ROOT" 2>/dev/null)

    local required_fields=("files_scanned" "total_emojis" "unmapped" "unused" "usage")
    for field in "${required_fields[@]}"; do
        if ! echo "$output" | jq -e ".result | has(\"$field\")" >/dev/null 2>&1; then
            test_fail "result missing required field: $field" "output: $output"
            return 1
        fi
    done
    return 0
}

# ============================================================================
# Test 6: files_scanned is greater than 0 (command files exist)
# ============================================================================
test_emoji_audit_finds_command_files() {
    local output files_scanned
    output=$(bash "$EMOJI_AUDIT_SCRIPT" "$REPO_ROOT" 2>/dev/null)
    files_scanned=$(echo "$output" | jq '.result.files_scanned' 2>/dev/null || echo "0")

    if [[ "$files_scanned" -lt 1 ]]; then
        test_fail "files_scanned should be >= 1" "got: $files_scanned"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 7: Reference map has at least 20 entries (unused array upper bound check)
# ============================================================================
test_emoji_audit_reference_map_has_20_entries() {
    local output total_reference
    output=$(bash "$EMOJI_AUDIT_SCRIPT" "$REPO_ROOT" 2>/dev/null)

    # unused entries are emojis in the reference map not found in commands
    # usage entries are emojis from the reference map that ARE found
    # together, unused + usage keys cover all reference map entries
    local unused_count usage_count total
    unused_count=$(echo "$output" | jq '.result.unmapped | length' 2>/dev/null || echo "0")
    usage_count=$(echo "$output" | jq '.result.usage | keys | length' 2>/dev/null || echo "0")
    local unused_ref_count
    unused_ref_count=$(echo "$output" | jq '.result.unused | length' 2>/dev/null || echo "0")
    total=$((usage_count + unused_ref_count))

    if [[ "$total" -lt 20 ]]; then
        test_fail "reference map should have at least 20 entries" "got: $total (usage=$usage_count, unused_ref=$unused_ref_count)"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 8: total_emojis is a non-negative integer
# ============================================================================
test_emoji_audit_total_emojis_is_integer() {
    local output total_emojis
    output=$(bash "$EMOJI_AUDIT_SCRIPT" "$REPO_ROOT" 2>/dev/null)
    total_emojis=$(echo "$output" | jq '.result.total_emojis' 2>/dev/null || echo "-1")

    if ! [[ "$total_emojis" =~ ^[0-9]+$ ]]; then
        test_fail "total_emojis should be a non-negative integer" "got: $total_emojis"
        return 1
    fi
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running emoji-audit tests"
log_info "Repo root: $REPO_ROOT"
log_info "Script path: $EMOJI_AUDIT_SCRIPT"

run_test test_emoji_audit_script_exists "emoji-audit.sh script file exists"
run_test test_emoji_audit_runs_without_error "emoji-audit.sh runs without error"
run_test test_emoji_audit_produces_valid_json "emoji-audit.sh produces valid JSON"
run_test test_emoji_audit_ok_true "emoji-audit.sh output has ok:true"
run_test test_emoji_audit_result_fields "emoji-audit.sh result has required fields"
run_test test_emoji_audit_finds_command_files "emoji-audit.sh finds command files"
run_test test_emoji_audit_reference_map_has_20_entries "reference map has at least 20 entries"
run_test test_emoji_audit_total_emojis_is_integer "total_emojis is a non-negative integer"

test_summary
