# Phase 08: Slash Command Wiring - Research

**Researched:** 2026-04-04
**Domain:** CLI migration, YAML command generation, shell-to-Go binary wiring
**Confidence:** HIGH

## Summary

Phase 08 rewires all 45 Claude Code and 45 OpenCode slash commands to invoke the Go binary (`aether <cmd>`) instead of the shell dispatcher (`bash .aether/aether-utils.sh <cmd>`). The investigation reveals this is NOT a simple find-and-replace. There are 1,074 total occurrences across 142 files (YAML sources, generated .md files, and playbooks), but the critical complexity lies in API differences between shell and Go.

The Go binary has 238 registered commands covering 99% of the subcommands referenced in YAML files. Only 3 shell subcommands lack direct Go equivalents: `flag-create` (use `flag-add`), `version-check` (use `version-check-cached`), and `normalize-args` (hardcoded in the generator preamble). However, 42 commands use positional argument syntax in the YAML files while their Go equivalents require `--flag` syntax. Additionally, 4 Go commands (`flag-list`, `status`, `history`, `phase`) output pretty-printed tables instead of the JSON envelope that slash commands expect for `jq` processing.

**Primary recommendation:** Update the YAML source files to use `aether` with `--flag` syntax, add a `--json` output flag to the 4 table-rendering Go commands, update the `NORMALIZE_PREAMBLE` in `generate-commands.js`, then regenerate all .md files using the generator. The playbook files need the same treatment. Add a shell-fallback wrapper function for any remaining gaps.

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WIRE-01 | All 45 Claude slash commands invoke `aether <cmd>` instead of `bash .aether/aether-utils.sh <cmd>` | "Invocation Patterns" section: 202 occurrences across 42 files. YAML-to-.md generation pipeline documented. |
| WIRE-02 | All 45 OpenCode slash commands invoke `aether <cmd>` instead of `bash .aether/aether-utils.sh <cmd>` | Same YAML generator produces OpenCode files. 256 occurrences across 45 files. Generator preamble also needs updating. |
| WIRE-03 | Shell fallback with visible deprecation notice for commands without Go equivalents | "Gap Analysis" section: only 3 gaps. Shell fallback wrapper pattern documented. |

</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| cobra | v1.8+ | Go CLI framework | Already in use -- all 238 commands registered |
| js-yaml | npm | YAML parsing for command generator | Already in use in `bin/generate-commands.js` |
| go-pretty/v6 | v6 | Table rendering in Go | Already used by 4 display commands |
| jq | any | JSON extraction in slash commands | Already used extensively in YAML command bodies |

### Supporting
| Library/Tool | Version | Purpose | When to Use |
|-------------|---------|---------|-------------|
| generate-commands.js | local | YAML-to-.md command generation | Primary tool for this phase -- regenerate all .md files from YAML |
| npm test | any | Run generate-commands test suite | Verify generator changes don't break existing behavior |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| YAML + generator | Direct .md editing | Direct editing would bypass the generator, causing drift on next regeneration. YAML is the source of truth per CLAUDE.md. |
| --json flag per command | Global JSON output mode | Global mode would require cobra root flag plumbing. Per-command flag is simpler and matches cobra conventions. |

**Installation:** No new packages required. All tools are already in the project.

## Architecture Patterns

### Recommended Project Structure
```
.aether/commands/*.yaml          # Source of truth -- EDIT HERE
    |
    v  (bin/generate-commands.js)
.claude/commands/ant/*.md        # Generated Claude commands
.opencode/commands/ant/*.md      # Generated OpenCode commands
.aether/docs/command-playbooks/  # Manual edits (not generated)
cmd/*.go                         # Go binary command implementations
```

### Pattern 1: YAML Source of Truth
**What:** All slash command content lives in `.aether/commands/*.yaml`. The `generate-commands.js` tool produces both Claude and OpenCode .md files.
**When to use:** Always edit YAML, never edit generated .md files directly.

