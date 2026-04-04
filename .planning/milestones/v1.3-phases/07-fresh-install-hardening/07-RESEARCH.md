# Phase 7: Fresh Install Hardening - Research

**Researched:** 2026-03-19
**Domain:** npm packaging, shell scripting, end-to-end install validation
**Confidence:** HIGH

## Summary

Phase 7 addresses two distinct but related requirements: (1) a fresh install smoke test that validates the complete `lay-eggs -> init -> plan -> build -> continue` lifecycle runs without errors on a clean repo, and (2) hardening `validate-package.sh` to reject packages that contain test artifacts in data files like QUEEN.md and pheromones.json.

The current state has several concrete problems discovered during research. First, `.aether/QUEEN.md` is shipped in the npm package and currently contains test artifact entries ("colony-a" through "colony-e") plus a real entry ("1771335865738") that were accumulated during development. This file propagates to every user's `~/.aether/system/` hub and from there into their projects via `/ant:lay-eggs`. Second, a temp file `.aether/QUEEN.md.tmp.98208.metaupd` (0 bytes) is also included in the published package due to missing npmignore patterns. Third, `validate-package.sh` currently only checks that required files exist and that private directories are listed in .npmignore -- it performs zero content inspection for test artifacts. The existing `test-lifecycle.sh` and `test-install.sh` tests in `tests/e2e/` cover the npm install-to-hub flow and subcommand-level lifecycle operations, but neither simulates a true fresh-install-to-build scenario end-to-end.

**Primary recommendation:** Split into two plans: (1) Harden validate-package.sh with test artifact detection for QUEEN.md and temp file rejection; (2) Create a fresh install smoke test script that simulates the complete lay-eggs through continue lifecycle in an isolated environment.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| INST-01 | Fresh install smoke test validates lay-eggs -> init -> plan -> build on clean repo without errors | The existing e2e test infrastructure (`tests/e2e/e2e-helpers.sh`, `test-lifecycle.sh`, `test-install.sh`) provides patterns for isolated environment setup, result tracking, and JSON extraction. A new script needs to simulate the full CLI install -> lay-eggs -> init flow by creating an isolated temp HOME, running `node bin/cli.js install`, then executing the aether-utils.sh subcommands that each slash command invokes, in sequence. |
| INST-04 | Pre-publish validation (validate-package.sh) rejects packages containing test artifacts in data files | `validate-package.sh` currently only checks file existence and npmignore rules. It needs additional checks: (1) QUEEN.md content inspection for non-template entries, (2) pheromones.json template purity check, (3) temp file glob rejection (*.tmp*), (4) CONTEXT.md exclusion from the package manifest. Research confirmed QUEEN.md contains 6 non-template entries that would ship to users. |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| bash | 3.2+ | Test scripts, validate-package.sh | macOS default, all existing tests use bash |
| jq | 1.6+ | JSON inspection in validation | Already used throughout aether-utils.sh |
| npm pack --dry-run | npm 9+ | Package content verification | The canonical way to inspect what npm will publish |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| mktemp | system | Isolated test environments | Already used in test-install.sh and test-lifecycle.sh |
| grep | system | Content pattern matching in validate-package.sh | Simple string matching for artifact detection |
| ava | 6.x | Unit tests (if needed) | Existing test framework in package.json |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| bash e2e scripts | ava + JS e2e tests | bash is more natural for CLI/shell testing; all existing e2e tests are bash |
| grep for content checks | node script for validation | bash is consistent with validate-package.sh; node would add complexity |

## Architecture Patterns

### Current File Layout (Relevant to Phase 7)
```
bin/
  validate-package.sh       # Pre-publish validation (needs hardening)
  cli.js                    # CLI including setupHub() and install command
  npx-install.js            # npx installer

.aether/
  QUEEN.md                  # CONTAMINATED: contains test artifacts, ships in package
  QUEEN.md.tmp.98208.metaupd  # STALE: temp file leaking into package
  .npmignore                # Missing patterns for QUEEN.md, *.tmp*
  templates/
    QUEEN.md.template       # Clean template (this is what should ship)
    pheromones.template.json # Clean template
    constraints.template.json # Clean template

tests/
  e2e/
    test-install.sh         # Tests npm install -> hub creation (12 tests)
    test-lifecycle.sh       # Tests subcommand lifecycle flow (7 steps)
    e2e-helpers.sh          # Shared test infrastructure
  bash/
    test-helpers.sh         # Assertion functions (assert_output_contains, etc.)
```

