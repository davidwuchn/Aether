# Phase 2: System Integrity - Research

**Researched:** 2026-04-07
**Domain:** Go CLI safety, data-loss prevention, smoke testing, deprecated code removal
**Confidence:** HIGH

## Summary

This phase eliminates data-loss risks from three hygiene commands (`backup-prune-global`, `temp-clean`, `data-clean`), fixes a false-positive bug in `isTestArtifact`, removes 13 deprecated Go commands and 45 deprecated shell scripts, creates a comprehensive smoke test suite for all ~254 subcommands, and standardizes error formatting. The codebase is a mature Go CLI with 943 existing tests across 43 test files. The Phase 1 infrastructure (AuditLogger, WriteBoundary) provides audit logging capability that can be wired into destructive commands.

The most technically nuanced work is the `isTestArtifact` fix. The current implementation uses fragile substring matching on signal content, which means any user pheromone containing the words "test" or "demo" gets flagged as a test artifact and deleted by `data-clean`. The fix should check the `source` field of the signal -- pheromone signals have a typed `Source` field (values observed: `"cli"`, `"user"`, `"auto"`, `"promotion"`) and the function should only flag signals whose source indicates they are test-generated.

**Primary recommendation:** Fix `isTestArtifact` by checking `source` field, add `--confirm` flags modeled on the existing `data-clean` pattern, create table-driven smoke tests iterating over all registered subcommands, and remove deprecated code in a single clean sweep.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Create a dedicated smoke test suite (Go test file) that exercises each subcommand against a temp directory with no colony state. Fast, repeatable, part of CI.
- **D-02:** Every registered subcommand gets a test case -- no panics, no silent failures, reasonable output (help text, error message, or deprecation notice). All subcommands, not just core ones.
- **D-03:** Full removal -- delete all 13 deprecated command registrations (semantic-*, survey-*, suggest-*) and the 41 deprecated shell scripts in `.aether/utils/`. Clean break, no dead code.
- **D-04:** Safety verification via grep -- confirm nothing references deprecated code before deleting.

### Claude's Discretion
- Exact smoke test structure (table-driven tests vs individual test functions)
- Error message formatting standard (prefix + description + remediation hint)
- How to handle commands that require colony state vs ones that don't
- Whether isTestArtifact fix uses a source field or a different approach
- Whether destructive commands get --confirm flag or audit logging or both
- TTY check implementation for blocking agent-initiated destructive commands

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INTG-01 | All primary `aether` commands run without error on a clean installation | Smoke test suite (D-01, D-02) -- test each of ~254 subcommands in temp dir |
| INTG-02 | No orphaned shell scripts remain in active code paths (deprecated scripts clearly marked) | Delete 45 shell scripts in `.aether/utils/` (D-03); grep verification (D-04) |
| INTG-03 | Error handling across all Go modules produces consistent, readable, actionable log output | Standardize `outputError` pattern with prefix+description+hint format |
| INTG-04 | The `isTestArtifact` function no longer false-positives on legitimate user data | Check `source` field instead of content substring matching |
| INTG-05 | `backup-prune-global` and `temp-clean` require confirmation before deleting files or create audit trail | Add `--confirm` flag modeled on `data-clean` existing pattern |
| INTG-06 | All 524+ existing tests continue passing with no regressions | Run `go test ./...` -- currently 943 tests passing (1 pre-existing failure in pkg/exchange) |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.26.1 | Runtime | Project language, verified via `go version` |
| Cobra | (from go.mod) | CLI framework | Already in use -- all commands built with cobra.Command |
| testing | stdlib | Test framework | Project convention -- 43 test files, 943 test cases |
| encoding/json | stdlib | JSON handling | All command I/O uses JSON envelopes |
| storage.AuditLogger | internal | Audit trail | Phase 1 infrastructure -- reusable for hygiene command logging |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| storage.Store | internal | File operations | All hygiene commands already use Store |
| storage.AppendJSONL | internal | Audit logging | Phase 1 -- can log destructive command executions |
| colony.PheromoneSignal | internal | Typed signal struct | Has `Source` field for isTestArtifact fix |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `--confirm` flag pattern | Interactive prompt (`confirm.Ask`) | Interactive prompts don't work in CI/agent contexts; `--confirm` is already the established pattern |
| Source field check for isTestArtifact | Separate test_signals.json tracking list | More complex, fragile -- source field is already present on all signals |
| Table-driven smoke tests | Individual test functions | Table-driven is more maintainable for 254 commands; individual tests would be ~254 functions |

