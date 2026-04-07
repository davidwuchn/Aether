---
phase: 02-system-integrity
reviewed: 2026-04-07T20:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - cmd/suggest.go
  - cmd/maintenance.go
  - cmd/helpers.go
  - cmd/maintenance_test.go
  - cmd/chamber_suggest_maintenance_test.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 02: Code Review Report (suggest/maintenance/chamber)

**Reviewed:** 2026-04-07T20:00:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Reviewed 5 files implementing the test artifact detection (`suggest.go`), three maintenance subcommands (`maintenance.go`), shared output helpers (`helpers.go`), and their tests. The code is well-structured with good defensive patterns: nil store guards, dry-run-by-default for destructive operations, and proper error message conventions. Test coverage is thorough with table-driven tests for `isTestArtifact` and confirmation gate tests for all three maintenance commands.

Two warnings were found: the `data-clean` command mutates the pheromones file format through a `map[string]interface{}` round-trip that loses struct-level JSON semantics, and silent error swallowing on file deletion in `temp-clean`.

## Warnings

### WR-01: data-clean re-serializes pheromones.json as map, losing struct-level JSON semantics

**File:** `/Users/callumcowie/repos/Aether/cmd/maintenance.go:40-76`
**Issue:** The `data-clean` command reads `pheromones.json` into `map[string]interface{}`, filters the signals array, then writes it back via `store.SaveJSON`. This means the file is re-serialized from a generic map rather than the `colony.PheromoneFile` struct. Consequences:

1. **Field ordering changes.** `json.MarshalIndent` on a map produces alphabetically-sorted keys. The original struct-serialized file has fields in struct-definition order. After `data-clean`, the file will have a different key order (e.g., `"active"` before `"content"` before `"id"`), making diffs noisy.
2. **`omitempty` semantics are lost.** The `PheromoneSignal` struct uses `omitempty` on `expires_at`, `strength`, `reason`, `content_hash`, `reinforcement_count`, `archived_at`, `tags`, and `scope`. When round-tripped through `map[string]interface{}`, Go's JSON encoder will include zero-value fields (e.g., `null` for pointer fields, `0` for int, `false` for bool) that `omitempty` would have omitted. This inflates the file and changes its structure.
3. **Version/colony_id handling.** The top-level `PheromoneFile` struct also uses `omitempty` for `version` and `colony_id`. These would also lose their omission behavior.

While the data remains functionally readable (the test at `maintenance_test.go:366-370` confirms this), the file format drifts from what the rest of the codebase expects to produce. Other commands that write pheromones (e.g., `pheromone-write`) use `colony.PheromoneFile`, so running `data-clean` would make the file inconsistent with freshly-written pheromones files.

**Fix:** Deserialize into `colony.PheromoneFile`, filter signals, then re-serialize from the struct:

```go
// Replace lines 40-76 in maintenance.go:
var pheromonesFile colony.PheromoneFile
if err := json.Unmarshal(data, &pheromonesFile); err != nil {
    outputError(1, fmt.Sprintf("data-clean: failed to parse pheromones.json: %v. Check file is valid JSON.", err), nil)
    return nil
}

var kept []colony.PheromoneSignal
removed := 0
for _, signal := range pheromonesFile.Signals {
    signalMap := signalToMap(signal) // helper to convert struct to map for isTestArtifact
    if isTestArtifact(signalMap) {
        removed++
        continue
    }
    kept = append(kept, signal)
}

if confirm && removed > 0 {
    pheromonesFile.Signals = kept
    if err := store.SaveJSON("pheromones.json", pheromonesFile); err != nil {
        outputError(2, fmt.Sprintf("data-clean: failed to save pheromones.json: %v. Check disk space and permissions.", err), nil)
        return nil
    }
}
```

This requires adding a `colony` import to `maintenance.go` and a helper function to convert `colony.PheromoneSignal` to `map[string]interface{}` for the `isTestArtifact` check, or alternatively refactoring `isTestArtifact` to accept the struct directly.

