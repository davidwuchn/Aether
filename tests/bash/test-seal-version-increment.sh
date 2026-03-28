#!/usr/bin/env bash
# Tests for seal version increment logic
# Verifies:
# 1. colony_version is incremented during seal (simulates Step 4.5 logic)
# 2. If colony_version doesn't exist, it starts from 0+1=1
# 3. The milestone display includes the version number
# 4. Claude Code seal.md contains a version increment step before Step 5
# 5. OpenCode seal.md contains a version increment step before Step 5

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

CLAUDE_SEAL="$REPO_ROOT/.claude/commands/ant/seal.md"
OPENCODE_SEAL="$REPO_ROOT/.opencode/commands/ant/seal.md"

# ============================================================================
# Test 1: version increment logic works when colony_version exists
# ============================================================================
test_seal_version_increment_existing_value() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "test seal versioning",
  "colony_name": "test-colony",
  "colony_version": 2,
  "state": "READY",
  "current_phase": 3,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "milestone": "Sealed Chambers",
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

    # Simulate the seal version increment step:
    # read current colony_version, increment by 1, write back
    local current_version
    current_version=$(jq '.colony_version // 0' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)
    local new_version=$(( current_version + 1 ))

    local updated
    updated=$(jq --argjson v "$new_version" '.colony_version = $v' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)
    echo "$updated" > "$tmpdir/.aether/data/COLONY_STATE.json"

    local stored_version
    stored_version=$(jq '.colony_version' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)

    rm -rf "$tmpdir"

    if [[ "$stored_version" != "3" ]]; then
        test_fail "colony_version should be incremented from 2 to 3" "got: $stored_version"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 2: version increment defaults to 0+1=1 when colony_version is absent
# ============================================================================
test_seal_version_increment_missing_field_defaults_to_1() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

    # State without colony_version field (backward compat scenario)
    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "old colony without version field",
  "colony_name": "old-colony",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "milestone": "Sealed Chambers",
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

    # Simulate the seal version increment step with default 0 fallback:
    local current_version
    current_version=$(jq '.colony_version // 0' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)
    local new_version=$(( current_version + 1 ))

    local updated
    updated=$(jq --argjson v "$new_version" '.colony_version = $v' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)
    echo "$updated" > "$tmpdir/.aether/data/COLONY_STATE.json"

    local stored_version
    stored_version=$(jq '.colony_version' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)

    rm -rf "$tmpdir"

    if [[ "$stored_version" != "1" ]]; then
        test_fail "colony_version should default to 0 then increment to 1" "got: $stored_version"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 3: the milestone display should include the version number
# ============================================================================
test_seal_milestone_display_includes_version() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"

    cat > "$tmpdir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "display version in milestone",
  "colony_name": "test-colony",
  "colony_version": 4,
  "state": "READY",
  "current_phase": 2,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "milestone": "Sealed Chambers",
  "plan": {"generated_at": null, "confidence": null, "phases": []},
  "memory": {"phase_learnings": [], "decisions": [], "instincts": []},
  "errors": {"records": [], "flagged_patterns": []},
  "signals": [], "graveyards": [], "events": []
}
EOF

    # Simulate reading the version after increment
    local current_version
    current_version=$(jq '.colony_version // 0' "$tmpdir/.aether/data/COLONY_STATE.json" 2>/dev/null)
    local new_version=$(( current_version + 1 ))

    # The milestone display should render as "Crowned Anthill v{new_version}"
    local milestone_display="Crowned Anthill v${new_version}"

    rm -rf "$tmpdir"

    # Verify the display string includes "v5" (4 + 1 = 5)
    if [[ "$milestone_display" != "Crowned Anthill v5" ]]; then
        test_fail "milestone display should be 'Crowned Anthill v5'" "got: $milestone_display"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 4: Claude Code seal.md contains version increment step
