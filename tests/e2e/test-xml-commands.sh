#!/usr/bin/env bash
# test-xml-commands.sh — Command-level XML integration requirement verification
# XMLCMD-01: Slash command subcommands produce valid export/import results
# XMLCMD-02: Cross-colony signal transfer produces working signals
# XMLCMD-03: Seal lifecycle exports standalone pheromones.xml
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
echo "XML Area: XML Command-Level Integration Requirements"
echo "================================================================"

# ============================================================================
# Environment Setup
# ============================================================================

E2E_TMP_DIR=$(setup_e2e_env)
trap teardown_e2e_env EXIT

init_results

UTILS="$E2E_TMP_DIR/.aether/aether-utils.sh"

# Check if xmllint is available — required for XML tests
if ! command -v xmllint >/dev/null 2>&1; then
  echo "  SKIP: xmllint not available — XML features require libxml2"
  echo "  Install: xcode-select --install on macOS"
  record_result "XMLCMD-01" "FAIL" "xmllint not available"
  record_result "XMLCMD-02" "FAIL" "xmllint not available"
  record_result "XMLCMD-03" "FAIL" "xmllint not available"
  # Write external results if requested
  if [[ -n "$EXTERNAL_RESULTS_FILE" ]]; then
    echo "XMLCMD-01=FAIL" >> "$EXTERNAL_RESULTS_FILE"
    echo "XMLCMD-02=FAIL" >> "$EXTERNAL_RESULTS_FILE"
    echo "XMLCMD-03=FAIL" >> "$EXTERNAL_RESULTS_FILE"
  fi
  print_area_results "XML-Commands"
  exit 1
fi

# ============================================================================
# XMLCMD-01: Slash command subcommands produce valid export/import results
# Strategy: Run pheromone-export-xml with path arg, verify ok:true + well-formed,
#           then import with colony prefix and verify prefix in imported IDs
# ============================================================================

echo ""
echo "--- XMLCMD-01: Slash command subcommands produce valid export/import results ---"

xmlcmd01_pass=true
xmlcmd01_notes=""

export_xml="$E2E_TMP_DIR/pheromones-cmd-test.xml"

# Step 1: Export pheromones to XML
echo "  Running pheromone-export-xml with path arg..."
raw_export=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-export-xml "$export_xml")
export_out=$(extract_json "$raw_export")

if echo "$export_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: pheromone-export-xml returned ok:true"
else
  xmlcmd01_pass=false
  xmlcmd01_notes="$xmlcmd01_notes [FAIL: pheromone-export-xml ok!=true: $export_out]"
  echo "  FAIL: pheromone-export-xml did not return ok:true"
  echo "  Got: $export_out"
fi

# Step 1b: Verify file exists
if [[ -f "$export_xml" ]]; then
  echo "  PASS: exported XML file exists"
else
  xmlcmd01_pass=false
  xmlcmd01_notes="$xmlcmd01_notes [FAIL: exported XML file not created]"
  echo "  FAIL: exported XML file not created"
fi

# Step 1c: Verify well-formed XML
if [[ -f "$export_xml" ]]; then
  if xmllint --noout "$export_xml" 2>/dev/null; then
    echo "  PASS: exported XML is well-formed"
  else
    xmlcmd01_pass=false
    xmlcmd01_notes="$xmlcmd01_notes [FAIL: exported XML not well-formed]"
    echo "  FAIL: exported XML is not well-formed"
  fi
fi