**Installation:** No new packages needed. All dependencies are internal or stdlib.

## Architecture Patterns

### Recommended Project Structure

No structural changes needed. All work is within existing files:

```
cmd/
  maintenance.go          -- Safety gates for backup-prune-global, temp-clean (INTG-05)
  suggest.go              -- isTestArtifact fix + file deletion after command removal (INTG-04, INTG-02)
  deprecated_cmds.go      -- Delete file entirely (INTG-02)
  helpers.go              -- outputError enhancement for consistent formatting (INTG-03)
  smoke_test.go           -- NEW: Smoke test suite (INTG-01)

.aether/utils/            -- Delete all 45 files (INTG-02)
```

### Pattern 1: Confirmation Gate (already established in codebase)

The `data-clean` command already implements the exact pattern needed for `backup-prune-global` and `temp-clean`:

**What:** Commands that modify/delete files require explicit `--confirm` flag. Without it, they produce a dry-run preview and delete nothing.

**When to use:** Any destructive file operation.

**Example (existing pattern from `cmd/maintenance.go`):**
```go
confirm, _ := cmd.Flags().GetBool("confirm")
// ... compute what would be removed ...
if confirm && removed > 0 {
    // Actually delete
}
// Report dry_run status
outputOK(map[string]interface{}{
    "removed": reportedRemoved,
    "dry_run": !confirm,
})
```

### Pattern 2: Test Infrastructure (existing helpers)

The codebase has well-established test utilities in `cmd/testing_main_test.go`:

- `saveGlobals(t)` -- saves and restores package-level globals (store, stdout, stderr, etc.)
- `resetRootCmd(t)` -- resets rootCmd state and all subcommand flags
- `newTestStoreWithRoot(t)` -- creates temp dir with store and env vars set
- `createTestStore(dataDir)` -- simple store creation

### Pattern 3: Smoke Test Structure (recommended)

Table-driven tests iterating over all registered subcommands:

```go
func TestSmokeCommands(t *testing.T) {
    commands := rootCmd.Commands()
    for _, cmd := range commands {
        cmd := cmd // capture range variable
        t.Run(cmd.Name(), func(t *testing.T) {
            saveGlobals(t)
            resetRootCmd(t)
            var buf bytes.Buffer
            stdout = &buf
            stderr = &bytes.Buffer{}
            defer func() { stdout = os.Stdout; stderr = os.Stderr }()

            tmpDir := t.TempDir()
            dataDir := tmpDir + "/.aether/data"
            os.MkdirAll(dataDir, 0755)
            origRoot := os.Getenv("AETHER_ROOT")
            os.Setenv("AETHER_ROOT", tmpDir)
            defer os.Setenv("AETHER_ROOT", origRoot)

            s, err := createTestStore(dataDir)
            if err != nil { t.Fatal(err) }
            store = s

            rootCmd.SetArgs([]string{cmd.Name()})
            err = rootCmd.Execute()
            // Verify: no panic, output is valid JSON or help text
            if err != nil && !cmd.HasSubCommands() {
                // Error is acceptable -- just verify JSON envelope
            }
        })
    }
}
```

### Anti-Patterns to Avoid
- **Removing commands without checking references:** Must grep for all 13 command names + 45 script names before deletion (D-04)
- **Breaking TestCommandCount:** Currently asserts >= 145 commands. Removing 13 commands drops from ~254 to ~241, still well above threshold. But verify.
- **Modifying outputError signature:** All 200+ call sites use `outputError(code, message, nil)`. Changing the signature requires updating all callers or using a wrapper.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Confirmation gates | Custom prompt system | `--confirm` bool flag (already in data-clean) | Simple, CI-compatible, established pattern |
| Test isolation | Custom test framework | Existing `saveGlobals`/`resetRootCmd` helpers | Already handles 943 tests reliably |
| Audit logging | Custom log format | `storage.AppendJSONL` from Phase 1 | Structured, append-only, cross-process safe |
| File locking | Custom lock mechanism | `storage.FileLocker` | Already used by all file operations |

**Key insight:** The codebase already has all the infrastructure needed. This phase is about wiring existing patterns into uncovered commands and removing dead code.

## Common Pitfalls

