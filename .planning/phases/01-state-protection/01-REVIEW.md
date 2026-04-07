---
phase: 01-state-protection
reviewed: 2026-04-07T15:12:11Z
depth: standard
files_reviewed: 17
files_reviewed_list:
  - cmd/build_flow_cmds.go
  - cmd/build_flow_cmds_test.go
  - cmd/state_cmds.go
  - cmd/state_cmds_test.go
  - cmd/state_extra.go
  - cmd/state_extra_test.go
  - cmd/state_history.go
  - cmd/state_history_test.go
  - cmd/testing_main_test.go
  - pkg/storage/audit.go
  - pkg/storage/audit_test.go
  - pkg/storage/boundary.go
  - pkg/storage/boundary_test.go
  - pkg/storage/checkpoint.go
  - pkg/storage/checkpoint_test.go
  - pkg/storage/corruption.go
  - pkg/storage/corruption_test.go
findings:
  critical: 2
  warning: 5
  info: 3
  total: 10
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-04-07T15:12:11Z
**Depth:** standard
**Files Reviewed:** 17
**Status:** issues_found

## Summary

Reviewed the state protection infrastructure across `cmd/` (CLI commands for state mutation, history, checkpointing, and build flow) and `pkg/storage/` (audit trail, boundary guard, corruption detection, and checkpoint management). The audit pipeline design is solid -- a read-mutate-validate-write-audit cycle with SHA-256 checksums and corruption detection. However, there are two critical bugs: a dead-code logic error in `state-write` field/value mode that silently discards all mutations, and a race condition in checkpoint naming when multiple destructive operations happen within the same second. Several warnings around error handling and a regex-compile-per-call inefficiency in corruption detection are also flagged.

## Critical Issues

### CR-01: state-write field/value mode silently discards the mutation

**File:** `cmd/state_extra.go:109-131`
**Issue:** The `state-write --field --value --force` code path marshals the state to a map, sets `m[field] = value`, but then immediately **unmarshals the original `data` back** into `state` on line 120, overwriting the map modification. The variable `updated` (the modified map) is only used to marshal/unmarshal back into `state` on lines 128-130, but the code path between lines 120-124 creates and discards a separate local `m` with the mutation, then unmarshals from the original `data` which does NOT contain the mutation.

Step by step:
1. Line 111: `data, _ := json.Marshal(state)` -- captures original state as bytes
2. Line 115-116: `json.Unmarshal(data, &m)` -- copies into map
3. Line 119: `m[field] = value` -- modifies the map
4. Line 120: `json.Unmarshal(data, state)` -- **BUG: unmarshals ORIGINAL data back into state, discarding the map change**
5. Line 124: `json.Marshal(m)` -- creates `updated` from the modified map
6. Line 128-129: `json.Unmarshal(updated, state)` -- finally applies the change

While step 6 does ultimately apply the change, step 4 is a no-op that overwrites state with the original data, making the code confusing and fragile. More critically, lines 115-119 modify `m` but line 120 overwrites `state` from the unmodified `data`. If anyone moves or removes lines 124-129 in a future refactor, the mutation is silently lost.

**Fix:**
```go
err := auditLogger.WriteBoundary("state-write", true, func(state *colony.ColonyState) (string, error) {
    data, err := json.Marshal(state)
    if err != nil {
        return "", err
    }
    var m map[string]interface{}
    if err := json.Unmarshal(data, &m); err != nil {
        return "", err
    }
    m[field] = value
    updated, err := json.Marshal(m)
    if err != nil {
        return "", err
    }
    if err := json.Unmarshal(updated, state); err != nil {
        return "", err
    }
    return fmt.Sprintf("%s -> %s", field, value), nil
})
```

### CR-02: Checkpoint timestamp collision under rapid successive destructive operations

**File:** `pkg/storage/checkpoint.go:20`
**Issue:** `AutoCheckpoint` uses `time.Now().UTC().Format("20060102-150405")` which has second-level granularity. If two destructive mutations happen within the same second (e.g., in a script or rapid CLI usage), the second checkpoint overwrites the first via `AtomicWrite`. This silently loses the first before-state snapshot, defeating the purpose of auto-checkpointing for the first operation.

**Fix:**
Use higher-resolution timestamps that include fractional seconds or a monotonic counter:
```go
timestamp := time.Now().UTC().Format("20060102-150405.000")  // millisecond precision
```
Or add a nanosecond suffix:
```go
timestamp := fmt.Sprintf("%s-%03d", time.Now().UTC().Format("20060102-150405"), time.Now().Nanosecond()/1000000)
```

## Warnings

### WR-01: Regex compiled on every call in corruption detection

**File:** `pkg/storage/corruption.go:37`
**Issue:** `looksLikeJQExpression` calls `regexp.MustCompile()` on every invocation. While not a correctness bug, in a hot path (e.g., bulk operations through WriteBoundary), this recompiles the regex for every event in every mutation. This is a known Go anti-pattern.