# Step 2: Import with colony prefix
if [[ -f "$export_xml" ]]; then
  echo "  Running pheromone-import-xml with colony prefix 'test-source'..."
  raw_import=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-import-xml "$export_xml" "test-source" 2>&1 || true)
  import_out=$(extract_json "$raw_import")

  if echo "$import_out" | jq -e '.ok == true' >/dev/null 2>&1; then
    echo "  PASS: pheromone-import-xml returned ok:true"
  else
    xmlcmd01_pass=false
    xmlcmd01_notes="$xmlcmd01_notes [FAIL: pheromone-import-xml ok!=true: $import_out]"
    echo "  FAIL: pheromone-import-xml did not return ok:true"
    echo "  Got: $import_out"
  fi

  import_count=$(echo "$import_out" | jq -r '.result.signal_count // 0' 2>/dev/null)
  if [[ "$import_count" -gt 0 ]]; then
    echo "  PASS: imported $import_count signal(s)"
  else
    xmlcmd01_pass=false
    xmlcmd01_notes="$xmlcmd01_notes [FAIL: signal_count is 0]"
    echo "  FAIL: signal_count is 0 after import"
  fi

  # Step 3: Verify imported signals have colony prefix in IDs
  pheromones_file="$E2E_TMP_DIR/.aether/data/pheromones.json"
  prefix_found=false
  if [[ -f "$pheromones_file" ]]; then
    prefix_ids=$(jq -r '.signals[].id' "$pheromones_file" 2>/dev/null | grep "test-source" || true)
    if [[ -n "$prefix_ids" ]]; then
      prefix_found=true
      echo "  PASS: imported signal IDs contain 'test-source' prefix"
    fi
  fi
  if [[ "$prefix_found" == "false" ]]; then
    xmlcmd01_pass=false
    xmlcmd01_notes="$xmlcmd01_notes [FAIL: no imported IDs contain colony prefix]"
    echo "  FAIL: no imported signal IDs contain 'test-source' prefix"
  fi
fi

if [[ "$xmlcmd01_pass" == "true" ]]; then
  record_result "XMLCMD-01" "PASS" "export ok, well-formed XML, import with prefix ok, prefix in IDs"
else
  record_result "XMLCMD-01" "FAIL" "$xmlcmd01_notes"
fi

# ============================================================================
# XMLCMD-02: Cross-colony signal transfer produces working signals
# Strategy: Create source colony with 3 signals, export to XML,
#           create target colony with 1 signal, import, verify 4 total,
#           all imported active, all have colony prefix
# ============================================================================

echo ""
echo "--- XMLCMD-02: Cross-colony signal transfer produces working signals ---"

xmlcmd02_pass=true
xmlcmd02_notes=""

# Step 1-2: Create source colony with 3 known signals (FOCUS, REDIRECT, FEEDBACK)
source_pheromones="$E2E_TMP_DIR/.aether/data/pheromones.json"
cat > "$source_pheromones" << 'SRCEOF'
{
  "signals": [
    {
      "id": "sig_focus_src001",
      "type": "FOCUS",
      "content": "Source focus on API design",
      "strength": 0.85,
      "effective_strength": 0.85,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "source-colony"
    },
    {
      "id": "sig_redirect_src001",
      "type": "REDIRECT",
      "content": "Source redirect avoid global state",
      "strength": 0.95,
      "effective_strength": 0.95,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "source-colony"
    },
    {
      "id": "sig_feedback_src001",
      "type": "FEEDBACK",
      "content": "Source feedback prefer pure functions",
      "strength": 0.6,
      "effective_strength": 0.6,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "source-colony"
    }
  ],
  "midden": []
}
SRCEOF

# Step 3: Export source signals to XML
source_xml="$E2E_TMP_DIR/source-colony-signals.xml"
echo "  Exporting 3 source colony signals to XML..."
raw_src_export=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-export-xml "$source_xml")
src_export_out=$(extract_json "$raw_src_export")

if echo "$src_export_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: source export ok (3 signals written)"
else
  xmlcmd02_pass=false
  xmlcmd02_notes="$xmlcmd02_notes [FAIL: source export ok!=true: $src_export_out]"
  echo "  FAIL: source export did not return ok:true"
  echo "  Got: $src_export_out"
fi

# Step 4: Create target colony with 1 existing signal
cat > "$source_pheromones" << 'TGTEOF'
{
  "signals": [
    {
      "id": "sig_focus_target001",
      "type": "FOCUS",
      "content": "Target colony original signal",
      "strength": 0.7,
      "effective_strength": 0.7,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "target-colony"
    }
  ],
  "midden": []
}
TGTEOF

# Step 5: Import source XML into target with prefix "source-colony"
echo "  Importing source signals into target with prefix 'source-colony'..."
raw_tgt_import=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-import-xml "$source_xml" "source-colony" 2>&1 || true)
tgt_import_out=$(extract_json "$raw_tgt_import")

