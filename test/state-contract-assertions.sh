#!/usr/bin/env bash
# Test: State contract assertions
# Verifies that each state type lives where the design document claims.
# These are READ-ONLY tests -- they assert facts about current file locations
# without modifying anything.

set -uo pipefail
# Note: set -e disabled because ((PASS++)) returns 1 when PASS was 0

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DATA_DIR="$REPO_ROOT/.aether/data"
HUB_DIR="$HOME/.aether"

PASS=0
FAIL=0
SKIP=0

assert_file_exists() {
    local label="$1" path="$2" expected="$3"  # expected: "repo" or "hub"
    if [[ -f "$path" ]]; then
        echo "  PASS: $label exists at $path (expected: $expected)"
        ((PASS++))
    else
        echo "  FAIL: $label expected at $path but not found"
        ((FAIL++))
    fi
}

assert_dir_exists() {
    local label="$1" path="$2" expected="$3"
    if [[ -d "$path" ]]; then
        echo "  PASS: $label directory exists at $path (expected: $expected)"
        ((PASS++))
    else
        echo "  FAIL: $label directory expected at $path but not found"
        ((FAIL++))
    fi
}

assert_file_uses_var() {
    local label="$1" file="$2" var_pattern="$3"
    if grep -q "$var_pattern" "$file" 2>/dev/null; then
        echo "  PASS: $label uses $var_pattern"
        ((PASS++))
    else
        echo "  FAIL: $label does not use $var_pattern in $file"
        ((FAIL++))
    fi
}

echo "=== State Contract Assertion Tests ==="
echo ""

echo "--- BRANCH-LOCAL state (lives in repo, different per branch) ---"

echo ""
echo "1. COLONY_STATE.json"
assert_file_exists "COLONY_STATE.json" "$DATA_DIR/COLONY_STATE.json" "repo"
assert_file_uses_var "state-api.sh" "$REPO_ROOT/.aether/utils/state-api.sh" 'DATA_DIR/COLONY_STATE.json'

echo ""
echo "2. pheromones.json"
assert_file_exists "pheromones.json" "$DATA_DIR/pheromones.json" "repo"
assert_file_uses_var "pheromone.sh" "$REPO_ROOT/.aether/utils/pheromone.sh" 'COLONY_DATA_DIR/pheromones.json'

echo ""
echo "3. midden/"
assert_dir_exists "midden directory" "$DATA_DIR/midden" "repo"
assert_file_uses_var "midden.sh" "$REPO_ROOT/.aether/utils/midden.sh" 'COLONY_DATA_DIR/midden'

echo ""
echo "4. flags (via COLONY_DATA_DIR)"
assert_file_uses_var "flag.sh" "$REPO_ROOT/.aether/utils/flag.sh" 'COLONY_DATA_DIR/flags.json'

echo ""
echo "5. session.json"
assert_file_exists "session.json" "$DATA_DIR/session.json" "repo"

echo ""
echo "6. learning-observations.json"
assert_file_exists "learning-observations.json" "$DATA_DIR/learning-observations.json" "repo"
assert_file_uses_var "learning.sh" "$REPO_ROOT/.aether/utils/learning.sh" 'COLONY_DATA_DIR/learning-observations.json'

echo ""
echo "7. rolling-summary.log"
assert_file_exists "rolling-summary.log" "$DATA_DIR/rolling-summary.log" "repo"

echo ""
echo "8. spawn-tree.txt"
assert_file_exists "spawn-tree.txt" "$DATA_DIR/spawn-tree.txt" "repo"
assert_file_uses_var "spawn.sh" "$REPO_ROOT/.aether/utils/spawn.sh" 'COLONY_DATA_DIR/spawn-tree.txt'

echo ""
echo "9. activity.log"
assert_file_exists "activity.log" "$DATA_DIR/activity.log" "repo"

