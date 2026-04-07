---
phase: 06-branching-worktree-discipline
reviewed: 2026-04-08T12:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - cmd/worktree.go
  - cmd/worktree_test.go
  - pkg/colony/colony.go
findings:
  critical: 2
  warning: 4
  info: 3
  total: 9
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-04-08T12:00:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Reviewed the worktree management commands (allocate, list, orphan-scan, merge-back) in `cmd/worktree.go`, their tests in `cmd/worktree_test.go`, and the colony types in `pkg/colony/colony.go`. The code is well-structured with good test coverage for branch validation, state management, and merge-back gates. However, two critical bugs were found: a `defer` inside a nested loop that leaks context resources and delays cancellation, and a shared `gitCtx` context that may expire before its last use. Several warning-level issues around error handling and environment variable restoration were also identified.

## Critical Issues

### CR-01: defer inside nested loop leaks context cancellation and accumulates deferred calls

**File:** `cmd/worktree.go:567-568`
**Issue:** `defer checkCancel()` is called inside a nested loop (for each modified file, for each worktree path). In Go, `defer` runs when the *enclosing function* returns, not when the loop iteration ends. This means:
1. All context cancellation functions accumulate and only fire when `checkClashesForWorktree` returns -- effectively none of the inner contexts are cancelled until the function exits.
2. If the outer context (line 510) has a deadline, all inner contexts inherit from `context.Background()` with a fresh `GitTimeout`, so they may outlive the outer function's intended scope.
3. The accumulation of deferred calls is O(files * worktrees), which could be significant for large repos.

**Fix:** Cancel the context inline instead of deferring:
```go
for _, file := range modifiedFiles {
    if file == "" {
        continue
    }
    for _, wtPath := range allPaths {
        wtReal, err := filepath.EvalSymlinks(wtPath)
        if err != nil {
            wtReal = wtPath
        }
        if wtReal == entryReal {
            continue
        }

        checkCtx, checkCancel := context.WithTimeout(context.Background(), GitTimeout)
        checkOut, checkErr := exec.CommandContext(checkCtx, "git", "-C", wtPath, "diff", "--name-only", baseBranch, "--", file).Output()
        checkCancel() // cancel immediately after use
        if checkErr != nil {
            continue
        }
        if strings.TrimSpace(string(checkOut)) != "" {
            clashes = append(clashes, file)
            break
        }
    }
}
```

### CR-02: Branch deletion at line 718 reuses gitCtx which may have expired

**File:** `cmd/worktree.go:680,718`
**Issue:** `gitCtx` is created at line 680 with `GitTimeout` and is used for the checkout (line 684), merge (line 696), and then again for branch deletion at line 718. The checkout and merge commands both consume wall-clock time. If their combined execution approaches or exceeds `GitTimeout`, the branch deletion at line 718 will fail immediately due to context deadline exceeded. The error is silently swallowed (line 719-722), so the branch is never cleaned up.

**Fix:** Use the separate `pruneCtx` (already created at line 708) for the branch deletion:
```go
// 5c: Delete branch (tolerate "not found")
branchDelErr := exec.CommandContext(pruneCtx, "git", "-C", aetherRoot, "branch", "-d", entry.Branch).Run()
```
Or create a fresh context for the branch deletion to ensure it has a full timeout budget.

## Warnings

### WR-01: os.Setenv defer restores to potentially wrong value in tests

**File:** `cmd/worktree_test.go:223-224,293-294,384-385,429-430,559,628,1081,1108,1242,1354,1574,1773`
**Issue:** The pattern `defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))` captures the current value of `AETHER_ROOT` at defer time. If a test function is called when `AETHER_ROOT` is already set (from a prior test or the environment), the deferred restore resets to that prior value. But when multiple tests run in sequence that each set and defer-restore `AETHER_ROOT`, the deferred restores execute in LIFO order. If test A sets `X=/a`, defers restore to original `/old`, then test B sets `X=/b`, defers restore to `/a`, the restores happen as: restore to `/a` (B's defer), then restore to `/old` (A's defer). This works correctly *within a single test function*, but the `newWorktreeTestStore` helper at line 540 calls `os.Setenv` without a corresponding defer, relying on the caller to do the defer. If a caller forgets (as in `TestWorktreeAllocateAgentPhase` at line 559 which does have it), subsequent tests may see a stale `AETHER_ROOT`.

The `newWorktreeTestStore` helper sets `AETHER_ROOT` (line 540) but does not defer its restoration, which is a footgun for callers.

