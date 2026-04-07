---
phase: 03-build-depth-controls
reviewed: 2026-04-07T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - pkg/colony/colony.go
  - pkg/colony/depth.go
  - pkg/colony/depth_test.go
  - cmd/context.go
  - cmd/context_test.go
  - cmd/status.go
  - cmd/colony_cmds.go
  - cmd/state_cmds.go
  - cmd/init_cmd.go
  - cmd/state_cmds_test.go
  - cmd/init_cmd_test.go
  - .aether/docs/command-playbooks/build-context.md
  - .aether/docs/command-playbooks/build-wave.md
  - .aether/docs/command-playbooks/build-verify.md
  - .aether/docs/command-playbooks/build-prep.md
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-04-07
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Reviewed Go source code for the build-depth-controls feature (D-03 through D-08), which adds a four-tier depth system (light/standard/deep/full) controlling build thoroughness, worker spawning, context budgets, and verification intensity. The implementation spans the `colony` package (types, depth budget), CLI commands (init, state-mutate, colony-depth, context-budget, status), and build playbook documentation.

The core logic is solid: depth validation is enforced consistently at all entry points (init, state-mutate field mode, state-mutate expression mode, context-budget, colony-depth set), tests cover all depth levels and invalid inputs, and the playbook docs correctly gate specialist agents on depth thresholds. No security vulnerabilities or crashes found.

Issues are limited to edge-case robustness in expression-mode depth validation, a potential panic in the status dashboard, and minor code quality observations.

## Warnings

### WR-01: Expression-mode depth validation does not block the write -- partial state corruption window

**File:** `cmd/state_cmds.go:170-178`
**Issue:** In `executeExpression`, after applying all sub-expressions to the raw JSON bytes, the code unmarshals into a `ColonyState` struct to validate typed fields (including `colony_depth`). If validation fails, it calls `outputError` and returns nil. However, by this point the raw JSON data has already been mutated in memory -- the function simply never writes it to disk. This is correct behavior for the happy path, but there is a subtle concern: if `json.Unmarshal` fails (line 173 returns an error), the validation is silently skipped entirely, and any invalid depth value would be written to disk without being caught.

**Fix:**
```go
var validateState colony.ColonyState
if err := json.Unmarshal(data, &validateState); err != nil {
    // If we can't unmarshal, the JSON is malformed -- don't write it
    outputError(1, fmt.Sprintf("expression produced invalid JSON: %v", err), nil)
    return nil
}
if validateState.ColonyDepth != "" && !validateState.ColonyDepth.Valid() {
    outputError(1, fmt.Sprintf("invalid colony depth %q: must be light, standard, deep, or full", validateState.ColonyDepth), nil)
    return nil
}
```

### WR-02: Nil pointer dereference if `state.Goal` is nil in `renderDashboard`

**File:** `cmd/status.go:59`
**Issue:** `renderDashboard` dereferences `state.Goal` with `*state.Goal` without checking for nil. While the caller at line 32-34 does check `state.Goal == nil` and returns early, `renderDashboard` is a public function exported from the package. Any caller that bypasses the nil check (or calls `renderDashboard` directly with a state that has a nil Goal) would panic.

**Fix:**
```go
goal := "No goal set"
if state.Goal != nil {
    goal = *state.Goal
}
```

### WR-03: `initDepth` flag is a package-level mutable variable without proper reset in tests

**File:** `cmd/init_cmd.go:22` and `cmd/init_cmd_test.go`
**Issue:** `initDepth` is declared as a `var` at package level (line 22). The cobra flag `--depth` binds to it via `StringVar` (line 203). In tests, each test calls `saveGlobals(t)` and `resetRootCmd(t)`, but there is no explicit reset of `initDepth` between tests. If cobra's flag parsing sets `initDepth` in one test (e.g., `TestInitCmd_DepthLight` sets it to `"light"`), and a subsequent test (e.g., `TestInitCmd_DepthDefault`) expects the default empty string, the residual value could cause flaky test behavior if `resetRootCmd` does not fully clear flag state.

**Fix:** Add `initDepth = ""` to the test setup or to the `saveGlobals`/`resetRootCmd` helper:
```go
func resetInitDepth(t *testing.T) {
    t.Helper()
    initDepth = ""
}
```
Call this at the start of each init test, or include it in `resetRootCmd`.

## Info

### IN-01: Redundant `COLONY_STATE.json` load in `prContextCmd`

**File:** `cmd/context.go:384-385, 408, 444, 483, 510`
**Issue:** `prContextCmd` loads `COLONY_STATE.json` into `colState` five separate times within the same `RunE` function (lines 384, 408, 444, 483, 510). The first load populates instincts, and subsequent loads re-read the same file for different fields. This is not a bug (the state doesn't change during execution), but it is wasteful I/O and could be consolidated into a single load.

**Fix:** Load `COLONY_STATE.json` once at the top of the function and reuse the `colState` variable throughout.

### IN-02: `readQUEENMd` parses any `## Wisdom` or `## Patterns` section header broadly

**File:** `cmd/context.go:1087-1090`
**Issue:** The section matching logic uses `strings.HasPrefix(sectionName, "Wisdom") || strings.HasPrefix(sectionName, "Patterns")`, which would match sections like `## Wisdom-Related Notes` or `## Patterns of Behavior` even if they are not the canonical wisdom/patterns sections. This is unlikely to cause issues in practice but could produce unexpected key-value pairs.

**Fix:** Use exact matching or a more specific pattern:
```go
inWisdomSection = sectionName == "Wisdom" || sectionName == "Patterns"
```

### IN-03: `formatTimestamp` uses string manipulation instead of time parsing

**File:** `cmd/status.go:334-352`
**Issue:** `formatTimestamp` manually strips timezone suffixes and truncates via string operations rather than parsing with `time.Parse`. This works for RFC3339 timestamps but would produce incorrect results for other formats. Since the codebase consistently uses RFC3339, this is informational only.

**Fix:** Consider using `time.Parse(time.RFC3339, ts)` and `parsed.Format("2006-01-02 15:04")` for robustness.

---

_Reviewed: 2026-04-07_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
