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

# Scan tech stack -- detect languages, frameworks, and package managers
# Usage: _scan_tech_stack <repo_root>
# Returns: raw JSON via stdout (caller wraps in json_ok)
_scan_tech_stack() {
  local root="${1:-.}"
  local languages="[]" frameworks="[]" package_managers="[]"

  # Language detection via file presence
  [[ -f "$root/tsconfig.json" ]] && languages=$(echo "$languages" | jq '. + ["typescript"]')
  [[ -f "$root/package.json" ]] && languages=$(echo "$languages" | jq '. + ["javascript"]')
  [[ -f "$root/requirements.txt" || -f "$root/pyproject.toml" ]] && languages=$(echo "$languages" | jq '. + ["python"]')
  [[ -f "$root/go.mod" ]] && languages=$(echo "$languages" | jq '. + ["go"]')
  [[ -f "$root/Cargo.toml" ]] && languages=$(echo "$languages" | jq '. + ["rust"]')
  [[ -f "$root/Gemfile" ]] && languages=$(echo "$languages" | jq '. + ["ruby"]')
  [[ -f "$root/pom.xml" || -f "$root/build.gradle" ]] && languages=$(echo "$languages" | jq '. + ["java"]')

  # Framework detection via file presence and package.json deps
  if [[ -f "$root/next.config.js" || -f "$root/next.config.ts" || -f "$root/next.config.mjs" ]]; then
    frameworks=$(echo "$frameworks" | jq '. + ["nextjs"]')
  fi
  if [[ -f "$root/angular.json" ]]; then
    frameworks=$(echo "$frameworks" | jq '. + ["angular"]')
  fi
  if [[ -f "$root/vue.config.js" || -f "$root/vite.config.ts" || -f "$root/vite.config.js" ]]; then
    frameworks=$(echo "$frameworks" | jq '. + ["vue"]')
  fi

  # Framework detection from package.json dependencies (targeted jq, no full reads)
  if [[ -f "$root/package.json" ]]; then
    local pkg_deps
    pkg_deps=$(jq -r '[(.dependencies // {} | keys[]), (.devDependencies // {} | keys[])] | join("\n")' "$root/package.json" 2>/dev/null || true)

    if echo "$pkg_deps" | grep -qx 'react'; then
      frameworks=$(echo "$frameworks" | jq '. + ["react"]')
    fi
    if echo "$pkg_deps" | grep -qx 'express'; then
      frameworks=$(echo "$frameworks" | jq '. + ["express"]')
    fi
    if echo "$pkg_deps" | grep -qx 'fastify'; then
      frameworks=$(echo "$frameworks" | jq '. + ["fastify"]')
    fi
    if echo "$pkg_deps" | grep -qx 'svelte'; then
      frameworks=$(echo "$frameworks" | jq '. + ["svelte"]')
    fi
    if echo "$pkg_deps" | grep -qx 'nest'; then
      frameworks=$(echo "$frameworks" | jq '. + ["nestjs"]')
    fi
  fi

  # Package manager detection
  if [[ -f "$root/package.json" ]]; then
    if [[ -f "$root/pnpm-lock.yaml" ]]; then
      package_managers=$(echo "$package_managers" | jq '. + ["pnpm"]')
    elif [[ -f "$root/yarn.lock" ]]; then
      package_managers=$(echo "$package_managers" | jq '. + ["yarn"]')
    elif [[ -f "$root/package-lock.json" ]]; then
      package_managers=$(echo "$package_managers" | jq '. + ["npm"]')
    else
      package_managers=$(echo "$package_managers" | jq '. + ["npm"]')
    fi
  fi
  [[ -f "$root/go.mod" ]] && package_managers=$(echo "$package_managers" | jq '. + ["go-modules"]')
  [[ -f "$root/Cargo.toml" ]] && package_managers=$(echo "$package_managers" | jq '. + ["cargo"]')
  if [[ -f "$root/Gemfile" ]]; then
    package_managers=$(echo "$package_managers" | jq '. + ["bundler"]')
  fi
  [[ -f "$root/requirements.txt" ]] && package_managers=$(echo "$package_managers" | jq '. + ["pip"]')
  if [[ -f "$root/pyproject.toml" ]]; then
    package_managers=$(echo "$package_managers" | jq '. + ["poetry"]')
  fi

  jq -n \
    --argjson langs "$languages" \
    --argjson fwks "$frameworks" \
    --argjson pkg_mgrs "$package_managers" \
    '{languages: $langs, frameworks: $fwks, package_managers: $pkg_mgrs}'
}