### WR-02: Silent error swallowing on os.Remove in temp-clean

**File:** `/Users/callumcowie/repos/Aether/cmd/maintenance.go:231-233`
**Issue:** The `temp-clean` command silently ignores errors from `os.Remove(path)`:

```go
for _, path := range removable {
    os.Remove(path)
}
```

If a file is deleted between the directory listing and the removal attempt (TOCTOU race), or if permissions prevent deletion, the error is silently swallowed. The command then reports `cleaned: N` where N is the number of files it *attempted* to clean, not the number it actually cleaned. This gives the user a false sense of completion.

Compare with `backup-prune-global` (line 170-172) which at least logs the error:

```go
if err := os.Remove(filepath.Join(backupDir, files[i].name)); err != nil {
    log.Printf("backup: failed to remove backup %s: %v", files[i].name, err)
}
```

**Fix:** Track actual deletions and log failures:

```go
cleaned := 0
for _, path := range removable {
    if err := os.Remove(path); err != nil {
        log.Printf("temp-clean: failed to remove %s: %v", path, err)
        continue
    }
    cleaned++
}

outputOK(map[string]interface{}{
    "cleaned": cleaned,
})
```

## Info

### IN-01: Redundant defer os.RemoveAll with t.TempDir()

**File:** `/Users/callumcowie/repos/Aether/cmd/chamber_suggest_maintenance_test.go:48,90,126,149,181,213,233,261,298,337,380,414,441,482`
**Issue:** Multiple tests call `t.TempDir()` (which auto-cleans after the test) and then also `defer os.RemoveAll(tmpDir)`. The explicit cleanup is redundant. This is a pre-existing codebase pattern (present in 100+ test locations) so it is noted for awareness only and not actionable in isolation.

**Fix:** Remove the redundant `defer os.RemoveAll(tmpDir)` calls. Or, if this is intentionally defensive, no change needed.

### IN-02: Test helper defined in unrelated test file

**File:** `/Users/callumcowie/repos/Aether/cmd/chamber_suggest_maintenance_test.go:19-37`
**Issue:** `newTestStoreWithRoot` is defined in `chamber_suggest_maintenance_test.go` but is also used by `maintenance_test.go` (lines 238, 286). This creates a hidden dependency between test files -- if someone renames or moves `chamber_suggest_maintenance_test.go`, `maintenance_test.go` breaks. The sibling helper `newTestStore` lives in `write_cmds_test.go`, which has the same cross-file dependency issue.

Since all files are in `package cmd`, this compiles fine, but the helper would be better placed in a shared test file (e.g., `testing_helpers_test.go`) or in `testing_main_test.go` alongside `saveGlobals` and `resetRootCmd`.

**Fix:** Move `newTestStoreWithRoot` to `testing_main_test.go` or a dedicated `test_helpers_test.go` file.

### IN-03: data-clean dry-run reports 0 removed instead of would_remove count

**File:** `/Users/callumcowie/repos/Aether/cmd/maintenance.go:79-83`
**Issue:** In dry-run mode, `data-clean` reports `removed: 0` instead of a `would_remove` count. Compare with `backup-prune-global` (line 160-161) which reports `would_prune: N` and `temp-clean` (line 224-225) which reports `would_clean: N`. The user has no way to know how many artifacts would be removed without running with `--confirm`.

```go
// Current behavior:
reportedRemoved := removed
if !confirm {
    reportedRemoved = 0  // User can't see what would be removed
}
```

**Fix:** Add a `would_remove` field in dry-run mode, consistent with the other commands:

```go
if !confirm {
    outputOK(map[string]interface{}{
        "scanned":      true,
        "removed":      0,
        "would_remove": removed,
        "dry_run":      true,
    })
    return nil
}
```

---

_Reviewed: 2026-04-07T20:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
