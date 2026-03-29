#!/bin/bash
# generate-commands.sh - Validate YAML source -> generated command file sync
#
# This script validates that generated .md files (.claude/commands/ant/*.md and
# .opencode/commands/ant/*.md) match the output from YAML sources (.aether/commands/*.yaml).
#
# Usage:
#   ./bin/generate-commands.sh [check|diff|help]
#
# Commands:
#   check  - Validate generated files match YAML sources
#   diff   - Show differences between command sets
#   help   - Show this help message

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

CLAUDE_DIR="$PROJECT_DIR/.claude/commands/ant"
OPENCODE_DIR="$PROJECT_DIR/.opencode/commands/ant"
CLAUDE_AGENT_DIR="$PROJECT_DIR/.claude/agents/ant"
OPENCODE_AGENT_DIR="$PROJECT_DIR/.opencode/agents"
AETHER_AGENT_MIRROR_DIR="$PROJECT_DIR/.aether/agents-claude"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Compute SHA hash with error handling
# Returns 0 on success, 1 on failure
# Echoes hash on success, error message on failure
compute_hash() {
    local file="$1"

    if [[ ! -r "$file" ]]; then
        echo "NOT_READABLE"
        return 1
    fi

    local hash
    hash=$(shasum "$file" 2>/dev/null | cut -d' ' -f1)
    if [[ -z "$hash" ]]; then
        echo "HASH_FAILED"
        return 1
    fi

    echo "$hash"
    return 0
}

# Count commands in each directory
count_commands() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        find "$dir" -name "*.md" | wc -l | tr -d ' '
    else
        echo "0"
    fi
}

# List command files (PLAN-006 fix #13 - warn about non-.md files)
list_commands() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        # Check for non-.md files and warn
        local non_md_count
        non_md_count=$(find "$dir" -type f ! -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
        if [[ "$non_md_count" -gt 0 ]]; then
            log_warn "$non_md_count non-.md file(s) found in $dir (ignored)"
        fi

        find "$dir" -name "*.md" -exec basename {} \; | sort
    fi
}

# List agent definition files (*.md) by basename
list_agents() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        find "$dir" -name "*.md" -type f -exec basename {} \; | sort
    fi
}

# Check if directories are in sync (by file count and names)
check_sync() {
    log_info "Checking command sync status..."

    local claude_count=$(count_commands "$CLAUDE_DIR")
    local opencode_count=$(count_commands "$OPENCODE_DIR")

    echo "Claude Code commands: $claude_count"
    echo "OpenCode commands:    $opencode_count"

    # PLAN-006 fix #10 - warn about empty directories
    if [[ "$claude_count" -eq 0 ]] && [[ "$opencode_count" -eq 0 ]]; then
        log_warn "Both command directories are empty"
        echo "This may indicate a misconfiguration"
    fi

    # PLAN-006 fix #11 - warn about large command counts
    local max_commands=500
    if [[ "$claude_count" -gt "$max_commands" ]] || [[ "$opencode_count" -gt "$max_commands" ]]; then
        log_warn "Large number of commands ($claude_count/$opencode_count)"
        echo "This may cause performance issues during sync checks"
    fi

    if [[ "$claude_count" != "$opencode_count" ]]; then
        log_error "Command counts don't match!"
        return 1
    fi

    # Check file names match
    local claude_files=$(list_commands "$CLAUDE_DIR")
    local opencode_files=$(list_commands "$OPENCODE_DIR")

    if [[ "$claude_files" != "$opencode_files" ]]; then
        log_error "Command file names don't match!"
        echo ""
        echo "Only in Claude Code:"
        comm -23 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        echo ""
        echo "Only in OpenCode:"
        comm -13 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        return 1
    fi

    log_info "Commands are in sync ($claude_count commands)"
    return 0
}

# Check YAML generation sync (Pass 2)
# Validates that generated .md files match YAML sources
check_yaml_generation() {
    log_info "Checking YAML generation sync..."

    # Verify YAML source directory exists
    local yaml_dir="$PROJECT_DIR/.aether/commands"
    if [[ ! -d "$yaml_dir" ]]; then
        log_error "YAML source directory not found: $yaml_dir"
        return 1
    fi

    local yaml_count
    yaml_count=$(find "$yaml_dir" -name "*.yaml" | wc -l | tr -d ' ')
    if [[ "$yaml_count" -eq 0 ]]; then
        log_error "No YAML source files found in $yaml_dir"
        return 1
    fi

    log_info "Found $yaml_count YAML source files"

    # Run generator in check mode
    if node "$PROJECT_DIR/bin/generate-commands.js" --check; then
        log_info "Generated files match YAML sources ($yaml_count commands)"
        return 0
    else
        log_error "Generated files are out of date. Run 'npm run generate' to update."
        return 1
    fi
}