# Scan directory structure -- measure repo surface
# Usage: _scan_directory_structure <repo_root>
# Returns: raw JSON via stdout
_scan_directory_structure() {
  local root="${1:-.}"
  local exclude_flags
  exclude_flags=$(_scan_find_exclude_flags)

  # Count files (cap depth at 5 for performance)
  local file_count
  file_count=$(find "$root" -maxdepth 5 -type f $exclude_flags 2>/dev/null | wc -l | tr -d ' ')

  # Calculate max directory depth
  local max_depth
  max_depth=$(find "$root" -type d $exclude_flags 2>/dev/null | awk -F/ '{print NF-2}' | sort -rn | head -1)
  [[ -z "$max_depth" || "$max_depth" == "0" ]] && max_depth=1

  # List top-level directories (exclude hidden dirs and excluded dirs)
  local top_dirs
  top_dirs=$(ls -1d "$root"/*/ 2>/dev/null | while read -r d; do
    local dirname
    dirname=$(basename "$d")
    # Skip hidden dirs
    [[ "$dirname" == .* ]] && continue
    # Skip excluded dirs
    local skip=false
    for excluded in "${_SCAN_EXCLUDE_DIRS[@]}"; do
      [[ "$dirname" == "$excluded" ]] && skip=true && break
    done
    [[ "$skip" == "true" ]] && continue
    echo "$dirname"
  done | jq -R . | jq -s .)

  jq -n \
    --argjson dirs "$top_dirs" \
    --argjson file_count "$file_count" \
    --argjson max_depth "$max_depth" \
    '{top_level_dirs: $dirs, file_count: $file_count, max_depth: $max_depth}'
}

# Scan git history -- summarize git repo state
# Usage: _scan_git_history <repo_root>
# Returns: raw JSON via stdout
_scan_git_history() {
  local root="${1:-.}"

  # Check for .git directory
  if [[ ! -d "$root/.git" ]]; then
    jq -n '{is_git_repo: false, commit_count: 0, recent_commits: []}'
    return
  fi

  # Count commits
  local commit_count
  commit_count=$(git -C "$root" rev-list --count HEAD 2>/dev/null || echo 0)

  # Get recent commits (oneline format)
  local recent_log
  recent_log=$(git -C "$root" log --oneline -n 10 2>/dev/null || echo "")

  # Parse recent commits into JSON array
  local recent_commits="[]"
  if [[ -n "$recent_log" ]]; then
    recent_commits=$(echo "$recent_log" | while read -r line; do
      local hash message
      hash=$(echo "$line" | awk '{print $1}')
      message=$(echo "$line" | cut -d' ' -f2-)
      jq -n --arg hash "$hash" --arg message "$message" '{hash: $hash, message: $message}'
    done | jq -s '.')
  fi

  jq -n \
    --argjson commit_count "$commit_count" \
    --argjson recent_commits "$recent_commits" \
    '{is_git_repo: true, commit_count: $commit_count, recent_commits: $recent_commits}'
}

# Scan survey status -- check territory survey freshness (SCAN-02)
# Usage: _scan_survey_status <repo_root>
# Returns: raw JSON via stdout
_scan_survey_status() {
  local root="${1:-.}"
  local survey_dir="$root/.aether/data/survey"
  local state_file="$root/.aether/data/COLONY_STATE.json"

  # Check if survey directory exists
  if [[ ! -d "$survey_dir" ]]; then
    jq -n '{has_survey: false, is_stale: false, suggestion: {action: "colonize", reason: "No territory survey found. Run /ant:colonize to map the codebase before planning."}}'
    return
  fi

  # Check survey completeness (7 required docs)
  local required="PROVISIONS.md TRAILS.md BLUEPRINT.md CHAMBERS.md DISCIPLINES.md SENTINEL-PROTOCOLS.md PATHOGENS.md"
  local missing=""
  for doc in $required; do
    [[ ! -f "$survey_dir/$doc" ]] && missing="$missing $doc"
  done

  if [[ -n "$missing" ]]; then
    local missing_json
    missing_json=$(echo "$missing" | jq -R 'split(" ") | map(select(length > 0))')
    jq -n \
      --argjson missing "$missing_json" \
      '{has_survey: true, is_stale: false, is_complete: false, missing: $missing, suggestion: {action: "colonize", reason: "Survey is incomplete (missing documents). Run /ant:colonize --force-resurvey to remap."}}'
    return
  fi

  # Check staleness from COLONY_STATE.json territory_surveyed field
  local surveyed_at=""
  if [[ -f "$state_file" ]]; then
    surveyed_at=$(jq -r '.territory_surveyed // empty' "$state_file" 2>/dev/null || echo "")
  fi

  if [[ -z "$surveyed_at" ]]; then
    # No timestamp in state -- fall back to file modification times
    local oldest_ts
    if [[ "$(uname)" == "Linux" ]]; then
      oldest_ts=$(find "$survey_dir" -name "*.md" -exec stat -c %Y {} \; 2>/dev/null | sort -n | head -1)
    else
      oldest_ts=$(find "$survey_dir" -name "*.md" -exec stat -f %m {} \; 2>/dev/null | sort -n | head -1)
    fi
    if [[ -n "$oldest_ts" ]]; then
      local now_ts
      now_ts=$(date +%s)
      local age_days=$(( (now_ts - oldest_ts) / 86400 ))
      if [[ "$age_days" -gt 7 ]]; then
        jq -n \
          --argjson age "$age_days" \
          '{has_survey: true, is_stale: true, age_days: $age, suggestion: {action: "colonize", reason: "Survey is \($age) days old. Run /ant:colonize --force-resurvey for fresh data."}}'
        return
      fi
    fi
  else
    # Parse ISO-8601 timestamp and compare
    local surveyed_epoch
    if [[ "$(uname)" == "Linux" ]]; then
      surveyed_epoch=$(date -d "$surveyed_at" "+%s" 2>/dev/null || echo 0)
    else
      surveyed_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$surveyed_at" "+%s" 2>/dev/null || echo 0)
    fi
    local now_epoch
    now_epoch=$(date +%s)
    local age_days=$(( (now_epoch - surveyed_epoch) / 86400 ))
    if [[ "$age_days" -gt 7 ]]; then
      jq -n \
        --argjson age "$age_days" \
        --arg surveyed_at "$surveyed_at" \
        '{has_survey: true, is_stale: true, age_days: $age, surveyed_at: $surveyed_at, suggestion: {action: "colonize", reason: "Survey is \($age) days old. Run /ant:colonize --force-resurvey for fresh data."}}'
      return
    fi
  fi

  # Survey is fresh and complete
  jq -n '{has_survey: true, is_stale: false, is_complete: true}'
}

# Scan prior colonies -- detect active colony state and archived colonies
# Usage: _scan_prior_colonies <repo_root>
# Returns: raw JSON via stdout
_scan_prior_colonies() {
  local root="${1:-.}"
  local chambers_dir="$root/.aether/chambers"
  local state_file="$root/.aether/data/COLONY_STATE.json"

  local colonies="[]"
  local has_active="false"
  local active_goal=""

  # Check for active colony
  if [[ -f "$state_file" ]]; then
    local goal state
    goal=$(jq -r '.goal // empty' "$state_file" 2>/dev/null || echo "")
    state=$(jq -r '.state // empty' "$state_file" 2>/dev/null || echo "")
    if [[ -n "$goal" && "$state" != "SEALED" ]]; then
      has_active="true"
      active_goal="$goal"
    fi
  fi

  # Check for archived colonies in chambers
  if [[ -d "$chambers_dir" ]]; then
    for chamber in "$chambers_dir"/*/; do
      [[ -d "$chamber" ]] || continue
      local chamber_name
      chamber_name=$(basename "$chamber")

      # Skip hidden dirs
      [[ "$chamber_name" == .* ]] && continue

      local chamber_state="$chamber/COLONY_STATE.json"
      [[ -f "$chamber_state" ]] || continue

      local chamber_goal chamber_date
      chamber_goal=$(jq -r '.goal // "unknown"' "$chamber_state" 2>/dev/null || echo "unknown")
      chamber_date=$(jq -r '.initialized_at // "unknown"' "$chamber_state" 2>/dev/null || echo "unknown")

      colonies=$(echo "$colonies" | jq \
        --arg name "$chamber_name" \
        --arg goal "$chamber_goal" \
        --arg date "$chamber_date" \
        '. + [{name: $name, goal: $goal, initialized_at: $date}]')
    done
  fi

  jq -n \
    --argjson has_active "$has_active" \
    --arg active_goal "$active_goal" \
    --argjson colonies "$colonies" \
    '{has_active_colony: $has_active, active_goal: $active_goal, archived_colonies: $colonies}'
}

# Scan complexity -- estimate repo complexity (SCAN-03)
# Usage: _scan_complexity <repo_root>
# Returns: raw JSON via stdout
_scan_complexity() {
  local root="${1:-.}"
  local exclude_flags
  exclude_flags=$(_scan_find_exclude_flags)

  # File count (excluding common directories)
  local file_count
  file_count=$(find "$root" -maxdepth 5 -type f $exclude_flags 2>/dev/null | wc -l | tr -d ' ')

  # Max directory depth
  local max_depth
  max_depth=$(find "$root" -type d $exclude_flags 2>/dev/null | awk -F/ '{print NF-2}' | sort -rn | head -1)
  [[ -z "$max_depth" || "$max_depth" == "0" ]] && max_depth=1

  # Dependency count from package manifests
  local dep_count=0
  if [[ -f "$root/package.json" ]]; then
    dep_count=$(jq '[.dependencies // {}, .devDependencies // {}] | add | keys | length' "$root/package.json" 2>/dev/null || echo 0)
  elif [[ -f "$root/Cargo.toml" ]]; then
    dep_count=$(grep -c '^\[' "$root/Cargo.toml" 2>/dev/null || echo 0)
  elif [[ -f "$root/go.mod" ]]; then
    dep_count=$(grep -c '^[a-z]' "$root/go.mod" 2>/dev/null || echo 0)
  fi

  # Classification thresholds
  local size="small"
  if [[ "$file_count" -gt 500 ]] || [[ "$max_depth" -gt 8 ]] || [[ "$dep_count" -gt 50 ]]; then
    size="large"
  elif [[ "$file_count" -gt 100 ]] || [[ "$max_depth" -gt 5 ]] || [[ "$dep_count" -gt 15 ]]; then
    size="medium"
  fi

  jq -n \
    --arg size "$size" \
    --argjson file_count "$file_count" \
    --argjson max_depth "$max_depth" \
    --argjson dep_count "$dep_count" \
    '{size: $size, metrics: {file_count: $file_count, max_directory_depth: $max_depth, dependency_count: $dep_count}}'
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

  # Run sub-scans (each returns raw JSON, caller wraps in json_ok)
  local tech_stack directory_structure git_history survey_status prior_colonies complexity

  tech_stack=$(_scan_tech_stack "$target_dir")
  directory_structure=$(_scan_directory_structure "$target_dir")
  git_history=$(_scan_git_history "$target_dir")
  survey_status=$(_scan_survey_status "$target_dir")
  prior_colonies=$(_scan_prior_colonies "$target_dir")
  complexity=$(_scan_complexity "$target_dir")

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