**Fix:** Have `newWorktreeTestStore` use `t.Cleanup` for reliable restoration:
```go
func newWorktreeTestStore(t *testing.T, stateJSON string) (*storage.Store, string) {
    t.Helper()
    tmpDir := t.TempDir()
    dataDir := tmpDir + "/.aether/data"
    os.MkdirAll(dataDir, 0755)
    os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(stateJSON), 0644)
    oldRoot := os.Getenv("AETHER_ROOT")
    os.Setenv("AETHER_ROOT", tmpDir)
    t.Cleanup(func() { os.Setenv("AETHER_ROOT", oldRoot) })
    s, _ := storage.NewStore(dataDir)
    return s, tmpDir
}
```

### WR-02: rand.Read error is silently ignored

**File:** `cmd/worktree.go:83`
**Issue:** `rand.Read(rnd)` returns an error that is discarded. While `crypto/rand.Read` only fails in exceptional circumstances (e.g., entropy source failure on Linux), ignoring the error means the function could return an ID with a zero random suffix, increasing the likelihood of ID collisions.

**Fix:**
```go
rnd := make([]byte, 4)
if _, err := rand.Read(rnd); err != nil {
    return "", fmt.Errorf("generate worktree ID: %w", err)
}
return fmt.Sprintf("wt_%d_%s", time.Now().Unix(), hex.EncodeToString(rnd)), nil
```
Callers of `generateWorktreeID` would then need to handle the error return.

### WR-03: Merge conflict leaves working tree in dirty state without rollback

**File:** `cmd/worktree.go:696-705`
**Issue:** When `git merge` fails at line 696 (returns non-nil error), the merge may have partially applied, leaving the working tree in a conflicted state. The code creates a blocker flag but does not abort the merge (`git merge --abort`). Subsequent operations on the repo may encounter unexpected conflicts. The user would need to manually run `git merge --abort`.

**Fix:** After detecting merge failure, abort the merge to return the working tree to a clean state:
```go
mergeOut, mergeErr := exec.CommandContext(gitCtx, "git", "-C", aetherRoot, "merge", entry.Branch).CombinedOutput()
if mergeErr != nil {
    // Abort the merge to return working tree to clean state
    exec.CommandContext(gitCtx, "git", "-C", aetherRoot, "merge", "--abort").Run()
    blockerDesc := fmt.Sprintf("Merge failed for %s: %s", branch, string(mergeOut))
    // ... existing blocker creation ...
}
```

### WR-04: validateBranchName allows human-track names with special characters

**File:** `cmd/worktree.go:55-58`
**Issue:** Human-track branch names with prefixes like `feature/` only require that the name is longer than the prefix. This allows names like `feature/auth!@#` (confirmed by the test at line 1051 in the test file with comment "prefix validation only; git rejects bad chars"). While git will reject truly invalid characters, the validation layer should ideally catch obviously problematic patterns (e.g., whitespace, control characters, `~^:?*[]\`) before passing them to git. The test at line 1050 shows `phase-1/builder 1` (with space) is correctly rejected for the agent track, but `feature/name with spaces` would pass the human-track check.

**Fix:** Add a character whitelist check after the prefix match:
```go
for _, prefix := range humanBranchPrefixes {
    if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
        desc := name[len(prefix):]
        if strings.ContainsAny(desc, " \t\n~^:?*[]\\") {
            return fmt.Errorf("invalid branch name %q: description contains invalid characters", name)
        }
        return nil
    }
}
```

## Info

### IN-01: Test test name at line 1051 has a misleading comment

**File:** `cmd/worktree_test.go:1051`
**Issue:** The test case `"special chars accepted by prefix", "feature/auth!@#", false` has the comment "prefix validation only; git rejects bad chars". This documents a known gap where the validation intentionally delegates character checking to git. This is fine as a conscious decision, but the comment should be in the code (worktree.go) rather than only in the test, so future maintainers understand the design intent.

### IN-02: Unused import of `context` package consideration in colony.go

**File:** `pkg/colony/colony.go:1-9`
**Issue:** The colony.go file imports `fmt` and `time` but not `context`. The types are pure data structures with no behavioral methods. This is clean and appropriate. No action needed -- noting for completeness.

### IN-03: WorktreeEntry.LastCommitAt not set during allocation

**File:** `cmd/worktree.go:207-216`
**Issue:** When a worktree is allocated (line 207-216), `LastCommitAt` is left as its zero value (empty string). The orphan scan at line 410 sets it when it discovers the last commit time, but between allocation and the first orphan scan, `LastCommitAt` is empty. The `isWorktreeOrphaned` function at line 94-99 handles this by checking `commitAt.IsZero()` and treating it as orphaned. This is correct behavior, but the explicit empty field could be surprising. Consider setting `LastCommitAt` to the allocation time initially for clarity, or documenting that it is populated lazily by orphan scan.

---

_Reviewed: 2026-04-08T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