**Generator flow:**
```
.aether/commands/status.yaml
  |
  +---> .claude/commands/ant/status.md    (Claude: {{TOOL_PREFIX}} -> "Run using the Bash tool...")
  |
  +---> .opencode/commands/ant/status.md  (OpenCode: {{TOOL_PREFIX}} -> "Run:", adds normalize-args preamble)
```

**Template markers processed by generator:**
- `{{TOOL_PREFIX "desc"}}` -> Claude: `Run using the Bash tool with description "desc":` / OpenCode: `Run:`
- `{{ARGUMENTS}}` -> Claude: `$ARGUMENTS` / OpenCode: `$normalized_args`
- `{{#claude}}...{{/claude}}` -> Include only in Claude output
- `{{#opencode}}...{{/opencode}}` -> Include only in OpenCode output

### Pattern 2: Replacement Strategy
**What:** Replace `bash .aether/aether-utils.sh` with `aether` in YAML files, adjusting argument syntax.
**When to use:** For every invocation in YAML files.

**Transformation rules:**

| Shell Pattern | Go Pattern | Notes |
|---------------|------------|-------|
| `bash .aether/aether-utils.sh version-check-cached` | `aether version-check-cached` | No args -- direct replacement |
| `bash .aether/aether-utils.sh generate-progress-bar 3 6 20` | `aether generate-progress-bar --current 3 --total 6 --width 20` | Positional to flags |
| `bash .aether/aether-utils.sh pheromone-write FOCUS "text" --strength 0.8` | `aether pheromone-write --type FOCUS --content "text" --strength 0.8` | Mixed positional/flags to all-flags |
| `bash .aether/aether-utils.sh spawn-log "Queen" "builder" "name" "desc"` | `aether spawn-log --caste "builder" --depth 1` | Positional to flags |
| `bash .aether/aether-utils.sh flag-list \| jq '.result'` | `aether flag-list --json \| jq '.result'` | Needs --json flag addition |

### Pattern 3: Generator Preamble Update
**What:** The OpenCode preamble in `generate-commands.js` hardcodes `bash .aether/aether-utils.sh normalize-args`.
**When to use:** Must be updated as part of this phase.

**Current (line 24-30 of bin/generate-commands.js):**
```javascript
const NORMALIZE_PREAMBLE = `### Step -1: Normalize Arguments