### Pitfall 1: TestCommandCount Regression
**What goes wrong:** Removing 13 deprecated commands causes `TestCommandCount` to fail (currently asserts >= 145).
**Why it happens:** The test counts all registered commands. Removing commands drops the count.
**How to avoid:** After removal, verify the count is still >= 145. With ~254 commands minus 13 = ~241, this is well above threshold. But update the threshold to a more precise value (e.g., >= 240) to catch future accidental drops.
**Warning signs:** `TestCommandCount` fails immediately after deprecated command removal.

### Pitfall 2: isTestArtifact Still Uses map[string]interface{} Not Typed Struct
**What goes wrong:** The `data-clean` command reads pheromones.json as `map[string]interface{}` and passes signals to `isTestArtifact` as `map[string]interface{}`. But the actual signal type is `colony.PheromoneSignal` with a `Source` field.
**Why it happens:** The code was written before the typed struct existed, and uses untyped JSON for flexibility.
**How to avoid:** In `isTestArtifact`, extract `source` from the map: `source, _ := signal["source"].(string)`. Then check: if source is `"user"` or `"cli"`, return false immediately (never flag user-created signals).
**Warning signs:** A signal with `source: "user"` still gets flagged after the fix.

### Pitfall 3: Smoke Tests Flaky Due to Command Side Effects
**What goes wrong:** Some commands write files, create directories, or spawn processes. Running them in parallel or in shared temp dirs causes interference.
**Why it happens:** Each test must be fully isolated with its own temp directory.
**How to avoid:** Each test gets its own `t.TempDir()`. No shared state between subtests.
**Warning signs:** Intermittent test failures with "file already exists" or "store not initialized".

### Pitfall 4: Deprecated Command Tests Must Be Removed Too
**What goes wrong:** `deprecated_cmds_test.go` (492 lines) and `suggest_cmds_test.go` (258 lines) test the deprecated commands. Removing the commands without removing the tests causes compile errors.
**Why it happens:** Tests import and reference the command variables directly.
**How to avoid:** Delete both test files along with their corresponding source files.
**Warning signs:** `go test ./...` fails with "undefined" errors after removing deprecated commands.

### Pitfall 5: Pre-existing Test Failure in pkg/exchange
**What goes wrong:** `TestImportPheromonesFromRealShellXML` already fails (empty Type and Priority fields in imported signals). This is a pre-existing failure, not caused by this phase.
**Why it happens:** The test was written against old XML format that doesn't include Type/Priority fields.
**How to avoid:** Do NOT fix this test in this phase (out of scope). Document it as pre-existing. INTG-06 requires "no regressions" -- this failure predates Phase 2.
**Warning signs:** If the test starts passing after changes, investigate why.

### Pitfall 6: backup-prune-global Uses store but temp-clean Does Not
**What goes wrong:** `backup-prune-global` uses `store.BasePath()` for the backup directory, while `temp-clean` uses `storage.ResolveAetherRoot()` directly. Adding `--confirm` to both requires different nil-store handling.
**Why it happens:** They were written at different times with different patterns.
**How to avoid:** `backup-prune-global` already checks `if store == nil`. For `temp-clean`, the confirm flag logic is independent of store -- it just needs to be added before the delete loop.
**Warning signs:** `temp-clean --confirm` works but `backup-prune-global --confirm` doesn't (or vice versa).

## Code Examples

Verified patterns from codebase:

### isTestArtifact Fix (source field check)

Current code (`cmd/suggest.go:62-84`):
```go
func isTestArtifact(signal map[string]interface{}) bool {
    id, _ := signal["id"].(string)
    contentRaw := signal["content"]
    content := ""
    if contentMap, ok := contentRaw.(map[string]interface{}); ok {
        content, _ = contentMap["text"].(string)
    } else if contentStr, ok := contentRaw.(string); ok {
        content = contentStr
    }
    if strings.HasPrefix(id, "test_") || strings.HasPrefix(id, "demo_") {
        return true
    }
    lower := strings.ToLower(content)
    if strings.Contains(lower, "test signal") || strings.Contains(lower, "demo pattern") {
        return true
    }
    return false
}
```