if echo "$tgt_import_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: import into target returned ok:true"
else
  xmlcmd02_pass=false
  xmlcmd02_notes="$xmlcmd02_notes [FAIL: target import ok!=true: $tgt_import_out]"
  echo "  FAIL: import into target did not return ok:true"
  echo "  Got: $tgt_import_out"
fi

# Step 6: Verify pheromone-read works on target
echo "  Running pheromone-read on target..."
raw_read=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-read 2>&1 || true)
read_out=$(extract_json "$raw_read")

if echo "$read_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: pheromone-read returned ok:true"
else
  xmlcmd02_pass=false
  xmlcmd02_notes="$xmlcmd02_notes [FAIL: pheromone-read ok!=true: $read_out]"
  echo "  FAIL: pheromone-read did not return ok:true"
  echo "  Got: $read_out"
fi

# Step 7: Assert target now has 4 signals (1 original + 3 imported)
target_signal_count=$(jq '.signals | length' "$source_pheromones" 2>/dev/null || echo "0")
if [[ "$target_signal_count" -eq 4 ]]; then
  echo "  PASS: target has 4 signals (1 original + 3 imported)"
else
  xmlcmd02_pass=false
  xmlcmd02_notes="$xmlcmd02_notes [FAIL: expected 4 signals, got $target_signal_count]"
  echo "  FAIL: expected 4 signals, got $target_signal_count"
fi

# Step 8: Assert all imported signals have active: true
inactive_count=$(jq '[.signals[] | select(.active != true)] | length' "$source_pheromones" 2>/dev/null || echo "0")
if [[ "$inactive_count" -eq 0 ]]; then
  echo "  PASS: all signals are active"
else
  xmlcmd02_pass=false
  xmlcmd02_notes="$xmlcmd02_notes [FAIL: $inactive_count signals not active]"
  echo "  FAIL: $inactive_count signal(s) are not active"
fi

# Step 9: Assert imported signal IDs contain "source-colony" prefix
imported_with_prefix=$(jq '[.signals[] | select(.id | contains("source-colony"))] | length' "$source_pheromones" 2>/dev/null || echo "0")
if [[ "$imported_with_prefix" -eq 3 ]]; then
  echo "  PASS: 3 imported signals have 'source-colony' prefix in IDs"
else
  xmlcmd02_pass=false
  xmlcmd02_notes="$xmlcmd02_notes [FAIL: expected 3 signals with prefix, got $imported_with_prefix]"
  echo "  FAIL: expected 3 signals with 'source-colony' prefix, got $imported_with_prefix"
fi

if [[ "$xmlcmd02_pass" == "true" ]]; then
  record_result "XMLCMD-02" "PASS" "cross-colony transfer: 3 signals exported, imported into target with 1 existing, 4 total, all active, prefix applied"
else
  record_result "XMLCMD-02" "FAIL" "$xmlcmd02_notes"
fi

# ============================================================================
# XMLCMD-03: Seal lifecycle exports standalone pheromones.xml
# Strategy: Set up env with signals, run pheromone-export-xml to exchange path
#           (simulating seal Step 6.5), verify file exists + well-formed,
#           import into fresh target and verify round-trip
# ============================================================================

echo ""
echo "--- XMLCMD-03: Seal lifecycle exports standalone pheromones.xml ---"

xmlcmd03_pass=true
xmlcmd03_notes=""

# Step 1: Set up env with active signals
cat > "$E2E_TMP_DIR/.aether/data/pheromones.json" << 'SEALEOF'
{
  "signals": [
    {
      "id": "sig_focus_seal001",
      "type": "FOCUS",
      "content": "Seal test focus signal",
      "strength": 0.8,
      "effective_strength": 0.8,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "seal-test"
    },
    {
      "id": "sig_redirect_seal001",
      "type": "REDIRECT",
      "content": "Seal test redirect signal",
      "strength": 0.9,
      "effective_strength": 0.9,
      "active": true,
      "created_at": "2026-02-18T00:00:00Z",
      "expires_at": "phase_end",
      "source": "seal-test"
    }
  ],
  "midden": []
}
SEALEOF