### Pattern 1: Isolated Environment Testing
**What:** Create a temp directory with `HOME` overridden to isolate the test from the user's real hub
**When to use:** Always, for fresh install tests
**Example:**
```bash
# Source: tests/e2e/test-install.sh (lines 13-14)
TMP_DIR=$(mktemp -d)
export HOME="$TMP_DIR"  # Isolate test environment
```

### Pattern 2: Subcommand Proxy Testing
**What:** The existing lifecycle test does NOT run slash commands directly (they require Claude). Instead it calls the underlying `aether-utils.sh` subcommands that each slash command invokes.
**When to use:** For verifying the init/plan/build/continue flow without an LLM
**Example:**
```bash
# Source: tests/e2e/test-lifecycle.sh (lines 132-134)
raw_si=$(bash "$UTILS" session-init "" "lifecycle-test" 2>&1 || true)
si_out=$(extract_json "$raw_si")
if echo "$si_out" | jq -e '.ok == true' >/dev/null 2>&1; then
```

### Pattern 3: validate-package.sh Content Checks
**What:** Add grep-based content inspection after file existence checks
**When to use:** For detecting test artifacts in files that will be published
**Example:**
```bash
# Check QUEEN.md doesn't contain non-template entries
# Template has "No philosophies recorded yet" placeholders
# Contaminated QUEEN.md has actual entries like "colony-a", "colony-b"
if [ -f "$AETHER_DIR/QUEEN.md" ]; then
  # Count lines that look like promoted wisdom entries
  ARTIFACT_COUNT=$(grep -c '^\- \*\*' "$AETHER_DIR/QUEEN.md" 2>/dev/null || echo "0")
  if [ "$ARTIFACT_COUNT" -gt 0 ]; then
    echo "ERROR: QUEEN.md contains $ARTIFACT_COUNT promoted entries (test artifacts?)" >&2
    echo "  Reset with: cp .aether/templates/QUEEN.md.template .aether/QUEEN.md" >&2
    exit 1
  fi
fi
```

### Anti-Patterns to Avoid
- **Testing with the user's real HOME:** Always override HOME to a temp directory. Never touch `~/.aether/` during tests.
- **Assuming LLM availability:** Smoke tests must verify the subcommand layer, not the slash command layer. Slash commands require Claude/OpenCode.
- **Checking only file existence:** The current validate-package.sh only checks "does the file exist?" -- it needs to also check "is the file clean?"

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Temp directory isolation | Custom cleanup logic | `mktemp -d` + `trap cleanup EXIT` | Already the pattern in all e2e tests |
| JSON validation | Manual parsing | `jq -e` checks | Already used everywhere in aether-utils.sh |
| Package content listing | Custom file walker | `npm pack --dry-run` | The canonical npm tool for this |
| Test result tracking | Custom framework | `record_lifecycle_result` from e2e-helpers.sh | Already exists and is bash 3.2 compatible |

**Key insight:** The test infrastructure in `tests/e2e/e2e-helpers.sh` already handles isolated environments, result tracking, and JSON extraction. The fresh install test should reuse these patterns rather than inventing new ones.

## Common Pitfalls

### Pitfall 1: QUEEN.md Shipped with Development Data
**What goes wrong:** The npm package includes `.aether/QUEEN.md` which contains entries accumulated during development (colony-a through colony-e test entries, plus real development entries). Every user who installs gets this contaminated wisdom file.
**Why it happens:** QUEEN.md is not excluded from the package (`files` in package.json includes `.aether/`), not listed in `.aether/.npmignore`, and not mentioned in `HUB_EXCLUDE_FILES` in cli.js.
**How to avoid:** Either (a) exclude QUEEN.md from the npm package entirely and let it be created from template during install, or (b) reset QUEEN.md to template state before publishing and add a validate-package.sh check. Option (a) is safer.
**Warning signs:** `npm pack --dry-run` showing `.aether/QUEEN.md` in the output with non-zero size.

### Pitfall 2: Temp Files Leaking into Package
**What goes wrong:** Files like `.aether/QUEEN.md.tmp.98208.metaupd` (created by atomic write operations that crashed or left behind) get included in the npm package.
**Why it happens:** No wildcard exclusion for temp files in `.aether/.npmignore`.
**How to avoid:** Add `*.tmp*` glob pattern to `.aether/.npmignore`.
**Warning signs:** `npm pack --dry-run` showing `.tmp` files.