Run: \`normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")\`

This ensures arguments work correctly in both Claude Code and OpenCode. Use \`$normalized_args\` throughout this command.

`;
```

**Options:**
1. Replace with `aether normalize-args "$@"` -- requires adding `normalize-args` command to Go binary
2. Remove the preamble entirely -- OpenCode commands would lose argument normalization
3. Keep shell fallback for normalize-args only -- acceptable transitional approach

### Pattern 4: Shell Fallback Wrapper
**What:** For the 3 commands without Go equivalents, fall back to shell with deprecation notice.
**When to use:** Commands referenced in YAML that have no Go binary equivalent.

**Fallback pattern:**
```bash
aether <cmd> 2>/dev/null || bash .aether/aether-utils.sh <cmd>
```

Or, for explicit deprecation notice:
```bash
echo "[DEPRECATED] Falling back to shell for <cmd>" >&2; bash .aether/aether-utils.sh <cmd>
```

### Anti-Patterns to Avoid
- **Editing generated .md files directly:** They will be overwritten on next `node bin/generate-commands.js`. Always edit YAML sources.
- **Removing `jq` extraction patterns prematurely:** Go outputs JSON envelopes, so `jq -r '.result'` is still needed for most commands. Do NOT assume Go output can be used raw.
- **Assuming Go output matches shell output:** Multiple commands have different JSON structures, different key casing, and different output formats. Verify each command individually.
- **Ignoring the playbook files:** The 11 playbook files have 271 occurrences of shell invocations that also need updating (covered in Phase 09, but should not break).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Command generation | Custom sed/awk to update .md files | `bin/generate-commands.js` | Generator handles template processing, provider-specific logic, and preamble injection |
| JSON output format | Custom JSON formatter per command | Existing `outputOK()` / `outputError()` helpers in Go | Standardized envelope: `{"ok":true,"result":...}` |
| Argument parsing | Manual positional arg handling | Cobra flags | Shell uses positional args, Go uses flags -- standard cobra pattern |
| Shell fallback | Custom dispatch logic | Simple `aether cmd || bash fallback` pattern | Only 3 commands need fallback |

**Key insight:** The YAML-to-.md generation pipeline is the single lever for this entire phase. Updating YAML sources and regenerating is the correct approach. Direct .md editing would create permanent drift.

## Gap Analysis

### Go Binary Coverage

| Metric | Count |
|--------|-------|
| Go binary registered commands | 238 |
| Unique shell subcommands in YAML files | 115 |
| Shell subcommands with Go equivalent | 112 (97%) |
| Shell subcommands WITHOUT Go equivalent | 3 (3%) |

### Commands Without Go Equivalents

| Shell Command | Go Alternative | Used In | Resolution |
|---------------|---------------|---------|------------|
| `normalize-args` | None | Generator preamble (all OpenCode commands) | Add to Go binary, or keep shell fallback in preamble |
| `flag-create` | `flag-add` | verify-castes.yaml (1 occurrence) | Replace with `flag-add` |
| `version-check` | `version-check-cached` | verify-castes.yaml, resume-colony.yaml (2 occurrences) | Replace with `version-check-cached` |

### Output Format Discrepancies (CRITICAL)

Four Go commands output pretty-printed tables instead of JSON, breaking `jq` pipelines in slash commands:

| Command | Current Go Output | Expected by Slash Commands | Resolution |
|---------|-------------------|---------------------------|------------|
| `flag-list` | Pretty table (go-pretty) | JSON `{"ok":true,"result":{"flags":[...]}}` | Add `--json` flag |
| `status` | Rendered ASCII dashboard | N/A (slash command builds its own dashboard from subcommands) | No change needed -- slash command doesn't call `aether status` directly |
| `history` | Pretty table | JSON `{"ok":true,"result":{"events":[...]}}` | Add `--json` flag |
| `phase` | Pretty table | JSON with phase details | Add `--json` flag |

**Note:** `status` is NOT a problem because the slash `/ant:status` command calls individual subcommands (pheromone-count, milestone-detect, etc.) to build its own dashboard -- it does NOT invoke `aether status` as a single call. So the pretty-table rendering of `aether status` does not affect slash command wiring.

### Argument Syntax Mismatches (42 commands)

The following commands use positional arguments in YAML but require `--flags` in Go:

```
activity-log, autofix-checkpoint, autofix-rollback, chamber-verify,
changelog-collect-plan-data, check-antipattern, colony-archive-xml,
error-add, error-flag-pattern, flag-acknowledge, flag-add,
flag-auto-resolve, flag-resolve, generate-ant-name,
generate-commit-message, generate-progress-bar, grave-add,
learning-promote-auto, memory-capture, midden-write, phase-insert,
pheromone-display, pheromone-export-xml, pheromone-import-xml,
pheromone-write, print-next-up, registry-add, registry-export-xml,
registry-import-xml, session-init, session-update, spawn-complete,
spawn-log, state-checkpoint, state-write, survey-load, swarm-cleanup,
swarm-findings-add, swarm-findings-init, swarm-solution-set,
wisdom-export-xml, wisdom-import-xml
```

Each requires individual mapping from positional to flag syntax.

## Common Pitfalls

### Pitfall 1: Go Binary Not in PATH
**What goes wrong:** Slash commands invoke `aether <cmd>` but `aether` is not installed or not in PATH.
**Why it happens:** The Go binary is installed at `/opt/homebrew/bin/aether` (currently v5.3.5) but other users/environments may not have it.
**How to avoid:** The YAML commands should use `aether` (bare command, relies on PATH). The npm postinstall script (Phase 11) will handle installation. For development, ensure `go install ./cmd/aether` puts it in `$GOPATH/bin`.
**Warning signs:** `aether: command not found` errors when running slash commands.

### Pitfall 2: jq Extraction Breaks on Go Output Format Differences
**What goes wrong:** Go binary outputs JSON with different key names or structure than shell, breaking `jq -r '.result.something'` pipelines.
**Why it happens:** Some Go commands were implemented independently and may use different JSON key names. For example, shell `pheromone-count` returns `{focus: N, redirect: N}` (lowercase) while Go returns `{FOCUS: N, REDIRECT: N}` (uppercase).
**How to avoid:** Verify jq extraction patterns work with Go output for every command that pipes through jq. Either fix Go output to match shell, or update jq patterns.
**Warning signs:** Slash commands showing "null" or empty values where data should appear.

### Pitfall 3: Generator Regeneration Overwrites Manual Edits
**What goes wrong:** Someone edits `.claude/commands/ant/*.md` files directly, then regeneration overwrites their changes.
**Why it happens:** The files have `<!-- Generated from .aether/commands/... - DO NOT EDIT DIRECTLY -->` headers but developers may miss them.
**How to avoid:** All edits go in `.aether/commands/*.yaml`. Regenerate with `node bin/generate-commands.js` after any YAML change.
**Warning signs:** The `--check` flag on `node bin/generate-commands.js --check` reports mismatches.

### Pitfall 4: Playbook Files Are NOT Generated
**What goes wrong:** Treating playbook files like generated .md files and trying to update them via the generator.
**Why it happens:** Playbooks are manually maintained files in `.aether/docs/command-playbooks/` -- they are NOT produced by the generator.
**How to avoid:** Playbook files are updated in Phase 09, not this phase. This phase only touches YAML sources and the generator.
**Warning signs:** Attempting to run generator to update playbooks (it won't).

### Pitfall 5: Missing --json Flag Causes Table Output in Pipelines
**What goes wrong:** Commands like `flag-list` output a pretty table when piped through `jq`, causing parse errors.
**Why it happens:** Go commands that use `go-pretty/table` always render tables regardless of stdout being a pipe.
**How to avoid:** Add `--json` flag to the affected Go commands (`flag-list`, `history`, `phase`). Update YAML invocations to include `--json` when the output is piped to `jq`.
**Warning signs:** `jq: error: Invalid numeric literal` when processing command output.

## Code Examples

### YAML Command Invocation (Current)

```yaml
# From .aether/commands/status.yaml - current shell invocation
{{TOOL_PREFIX "Detecting colony milestone..."}} `bash .aether/aether-utils.sh milestone-detect`
```

### YAML Command Invocation (After Wiring)

```yaml
# After wiring - Go binary invocation
{{TOOL_PREFIX "Detecting colony milestone..."}} `aether milestone-detect`
```

### YAML Positional-to-Flags Conversion

```yaml
# Before (shell positional args):
bash .aether/aether-utils.sh generate-progress-bar "$current_phase" "$total_phases" 20

# After (Go flags):
aether generate-progress-bar --current "$current_phase" --total "$total_phases" --width 20
```

### YAML jq Extraction Pattern (Unchanged)

```yaml
# jq patterns work the same way since Go also outputs JSON envelope
depth_result=$(aether colony-depth get 2>/dev/null || echo '{"ok":true,"result":{"depth":"standard","source":"default"}}')
colony_depth=$(echo "$depth_result" | jq -r '.result.depth // "standard"')
```

### Generator Preamble Update (bin/generate-commands.js)

```javascript
// Option A: Use Go binary (requires normalize-args command in Go)
const NORMALIZE_PREAMBLE = `### Step -1: Normalize Arguments

Run: \`normalized_args=$(aether normalize-args "$@")\`

This ensures arguments work correctly in both Claude Code and OpenCode. Use \`$normalized_args\` throughout this command.

`;

// Option B: Shell fallback (transitional)
const NORMALIZE_PREAMBLE = `### Step -1: Normalize Arguments

Run: \`normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")\`

This ensures arguments work correctly in both Claude Code and OpenCode. Use \`$normalized_args\` throughout this command.

`;
```

### Go --json Flag Addition Pattern

```go
// Pattern for adding --json flag to table-rendering commands
var flagListJSON bool

var flagsCmd = &cobra.Command{
    Use:   "flag-list",
    Short: "List all flags",
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... load and filter flags ...

        if flagListJSON {
            outputOK(map[string]interface{}{
                "flags": filtered,
            })
            return nil
        }

        renderFlagsTable(filtered)
        return nil
    },
}

func init() {
    flagsCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output as JSON")
    // ... other flags ...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Direct .md file editing | YAML source + generator | Phase 13 of v1.x | YAML is now canonical source |
| Shell dispatcher only | Shell + Go binary side by side | v5.4 milestone | Go has 238 commands, shell has ~305 |
| `bash .aether/aether-utils.sh` | `aether` (this phase) | v6.0 target | All user-facing invocations go through Go |

**Deprecated/outdated:**
- Direct editing of `.claude/commands/ant/*.md` files: Use YAML + generator instead
- `flag-create` command: Use `flag-add` instead
- `version-check` command: Use `version-check-cached` instead

## Open Questions

1. **normalize-args in Go binary or keep shell?**
   - What we know: `normalize-args` is called in the OpenCode preamble for all 45 OpenCode commands. It has no Go equivalent.
   - What's unclear: Whether the normalization logic is complex enough to warrant a Go implementation.
   - Recommendation: Check the shell implementation complexity. If simple, add to Go. If complex, keep shell fallback in preamble as a transitional measure.

2. **Should Go commands accept positional args too?**
   - What we know: 42 YAML invocations use positional args but Go commands use flags.
   - What's unclear: Whether modifying Go commands to accept both positional and flag args would reduce YAML changes.
   - Recommendation: Update YAML files to use flags. Adding positional arg parsing to Go commands would be a larger change and against cobra conventions.

3. **pheromone-count key casing mismatch**
   - What we know: Shell returns lowercase keys (focus, redirect, feedback). Go returns uppercase (FOCUS, REDIRECT, FEEDBACK).
   - What's unclear: Which jq patterns in YAML files depend on lowercase.
   - Recommendation: Standardize on one casing. Prefer the Go uppercase since it matches the signal type constants. Update any jq patterns that depend on lowercase.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go binary (aether) | All slash commands | Yes | 5.3.5 (installed), local build available | Build from source |
| node | generate-commands.js | Yes | system | -- |
| jq | JSON extraction in slash commands | Yes | system | -- |
| js-yaml | YAML parsing | Yes | npm | -- |
| go-pretty/v6 | Table rendering | Yes | v6 | -- |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + npm/Jest |
| Config file | go.mod (Go), package.json (npm) |
| Quick run command | `node bin/generate-commands.js --check` |
| Full suite command | `go test ./cmd/... && npm test` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WIRE-01 | Claude commands call `aether` not `bash .aether/aether-utils.sh` | unit | `grep -c 'bash .aether/aether-utils.sh' .claude/commands/ant/*.md` (expect 0) | Yes |
| WIRE-02 | OpenCode commands call `aether` not `bash .aether/aether-utils.sh` | unit | `grep -c 'bash .aether/aether-utils.sh' .opencode/commands/ant/*.md` (expect 0) | Yes |
| WIRE-03 | Shell fallback only for commands without Go equivalents | unit | `grep 'bash .aether/aether-utils.sh' .claude/commands/ant/*.md .opencode/commands/ant/*.md` (should only match normalize-args or fallback wrapper) | Yes |
| WIRE-01 | Generated .md files match YAML sources | unit | `node bin/generate-commands.js --check` (exit 0) | Yes |
| WIRE-01 | All 45 Claude commands exist | smoke | `ls .claude/commands/ant/*.md \| wc -l` (expect 45) | Yes |
| WIRE-02 | All 45 OpenCode commands exist | smoke | `ls .opencode/commands/ant/*.md \| wc -l` (expect 45) | Yes |
| WIRE-03 | flag-list --json produces valid JSON | unit | `aether flag-list --json \| jq .ok` (expect true) | Wave 0 |
| WIRE-03 | history --json produces valid JSON | unit | `aether history --json \| jq .ok` (expect true) | Wave 0 |
| WIRE-03 | phase --json produces valid JSON | unit | `aether phase --json \| jq .ok` (expect true) | Wave 0 |

### Sampling Rate
- **Per task commit:** `node bin/generate-commands.js --check && grep -c 'bash .aether/aether-utils.sh' .claude/commands/ant/*.md .opencode/commands/ant/*.md`
- **Per wave merge:** `go test ./cmd/... && npm test`
- **Phase gate:** Zero occurrences of `bash .aether/aether-utils.sh` in generated .md files. All Go tests pass. Generator check passes.

### Validation Strategy for Output Parity

The key validation challenge is ensuring `aether <cmd>` produces output that existing `jq` extraction patterns can still parse. The approach:

1. **Static analysis:** Grep all `jq` patterns from YAML files that reference command output
2. **For each command referenced:** Run both `aether <cmd>` and the shell version, compare JSON structure
3. **Automated parity test:** Build a script that runs key commands through both paths and diffs the `.result` keys

```bash
# Example parity test pattern
go_output=$(aether pheromone-count 2>/dev/null)
shell_output=$(bash .aether/aether-utils.sh pheromone-count 2>/dev/null)
echo "Go:    $go_output"
echo "Shell: $shell_output"
# Check if jq extraction patterns work on both
echo "Go jq:    $(echo "$go_output" | jq -r '.result.FOCUS // .result.focus // "FAIL"')"
echo "Shell jq: $(echo "$shell_output" | jq -r '.result.focus // "FAIL"')"
```

### Wave 0 Gaps
- [ ] `cmd/flags.go` -- add `--json` flag to `flag-list` command
- [ ] `cmd/history.go` -- add `--json` flag to `history` command
- [ ] `cmd/phase.go` -- add `--json` flag to `phase` command
- [ ] Parity test script -- create `tests/bash/test-shell-go-parity.sh` or similar

*(If no gaps: existing test infrastructure covers all phase requirements)*

## Sources

### Primary (HIGH confidence)
- Source code analysis of `cmd/*.go` -- verified Go command API signatures
- Source code analysis of `bin/generate-commands.js` -- verified YAML-to-.md generation pipeline
- Source code analysis of `.aether/commands/*.yaml` -- verified 45 YAML source files and 345 shell invocations
- Binary testing -- built and ran Go binary, verified output formats against shell equivalents

### Secondary (MEDIUM confidence)
- `.planning/ROADMAP.md` -- verified phase dependencies and success criteria
- `.planning/REQUIREMENTS.md` -- verified MIGRATE-01 through MIGRATE-05 traceability

### Tertiary (LOW confidence)
- N/A -- all findings verified against source code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all tools already in use in the project
- Architecture: HIGH -- YAML generator pipeline fully analyzed, transformation rules verified
- Pitfalls: HIGH -- discovered through direct testing of Go vs shell output differences
- Gap analysis: HIGH -- exhaustive grep of all YAML files against Go binary command list

**Research date:** 2026-04-04
**Valid until:** 2026-05-04 (stable -- no fast-moving dependencies)
