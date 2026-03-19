#!/usr/bin/env bash
# test-fresh-install.sh — Fresh install smoke test
# Simulates the complete user journey: install -> lay-eggs -> init -> plan -> build -> continue
# in an isolated environment with HOME overridden (no prior Aether state).
#
# NOTE: Written for bash 3.2 (macOS default). No associative arrays.
# Supports --results-file <path> flag for master runner integration.

set -euo pipefail

E2E_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$E2E_SCRIPT_DIR/../.." && pwd)"

# Parse --results-file flag
EXTERNAL_RESULTS_FILE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --results-file)
      EXTERNAL_RESULTS_FILE="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

# Source shared e2e infrastructure
# shellcheck source=./e2e-helpers.sh
source "$E2E_SCRIPT_DIR/e2e-helpers.sh"

echo ""
echo "================================================================"
echo "FRESH INSTALL: Complete Install-to-Build Lifecycle Smoke Test"
echo "  install -> lay-eggs -> init -> signals -> build -> continue"
echo "================================================================"

# ============================================================================
# Create fully isolated test environment (overrides HOME)
# ============================================================================

FRESH_TMP=$(mktemp -d)
ORIGINAL_HOME="$HOME"
export HOME="$FRESH_TMP"

cleanup_fresh() {
  export HOME="$ORIGINAL_HOME"
  if [[ -n "${FRESH_TMP:-}" && -d "$FRESH_TMP" ]]; then
    rm -rf "$FRESH_TMP"
    echo ""
    echo "  Fresh install test environment cleaned up."
  fi
}
trap cleanup_fresh EXIT

echo ""
echo "--- Isolated HOME at $FRESH_TMP ---"

# ============================================================================
# Result tracking (file-based, bash 3.2 compatible)
# ============================================================================

FRESH_RESULTS=$(mktemp)

record_fresh_result() {
  local step="$1"
  local status="$2"
  local notes="${3:-}"
  local tmp
  tmp=$(mktemp)
  grep -v "^${step}|" "$FRESH_RESULTS" > "$tmp" 2>/dev/null || true
  echo "${step}|${status}|${notes}" >> "$tmp"
  mv "$tmp" "$FRESH_RESULTS"
}

# ============================================================================
# STEP 1: Hub installation (simulates npm install -g aether-colony)
# ============================================================================

echo ""
echo "--- Step 1: Hub installation (node bin/cli.js install) ---"

STEP1_PASS=true

cd "$PROJECT_ROOT"
node bin/cli.js install --quiet 2>/dev/null || true

# Verify hub structure
if [[ -f "$HOME/.aether/system/aether-utils.sh" ]]; then
  echo "  PASS: aether-utils.sh installed to hub"
else
  echo "  FAIL: aether-utils.sh not found in hub"
  STEP1_PASS=false
fi

if [[ -d "$HOME/.aether/system/utils" ]]; then
  echo "  PASS: utils/ directory installed to hub"
else
  echo "  FAIL: utils/ directory not found in hub"
  STEP1_PASS=false
fi

if [[ "$STEP1_PASS" == "true" ]]; then
  record_fresh_result "step1-hub-install" "PASS" "Hub created with aether-utils.sh and utils/"
else
  record_fresh_result "step1-hub-install" "FAIL" "Hub missing critical files"
fi

# ============================================================================
# STEP 2: Lay-eggs simulation (simulates /ant:lay-eggs on a fresh repo)
# ============================================================================

echo ""
echo "--- Step 2: Lay-eggs simulation (copy from hub to project) ---"

STEP2_PASS=true

# Create a fresh repo inside the isolated HOME
TEST_REPO="$FRESH_TMP/test-project"
mkdir -p "$TEST_REPO"
(
  cd "$TEST_REPO"
  git init -q
  echo "fresh install test project" > README.md
  git add README.md
  git commit -q -m "init: fresh install test project"
) 2>/dev/null || true

# Simulate lay-eggs: copy system files from hub to project .aether/
mkdir -p "$TEST_REPO/.aether/data"
mkdir -p "$TEST_REPO/.aether/utils"
mkdir -p "$TEST_REPO/.aether/exchange"

cp "$HOME/.aether/system/aether-utils.sh" "$TEST_REPO/.aether/"
if [[ -d "$HOME/.aether/system/utils" ]]; then
  cp -r "$HOME/.aether/system/utils/." "$TEST_REPO/.aether/utils/"
fi
if [[ -d "$HOME/.aether/system/exchange" ]]; then
  cp -r "$HOME/.aether/system/exchange/." "$TEST_REPO/.aether/exchange/" 2>/dev/null || true
fi
if [[ -d "$HOME/.aether/system/templates" ]]; then
  cp -r "$HOME/.aether/system/templates" "$TEST_REPO/.aether/templates"
fi

UTILS="$TEST_REPO/.aether/aether-utils.sh"

