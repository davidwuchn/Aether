#!/usr/bin/env bash
# Tests for colony_version field in templates
# Verifies:
# 1. colony_version exists in colony-state.template.json with default value 1
# 2. colony-state-reset.jq.template resets colony_version to 0

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

COLONY_STATE_TEMPLATE="$REPO_ROOT/.aether/templates/colony-state.template.json"
RESET_TEMPLATE="$REPO_ROOT/.aether/templates/colony-state-reset.jq.template"

# ============================================================================
# Test 1: colony_version field exists in colony-state.template.json
# ============================================================================
test_colony_version_field_exists() {
    if ! jq -e 'has("colony_version")' "$COLONY_STATE_TEMPLATE" >/dev/null 2>&1; then
        test_fail "colony_version field not found in colony-state.template.json" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 2: colony_version default value is 1 in colony-state.template.json
# ============================================================================
test_colony_version_default_value() {
    local value
    value=$(jq '.colony_version' "$COLONY_STATE_TEMPLATE" 2>/dev/null)

    if [[ "$value" != "1" ]]; then
        test_fail "colony_version default should be 1" "got: $value"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 3: colony_version comment field exists in colony-state.template.json
# ============================================================================
test_colony_version_comment_exists() {
    if ! jq -e 'has("_comment_colony_version")' "$COLONY_STATE_TEMPLATE" >/dev/null 2>&1; then
        test_fail "_comment_colony_version field not found in colony-state.template.json" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 4: colony_version is placed after colony_name and before state
# ============================================================================
test_colony_version_field_order() {
    local keys
    keys=$(jq -r '[keys_unsorted[]] | to_entries | map(select(.value | IN("colony_name","colony_version","state"))) | map(.key) | @json' "$COLONY_STATE_TEMPLATE" 2>/dev/null)

    # Extract positions
    local colony_name_pos colony_version_pos state_pos
    colony_name_pos=$(jq -r '[keys_unsorted[]] | index("colony_name")' "$COLONY_STATE_TEMPLATE" 2>/dev/null)
    colony_version_pos=$(jq -r '[keys_unsorted[]] | index("colony_version")' "$COLONY_STATE_TEMPLATE" 2>/dev/null)
    state_pos=$(jq -r '[keys_unsorted[]] | index("state")' "$COLONY_STATE_TEMPLATE" 2>/dev/null)

    if [[ -z "$colony_version_pos" || "$colony_version_pos" == "null" ]]; then
        test_fail "colony_version not found in template keys" ""
        return 1
    fi

    if [[ "$colony_version_pos" -le "$colony_name_pos" ]]; then
        test_fail "colony_version should come after colony_name" "colony_name=$colony_name_pos, colony_version=$colony_version_pos"
        return 1
    fi

    if [[ "$colony_version_pos" -ge "$state_pos" ]]; then
        test_fail "colony_version should come before state" "colony_version=$colony_version_pos, state=$state_pos"
        return 1
    fi

    return 0
}

# ============================================================================
# Test 5: reset template contains .colony_version = 0 line
# ============================================================================
test_reset_template_has_colony_version() {
    if ! grep -q '\.colony_version = 0' "$RESET_TEMPLATE"; then
        test_fail ".colony_version = 0 not found in colony-state-reset.jq.template" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 6: reset template actually resets colony_version to 0 when applied via jq
# ============================================================================
test_reset_template_applies_correctly() {
    local tmpdir
    tmpdir=$(mktemp -d)

    # Create a state JSON with colony_version = 3
    cat > "$tmpdir/state.json" << 'EOF'
{
  "version": "3.0",
  "goal": "test goal",
  "colony_name": "test-colony",
  "colony_version": 3,
  "state": "CROWNED_ANTHILL",
  "current_phase": 5,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "milestone": null,
  "plan": {
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
    "phase_learnings": [],
    "decisions": [],
    "instincts": []
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "signals": [],
  "graveyards": [],
  "events": []
}
EOF

    local result
    result=$(jq -f "$RESET_TEMPLATE" "$tmpdir/state.json" 2>/dev/null)

    local colony_version
    colony_version=$(echo "$result" | jq '.colony_version' 2>/dev/null)

    rm -rf "$tmpdir"

    if [[ "$colony_version" != "0" ]]; then
        test_fail "colony_version should be reset to 0 after applying reset template" "got: $colony_version"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 7: reset template places .colony_version = 0 after .goal = null
# ============================================================================
test_reset_template_colony_version_after_goal() {
    local goal_line colony_version_line
    goal_line=$(grep -n '\.goal = null' "$RESET_TEMPLATE" | head -1 | cut -d: -f1)
    colony_version_line=$(grep -n '\.colony_version = 0' "$RESET_TEMPLATE" | head -1 | cut -d: -f1)

    if [[ -z "$goal_line" || -z "$colony_version_line" ]]; then
        test_fail "Could not find .goal or .colony_version lines in reset template" ""
        return 1
    fi

    if [[ "$colony_version_line" -le "$goal_line" ]]; then
        test_fail ".colony_version = 0 should appear after .goal = null" "goal_line=$goal_line, colony_version_line=$colony_version_line"
        return 1
    fi

    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running colony_version template tests"
log_info "Repo root: $REPO_ROOT"

run_test test_colony_version_field_exists "colony_version field exists in colony-state template"
run_test test_colony_version_default_value "colony_version default value is 1"
run_test test_colony_version_comment_exists "_comment_colony_version instruction field exists"
run_test test_colony_version_field_order "colony_version placed after colony_name and before state"
run_test test_reset_template_has_colony_version "reset template contains .colony_version = 0"
run_test test_reset_template_applies_correctly "reset template resets colony_version to 0 when applied"
run_test test_reset_template_colony_version_after_goal "reset template: .colony_version = 0 after .goal = null"

test_summary
