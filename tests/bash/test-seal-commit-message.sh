#!/usr/bin/env bash
# Tests for generate-commit-message seal type
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
AETHER_UTILS="${SCRIPT_DIR}/../../.aether/aether-utils.sh"

passed=0
failed=0

assert_eq() {
  local desc="$1" expected="$2" actual="$3"
  if [[ "$expected" == "$actual" ]]; then
    echo "  PASS: $desc"
    ((passed++))
  else
    echo "  FAIL: $desc"
    echo "    expected: $expected"
    echo "    actual:   $actual"
    ((failed++))
  fi
}

assert_match() {
  local desc="$1" pattern="$2" actual="$3"
  if echo "$actual" | grep -qE "$pattern"; then
    echo "  PASS: $desc"
    ((passed++))
  else
    echo "  FAIL: $desc"
    echo "    pattern: $pattern"
    echo "    actual:  $actual"
    ((failed++))
  fi
}

echo "=== generate-commit-message seal type ==="

# Test 1: seal type returns valid JSON with expected keys
echo ""
echo "Test 1: seal type returns valid JSON"
result=$(bash "$AETHER_UTILS" generate-commit-message seal 5 "Build the widget system" "12" 2>/dev/null)
ok=$(echo "$result" | jq -r '.ok // false')
assert_eq "returns ok: true" "true" "$ok"
msg=$(echo "$result" | jq -r '.result.message // empty')
body=$(echo "$result" | jq -r '.result.body // empty')
assert_match "has message field" ".+" "$msg"
assert_match "has body field" ".+" "$body"

# Test 2: message starts with aether-seal: and includes goal text
echo ""
echo "Test 2: message format starts with aether-seal:"
assert_match "message starts with aether-seal:" "^aether-seal:" "$msg"
assert_match "message includes goal text" "widget" "$msg"

# Test 3: long goal gets truncated to under 72 chars
echo ""
echo "Test 3: truncation of long goal to 72 chars"
long_result=$(bash "$AETHER_UTILS" generate-commit-message seal 10 "This is a very long colony goal that exceeds the seventy two character limit for commit messages" "30" 2>/dev/null)
long_msg=$(echo "$long_result" | jq -r '.result.message // empty')
msg_len=${#long_msg}
if [[ $msg_len -le 72 ]]; then
  echo "  PASS: message is under 72 chars (${msg_len})"
  ((passed++))
else
  echo "  FAIL: message exceeds 72 chars (${msg_len})"
  echo "    message: $long_msg"
  ((failed++))
fi

# Test 4: body contains phase count and colony age
echo ""
echo "Test 4: body contains phase count and colony age"
assert_match "body has phase count" "5 phase" "$body"
assert_match "body has colony age" "12 day" "$body"

# Test 5: seal with no colony age still works
echo ""
echo "Test 5: seal with no colony age"
no_age_result=$(bash "$AETHER_UTILS" generate-commit-message seal 3 "Quick project" 2>/dev/null)
no_age_ok=$(echo "$no_age_result" | jq -r '.ok // false')
assert_eq "returns ok without age" "true" "$no_age_ok"

echo ""
echo "=== Results: $passed passed, $failed failed ==="
exit $failed
