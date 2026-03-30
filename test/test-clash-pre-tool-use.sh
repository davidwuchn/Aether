#!/usr/bin/env bash
# Test: Clash PreToolUse hook
# Tests the JavaScript PreToolUse hook that Claude Code calls before Edit/Write.
#
# These tests pipe JSON input to the hook and verify the output decision.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOK_PATH="$REPO_ROOT/.aether/utils/hooks/clash-pre-tool-use.js"

PASS=0
FAIL=0
SKIP=0

pass() { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }

# Helper: Run the hook with given stdin JSON
run_hook() {
    echo "$1" | node "$HOOK_PATH" 2>/dev/null
    local exit_code=$?
    echo "$exit_code"
}

echo "=== Clash PreToolUse Hook Tests ==="
echo ""

# --- Test 1: Allow when tool is not Edit or Write ---
echo "1. Allow when tool is Bash (not Edit/Write)"
result=$(echo '{"tool_name":"Bash","tool_input":{"command":"ls"}}' | node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
if [[ $exit_code -eq 0 ]]; then
    pass "Exits 0 for non-Edit/Write tools (allow)"
else
    fail "Expected exit 0 for Bash tool, got exit $exit_code"
fi

# --- Test 2: Allow when no file_path in tool_input ---
echo "2. Allow when tool is Edit but no file_path"
result=$(echo '{"tool_name":"Edit","tool_input":{}}' | node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
if [[ $exit_code -eq 0 ]]; then
    pass "Exits 0 when no file_path present (allow)"
else
    fail "Expected exit 0 for missing file_path, got exit $exit_code"
fi

# --- Test 3: Allow for allowlisted .aether/data/ files ---
echo "3. Allow for .aether/data/ files (allowlist bypass)"
result=$(echo '{"tool_name":"Edit","tool_input":{"file_path":"/some/repo/.aether/data/COLONY_STATE.json"}}' | node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
if [[ $exit_code -eq 0 ]]; then
    pass "Exits 0 for .aether/data/ files (allowlisted)"
else
    fail "Expected exit 0 for .aether/data/ file, got exit $exit_code"
fi

# --- Test 4: Allow when not in a git worktree (no conflict possible) ---
echo "4. Allow when clash-detect returns no conflict"
# Use a temp dir as cwd so the hook doesn't find the real clash-detect
MOCK_DIR=$(mktemp -d)
MOCK_CWD=$(mktemp -d)
cat > "$MOCK_DIR/clash-detect" << 'MOCK_EOF'
#!/bin/bash
echo '{"ok":true,"result":{"conflict":false}}'
MOCK_EOF
chmod +x "$MOCK_DIR/clash-detect"
result=$(echo "{\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"$MOCK_CWD/src/index.ts\"},\"cwd\":\"$MOCK_CWD\"}" | PATH="$MOCK_DIR:$PATH" node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
rm -rf "$MOCK_DIR" "$MOCK_CWD"
if [[ $exit_code -eq 0 ]]; then
    pass "Exits 0 when clash-detect reports no conflict (allow)"
else
    fail "Expected exit 0 for no conflict, got exit $exit_code"
fi

# --- Test 5: Block when clash-detect returns conflict ---
echo "5. Block when clash-detect returns conflict"
MOCK_DIR=$(mktemp -d)
MOCK_CWD=$(mktemp -d)
cat > "$MOCK_DIR/clash-detect" << 'MOCK_EOF'
#!/bin/bash
echo '{"ok":true,"result":{"conflict":true,"conflicting_worktrees":["feature-x"]}}'
MOCK_EOF
chmod +x "$MOCK_DIR/clash-detect"
result=$(echo "{\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"$MOCK_CWD/src/index.ts\"},\"cwd\":\"$MOCK_CWD\"}" | PATH="$MOCK_DIR:$PATH" node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
rm -rf "$MOCK_DIR" "$MOCK_CWD"
if [[ $exit_code -eq 2 ]]; then
    pass "Exits 2 when clash-detect reports conflict (block)"
else
    fail "Expected exit 2 for conflict, got exit $exit_code, output: $result"
fi

# --- Test 6: Allow when clash-detect errors out (fail-open) ---
echo "6. Allow when clash-detect fails (fail-open safety)"
MOCK_DIR=$(mktemp -d)
MOCK_CWD=$(mktemp -d)
cat > "$MOCK_DIR/clash-detect" << 'MOCK_EOF'
#!/bin/bash
echo '{"ok":false,"error":"something went wrong"}' >&2
exit 1
MOCK_EOF
chmod +x "$MOCK_DIR/clash-detect"
result=$(echo "{\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"$MOCK_CWD/src/index.ts\"},\"cwd\":\"$MOCK_CWD\"}" | PATH="$MOCK_DIR:$PATH" node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
rm -rf "$MOCK_DIR" "$MOCK_CWD"
if [[ $exit_code -eq 0 ]]; then
    pass "Exits 0 when clash-detect errors (fail-open)"
else
    fail "Expected exit 0 for clash-detect error (fail-open), got exit $exit_code"
fi

# --- Test 7: Write tool is also checked ---
echo "7. Write tool is also checked for clashes"
MOCK_DIR=$(mktemp -d)
MOCK_CWD=$(mktemp -d)
cat > "$MOCK_DIR/clash-detect" << 'MOCK_EOF'
#!/bin/bash
echo '{"ok":true,"result":{"conflict":true,"conflicting_worktrees":["other-branch"]}}'
MOCK_EOF
chmod +x "$MOCK_DIR/clash-detect"
result=$(echo "{\"tool_name\":\"Write\",\"tool_input\":{\"file_path\":\"$MOCK_CWD/src/new-file.ts\",\"content\":\"hello\"},\"cwd\":\"$MOCK_CWD\"}" | PATH="$MOCK_DIR:$PATH" node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
rm -rf "$MOCK_DIR" "$MOCK_CWD"
if [[ $exit_code -eq 2 ]]; then
    pass "Exits 2 for Write tool with conflict"
else
    fail "Expected exit 2 for Write tool conflict, got exit $exit_code"
fi

# --- Test 8: Hook handles malformed JSON gracefully ---
echo "8. Allow when input is malformed JSON (fail-open)"
result=$(echo 'not-json' | node "$HOOK_PATH" 2>/dev/null)
exit_code=$?
if [[ $exit_code -eq 0 ]]; then
    pass "Exits 0 for malformed JSON (fail-open)"
else
    fail "Expected exit 0 for malformed JSON, got exit $exit_code"
fi

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
