---
phase: 05-orchestration-layer
reviewed: 2026-04-07T23:15:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - cmd/autopilot.go
  - cmd/orchestrator.go
  - cmd/orchestrator_test.go
  - pkg/agent/task_graph.go
  - pkg/agent/task_graph_test.go
  - pkg/agent/task_router.go
  - pkg/agent/task_router_test.go
  - pkg/agent/orchestrator.go
  - pkg/agent/orchestrator_test.go
  - pkg/colony/colony.go
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-04-07T23:15:00Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

Reviewed the orchestration layer implementation spanning 10 files: autopilot CLI commands, orchestrator CLI commands with tests, task graph construction and routing logic, the phase orchestrator engine, and colony state types.

The code is generally well-structured with good test coverage. The task graph uses Kahn's algorithm with proper cycle detection, the router employs a sensible two-pass approach (explicit hints then keyword fallback), and the phase orchestrator handles concurrent dispatch with dependency ordering.

Three warnings were identified: two dead-code methods on PhaseOrchestrator that are defined but never called, a silently discarded `json.Marshal` error, and a keyword matching function whose doc comment claims whole-word matching but actually uses substring matching. Three informational items cover unused dependencies in test files.

No security vulnerabilities or critical bugs were found.

## Warnings

### WR-01: Dead-code methods on PhaseOrchestrator

**File:** `pkg/agent/orchestrator.go:197-244`
**Issue:** `validateOutput` (lines 197-206) and `updateState` (lines 208-244) are defined on `PhaseOrchestrator` but never called anywhere in the codebase. Neither `cmd/` nor any other `pkg/agent/` file references them. Dead code increases maintenance burden and signals incomplete integration -- `updateState` in particular appears intended to persist orchestrator progress to `COLONY_STATE.json` after each phase run, but `Run()` never invokes it, meaning orchestrator state is never persisted by the engine itself.
**Fix:** Either integrate `updateState` into the `Run()` method (e.g., call `o.updateState(phase.ID, "completed")` after the main loop at line 97) and `validateOutput` into the result-building step, or remove both methods if they are not yet needed.

### WR-02: Silently discarded json.Marshal error

**File:** `pkg/agent/orchestrator.go:128`
**Issue:** `payloadBytes, _ := json.Marshal(payload)` discards the error return. While the payload is a simple `map[string]interface{}` with string and slice values that will never fail to marshal, this is a fragile pattern. If the payload structure evolves to include types that cannot be marshaled (channels, functions), this would silently produce an empty or nil `Payload` in the dispatched event with no indication of failure.
**Fix:**
```go
payloadBytes, err := json.Marshal(payload)
if err != nil {
    o.recordResult(task, agents, false, "", fmt.Errorf("marshal payload for task %s: %w", task.ID, err), time.Since(start))
    return nil
}
```

### WR-03: matchesKeyword doc comment claims whole-word matching but uses substring matching

**File:** `pkg/agent/task_router.go:71-77`
**Issue:** The function comment states "checks if the text contains any of the given keywords as whole words" but the implementation uses `strings.Contains(text, kw)` which matches substrings. For example, the task "investigate performance" contains the substring "test" (in "investigate"), which would match the watcher keywords at line 41 before scout keywords at line 35. The comment on line 32 acknowledges this ordering concern for the word "investigate" containing "test", but the real fix is to match whole words as the doc claims. Currently the keyword ordering on lines 35-46 is the only defense against false substring matches.
**Fix:** Either update the doc comment to accurately describe substring matching, or implement word-boundary matching:
```go
func matchesKeyword(text string, keywords ...string) bool {
    for _, kw := range keywords {
        // Match whole words only
        if text == kw || strings.HasPrefix(text, kw+" ") || strings.HasSuffix(text, " "+kw) || strings.Contains(text, " "+kw+" ") {
            return true
        }
    }
    return false
}
```

## Info

### IN-01: Unused import in orchestrator_test.go

**File:** `pkg/agent/orchestrator_test.go:12`
**Issue:** The `"github.com/calcosmic/Aether/pkg/colony"` import is used only by the helper function `makeOrchPhase`, so this is technically used. However, `"github.com/calcosmic/Aether/pkg/storage"` is imported in `orchestrator.go` (line 12) and used only in `updateState` (dead code from WR-01). If `updateState` is removed, the `storage` import becomes unused and should be cleaned up.
**Fix:** If WR-01 is resolved by removing dead code, also remove the `"github.com/calcosmic/Aether/pkg/storage"` import from `orchestrator.go`.

### IN-02: No validation of autopilot phase/status values

**File:** `cmd/autopilot.go:130-131`
**Issue:** `autopilot-update` accepts any string as a status value (e.g., `"status"` flag with no validation). Invalid statuses like `"bogus"` would be persisted. Similarly, `autopilot-init` accepts `--phases 0` without complaint. These are minor input validation gaps that would produce confusing state rather than bugs.
**Fix:** Consider validating status against known values (`"initialized"`, `"running"`, `"completed"`, `"failed"`, `"stopped"`) and phases against a minimum of 1.

### IN-03: ColonyState field StateBUILT has no transition to EXECUTING

**File:** `pkg/colony/colony.go:300-304`
**Issue:** The legal transitions map allows `StateEXECUTING -> StateBUILT` and `StateBUILT -> StateREADY` but there is no transition from `StateBUILT` to `StateEXECUTING`. This may be intentional (a built state must go back to ready first) but the asymmetry is worth noting. If a phase is built and needs re-execution, it must go BUILT -> READY -> EXECUTING, which may be the intended workflow.
**Fix:** Document the intended state machine flow if this is correct, or add `StateBUILT: {StateEXECUTING}` if re-execution should be direct.

---

_Reviewed: 2026-04-07T23:15:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