Recommended fix:
```go
func isTestArtifact(signal map[string]interface{}) bool {
    // User-created signals (source: "user" or "cli") are never test artifacts,
    // regardless of content. Only auto-generated or system signals can be artifacts.
    source, _ := signal["source"].(string)
    switch source {
    case "user", "cli":
        return false
    }

    id, _ := signal["id"].(string)
    contentRaw := signal["content"]
    content := ""
    if contentMap, ok := contentRaw.(map[string]interface{}); ok {
        content, _ = contentMap["text"].(string)
    } else if contentStr, ok := contentRaw.(string); ok {
        content = contentStr
    }

    // Only flag by content/prefix for non-user sources (auto, system, etc.)
    if strings.HasPrefix(id, "test_") || strings.HasPrefix(id, "demo_") {
        return true
    }
    lower := strings.ToLower(content)
    if strings.Contains(lower, "test signal") || strings.Contains(lower, "demo pattern") {
        return true
    }
    return false
}
```

### Confirmation Gate Pattern (backup-prune-global)

Model on existing `data-clean` pattern:
```go
var backupPruneGlobalCmd = &cobra.Command{
    Use:   "backup-prune-global",
    Short: "Prune old backups to a cap",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... existing store/backup-dir logic ...
        confirm, _ := cmd.Flags().GetBool("confirm")

        // ... compute files to prune (existing sort logic) ...

        if !confirm {
            // Dry-run: report what would be pruned, delete nothing
            outputOK(map[string]interface{}{
                "pruned": 0,
                "kept":   len(files),
                "would_prune": pruneCount,
                "dry_run": true,
            })
            return nil
        }

        // Only delete when --confirm is explicit
        for i := 0; i < pruneCount; i++ {
            // ... existing delete logic ...
        }

        outputOK(map[string]interface{}{
            "pruned": pruneCount,
            "kept":   cap,
        })
        return nil
    },
}
```

### Error Format Enhancement

Current pattern (inconsistent):
```go
outputError(1, "no store initialized", nil)                     // short
outputError(2, fmt.Sprintf("failed to save: %v", err), nil)     // wrapped error
outputError(1, "COLONY_STATE.json not found", nil)              // file reference
```

Recommended standardized format:
```go
outputError(code, "prefix: description. remediation hint", nil)
// Examples:
outputError(1, "store: no store initialized. Run 'aether init' first.", nil)
outputError(2, fmt.Sprintf("state: failed to save %s: %v", path, err), nil)
```

However, changing all 200+ call sites is high-risk for INTG-06 (no regressions). A safer approach is:
1. Define the convention (INTG-03 deliverable)
2. Apply to NEW/CHANGED commands in this phase only
3. Existing commands get updated incrementally in later phases

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No confirmation on destructive commands | `--confirm` flag pattern (data-clean) | Phase 1 era | backup-prune-global and temp-clean need this pattern |
| Content-based test artifact detection | Source-based detection | This phase | Prevents false positives on user pheromones |
| Deprecated commands as stubs | Full removal | This phase | Cleaner codebase, fewer commands to maintain |

**Deprecated/outdated:**
- 13 Go commands in `deprecated_cmds.go` and `suggest.go` -- remove entirely
- 45 shell scripts in `.aether/utils/` -- remove entirely
- 750 lines of deprecated command tests -- remove with the commands

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Pheromone signal `source` values are limited to `"user"`, `"cli"`, `"auto"`, `"promotion"` | isTestArtifact fix | If other source values exist (e.g., `"system"`, `"test"`), they might need different handling |
| A2 | All 45 files in `.aether/utils/` are deprecated and unreferenced by active code | Deprecated cleanup | If any are still referenced, removal breaks something |
| A3 | The pre-existing `TestImportPheromonesFromRealShellXML` failure is out of scope | INTG-06 | If it's expected to be fixed in this phase, INTG-06 success criterion is harder to meet |
| A4 | Smoke tests for 254 commands will complete in reasonable time (<60s) | Smoke test design | If some commands are slow (network calls, long operations), tests need timeouts/skipping |
| A5 | `TestCommandCount` threshold (>=145) won't need updating after removing 13 commands | Deprecated removal | 254-13=241 > 145, so threshold is fine as-is |

## Open Questions

1. **Should error formatting changes apply to ALL existing commands or just new/changed ones?**
   - What we know: There are 200+ `outputError` call sites across the codebase.
   - What's unclear: Whether INTG-03 means "define a standard and apply everywhere" or "define a standard and apply going forward."
   - Recommendation: Define the standard, apply to commands changed in this phase (maintenance.go, helpers.go), document the convention for future phases. Mass-updating all 200+ sites is high-risk for INTG-06.