echo ""
echo "10. last-build-result.json / last-build-claims.json"
assert_file_exists "last-build-result.json" "$DATA_DIR/last-build-result.json" "repo"
assert_file_exists "last-build-claims.json" "$DATA_DIR/last-build-claims.json" "repo"

echo ""
echo "11. pending-decisions.json"
assert_file_exists "pending-decisions.json" "$DATA_DIR/pending-decisions.json" "repo"

echo ""
echo "12. errors.log"
assert_file_exists "errors.log" "$DATA_DIR/errors.log" "repo"

echo ""
echo "13. queen-wisdom.json (local colony-level copy)"
assert_file_exists "queen-wisdom.json" "$DATA_DIR/queen-wisdom.json" "repo"

echo ""
echo "14. survey/ directory"
assert_dir_exists "survey" "$DATA_DIR/survey" "repo"

echo ""
echo "15. checkpoints/backups"
assert_dir_exists "backups" "$DATA_DIR/backups" "repo"

echo ""
echo "--- HUB-GLOBAL state (lives in ~/.aether/, shared across branches/repos) ---"

echo ""
echo "16. Hub QUEEN.md"
assert_file_exists "Hub QUEEN.md" "$HUB_DIR/QUEEN.md" "hub"
assert_file_uses_var "queen.sh reads global" "$REPO_ROOT/.aether/utils/queen.sh" 'HOME/.aether/QUEEN.md'

echo ""
echo "17. Hive Brain (wisdom.json)"
assert_file_exists "Hive wisdom.json" "$HUB_DIR/hive/wisdom.json" "hub"
assert_file_uses_var "hive.sh" "$REPO_ROOT/.aether/utils/hive.sh" 'HOME/.aether/hive/wisdom.json'

echo ""
echo "18. Eternal Memory"
assert_file_exists "Eternal memory.json" "$HUB_DIR/eternal/memory.json" "hub"
assert_file_uses_var "pheromone.sh eternal" "$REPO_ROOT/.aether/utils/pheromone.sh" 'HOME/.aether/eternal'

echo ""
echo "19. Registry"
assert_file_exists "Registry" "$HUB_DIR/registry.json" "hub"
assert_file_uses_var "registry-add" "$REPO_ROOT/.aether/aether-utils.sh" 'HOME/.aether/registry.json'

echo ""
echo "20. Hub skills directory"
assert_dir_exists "Hub skills" "$HUB_DIR/skills" "hub"

echo ""
echo "21. Hub activity.log"
assert_file_exists "Hub activity.log" "$HUB_DIR/data/activity.log" "hub"

echo ""
echo "22. Hub chambers (entombed colonies)"
assert_dir_exists "Hub chambers" "$HUB_DIR/chambers" "hub"

echo ""
echo "--- VERIFICATION: .aether/data/ is in .gitignore ---"

echo ""
echo "23. Branch-local state is NOT tracked by git"
if grep -q '.aether/data/' "$REPO_ROOT/.gitignore" 2>/dev/null; then
    echo "  PASS: .aether/data/ is in .gitignore (branch-local state is untracked)"
    ((PASS++))
else
    echo "  FAIL: .aether/data/ NOT in .gitignore -- branch-local state would be committed"
    ((FAIL++))
fi

echo ""
echo "24. Hub state is NOT inside any git repo"
if ! git -C "$HUB_DIR" rev-parse --is-inside-work-tree 2>/dev/null; then
    echo "  PASS: ~/.aether/ is not inside a git repo (hub state is branch-agnostic)"
    ((PASS++))
else
    echo "  FAIL: ~/.aether/ is inside a git repo -- hub state would be branch-sensitive"
    ((FAIL++))
fi

echo ""
echo "=== RESULTS ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo "Skipped: $SKIP"
echo ""

if [[ "$FAIL" -gt 0 ]]; then
    echo "STATUS: SOME ASSERTIONS FAILED"
    exit 1
else
    echo "STATUS: ALL ASSERTIONS PASSED"
    exit 0
fi