# Step 2: Run pheromone-export-xml to exchange path (simulating seal Step 6.5)
seal_export_path="$E2E_TMP_DIR/.aether/exchange/pheromones.xml"
echo "  Simulating seal Step 6.5: exporting to .aether/exchange/pheromones.xml..."
raw_seal_export=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-export-xml "$seal_export_path")
seal_export_out=$(extract_json "$raw_seal_export")

if echo "$seal_export_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: seal-style export returned ok:true"
else
  xmlcmd03_pass=false
  xmlcmd03_notes="$xmlcmd03_notes [FAIL: seal export ok!=true: $seal_export_out]"
  echo "  FAIL: seal-style export did not return ok:true"
  echo "  Got: $seal_export_out"
fi

# Step 3: Assert file exists at .aether/exchange/pheromones.xml
if [[ -f "$seal_export_path" ]]; then
  echo "  PASS: pheromones.xml exists at .aether/exchange/"
else
  xmlcmd03_pass=false
  xmlcmd03_notes="$xmlcmd03_notes [FAIL: pheromones.xml not created at exchange path]"
  echo "  FAIL: pheromones.xml not created at .aether/exchange/"
fi

# Step 4: Assert well-formed XML
if [[ -f "$seal_export_path" ]]; then
  if xmllint --noout "$seal_export_path" 2>/dev/null; then
    echo "  PASS: pheromones.xml is well-formed XML"
  else
    xmlcmd03_pass=false
    xmlcmd03_notes="$xmlcmd03_notes [FAIL: pheromones.xml not well-formed]"
    echo "  FAIL: pheromones.xml is not well-formed XML"
  fi
fi

# Step 5: Import standalone file into fresh pheromones.json with prefix "sealed-colony"
# Write a fresh pheromones.json (empty signals) to simulate a target colony
cat > "$E2E_TMP_DIR/.aether/data/pheromones.json" << 'FRESHEOF'
{
  "signals": [],
  "midden": []
}
FRESHEOF

echo "  Importing standalone pheromones.xml with prefix 'sealed-colony'..."
raw_seal_import=$(run_in_isolated_env "$E2E_TMP_DIR" pheromone-import-xml "$seal_export_path" "sealed-colony" 2>&1 || true)
seal_import_out=$(extract_json "$raw_seal_import")

if echo "$seal_import_out" | jq -e '.ok == true' >/dev/null 2>&1; then
  echo "  PASS: import of standalone pheromones.xml returned ok:true"
else
  xmlcmd03_pass=false
  xmlcmd03_notes="$xmlcmd03_notes [FAIL: import ok!=true: $seal_import_out]"
  echo "  FAIL: import of standalone pheromones.xml did not return ok:true"
  echo "  Got: $seal_import_out"
fi

# Step 6: Verify signal count matches (2 signals written to source, should import 2)
# Note: pheromone-export-xml returns {path, validated} not signal_count,
# so we count from the known source (2 signals written above)
seal_expected_count=2
seal_imported_count=$(jq '.signals | length' "$E2E_TMP_DIR/.aether/data/pheromones.json" 2>/dev/null || echo "0")

if [[ "$seal_imported_count" -eq "$seal_expected_count" ]]; then
  echo "  PASS: round-trip signal count matches ($seal_expected_count exported, $seal_imported_count imported)"
else
  xmlcmd03_pass=false
  xmlcmd03_notes="$xmlcmd03_notes [FAIL: count mismatch - expected $seal_expected_count, imported $seal_imported_count]"
  echo "  FAIL: signal count mismatch - expected $seal_expected_count, imported $seal_imported_count"
fi

if [[ "$xmlcmd03_pass" == "true" ]]; then
  record_result "XMLCMD-03" "PASS" "seal-style export creates importable pheromones.xml; round-trip signal count matches"
else
  record_result "XMLCMD-03" "FAIL" "$xmlcmd03_notes"
fi

# ============================================================================
# Output results
# ============================================================================

# Write external results file if requested (for master runner)
if [[ -n "$EXTERNAL_RESULTS_FILE" ]]; then
  while IFS='|' read -r req_id status notes; do
    echo "${req_id}=${status}" >> "$EXTERNAL_RESULTS_FILE"
  done < "$RESULTS_FILE"
fi

print_area_results "XML-Commands"
