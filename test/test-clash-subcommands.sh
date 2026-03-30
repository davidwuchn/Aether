#!/usr/bin/env bash
# Test: Clash subcommands (clash-setup, clash-check) in aether-utils.sh
# Tests the dispatcher integration for clash detection commands.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

PASS=0
FAIL=0
SKIP=0

pass() { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }

echo "=== Clash Subcommand Tests ==="
echo ""

# --- Test 1: clash-check dispatches correctly ---
echo "1. clash-check subcommand exists and runs"
result=$(bash "$AETHER_UTILS" clash-check --help 2>&1)
exit_code=$?
# It should either return valid JSON or a usage error
if [[ $exit_code -ne 0 ]] || echo "$result" | jq -e '.' >/dev/null 2>&1; then
    pass "clash-check subcommand is recognized"
else
    fail "clash-check not recognized or returned invalid output: $result"
fi

# --- Test 2: clash-check returns JSON ---
echo "2. clash-check returns valid JSON"
result=$(bash "$AETHER_UTILS" clash-check --file "nonexistent.txt" 2>&1)
if echo "$result" | jq -e '.' >/dev/null 2>&1; then
    pass "clash-check returns valid JSON"
else
    fail "clash-check returned invalid JSON: $result"
fi

# --- Test 3: clash-check --file missing returns error ---
echo "3. clash-check without --file returns error"
result=$(bash "$AETHER_UTILS" clash-check 2>&1)
if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
    pass "clash-check without --file returns error JSON"
else
    fail "Expected error JSON, got: $result"
fi

# --- Test 4: clash-check --file with non-existent file returns no conflict ---
echo "4. clash-check for non-existent file returns no conflict"
result=$(bash "$AETHER_UTILS" clash-check --file "this-file-does-not-exist.ts" 2>&1)
if echo "$result" | jq -e '.ok == true and .result.conflict == false' >/dev/null 2>&1; then
    pass "clash-check returns no conflict for non-existent file"
else
    fail "Expected no conflict, got: $result"
fi

# --- Test 5: clash-setup --install registers the hook ---
echo "5. clash-setup --install registers the hook"
# Use a temp settings file to avoid modifying the real one
TMP_SETTINGS=$(mktemp -t clash-settings)
cat > "$TMP_SETTINGS" << 'EOF'
{"hooks":{}}
EOF
result=$(CLASH_SETTINGS_PATH="$TMP_SETTINGS" bash "$AETHER_UTILS" clash-setup --install 2>&1)
if echo "$result" | jq -e '.ok == true and .result.hook_installed == true' >/dev/null 2>&1; then
    pass "clash-setup --install returns success"
else
    fail "Expected hook_installed:true, got: $result"
fi
# Verify the settings file was modified
if grep -q "clash-pre-tool-use" "$TMP_SETTINGS" 2>/dev/null; then
    pass "Hook was written to settings file"
else
    fail "Hook entry not found in settings file"
fi
rm -f "$TMP_SETTINGS"

# --- Test 6: clash-setup --uninstall removes the hook ---
echo "6. clash-setup --uninstall removes the hook"
TMP_SETTINGS=$(mktemp -t clash-settings)
cat > "$TMP_SETTINGS" << 'SETTINGS_EOF'
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "node .aether/utils/hooks/clash-pre-tool-use.js",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
SETTINGS_EOF
result=$(CLASH_SETTINGS_PATH="$TMP_SETTINGS" bash "$AETHER_UTILS" clash-setup --uninstall 2>&1)
if echo "$result" | jq -e '.ok == true and .result.hook_installed == false' >/dev/null 2>&1; then
    pass "clash-setup --uninstall returns success"
else
    fail "Expected hook_installed:false, got: $result"
fi
# Verify the hook entry was removed from PreToolUse
if ! grep -q "clash-pre-tool-use" "$TMP_SETTINGS" 2>/dev/null; then
    pass "Hook was removed from settings file"
else
    fail "Hook entry still found in settings file"
fi
rm -f "$TMP_SETTINGS"

# --- Test 7: clash-setup --install is idempotent ---
echo "7. clash-setup --install is idempotent"
TMP_SETTINGS=$(mktemp -t clash-settings)
cat > "$TMP_SETTINGS" << 'EOF'
{"hooks":{"PreToolUse":[{"matcher":"Edit|Write","hooks":[{"type":"command","command":"node .aether/utils/hooks/clash-pre-tool-use.js","timeout":5}]}]}}
EOF
result=$(CLASH_SETTINGS_PATH="$TMP_SETTINGS" bash "$AETHER_UTILS" clash-setup --install 2>&1)
if echo "$result" | jq -e '.ok == true' >/dev/null 2>&1; then
    pass "clash-setup --install is idempotent"
else
    fail "Expected success on re-install, got: $result"
fi
rm -f "$TMP_SETTINGS"

# --- Test 8: clash-setup without flags shows usage ---
echo "8. clash-setup without flags returns usage"
result=$(bash "$AETHER_UTILS" clash-setup 2>&1)
if echo "$result" | jq -e '.ok == false' >/dev/null 2>&1; then
    pass "clash-setup without flags returns error"
else
    fail "Expected error for missing flags, got: $result"
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
