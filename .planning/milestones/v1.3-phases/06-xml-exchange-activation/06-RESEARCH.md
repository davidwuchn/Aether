# Phase 6: XML Exchange Activation - Research

**Researched:** 2026-03-19
**Domain:** Bash shell scripting, XML exchange, slash command authoring, lifecycle integration
**Confidence:** HIGH

## Summary

Phase 6 activates an XML exchange system that already exists in full -- the code to export and import pheromone signals, wisdom entries, and colony registries between JSON and XML is implemented, tested, and working. The `pheromone-export-xml`, `pheromone-import-xml`, `wisdom-export-xml`, `wisdom-import-xml`, `registry-export-xml`, `registry-import-xml`, and `colony-archive-xml` subcommands all exist in `aether-utils.sh` (lines 8318-8736). What is missing is user-facing wiring: no slash commands expose these capabilities, and the seal lifecycle only partially integrates XML export (Step 6.5 exists but uses `colony-archive-xml` which is a combined archive, not a standalone pheromone signal export).

The work is primarily **slash command authoring** (creating `.claude/commands/ant/export-signals.md` and `.claude/commands/ant/import-signals.md`) and **lifecycle integration** (ensuring seal.md's Step 6.5 exports pheromone signals specifically, not just the combined archive). The e2e test suite at `tests/e2e/test-xml.sh` currently tests low-level round-trip functionality (XML-01/02/03 in that file refer to different requirements than the REQUIREMENTS.md IDs) and needs to be updated to verify the new command-level requirements.

**Primary recommendation:** Create two new slash commands (`/ant:export-signals` and `/ant:import-signals`) that wrap existing `pheromone-export-xml` and `pheromone-import-xml` subcommands, verify the seal lifecycle auto-export works, and write integration tests that prove cross-colony signal transfer.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| XML-01 | XML exchange system wired into commands (/ant:export-signals and /ant:import-signals or equivalent) | Two new slash commands needed; underlying subcommands (`pheromone-export-xml`, `pheromone-import-xml`) already exist and are tested. Command format follows established patterns from focus.md, pheromones.md. |
| XML-02 | Pheromone XML export/import works end-to-end (export from one colony, import into another) | `pheromone-export-xml` exports `.aether/data/pheromones.json` to XML; `pheromone-import-xml` reads XML and merges into local pheromones.json with optional colony prefix. Round-trip is tested in `tests/bash/test-xml-roundtrip.sh`. Integration test needed to verify cross-colony scenario with prefix namespacing. |
| XML-03 | XML exchange integrated into pause/seal lifecycle (automatic export on colony seal) | seal.md Step 6.5 already calls `colony-archive-xml` which includes pheromones. Need to verify it exports standalone pheromone XML too (or adjust to export `pheromones.xml` separately). Also wire into `pause-colony.md` as best-effort export. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| aether-utils.sh | current (~9,808 lines) | Central CLI with 150+ subcommands including all XML exchange operations | Single entry point for all exchange operations |
| pheromone-xml.sh | current | JSON-to-XML bidirectional conversion for pheromones | Dedicated exchange module, already tested |
| wisdom-xml.sh | current | JSON-to-XML bidirectional conversion for wisdom | Dedicated exchange module, already tested |
| registry-xml.sh | current | JSON-to-XML bidirectional conversion for colony registry | Dedicated exchange module, already tested |
| xmllint | system | XML well-formedness validation | Ships with macOS (xcode-select), required dependency |
| jq | system | JSON processing | Already used throughout aether-utils.sh |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| xmlstarlet | Advanced XPath extraction during import | Optional enhancement; fallback to grep/sed exists |
| xml-core.sh | Shared XML helpers (escape, validate, json wrappers) | Sourced by all exchange modules |
| pheromone.xsd | XSD schema for pheromone XML validation | Optional validation during export |

## Architecture Patterns

### Existing Exchange File Layout
```
.aether/
  exchange/
    pheromone-xml.sh      # JSON <-> XML for pheromone signals
    wisdom-xml.sh         # JSON <-> XML for queen wisdom
    registry-xml.sh       # JSON <-> XML for colony registry
    pheromones.xml        # Sample/last-exported pheromone XML
    colony-registry.xml   # Sample/last-exported registry XML
    queen-wisdom.xml      # Sample/last-exported wisdom XML
  data/
    pheromones.json       # Active pheromone signals (source of truth)
  schemas/
    pheromone.xsd         # XSD validation schema
    queen-wisdom.xsd      # XSD validation schema
    colony-registry.xsd   # XSD validation schema
```

### Pattern 1: Slash Command Structure
**What:** All slash commands follow a consistent structure with frontmatter, instructions, step-by-step flow, and Next Up section.
**When to use:** Creating any new `/ant:*` command.
**Example:** (from focus.md -- the simplest signal command)
```markdown
---
name: ant:focus
description: "Emit FOCUS signal to guide colony attention"
---

You are the **Queen**. Emit a FOCUS pheromone signal.

## Instructions

### Step 1: Validate
[validate arguments]

### Step 2: Execute
[call aether-utils.sh subcommand]

### Step 3: Confirm
[display result]

### Step 4: Next Up
[generate state-based suggestions]
```

### Pattern 2: aether-utils.sh Subcommand Invocation
**What:** Slash commands invoke aether-utils.sh subcommands via `bash .aether/aether-utils.sh <subcommand> <args>` and parse JSON results.
**When to use:** Any command that needs to read/write colony data.
**Example:**
```bash
# Export pheromones to XML
result=$(bash .aether/aether-utils.sh pheromone-export-xml "/path/to/output.xml")
ok=$(echo "$result" | jq -r '.ok')

# Import pheromones from XML with colony prefix
result=$(bash .aether/aether-utils.sh pheromone-import-xml "/path/to/signals.xml" "source-colony")
signal_count=$(echo "$result" | jq -r '.result.signal_count')
```

### Pattern 3: Lifecycle Hook Integration
**What:** Seal and pause commands have numbered steps; new behavior is inserted as sub-steps (e.g., Step 6.5).
**When to use:** Adding XML export to seal/pause lifecycle.
**Existing example:** seal.md Step 6.5 already calls `colony-archive-xml` as best-effort.
**Key principle:** Lifecycle hooks are ALWAYS best-effort (non-blocking). XML export failure never prevents sealing or pausing.

### Pattern 4: OpenCode Command Parity
**What:** `.opencode/commands/ant/` must mirror `.claude/commands/ant/` (currently 37 vs 38 commands -- only `data-clean.md` differs).
**When to use:** After creating any new Claude Code command, create the OpenCode equivalent.

### Anti-Patterns to Avoid
- **Blocking lifecycle on XML export:** XML export failure MUST NEVER prevent seal/pause from completing. Always wrap in best-effort blocks.
- **Hardcoding paths:** Use `$SCRIPT_DIR` and `$DATA_DIR` variables from aether-utils.sh, not absolute paths.
- **Skipping colony prefix on import:** Always apply a colony prefix when importing foreign signals to prevent ID collisions. The `pheromone-import-xml` subcommand already supports `[colony_prefix]` as second arg.
- **Overwriting existing signals:** Import merge uses `group_by(.id) | map(last)` -- current colony signals win on ID collision. This is correct behavior.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON-to-XML conversion | Custom jq/sed pipeline | `pheromone-export-xml` subcommand | Handles namespaces, escaping, validation; 575 lines of tested code |
| XML-to-JSON conversion | Custom grep/awk parser | `pheromone-import-xml` subcommand | Uses xmlstarlet with grep/sed fallback; handles namespace stripping |
| XML validation | Manual well-formedness check | `pheromone-validate-xml` subcommand | Uses xmllint with XSD schema support |
| Signal merge with dedup | Custom merge logic | `pheromone-import-xml`'s built-in merge | Handles prefix namespacing, ID collision resolution |
| Combined archive export | Assembling XML sections | `colony-archive-xml` subcommand | Combines pheromones + wisdom + registry into single archive |
| XML escaping | Manual sed replacements | `xml-core.sh` helpers | `xml_json_ok`, `xml_json_err`, and escape helpers handle all edge cases |

**Key insight:** The entire XML exchange infrastructure is already built, tested, and working. Phase 6 is a **wiring** task, not a **building** task. The planner should create tasks that compose existing subcommands into slash commands and lifecycle hooks, not implement new conversion logic.

## Common Pitfalls

### Pitfall 1: xmllint Not Available
**What goes wrong:** XML export/import fails on systems without xmllint installed.
**Why it happens:** xmllint comes from libxml2 which requires Xcode Command Line Tools on macOS.
**How to avoid:** Every code path that uses XML must check `command -v xmllint` first and degrade gracefully. All existing subcommands already do this via `json_err "$E_FEATURE_UNAVAILABLE"`. Slash commands should catch this error and display a helpful message.
**Warning signs:** `json_err` with `E_FEATURE_UNAVAILABLE` in test output.

### Pitfall 2: Existing e2e Test IDs Clash with Requirement IDs
**What goes wrong:** `tests/e2e/test-xml.sh` defines XML-01/02/03 but these test different things than REQUIREMENTS.md XML-01/02/03. The e2e tests verify low-level round-trip; the requirements verify user-facing command wiring.
**Why it happens:** The e2e tests were written before the current roadmap requirements were defined.
**How to avoid:** Either rename the e2e test requirement IDs (risky -- may break test runner) or write new tests that explicitly target REQUIREMENTS.md XML-01/02/03 as separate test cases. Recommendation: add new test cases to the existing `test-xml.sh` e2e file or create a separate `test-xml-commands.sh`.
**Warning signs:** Test passes but requirement is not actually met.

### Pitfall 3: colony-archive-xml vs Standalone Pheromone Export
**What goes wrong:** XML-03 requires "automatic export of pheromone signals on seal." The current seal.md Step 6.5 calls `colony-archive-xml` which produces a combined archive (pheromones + wisdom + registry). This is more than what XML-03 strictly requires, but it DOES include pheromone signals.
**Why it happens:** `colony-archive-xml` was designed as a comprehensive archive, not a targeted signal export.
**How to avoid:** Decide whether the combined archive satisfies XML-03 (it does contain exported pheromone signals) or if a separate standalone `pheromone-export-xml` call is also needed. Recommendation: the combined archive satisfies the requirement, but also run `pheromone-export-xml` to produce a standalone `pheromones.xml` file that can be easily shared/imported by other colonies.
**Warning signs:** Archive exists but no standalone importable XML file.

### Pitfall 4: Import Without Colony Prefix Creates ID Collisions
**What goes wrong:** Importing signals from another colony without a prefix means signal IDs like `sig_focus_001` collide with local signals of the same ID.
**Why it happens:** Both colonies generate IDs from timestamps/counters.
**How to avoid:** The `pheromone-import-xml` subcommand's second argument is `colony_prefix`. The slash command MUST require or auto-generate a prefix. On collision, current colony wins (`map(last)` in jq merge).
**Warning signs:** Import reports success but signal count doesn't increase; imported signals silently overwritten.

### Pitfall 5: Pause-Colony XML Export Scope
**What goes wrong:** XML-03 mentions "pause/seal lifecycle" but the REQUIREMENTS.md only says "automatic export on colony seal." pause-colony does NOT currently export XML.
**Why it happens:** The pause command focuses on creating a text-based HANDOFF.md, not XML archives.
**How to avoid:** The strict requirement is seal-only. Adding XML export to pause is a nice-to-have but not required by XML-03. If added, it MUST be best-effort and non-blocking.
**Warning signs:** Over-scoping the phase by modifying pause-colony when the requirement only mandates seal.

## Code Examples

### Example 1: Export Signals (subcommand call)
```bash
# Source: aether-utils.sh line 8318-8344
# Export current colony's pheromones to XML
bash .aether/aether-utils.sh pheromone-export-xml ".aether/exchange/pheromones.xml"
# Returns: {"ok":true,"result":{"path":".aether/exchange/pheromones.xml","validated":false}}
```

### Example 2: Import Signals with Colony Prefix (subcommand call)
```bash
# Source: aether-utils.sh line 8346-8406
# Import XML signals with colony prefix to prevent ID collisions
bash .aether/aether-utils.sh pheromone-import-xml "/path/to/foreign-signals.xml" "source-colony-name"
# Returns: {"ok":true,"result":{"imported":true,"signal_count":5,"source":"/path/to/foreign-signals.xml"}}
```

### Example 3: Seal Lifecycle XML Export (existing Step 6.5)
```bash
# Source: seal.md lines 477-495
# Best-effort XML archive export during seal ceremony
if command -v xmllint >/dev/null 2>&1; then
  xml_result=$(bash .aether/aether-utils.sh colony-archive-xml ".aether/exchange/colony-archive.xml" 2>&1)
  xml_ok=$(echo "$xml_result" | jq -r '.ok // false' 2>/dev/null)
fi
```

### Example 4: Merge Logic (how import handles collisions)
```bash
# Source: aether-utils.sh line 8390-8397
# Merge: imported signals first, existing signals last
# map(last) keeps current colony's version on ID collision
jq -s --argjson new_signals "$pix_prefixed_signals" '
  .[0] as $existing |
  {
    signals: ([$new_signals[], $existing.signals[]] | group_by(.id) | map(last)),
    version: $existing.version,
    colony_id: $existing.colony_id
  }
' "$pix_pheromones"
```

### Example 5: Pheromone JSON Structure (source of truth)
```json
{
  "signals": [
    {
      "id": "sig_feedback_001",
      "type": "FEEDBACK",
      "priority": "low",
      "source": "worker_builder",
      "created_at": "2026-02-16T12:00:00Z",
      "active": true,
      "strength": 0.6,
      "content": { "text": "..." },
      "tags": [{ "value": "testing", "weight": 0.7, "category": "quality" }]
    }
  ],
  "version": "1.0.0",
  "colony_id": "aether-dev"
}
```

### Example 6: Exported Pheromone XML Structure
```xml
<?xml version="1.0" encoding="UTF-8"?>
<pheromones xmlns="http://aether.colony/schemas/pheromones"
            version="1.0.0" generated_at="2026-02-17T23:51:44Z" colony_id="aether-dev">
  <metadata>
    <source type="system">aether-pheromone-converter</source>
    <context>Colony pheromone signals</context>
  </metadata>
  <signal id="sig_focus_001" type="FOCUS" priority="normal" source="user"
          created_at="2026-02-16T10:00:00Z" expires_at="2026-02-17T10:00:00Z" active="true">
    <content>
      <text>XML migration and pheromone system implementation</text>
    </content>
  </signal>
</pheromones>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No XML exchange | Full exchange module with 3 converters | Feb 2026 | Pheromones, wisdom, and registry can all round-trip through XML |
| Seal has no XML step | Seal Step 6.5 exports colony-archive-xml | Feb/Mar 2026 | Combined archive produced during seal, but no standalone pheromone export |
| No slash commands for export/import | subcommands exist but no slash commands | Current gap | Users can't easily export/import signals without knowing bash subcommands |

**What exists but is not wired:**
- `pheromone-export-xml` subcommand (works, tested)
- `pheromone-import-xml` subcommand with merge and prefix (works, tested)
- `pheromone-validate-xml` subcommand (works)
- `colony-archive-xml` combined archive (works, used by seal)
- `xml-pheromone-merge` function for multi-colony merge (works, tested)
- XSD schemas for all XML formats (exist in `.aether/schemas/`)
- 7 bash test files covering round-trip, security, schemas, e2e

## Existing Test Coverage

| Test File | What It Tests | Status |
|-----------|---------------|--------|
| `tests/bash/test-xml-roundtrip.sh` | JSON-to-XML-to-JSON round-trip for pheromone signals | Passing |
| `tests/bash/test-pheromone-xml.sh` | Pheromone-to-XML with XSD validation | Passing |
| `tests/bash/test-xml-utils.sh` | XML utility functions (escape, validate, format) | Passing |
| `tests/bash/test-xml-security.sh` | XXE protection, entity injection prevention | Passing |
| `tests/bash/test-xml-schemas.sh` | XSD schema validation for all XML types | Passing |
| `tests/bash/test-phase3-xml.sh` | Phase 3 specific XML tests | Passing |
| `tests/e2e/test-xml.sh` | End-to-end XML round-trip (pheromone, wisdom, registry) | Passing |

**Gap:** No tests verify the slash command layer or the cross-colony import scenario end-to-end.

## Open Questions

1. **Command naming: `/ant:export-signals` vs `/ant:export-pheromones`**
   - What we know: The roadmap says "export and import pheromone signals." The existing subcommands use `pheromone-export-xml` naming.
   - What's unclear: Whether to use "signals" (user-friendly) or "pheromones" (system-consistent) in the slash command name.
   - Recommendation: Use `/ant:export-signals` and `/ant:import-signals` as the user-facing names, since "signals" is the term used in user-facing pheromone documentation and the roadmap. The underlying subcommand remains `pheromone-export-xml`.

2. **Default export path**
   - What we know: `pheromone-export-xml` defaults to `.aether/exchange/pheromones.xml`. Users may want to specify a custom path for sharing.
   - What's unclear: Should the command default to exchange/ or let users specify?
   - Recommendation: Default to `.aether/exchange/pheromones.xml` but accept an optional path argument. Display the output path in the confirmation message.

3. **Scope of "import into another" (XML-02)**
   - What we know: `pheromone-import-xml` merges into the local `pheromones.json` with optional colony prefix.
   - What's unclear: Does XML-02 require that the imported signals actually function (decay, appear in colony-prime output)?
   - Recommendation: Yes -- imported signals should have `active: true` and appear in `pheromone-read` output. The existing import code already sets `active: true` for imported signals and merges them into pheromones.json, which is the same file that `pheromone-read` and `colony-prime` consume.

4. **OpenCode parity**
   - What we know: OpenCode commands at `.opencode/commands/ant/` must mirror Claude Code commands. Currently 37 vs 38 commands (only data-clean.md differs).
   - What's unclear: Whether OpenCode parity is required in this phase or can be deferred.
   - Recommendation: Create the OpenCode equivalents alongside the Claude Code commands. The diff is currently small (1 file). Adding export-signals.md and import-signals.md to both locations keeps parity.

## Sources

### Primary (HIGH confidence)
- `.aether/exchange/pheromone-xml.sh` -- Full source of export/import/merge/validate functions (576 lines)
- `.aether/exchange/wisdom-xml.sh` -- Wisdom exchange module (319 lines)
- `.aether/exchange/registry-xml.sh` -- Registry exchange module (274 lines)
- `.aether/aether-utils.sh` lines 8315-8736 -- All XML exchange subcommands
- `.claude/commands/ant/seal.md` lines 477-495 -- Existing seal Step 6.5 XML export
- `.claude/commands/ant/focus.md` -- Reference slash command structure
- `.planning/REQUIREMENTS.md` -- Requirement definitions for XML-01/02/03
- `tests/e2e/test-xml.sh` -- Existing e2e test structure
- `tests/bash/test-xml-roundtrip.sh` -- Round-trip test patterns
- `.aether/data/pheromones.json` -- Live pheromone data structure

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/pause-colony.md` -- Pause lifecycle (for potential XML-03 extension)
- `.claude/commands/ant/pheromones.md` -- Pheromone management command patterns
- `.aether/utils/xml-core.sh` -- Shared XML helper functions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- All code exists and is verified by reading source files
- Architecture: HIGH -- Slash command patterns are well-established across 38 existing commands
- Pitfalls: HIGH -- Identified from reading actual code paths and merge logic
- Requirements mapping: HIGH -- Requirements are clearly defined and map directly to existing subcommands

**Research date:** 2026-03-19
**Valid until:** 2026-04-19 (stable domain -- bash shell scripts, no external dependency changes expected)