# Check agent sync strategy:
# 1) Claude <-> OpenCode: structural parity (count + file names)
# 2) Claude <-> .aether mirror: exact parity (count + names + content hash)
check_agent_sync() {
    log_info "Checking agent sync status..."

    local claude_count
    local opencode_count
    local mirror_count
    claude_count=$(count_commands "$CLAUDE_AGENT_DIR")
    opencode_count=$(count_commands "$OPENCODE_AGENT_DIR")
    mirror_count=$(count_commands "$AETHER_AGENT_MIRROR_DIR")

    echo "Claude agents:        $claude_count"
    echo "OpenCode agents:      $opencode_count"
    echo "Aether mirror agents: $mirror_count"

    if [[ "$claude_count" != "$opencode_count" ]]; then
        log_error "Claude/OpenCode agent counts don't match!"
        return 1
    fi

    if [[ "$claude_count" != "$mirror_count" ]]; then
        log_error "Claude/.aether mirror agent counts don't match!"
        return 1
    fi

    local claude_files
    local opencode_files
    local mirror_files
    claude_files=$(list_agents "$CLAUDE_AGENT_DIR")
    opencode_files=$(list_agents "$OPENCODE_AGENT_DIR")
    mirror_files=$(list_agents "$AETHER_AGENT_MIRROR_DIR")

    if [[ "$claude_files" != "$opencode_files" ]]; then
        log_error "Claude/OpenCode agent file names don't match!"
        echo ""
        echo "Only in Claude:"
        comm -23 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        echo ""
        echo "Only in OpenCode:"
        comm -13 <(echo "$claude_files") <(echo "$opencode_files") | sed 's/^/  /'
        return 1
    fi

    if [[ "$claude_files" != "$mirror_files" ]]; then
        log_error "Claude/.aether mirror agent file names don't match!"
        echo ""
        echo "Only in Claude:"
        comm -23 <(echo "$claude_files") <(echo "$mirror_files") | sed 's/^/  /'
        echo ""
        echo "Only in .aether mirror:"
        comm -13 <(echo "$claude_files") <(echo "$mirror_files") | sed 's/^/  /'
        return 1
    fi

    # Claude and mirror should be byte-identical.
    local file
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        local claude_file="$CLAUDE_AGENT_DIR/$file"
        local mirror_file="$AETHER_AGENT_MIRROR_DIR/$file"
        local claude_hash
        local mirror_hash
        claude_hash=$(compute_hash "$claude_file")
        mirror_hash=$(compute_hash "$mirror_file")
        if [[ "$claude_hash" != "$mirror_hash" ]]; then
            log_error "Claude/.aether mirror content drift: $file"
            return 1
        fi
    done <<< "$claude_files"

    log_info "Agents are in sync (Claude/OpenCode structural parity, Claude/.aether mirror exact parity)"
    return 0
}

# Show diff between command sets
show_diff() {
    log_info "Comparing command sets..."

    local yaml_dir="$PROJECT_DIR/.aether/commands"
    local yaml_count=0
    if [[ -d "$yaml_dir" ]]; then
        yaml_count=$(find "$yaml_dir" -name "*.yaml" | wc -l | tr -d ' ')
    fi
    echo "YAML source files: $yaml_count"

    # Use null delimiter for safe iteration (handles filenames with spaces)
    while IFS= read -r -d '' claude_file; do
        local file
        file=$(basename "$claude_file")
        local opencode_file="$OPENCODE_DIR/$file"

        if [[ ! -f "$opencode_file" ]]; then
            log_warn "$file exists only in Claude Code"
            continue
        fi

        # Compare file sizes as a quick check
        local claude_size=$(wc -l < "$claude_file" | tr -d ' ')
        local opencode_size=$(wc -l < "$opencode_file" | tr -d ' ')

        if [[ "$claude_size" != "$opencode_size" ]]; then
            echo "$file: $claude_size lines (Claude) vs $opencode_size lines (OpenCode)"
        fi
    done < <(find "$CLAUDE_DIR" -name "*.md" -type f -print0 2>/dev/null | sort -z)
}

# Display help
show_help() {
    echo "Aether YAML Generation Validation Tool"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  check   Validate generated .md files match YAML sources"
    echo "  diff    Show differences between command sets"
    echo "  help    Show this help message"
    echo ""
    echo "Directories:"
    echo "  YAML sources:    $PROJECT_DIR/.aether/commands"
    echo "  Claude output:   $CLAUDE_DIR"
    echo "  OpenCode output: $OPENCODE_DIR"
    echo "  Claude agents:   $CLAUDE_AGENT_DIR"
    echo "  OpenCode agents: $OPENCODE_AGENT_DIR"
    echo "  Aether mirror:   $AETHER_AGENT_MIRROR_DIR"
    echo ""
    echo "Note: Command specs are maintained in .aether/commands/*.yaml."
    echo "Run 'npm run generate' to regenerate .md files from YAML sources."
    echo "Use this tool to verify generated files are up to date."
}

# Main
case "${1:-check}" in
    check)
        # Pass 1: file count + name check
        check_sync
        # Pass 2: YAML generation check
        check_yaml_generation
        # Pass 3: agent sync policy checks
        check_agent_sync
        ;;
    diff)
        show_diff
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