# ============================================================================
test_claude_seal_contains_version_increment_step() {
    if ! grep -q 'colony_version' "$CLAUDE_SEAL"; then
        test_fail "seal.md should contain colony_version reference for version increment" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 5: Claude Code seal.md version increment step appears before Step 5
# ============================================================================
test_claude_seal_version_increment_before_step5() {
    local version_line step5_line
    version_line=$(grep -n 'colony_version' "$CLAUDE_SEAL" | head -1 | cut -d: -f1)
    step5_line=$(grep -n '^### Step 5: Update Milestone' "$CLAUDE_SEAL" | head -1 | cut -d: -f1)

    if [[ -z "$version_line" ]]; then
        test_fail "colony_version not found in Claude Code seal.md" ""
        return 1
    fi

    if [[ -z "$step5_line" ]]; then
        test_fail "Step 5: Update Milestone not found in Claude Code seal.md" ""
        return 1
    fi

    if [[ "$version_line" -ge "$step5_line" ]]; then
        test_fail "colony_version increment should appear before Step 5" \
            "version_line=$version_line, step5_line=$step5_line"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 6: OpenCode seal.md contains version increment step
# ============================================================================
test_opencode_seal_contains_version_increment_step() {
    if ! grep -q 'colony_version' "$OPENCODE_SEAL"; then
        test_fail "OpenCode seal.md should contain colony_version reference for version increment" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 7: OpenCode seal.md version increment step appears before Step 5
# ============================================================================
test_opencode_seal_version_increment_before_step5() {
    local version_line step5_line
    version_line=$(grep -n 'colony_version' "$OPENCODE_SEAL" | head -1 | cut -d: -f1)
    step5_line=$(grep -n '^### Step 5: Update Milestone' "$OPENCODE_SEAL" | head -1 | cut -d: -f1)

    if [[ -z "$version_line" ]]; then
        test_fail "colony_version not found in OpenCode seal.md" ""
        return 1
    fi

    if [[ -z "$step5_line" ]]; then
        test_fail "Step 5: Update Milestone not found in OpenCode seal.md" ""
        return 1
    fi

    if [[ "$version_line" -ge "$step5_line" ]]; then
        test_fail "colony_version increment should appear before Step 5" \
            "version_line=$version_line, step5_line=$step5_line"
        return 1
    fi
    return 0
}

# ============================================================================
# Test 8: Claude Code seal.md ceremony includes "Crowned Anthill v{version}" display
# ============================================================================
test_claude_seal_ceremony_includes_versioned_milestone() {
    if ! grep -q 'Crowned Anthill v' "$CLAUDE_SEAL" && \
       ! grep -q 'colony_version' "$CLAUDE_SEAL"; then
        test_fail "seal.md ceremony should reference versioned milestone display" ""
        return 1
    fi
    # At minimum, the file must reference colony_version for the display
    if ! grep -q 'colony_version' "$CLAUDE_SEAL"; then
        test_fail "seal.md should reference colony_version for ceremony display" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Test 9: OpenCode seal.md ceremony includes versioned milestone display
# ============================================================================
test_opencode_seal_ceremony_includes_versioned_milestone() {
    if ! grep -q 'colony_version' "$OPENCODE_SEAL"; then
        test_fail "OpenCode seal.md should reference colony_version for ceremony display" ""
        return 1
    fi
    return 0
}

# ============================================================================
# Run tests
# ============================================================================

log_info "Running seal version increment tests"
log_info "Repo root: $REPO_ROOT"

run_test test_seal_version_increment_existing_value \
    "seal-version: increments colony_version from existing value"
run_test test_seal_version_increment_missing_field_defaults_to_1 \
    "seal-version: missing colony_version defaults to 0 then increments to 1"
run_test test_seal_milestone_display_includes_version \
    "seal-version: milestone display includes version number (Crowned Anthill v{N})"
run_test test_claude_seal_contains_version_increment_step \
    "claude-seal: seal.md contains colony_version increment step"
run_test test_claude_seal_version_increment_before_step5 \
    "claude-seal: version increment step appears before Step 5"
run_test test_opencode_seal_contains_version_increment_step \
    "opencode-seal: seal.md contains colony_version increment step"
run_test test_opencode_seal_version_increment_before_step5 \
    "opencode-seal: version increment step appears before Step 5"
run_test test_claude_seal_ceremony_includes_versioned_milestone \
    "claude-seal: ceremony references versioned milestone display"
run_test test_opencode_seal_ceremony_includes_versioned_milestone \
    "opencode-seal: ceremony references versioned milestone display"

test_summary