2. **Should smoke tests skip commands that require network access or long-running operations?**
   - What we know: Some commands like `binary-download` likely require network. `serve` starts a server.
   - What's unclear: Which specific commands need special handling.
   - Recommendation: Identify commands with side effects (serve, binary-download) and mark them as skip-with-note in the smoke test table. Verify they don't panic on `--help` instead.

3. **Should the `errorPatternCheckCmd` (already deprecated via cobra.Deprecated field) be removed too?**
   - What we know: It uses `Deprecated: "use error-flag-pattern instead"` in cobra, not the `newDeprecatedCmd` helper.
   - What's unclear: Whether it counts as one of the "13 deprecated commands."
   - Recommendation: It's in `cmd/error_cmds.go` alongside active commands. Either remove it or convert to the `newDeprecatedCmd` pattern first, then remove. Check if any callers reference it.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified)

All work is within the Go codebase using stdlib and internal packages. No external tools, services, or CLIs needed beyond `go test`.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none |
| Quick run command | `go test ./cmd/ -run TestSmoke -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INTG-01 | All commands run without panic on fresh install | smoke | `go test ./cmd/ -run TestSmoke -count=1` | Wave 0: `cmd/smoke_test.go` |
| INTG-02 | No orphaned deprecated code | unit | `go test ./cmd/ -run TestNoDeprecatedCommands -count=1` | Wave 0: `cmd/deprecated_cmds_test.go` (modify) |
| INTG-03 | Error messages follow consistent format | unit | `go test ./cmd/ -run TestErrorFormat -count=1` | Wave 0: `cmd/helpers_test.go` (modify) |
| INTG-04 | isTestArtifact doesn't false-positive on user signals | unit | `go test ./cmd/ -run TestIsTestArtifact -count=1` | Wave 0: `cmd/maintenance_test.go` (new or modify) |
| INTG-05 | Destructive commands require --confirm | unit | `go test ./cmd/ -run TestBackupPrune|TestTempClean -count=1` | Wave 0: `cmd/maintenance_test.go` (new or modify) |
| INTG-06 | All existing tests pass | regression | `go test ./... -count=1` | Existing: 43 test files |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run "TestSmoke|TestIsTestArtifact|TestBackupPrune|TestTempClean" -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/smoke_test.go` -- covers INTG-01 (new file, table-driven smoke tests for all commands)
- [ ] `cmd/maintenance_test.go` -- covers INTG-04 and INTG-05 (new file or extend existing)
- [ ] `cmd/deprecated_cmds_test.go` -- will be DELETED with deprecated commands; replacement test for INTG-02 needed
- [ ] `cmd/suggest_cmds_test.go` -- will be DELETED with deprecated suggest commands
- [ ] Framework install: not needed (stdlib)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | yes | `isTestArtifact` source field check (INTG-04) -- prevents data loss from false positive classification |
| V6 Cryptography | no | N/A |

### Known Threat Patterns for Go CLI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Accidental data deletion | Tampering | `--confirm` flag (INTG-05), audit logging via AppendJSONL |
| Test artifact false positive | Tampering | Source field validation (INTG-04) |
| Deprecated code exploitation | Tampering | Full removal (INTG-02) |

## Sources

### Primary (HIGH confidence)
- Codebase grep and file reads -- all code patterns, function signatures, test infrastructure verified directly
- `cmd/maintenance.go` -- current isTestArtifact, data-clean, backup-prune-global, temp-clean
- `cmd/deprecated_cmds.go` -- 8 deprecated semantic/survey commands
- `cmd/suggest.go` -- 5 deprecated suggest commands, isTestArtifact function
- `pkg/colony/pheromones.go` -- PheromoneSignal struct with Source field
- `cmd/helpers.go` -- outputOK/outputError functions
- `cmd/testing_main_test.go` -- test infrastructure helpers
- `pkg/storage/audit.go` -- AuditLogger from Phase 1

### Secondary (MEDIUM confidence)
- `TestCommandCount` threshold (>=145) -- verified by running `go test`
- Pre-existing test failure in `pkg/exchange` -- verified by running tests

### Tertiary (LOW confidence)
- Source field values ("user", "cli", "auto", "promotion") -- observed in code but not exhaustively verified as the complete set

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all internal, verified via codebase
- Architecture: HIGH - patterns directly observed in existing code
- Pitfalls: HIGH - most derived from code analysis and test runs
- isTestArtifact fix: MEDIUM - source field approach is sound but source value list not exhaustively verified (A1)

**Research date:** 2026-04-07
**Valid until:** 30 days (stable domain, internal codebase)