### Pitfall 3: CONTEXT.md Shipped in Package
**What goes wrong:** `.aether/CONTEXT.md` is a per-colony file that should not be in the package. The `syncAetherToHub` function in cli.js already excludes it (line 838: `HUB_EXCLUDE_FILES = ['CONTEXT.md', 'HANDOFF.md']`), but the npm package itself includes it because `.aether/` is in the `files` array in package.json.
**Why it happens:** The hub sync and npm packaging are separate mechanisms. Hub sync filters correctly; npm packaging uses `.aether/.npmignore` which doesn't exclude CONTEXT.md.
**How to avoid:** Add `CONTEXT.md` to `.aether/.npmignore`. It is already excluded from hub sync, but should also be excluded from the tarball.
**Warning signs:** `npm pack --dry-run` showing `.aether/CONTEXT.md`.

### Pitfall 4: Fresh Install Assumes Hub Exists
**What goes wrong:** `/ant:lay-eggs` Step 1 checks for `~/.aether/system/aether-utils.sh`. If the hub is not set up (user hasn't run `npm install -g aether-colony`), lay-eggs will fail with a helpful message. But the smoke test needs to simulate both the hub setup AND the lay-eggs flow.
**Why it happens:** The CLI install (`node bin/cli.js install`) is what creates the hub. The smoke test must run this first.
**How to avoid:** The smoke test must follow the real user journey: (1) `npm install -g` (simulated by `node bin/cli.js install`), (2) `/ant:lay-eggs` (simulated by copying from hub), (3) `/ant:init`, (4) etc.

### Pitfall 5: validate-package.sh Runs at prepublishOnly but Not in CI
**What goes wrong:** The validation only runs when someone does `npm publish`. If test artifacts are committed, they stay until publish time.
**Why it happens:** No pre-commit or CI check for package cleanliness.
**How to avoid:** The fresh install smoke test (INST-01) can double as a CI guard -- if it detects artifacts, it fails the test suite.

## Code Examples

Verified patterns from the existing codebase:

### Fresh Install Hub Setup (from test-install.sh)
```bash
# Source: tests/e2e/test-install.sh lines 100-113
TMP_DIR=$(mktemp -d)
export HOME="$TMP_DIR"

cd "$PROJECT_ROOT"
node bin/cli.js install --quiet 2>/dev/null || true

# Verify hub created
if [ -d "$HOME/.aether" ]; then
    echo "PASS: ~/.aether/ created"
fi
if [ -f "$HOME/.aether/system/aether-utils.sh" ]; then
    echo "PASS: aether-utils.sh installed"
fi
```

### Lifecycle Subcommand Proxy (from test-lifecycle.sh)
```bash
# Source: tests/e2e/test-lifecycle.sh lines 130-164
# Simulate /ant:init via its underlying subcommands
raw_si=$(bash "$UTILS" session-init "" "lifecycle-test" 2>&1 || true)
si_out=$(extract_json "$raw_si")

if echo "$si_out" | jq -e '.ok == true' >/dev/null 2>&1; then
    echo "PASS: session-init returned ok:true"
fi
```

### Content Validation in validate-package.sh
```bash
# Pattern: Check QUEEN.md is clean (template-only content)
# The template contains "No philosophies recorded yet" etc.
# A contaminated file has "- **colony-..." entries
QUEEN_FILE="$AETHER_DIR/QUEEN.md"
if [ -f "$QUEEN_FILE" ]; then
    # Template lines use "*No ... recorded yet.*" placeholders
    # Promoted entries use "- **{id}** ({timestamp}): {content}" format
    if grep -qE '^\- \*\*[a-zA-Z0-9_-]+\*\*' "$QUEEN_FILE"; then
        echo "ERROR: QUEEN.md contains promoted entries (test artifacts)" >&2
        echo "  Reset: cp .aether/templates/QUEEN.md.template .aether/QUEEN.md" >&2
        exit 1
    fi
fi
```

### Template Purity Check for pheromones.json
```bash
# If pheromones.json exists in .aether/ (it shouldn't, but check)
PHER_FILE="$AETHER_DIR/data/pheromones.json"
if [ -f "$PHER_FILE" ]; then
    SIGNAL_COUNT=$(jq '.signals | length' "$PHER_FILE" 2>/dev/null || echo "0")
    if [ "$SIGNAL_COUNT" -gt 0 ]; then
        echo "ERROR: pheromones.json contains $SIGNAL_COUNT signals (test data)" >&2
        exit 1
    fi
fi
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| runtime/ staging directory | Direct .aether/ packaging (v4.0) | 2026-02 | validate-package.sh replaced sync-to-runtime.sh |
| No content validation | File existence + npmignore checks only | 2026-02 | Misses test artifact contamination |
| No fresh install test | test-install.sh tests hub creation only | 2026-02 | Doesn't cover lay-eggs -> init -> build flow |

**Current gaps:**
- QUEEN.md: Shipped with test data, no content check
- Temp files: No glob exclusion in .aether/.npmignore
- CONTEXT.md: Excluded from hub sync but included in npm tarball
- No end-to-end fresh install test covering the full lifecycle

## Key Findings for Implementation

### What needs to happen for INST-04 (validate-package.sh hardening):

1. **Add content checks to validate-package.sh:**
   - QUEEN.md: Reject if it contains promoted entries (regex: `^\- \*\*`)
   - Or better: exclude QUEEN.md from the package entirely (add to `.aether/.npmignore`) since it should always be created from template
   - Reject if any `.tmp*` files exist in `.aether/`
   - Reject if CONTEXT.md would be included (add to `.aether/.npmignore`)

2. **Clean .aether/.npmignore:**
   - Add: `QUEEN.md` (should be created from template, not shipped)
   - Add: `*.tmp*` (temp files from atomic writes)
   - Add: `CONTEXT.md` (per-colony file, already excluded from hub sync)
   - Add: `HANDOFF.md` (per-session file)

3. **Clean current QUEEN.md:**
   - Either reset to template or delete (since it will be excluded from package)
   - Remove `.aether/QUEEN.md.tmp.98208.metaupd`

### What needs to happen for INST-01 (fresh install smoke test):

1. **Create `tests/e2e/test-fresh-install.sh`:**
   - Set up isolated HOME (mktemp + HOME override)
   - Run `node bin/cli.js install --quiet` to create hub
   - Create a temp repo with git init
   - Simulate lay-eggs: copy system files from hub to temp repo's .aether/
   - Run queen-init, session-init, validate-state subcommands
   - Write a colony state from template
   - Initialize pheromones, constraints, midden from templates
   - Run pheromone-write, pheromone-read to verify signal flow
   - Run session-update to simulate plan/build/continue
   - Verify all operations return ok:true
   - Clean up

2. **Test structure should follow test-lifecycle.sh pattern:**
   - Step-by-step with clear PASS/FAIL per step
   - Use extract_json from e2e-helpers.sh
   - Support --results-file for runner integration
   - bash 3.2 compatible (no associative arrays)

## Open Questions

1. **Should QUEEN.md be excluded from the npm package entirely?**
   - What we know: QUEEN.md is generated from template during queen-init. The template ships in the package. The hub's global QUEEN.md is created by npx-install.js from the template (line 156-165 of npx-install.js). The per-repo QUEEN.md is created by queen-init subcommand.
   - What's unclear: Is there any flow where having a pre-populated QUEEN.md in the package (as opposed to the template) adds value?
   - Recommendation: Exclude QUEEN.md from the package. It should always be created from the template. This eliminates the artifact contamination vector entirely. Both the CLI install and the lay-eggs flow create QUEEN.md from the template, so shipping the file itself serves no purpose.

2. **Should the smoke test also verify `npm pack --dry-run` output?**
   - What we know: validate-package.sh has a --dry-run mode that runs npm pack --dry-run, but the output is just displayed, not verified programmatically.
   - What's unclear: Should the fresh install test also verify that the packed tarball is clean, or is that validate-package.sh's job?
   - Recommendation: Keep concerns separate. validate-package.sh checks package cleanliness (INST-04). The smoke test checks runtime correctness (INST-01). They complement each other.

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection: `bin/validate-package.sh` -- current validation logic
- Direct codebase inspection: `.aether/.npmignore` -- current exclusion patterns
- Direct codebase inspection: `.aether/QUEEN.md` -- confirmed test artifacts present
- Direct codebase inspection: `npm pack --dry-run` output -- confirmed QUEEN.md and .tmp file in package
- Direct codebase inspection: `bin/cli.js` lines 834-838 -- HUB_EXCLUDE_FILES list
- Direct codebase inspection: `tests/e2e/test-install.sh` -- existing install test patterns
- Direct codebase inspection: `tests/e2e/test-lifecycle.sh` -- existing lifecycle test patterns
- Direct codebase inspection: `.claude/commands/ant/lay-eggs.md` -- lay-eggs flow
- Direct codebase inspection: `.claude/commands/ant/init.md` -- init flow
- Direct codebase inspection: `.aether/templates/QUEEN.md.template` -- clean template content

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all tools (bash, jq, npm) already in use in the project
- Architecture: HIGH -- patterns directly observed in existing e2e test files
- Pitfalls: HIGH -- all issues confirmed by direct inspection of files and npm pack output

**Research date:** 2026-03-19
**Valid until:** 2026-04-19 (stable domain -- shell scripts and npm packaging)
