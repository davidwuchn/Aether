# Phase 8: Documentation Update - Research

**Researched:** 2026-03-19
**Domain:** Documentation accuracy and alignment with verified system behavior
**Confidence:** HIGH

## Summary

Phase 8 is a documentation-only phase. No code changes are required -- every file touched is a `.md` file. The goal is to ensure all documentation accurately describes verified, working behavior after the changes made in Phases 1-7. Through exhaustive investigation of the codebase, I have catalogued every specific inaccuracy, stale reference, and aspirational claim across the four documentation targets: CLAUDE.md, pheromones.md, known-issues.md, and README.md (plus related docs).

The key finding is that documentation drift is concentrated in four categories: (1) stale numeric counts (commands, lines, tests, subcommands), (2) the pheromone model description still implying workers independently read signals when in reality colony-prime injects context into worker prompts, (3) known-issues.md containing 15+ resolved/FIXED entries that should be removed, and (4) missing documentation for new commands added in Phases 2, 6, and 7 (data-clean, export-signals, import-signals, etc.).

**Primary recommendation:** Treat each DOCS requirement as an independent plan -- each has a clear target file set and a well-defined "done" state that can be verified with grep.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DOCS-01 | CLAUDE.md updated to match current reality (no references to eliminated features like runtime/) | Section "CLAUDE.md Inaccuracies" below provides exact line-by-line list of stale counts, missing commands, and outdated references |
| DOCS-02 | Pheromone documentation accurately describes injection model (colony-prime injects context, workers don't independently read signals) | Section "Pheromone Documentation Inaccuracies" provides exact lines in pheromones.md and CLAUDE.md where the model is described incorrectly |
| DOCS-03 | known-issues.md updated (stale FIXED statuses corrected, resolved items removed) | Section "known-issues.md Cleanup" provides complete list of 15 FIXED entries to remove and 8 open entries to retain |
| DOCS-04 | README and user-facing docs reflect verified behavior, not aspirational features | Section "README.md Inaccuracies" provides exact stale claims with correct values |
</phase_requirements>

## Standard Stack

This phase has no library dependencies. It is pure documentation editing.

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| grep/ripgrep | any | Verify no stale references remain after edits | Fastest way to confirm all instances of a term are removed |
| diff | any | Verify changes are correct | Standard review tool |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| wc -l | any | Verify line counts in aether-utils.sh | When updating line count claims |
| ls \| wc -l | any | Count commands/agents | When updating count claims |
| npm test | any | Get current test count | When updating test count claims |

## Architecture Patterns

### Pattern 1: Evidence-Based Documentation Update
**What:** Every documentation change is backed by a verified fact from the codebase
**When to use:** Always, for this entire phase

**Process:**
1. Identify the inaccurate claim (from research findings below)
2. Determine the correct value by running the relevant command
3. Update the documentation
4. Verify with grep that no other instances of the stale value remain

### Pattern 2: Surgical Removal for known-issues.md
**What:** Remove entire FIXED entries (header + body), preserve open entries intact
**When to use:** DOCS-03

**Process:**
1. Identify all entries with "FIXED" in their heading or status
2. Remove the entire entry block (from `###` heading to next `###` heading or `---` separator)
3. Update the Workarounds Summary table at bottom
4. Verify remaining entries are all genuinely open

### Anti-Patterns to Avoid
- **Aspirational documentation:** Never write "Workers read signals" if the actual mechanism is "colony-prime injects signals into worker prompts"
- **Approximate counts without verification:** Never write "~9,808 lines" without running `wc -l` first
- **Leaving FIXED entries in known-issues.md:** The success criterion requires resolved items be REMOVED, not just marked

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Finding all stale references | Manual reading | grep patterns from findings below | grep is comprehensive, manual review misses instances |
| Verifying counts | Remembering old values | Run `wc -l`, `ls \| wc -l`, `npm test` | Source of truth is the codebase, not memory |

## Common Pitfalls

### Pitfall 1: Missing secondary references
**What goes wrong:** You update CLAUDE.md but miss the same stale value in README.md or aether-colony.md
**Why it happens:** The same fact (e.g., "36 slash commands") appears in multiple files
**How to avoid:** After updating a value, grep the entire repo for the old value
**Warning signs:** grep still finds the old value after your edit

### Pitfall 2: Changing the pheromone wording but keeping the mental model
**What goes wrong:** You change "Workers read signals" to "Workers receive signals" but this still implies independent action
**Why it happens:** Habit of describing the old model
**How to avoid:** Always use the injection framing: "colony-prime injects signals into worker prompts" or "signals are injected by the Queen via colony-prime"
**Warning signs:** Any sentence where "workers" is the subject of an active verb related to signal consumption

### Pitfall 3: Not removing the FIXED entries entirely
**What goes wrong:** You mark entries as FIXED but leave them in the file
**Why it happens:** Seems like historical value to keep them
**How to avoid:** Success criteria explicitly says "resolved items have been removed"
**Warning signs:** grep for "FIXED" still returns results in known-issues.md after cleanup

### Pitfall 4: Updating counts that will change again
**What goes wrong:** You hardcode "537 tests" but a future phase adds more tests
**Why it happens:** Desire for precision
**How to avoid:** Use floor values with "+" suffix (e.g., "530+ passing") for things that grow. Only use exact counts for things that are stable (command count, agent count)
**Warning signs:** N/A -- this is a judgment call

### Pitfall 5: Forgetting the aether-colony.md rules file
**What goes wrong:** CLAUDE.md gets updated but the rules file distributed to all repos still has stale information
**Why it happens:** aether-colony.md is a separate file in .claude/rules/ that duplicates some CLAUDE.md content
**How to avoid:** Check aether-colony.md for the same stale claims. Currently it lists fewer commands than actually exist
**Warning signs:** grep the rules directory too

## Code Examples

Not applicable -- this phase is documentation editing, not code.

## Detailed Findings

### CLAUDE.md Inaccuracies (DOCS-01)

**Stale numeric counts:**

| Claim | Location | Actual Value | Source |
|-------|----------|-------------|--------|
| "~9,808 lines, 150 subcommands" (Quick Reference table) | Line 16 | 10,499 lines, 110 subcommands | `wc -l .aether/aether-utils.sh` = 10499; `help` command returns 110 commands |
| "9,808 lines, 150 subcommands" (Architecture diagram) | Line 30 | 10,499 lines, 110 subcommands | Same source |
| "150 subcommands" (Key Directories section) | Line 94 | 110 subcommands | `help` JSON output `.commands \| length` = 110 |
| "Tests \| 490+ passing" | Line 17 | 537 passing | `npm test` output |
| "36 slash commands (Claude + OpenCode)" | Line 14 | 40 Claude + 39 OpenCode | `ls .claude/commands/ant/ \| wc -l` = 40; `ls .opencode/commands/ant/ \| wc -l` = 39 |
| "36 slash commands" (Architecture diagram, lines 38, 40) | Lines 38, 40 | 40 Claude, 39 OpenCode | Same source |
| "36 slash commands" (.claude/ section, line 118) | Line 118 | 40 | Same source |

**Missing new commands from Phases 2, 6:**
- `/ant:data-clean` (Phase 2) -- not mentioned anywhere in CLAUDE.md
- `/ant:export-signals` (Phase 6) -- not mentioned anywhere in CLAUDE.md
- `/ant:import-signals` (Phase 6) -- not mentioned anywhere in CLAUDE.md
- `/ant:verify-castes` -- exists but not documented
- `/ant:migrate-state` -- exists but not documented
- `/ant:tunnels` -- exists but not documented

**Pheromone section (lines 196-222):**
- Does not describe the injection model (colony-prime injects signals into worker prompts via `prompt_section`)
- Does not mention `pheromone_protocol` sections in agent definitions (added in Phase 4)
- Does not reference the export/import commands added in Phase 6

**"The Core Insight" section (lines 360-376):**
- Claims "Pheromones don't update context" -- this is now FALSE after Phase 3 (pheromone-prime wired into colony-prime)
- Claims "Decisions don't become pheromones" -- partially addressed by Phase 4 (auto-emit)
- Claims "Learnings don't become instincts" -- addressed by Phase 5 (learning pipeline validation)
- Claims "Midden doesn't affect behavior" -- FALSE after Phase 4 (midden threshold auto-REDIRECT)
- This entire section describes problems that have been solved. Should be updated or removed.

**"RUNTIME UPDATE ARCHITECTURE.md" reference (lines 18, 46):**
- The file exists at repo root and is accurate (it was updated for v4.0). No change needed here, but it is worth noting for completeness.

**Last Updated line (line 5):**
- Says "2026-02-22" -- should be updated to reflect this milestone's changes

**Rule Modules section (lines 158-162):**
- Lists only `aether-colony.md` but the actual `.claude/rules/` directory only contains `aether-colony.md`, so this is correct
- However the section references "Previous separate rule files have been consolidated" which is historical and could be simplified

### Pheromone Documentation Inaccuracies (DOCS-02)

**pheromones.md specific issues:**

| Line | Current Text | Problem | Correct Text |
|------|-------------|---------|-------------|
| 24 | "Workers read FOCUS signals and weight this area higher" | Implies workers independently read signals | "FOCUS signals are injected into worker prompts via colony-prime and weight the indicated area higher in task execution" |
| 150 | "Pheromones combine. Workers check all active signals" | Implies workers independently check signals | "Pheromones combine. colony-prime injects all active signals into worker prompts, ordered by priority" |
| 159 | "Workers check high priority signals first" | Same active-worker framing | "High priority signals appear first in the injected context" |

**pheromones.md content that is correct and should NOT be changed:**
- Line 13: `pheromone-prime --compact` reference is accurate
- The TTL/decay mechanics are accurate
- Auto-emitted pheromone descriptions are accurate
- Signal combination table is accurate (just needs framing fix)

**CLAUDE.md Pheromone section (lines 196-222):**
- Does not mention the injection model at all
- Should reference that colony-prime assembles signals into `prompt_section` which is injected into worker spawn context
- Should reference the `pheromone_protocol` sections added to builder, watcher, and scout agent definitions in Phase 4

**README.md Pheromone section (lines 116-132):**
- Lines 128-129 correctly describe injection: "Auto-injected into worker prompts via `colony-prime --compact`"
- This section is largely accurate -- the README was updated more recently than CLAUDE.md/pheromones.md

**.claude/rules/aether-colony.md Pheromone section (lines 137-144):**
- Generic description that doesn't mention injection model
- Says "Signals guide colony behavior without hard-coding instructions" which is vague but not incorrect
- Could be enhanced but is not actively misleading

### known-issues.md Cleanup (DOCS-03)

**FIXED entries to REMOVE (15 entries):**

Critical Issues:
1. BUG-005: Missing lock release in flag-auto-resolve -- FIXED (Phase 16)
2. BUG-011: Missing error handling in flag-auto-resolve jq -- FIXED (Phase 16)

Medium Priority Issues:
3. BUG-002: Missing release_lock in flag-add error path -- FIXED (Phase 16)
4. BUG-003: Race condition in backup creation -- FIXED (Phase 16)

Architecture Issues:
5. ISSUE-002: Missing exec error handling -- FIXED (Phase 18-02)
6. ISSUE-003: Incomplete help command -- FIXED (Phase 18-03)
7. ISSUE-004: Template path hardcoded to staging directory -- FIXED (Phase 20)
8. ISSUE-007: Feature detection race condition -- FIXED (Phase 18-01)

Architecture Gaps:
9. GAP-001: No schema version validation -- FIXED (Phase 18-04)
10. GAP-002: No cleanup for stale spawn-tree entries -- FIXED (Phase 18-01)
11. GAP-003: No retry logic for failed spawns -- RESOLVED (Phase 18-02)
12. GAP-004: Missing queen-* documentation -- FIXED (Phase 18-03)
13. GAP-005: No validation of queen-read JSON output -- FIXED (Phase 18-04)
14. GAP-006: Missing queen-* command documentation -- FIXED (Phase 18-03)
15. GAP-009: context-update has no file locking -- FIXED (Phase 16)

Also remove: Fixed Issues section header entry "Checkpoint Allowlist System (Fixed 2026-02-15)" -- this is also a resolved issue.

**Entries to RETAIN (still open):**
1. BUG-004: Missing error code in flag-acknowledge (MEDIUM)
2. BUG-006: No lock release on JSON validation failure (MEDIUM)
3. BUG-007: 17+ instances of missing error codes (MEDIUM)
4. BUG-008: Missing error code in flag-add jq failure (HIGH)
5. BUG-009: Missing error codes in file checks (MEDIUM)
6. BUG-010: Missing error codes in context-update (MEDIUM)
7. BUG-012: Missing error code in unknown command (LOW)
8. ISSUE-001: Inconsistent error code usage (MEDIUM)
9. ISSUE-005: Potential infinite loop in spawn-tree (LOW)
10. ISSUE-006: Fallback json_err incompatible (LOW)
11. GAP-007: No error code standards documentation (open)
12. GAP-008: Missing error path test coverage (open)
13. GAP-010: Missing error code standards documentation (duplicate of GAP-007, consider removing)

**Workarounds Summary table:**
- Currently has 3 rows, 2 are struck through (FIXED). After cleanup: only 1 row remains (GAP-004 workaround).
- Actually GAP-004 is FIXED too. So the entire workarounds table should be reviewed.

**File footer:**
- Says "Generated from Oracle Research findings - 2026-02-15" -- should be updated with current date

### README.md Inaccuracies (DOCS-04)

| Claim | Location | Actual Value | Source |
|-------|----------|-------------|--------|
| "9 Active Agent Types" | Line 54 | 22 agent definitions (10 active spawnable castes per caste-system.md) | `ls .claude/agents/ant/ \| wc -l` = 22 |
| "35 Slash Commands" | Line 55 | 40 Claude commands | `ls .claude/commands/ant/ \| wc -l` = 40 |
| "installs 22 agents...plus 37 slash commands" | Line 77 | 22 agents + 40 commands | Same source |
| "Utility layer (80+ subcommands)" | Line 237 | 110 subcommands | `help` JSON output |
| "<your-repo>/.aether/ # Repo-local runtime" | Line 234 | Comment uses "runtime" which could confuse (runtime/ was eliminated) | Rephrase to "Repo-local colony files" or similar |

**Missing commands in README command tables:**
- `/ant:data-clean` -- not listed
- `/ant:export-signals` -- not listed
- `/ant:import-signals` -- not listed
- `/ant:verify-castes` -- not listed
- `/ant:migrate-state` -- not listed
- `/ant:tunnels` -- not listed
- `/ant:insert-phase` -- not listed

**README content that IS accurate:**
- Pheromone injection description (lines 128-129) -- correctly says "Auto-injected into worker prompts via colony-prime --compact"
- The Agent Castes table (lines 170-182)
- Spawn depth mechanics
- 6-Phase verification gates
- Colony memory (QUEEN.md) description
- Quick start workflow
- Safety features description
- CLI commands section

### Additional Documentation Files

**source-of-truth-map.md:**
- Updated date says "2026-02-22" -- stale
- Verified Inventory Snapshot shows "Slash commands (Claude) | 37" and "Slash commands (OpenCode) | 37" -- both stale (40 and 39 respectively)
- Shows "Sourced shell utilities | 17" -- should be verified
- Shows "Tests (all files) | 66" -- should be verified
- Shows "Command playbooks | 12" -- should be verified

**aether-colony.md (rules file):**
- Does not list export-signals, import-signals, data-clean, or other new commands
- Pheromone System section is generic but not actively incorrect

**context-continuity.md:**
- Phase 3 and Phase 4 are marked "next" but may have been partially implemented
- Uses "runtime command" phrasing throughout (this is acceptable -- it means "command that runs at runtime" not the eliminated `runtime/` directory)

## State of the Art

| Old State | Current State | When Changed | Impact on Docs |
|-----------|--------------|--------------|----------------|
| Workers independently read signals | colony-prime injects signals into worker prompts | Phase 3/4 (2026-03-19) | pheromones.md, CLAUDE.md must update model description |
| 36 slash commands | 40 Claude + 39 OpenCode | Phase 2/6/7 (2026-03-19) | All count references must update |
| 490+ tests | 537 tests | Phases 1-7 (2026-03-19) | CLAUDE.md test count must update |
| ~9,808 lines in aether-utils.sh | 10,499 lines | Phases 1-7 | CLAUDE.md line count must update |
| 150 subcommands | 110 subcommands | Verified via help command | CLAUDE.md subcommand count must update (was always wrong, likely aspirational) |
| 15 FIXED issues in known-issues.md | Should be 0 FIXED | Phase 16-20 fixes | All FIXED entries must be removed |
| No pheromone export/import commands | export-signals and import-signals exist | Phase 6 (2026-03-19) | README and CLAUDE.md must document these |
| "The Core Insight" listed 4 integration gaps | All 4 gaps addressed by Phases 3-5 | Phases 3-5 (2026-03-19) | CLAUDE.md section must be updated or rewritten |

## Open Questions

1. **Should "The Core Insight" section in CLAUDE.md be removed or rewritten?**
   - What we know: All 4 bullet points listing integration gaps are now addressed
   - What's unclear: Whether to remove the section entirely or rewrite it to describe the now-connected system
   - Recommendation: Rewrite to describe the current integrated state (e.g., "The integration is complete -- pheromones update context, decisions become pheromones, learnings become instincts, midden affects behavior")

2. **Should counts use exact values or "N+" floor values?**
   - What we know: Some counts (commands, agents) are stable; others (tests, lines) grow with every phase
   - What's unclear: Whether to use "537 tests" or "530+ tests"
   - Recommendation: Use exact values for stable counts (40 commands, 22 agents, 110 subcommands). Use "N+" for growing counts (530+ tests, 10,000+ lines).

3. **Should duplicate GAP entries in known-issues.md be consolidated?**
   - What we know: GAP-007 and GAP-010 are duplicates ("No error code standards documentation")
   - What's unclear: Whether to keep both or merge
   - Recommendation: Remove GAP-010 as duplicate, keep GAP-007

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection via grep, wc -l, ls, npm test -- all findings verified against actual file contents
- Phase verification reports (04-VERIFICATION.md, 06-VERIFICATION.md, 07-VERIFICATION.md) -- confirmed what changes were made
- Agent definitions (.claude/agents/ant/aether-builder.md, aether-watcher.md, aether-scout.md) -- confirmed pheromone_protocol sections exist
- REQUIREMENTS.md -- confirmed DOCS-01 through DOCS-04 requirement text

### Secondary (MEDIUM confidence)
- source-of-truth-map.md counts -- these are from the 2026-02-22 snapshot and are now stale; verified actual counts are different

## Metadata

**Confidence breakdown:**
- DOCS-01 (CLAUDE.md): HIGH -- every inaccuracy identified with exact line numbers and verified correct values
- DOCS-02 (Pheromone docs): HIGH -- injection model confirmed via agent definitions and colony-prime code; exact lines identified
- DOCS-03 (known-issues.md): HIGH -- every entry categorized as FIXED or open with cross-reference to phase verification reports
- DOCS-04 (README): HIGH -- every stale claim identified with verified correct values

**Research date:** 2026-03-19
**Valid until:** 2026-04-19 (documentation drift findings are stable unless new features are added)
