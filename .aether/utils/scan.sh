#!/usr/bin/env bash
# Scan utility -- repo scanning for smart init research data
# Provides: _scan_init_research, _scan_tech_stack, _scan_directory_structure,
#           _scan_git_history, _scan_survey_status, _scan_prior_colonies, _scan_complexity
#
# These functions are sourced by aether-utils.sh at startup.
# All shared infrastructure (json_ok, json_err, DATA_DIR, SCRIPT_DIR) is available.

# Directories to exclude from scanning
_SCAN_EXCLUDE_DIRS=(
  node_modules
  .git
  .aether
  dist
  build
  __pycache__
  .next
  target
  vendor
  .venv
  venv
  coverage
)

# Build -not -path flags from _SCAN_EXCLUDE_DIRS for use with find
_scan_find_exclude_flags() {
  local flags=""
  for dir in "${_SCAN_EXCLUDE_DIRS[@]}"; do
    flags+=" -not -path '*/${dir}/*'"
  done
  printf '%s' "$flags"
}

# Scan tech stack -- returns languages, frameworks, package managers
# STUB: Plan 29-02 will implement real scanning logic
_scan_tech_stack() {
  json_ok '{"languages":[],"frameworks":[],"package_managers":[]}'
}

# Scan directory structure -- returns top-level dirs, file count, max depth
# STUB: Plan 29-02 will implement real scanning logic
_scan_directory_structure() {
  json_ok '{"top_level_dirs":[],"file_count":0,"max_depth":0}'
}

# Scan git history -- returns repo status, commit count, recent commits
# STUB: Plan 29-02 will implement real scanning logic
_scan_git_history() {
  json_ok '{"is_git_repo":false,"commit_count":0,"recent_commits":[]}'
}

# Scan survey status -- returns whether a territory survey exists and is stale
# STUB: Plan 29-02 will implement real scanning logic
_scan_survey_status() {
  json_ok '{"has_survey":false,"is_stale":false}'
}

# Scan prior colonies -- returns active colony state and archived colonies
# STUB: Plan 29-02 will implement real scanning logic
_scan_prior_colonies() {
  json_ok '{"has_active_colony":false,"active_goal":"","archived_colonies":[]}'
}

# Scan complexity -- returns size classification and metrics
# STUB: Plan 29-02 will implement real scanning logic
_scan_complexity() {
  json_ok '{"size":"small","metrics":{"file_count":0,"max_directory_depth":0,"dependency_count":0}}'
}

# Main entry point: scan repo and produce structured research JSON
# Usage: _scan_init_research [--target <dir>]
# Options:
#   --target <dir>   Directory to scan (default: $AETHER_ROOT or current dir)
_scan_init_research() {
  local target_dir=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --target)
        target_dir="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done

  # Default target
  target_dir="${target_dir:-${AETHER_ROOT:-.}}"

  # Validate target exists
  if [[ ! -d "$target_dir" ]]; then
    json_err "$E_FILE_NOT_FOUND" "Target directory does not exist: $target_dir"
    return 1
  fi

  # Run sub-scans (stubs for now)
  local tech_stack directory_structure git_history survey_status prior_colonies complexity

  tech_stack=$(_scan_tech_stack | jq -r '.result')
  directory_structure=$(_scan_directory_structure | jq -r '.result')
  git_history=$(_scan_git_history | jq -r '.result')
  survey_status=$(_scan_survey_status | jq -r '.result')
  prior_colonies=$(_scan_prior_colonies | jq -r '.result')
  complexity=$(_scan_complexity | jq -r '.result')

  # Assemble final output via jq
  local result
  result=$(jq -n \
    --argjson tech_stack "$tech_stack" \
    --argjson directory_structure "$directory_structure" \
    --argjson git_history "$git_history" \
    --argjson survey_status "$survey_status" \
    --argjson prior_colonies "$prior_colonies" \
    --argjson complexity "$complexity" \
    '{
      schema_version: 1,
      tech_stack: $tech_stack,
      directory_structure: $directory_structure,
      git_history: $git_history,
      survey_status: $survey_status,
      prior_colonies: $prior_colonies,
      complexity: $complexity,
      scanned_at: (now | todate)
    }')

  json_ok "$result"
}