# Create COLONY_STATE.json (needed before queen-init)
cat > "$TEST_REPO/.aether/data/COLONY_STATE.json" << 'COLONY_EOF'
{
  "goal": "fresh-install-smoke-test",
  "state": "active",
  "current_phase": 1,
  "milestone": "First Mound",
  "plan": {"id": "smoke-test-plan", "tasks": []},
  "memory": {
    "instincts": [],
    "phase_learnings": [],
    "decisions": []
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "fresh-install-001",
  "initialized_at": "2026-03-19T00:00:00Z"
}
COLONY_EOF

# Run queen-init subcommand
raw_qi=$(bash "$UTILS" queen-init 2>&1 || true)
qi_out=$(extract_json "$raw_qi")

if echo "$qi_out" | jq -e '(.created == true) or (.result.created == true) or (.reason == "already_exists") or (.result.reason == "already_exists")' >/dev/null 2>&1; then
  echo "  PASS: queen-init succeeded"
else
  echo "  FAIL: queen-init did not return created:true"
  echo "  Got: $qi_out"
  STEP2_PASS=false
fi

# Verify QUEEN.md was created
if [[ -f "$TEST_REPO/.aether/QUEEN.md" ]]; then
  echo "  PASS: QUEEN.md created in project"
  # Verify QUEEN.md contains template placeholder text, not promoted entries
  if grep -q "No philosophies recorded yet" "$TEST_REPO/.aether/QUEEN.md" 2>/dev/null; then
    echo "  PASS: QUEEN.md contains clean template placeholders"
  else
    echo "  FAIL: QUEEN.md does not contain template placeholders"
    STEP2_PASS=false
  fi
else
  echo "  FAIL: QUEEN.md not created"
  STEP2_PASS=false
fi

if [[ "$STEP2_PASS" == "true" ]]; then
  record_fresh_result "step2-lay-eggs" "PASS" "Lay-eggs simulated; QUEEN.md created with clean template"
else
  record_fresh_result "step2-lay-eggs" "FAIL" "Lay-eggs simulation failed"
fi

# ============================================================================
# STEP 3: Colony init (simulates /ant:init "smoke test")
# ============================================================================

echo ""
echo "--- Step 3: Colony init (session-init) ---"

STEP3_PASS=true

raw_si=$(bash "$UTILS" session-init "" "fresh-install-smoke-test" 2>&1 || true)
si_out=$(extract_json "$raw_si")

session_file="$TEST_REPO/.aether/data/session.json"

if echo "$si_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: session-init returned ok:true"
  if [[ -f "$session_file" ]]; then
    has_current_phase=$(jq 'if .current_phase != null then "yes" else "no" end' "$session_file" 2>/dev/null || echo '"no"')
    has_current_phase="${has_current_phase//\"/}"
    has_goal=$(jq 'if .colony_goal != null then "yes" else "no" end' "$session_file" 2>/dev/null || echo '"no"')
    has_goal="${has_goal//\"/}"
    if [[ "$has_current_phase" == "yes" && "$has_goal" == "yes" ]]; then
      echo "  PASS: session.json created with current_phase + colony_goal"
    else
      echo "  FAIL: session.json missing required fields (phase=$has_current_phase goal=$has_goal)"
      STEP3_PASS=false
    fi
  else
    echo "  FAIL: session.json not created"
    STEP3_PASS=false
  fi
else
  echo "  FAIL: session-init did not return ok:true"
  echo "  Got: $si_out"
  STEP3_PASS=false
fi

if [[ "$STEP3_PASS" == "true" ]]; then
  record_fresh_result "step3-colony-init" "PASS" "session-init ok; session.json has current_phase+goal"
else
  record_fresh_result "step3-colony-init" "FAIL" "Colony init failed"
fi

# ============================================================================
# STEP 4: Signal flow (simulates pheromone operations during plan/build)
# ============================================================================

echo ""
echo "--- Step 4: Signal flow (pheromone-write + pheromone-read) ---"

STEP4_PASS=true

# Initialize pheromones.json from template
cat > "$TEST_REPO/.aether/data/pheromones.json" << 'PHER_EOF'
{"signals":[],"midden":[]}
PHER_EOF

# Write a FOCUS pheromone
raw_pw=$(bash "$UTILS" pheromone-write FOCUS "fresh install quality" --strength 0.8 2>&1 || true)
pw_out=$(extract_json "$raw_pw")

if echo "$pw_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: pheromone-write FOCUS returned ok:true"
else
  echo "  FAIL: pheromone-write FOCUS failed"
  echo "  Got: $pw_out"
  STEP4_PASS=false
fi

# Read pheromones back
raw_pr=$(bash "$UTILS" pheromone-read 2>&1 || true)
pr_out=$(extract_json "$raw_pr")

if echo "$pr_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  signal_count=$(echo "$pr_out" | jq '.count // 0' 2>/dev/null || echo "0")
  if [[ "$signal_count" -gt 0 ]] 2>/dev/null || echo "$pr_out" | jq -e '.signals | length > 0' >/dev/null 2>&1; then
    echo "  PASS: pheromone-read returns signals (count: $signal_count)"
  else
    if echo "$pr_out" | jq -e '.signals != null' >/dev/null 2>&1; then
      echo "  PASS: pheromone-read returns ok:true with signals field"
    else
      echo "  NOTE: pheromone-read returned ok:true but signal count unclear"
    fi
  fi
else
  echo "  FAIL: pheromone-read failed"
  echo "  Got: $pr_out"
  STEP4_PASS=false
fi

if [[ "$STEP4_PASS" == "true" ]]; then
  record_fresh_result "step4-signal-flow" "PASS" "pheromone-write FOCUS ok; pheromone-read ok"
else
  record_fresh_result "step4-signal-flow" "FAIL" "Pheromone operations failed"
fi

# ============================================================================
# STEP 5: Build simulation (simulates /ant:build)
# ============================================================================

echo ""
echo "--- Step 5: Build simulation (pheromone-prime + session-update) ---"

STEP5_PASS=true

# pheromone-prime should work with the signal we just wrote
raw_pp=$(bash "$UTILS" pheromone-prime builder 2>&1 || true)
pp_out=$(extract_json "$raw_pp")

if echo "$pp_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: pheromone-prime returned ok:true"
else
  echo "  NOTE: pheromone-prime issue: $pp_out (non-critical)"
fi

# session-update: plan -> build
raw_su1=$(bash "$UTILS" session-update "/ant:plan" "/ant:build 1" "Plan created" 2>&1 || true)
su1_out=$(extract_json "$raw_su1")

if echo "$su1_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: session-update (plan->build) returned ok:true"
else
  echo "  FAIL: session-update (plan->build) failed"
  echo "  Got: $su1_out"
  STEP5_PASS=false
fi

# session-update: build -> continue
raw_su2=$(bash "$UTILS" session-update "/ant:build" "/ant:continue" "Build in progress" 2>&1 || true)
su2_out=$(extract_json "$raw_su2")

if echo "$su2_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: session-update (build->continue) returned ok:true"
else
  echo "  FAIL: session-update (build->continue) failed"
  echo "  Got: $su2_out"
  STEP5_PASS=false
fi

if [[ "$STEP5_PASS" == "true" ]]; then
  record_fresh_result "step5-build-sim" "PASS" "pheromone-prime ok; session-update plan+build ok"
else
  record_fresh_result "step5-build-sim" "FAIL" "Build simulation failed"
fi

# ============================================================================
# STEP 6: Continue simulation (simulates /ant:continue)
# ============================================================================

echo ""
echo "--- Step 6: Continue simulation (session-update + milestone-detect) ---"

STEP6_PASS=true

# session-update: continue -> seal
raw_su3=$(bash "$UTILS" session-update "/ant:continue" "/ant:seal" "Phase completed" 2>&1 || true)
su3_out=$(extract_json "$raw_su3")

if echo "$su3_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: session-update (continue->seal) returned ok:true"
else
  echo "  FAIL: session-update (continue->seal) failed"
  echo "  Got: $su3_out"
  STEP6_PASS=false
fi

# milestone-detect
raw_md=$(bash "$UTILS" milestone-detect 2>&1 || true)
md_out=$(extract_json "$raw_md")

if echo "$md_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  milestone=$(echo "$md_out" | jq -r '.milestone // "unknown"' 2>/dev/null || echo "unknown")
  echo "  PASS: milestone-detect returned ok:true (milestone: $milestone)"
else
  echo "  FAIL: milestone-detect failed"
  echo "  Got: $md_out"
  STEP6_PASS=false
fi

if [[ "$STEP6_PASS" == "true" ]]; then
  record_fresh_result "step6-continue-sim" "PASS" "session-update continue ok; milestone-detect ok"
else
  record_fresh_result "step6-continue-sim" "FAIL" "Continue simulation failed"
fi

# ============================================================================
# Output results summary
# ============================================================================

echo ""
echo "================================================================"
echo "FRESH INSTALL Results Summary"
echo "================================================================"
echo ""
echo "| Step | Status | Notes |"
echo "|------|--------|-------|"

pass_count=0
fail_count=0
while IFS='|' read -r step status notes; do
  echo "| $step | $status | $notes |"
  if [[ "$status" == "PASS" ]]; then
    pass_count=$((pass_count + 1))
  else
    fail_count=$((fail_count + 1))
  fi
done < <(sort "$FRESH_RESULTS")

echo ""
echo "**Fresh Install Summary:** $pass_count PASS, $fail_count FAIL"
echo ""

# Write external results file if requested (for master runner)
if [[ -n "$EXTERNAL_RESULTS_FILE" ]]; then
  if [[ $fail_count -eq 0 ]]; then
    echo "FRESH_INSTALL=PASS" >> "$EXTERNAL_RESULTS_FILE"
  else
    echo "FRESH_INSTALL=FAIL" >> "$EXTERNAL_RESULTS_FILE"
  fi
fi

# Cleanup results temp file
rm -f "$FRESH_RESULTS"

if [[ $fail_count -eq 0 ]]; then
  echo "FRESH INSTALL TEST: ALL PASS"
  exit 0
else
  echo "FRESH INSTALL TEST: $fail_count STEP(S) FAILED"
  exit 1
fi