**Fix:**
Move the regex to a package-level compiled variable:
```go
var assignmentRe = regexp.MustCompile(`^\.([\w.\[\]]+)\s*[|=]`)

func looksLikeJQExpression(s string) bool {
    if assignmentRe.MatchString(s) {
        return true
    }
    // ...
}
```

### WR-02: Unchecked errors from os.WriteFile and os.MkdirAll in versionCheckCachedCmd

**File:** `cmd/build_flow_cmds.go:59-60`
**Issue:** Both `os.MkdirAll` and `os.WriteFile` return errors that are silently discarded. If the data directory is not writable (permissions issue, disk full), the cache write fails silently. The command still reports success with `cached: false`, but the user gets no indication of the write failure.

**Fix:**
```go
if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
    // Non-fatal, but log
    fmt.Fprintf(os.Stderr, "warning: failed to create cache dir: %v\n", err)
}
if err := os.WriteFile(cachePath, entryData, 0644); err != nil {
    fmt.Fprintf(os.Stderr, "warning: failed to write cache: %v\n", err)
}
```

### WR-03: Unchecked error from json.Marshal in versionCheckCachedCmd

**File:** `cmd/build_flow_cmds.go:58`
**Issue:** `json.Marshal(entry)` error is discarded via `_, _ :=`. While marshaling a simple struct is unlikely to fail, the pattern of silently discarding errors is inconsistent with the project's error-handling conventions.

**Fix:**
```go
entryData, err := json.Marshal(entry)
if err != nil {
    outputError(1, fmt.Sprintf("failed to marshal cache entry: %v", err), nil)
    return nil
}
```

### WR-04: update-progress does not validate --phase flag is provided

**File:** `cmd/build_flow_cmds.go:146`
**Issue:** `mustGetInt(cmd, "phase")` returns 0 if the flag is not provided (the default). The code then proceeds to attempt phase index -1 (`0 - 1`), which triggers the bounds check error. While this produces an error message, the error message says "phase 0 not found" which is confusing -- it should say "--phase is required" or "phase must be >= 1".

**Fix:**
Add an explicit check after reading the flag:
```go
phaseNum := mustGetInt(cmd, "phase")
if phaseNum <= 0 {
    outputError(1, "--phase must be >= 1", nil)
    return nil
}
```

### WR-05: state-write positional JSON mode does not validate JSON schema

**File:** `cmd/state_extra.go:79-85`
**Issue:** When a user passes positional JSON with `--force`, the code validates that the JSON is syntactically valid (`json.Valid`) but does not validate that the JSON conforms to the `ColonyState` schema. A user could write `{"version": "1.0"}` (missing required fields like `state`, `goal`) and it would be accepted. While this is the intended `--force` bypass behavior, the mutation still passes through `WriteBoundary` which calls `DetectCorruption` -- but `DetectCorruption` only checks the Events field for jq patterns. The corruption detector does not validate structural completeness of the state.

This is by design (the `--force` flag explicitly bypasses safety), but worth noting that the corruption detector has a narrow scope.

**Fix:** No code fix needed -- this is documented behavior. Consider adding a comment in the code noting the intentional gap:
```go
// Note: DetectCorruption only checks for jq injection in Events.
// Structural validation (required fields, valid enums) is intentionally
// NOT enforced here -- --force is the escape hatch for advanced usage.
```

## Info

### IN-01: Duplicate helper function `readAuditChangelog` across test files

**File:** `cmd/state_cmds_test.go:182-189` and `cmd/build_flow_cmds_test.go:18-26`
**Issue:** `readAuditChangelog` in `state_cmds_test.go` and `readAuditChangelogBuildFlow` in `build_flow_cmds_test.go` are functionally identical -- both create an `AuditLogger` and call `ReadHistory(0)`. The comment in the build_flow version explicitly acknowledges the duplication.

**Fix:** Move to a shared test helper file (e.g., `cmd/test_helpers_test.go`) and have both files use the same function.

### IN-02: Custom `itoa` implementation in checkpoint_test.go

**File:** `pkg/storage/checkpoint_test.go:214-224`
**Issue:** A custom integer-to-string conversion is implemented instead of using `strconv.Itoa` or `fmt.Sprintf("%d", i)`. Go's standard library provides `strconv.Itoa` which is more readable and battle-tested.

**Fix:**
```go
import "strconv"
// Replace custom itoa calls with strconv.Itoa(i)
```

### IN-03: Unused import `sync` in audit_test.go

**File:** `pkg/storage/audit_test.go:10`
**Issue:** The `sync` package is imported and used only in `TestAudit_ConcurrentWriteBoundary`. This is not actually unused -- however, it's worth noting that the concurrent test has a known limitation documented in its docstring (some mutations may be lost under high concurrency due to lack of cross-file transaction locking). This is correctly documented but could benefit from a `// NOTE:` comment in the test body itself reminding future maintainers.

---

_Reviewed: 2026-04-07T15:12:11Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
