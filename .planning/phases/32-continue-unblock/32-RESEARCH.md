# Phase 32: Continue Unblock - Research

**Researched:** 2026-04-22
**Domain:** Go runtime continue orchestration -- verify/learn/advance pipeline recovery
**Confidence:** HIGH

## Summary

The active colony in this very repo is stuck at phase 2 with state EXECUTING. The `aether continue` command runs successfully in tests (15/15 pass) but fails to advance this colony because the real-world continue flow encounters a cascade of issues that the test harness does not reproduce: manifest dispatches are stuck at "spawned" (never "completed"), builder claims are empty, tests fail at runtime, and the watcher never ran to completion.

The root problem is NOT that the continue pipeline has bypass paths (Phase 31 closed those). The problem is that the continue pipeline cannot advance a colony whose build phase was abandoned mid-execution -- workers were dispatched but never returned results. The `runCodexContinue` function correctly blocks advancement when evidence is missing, but it has no recovery path to help the user unblock. It just reports "blocked" with a redispatch command that the user may not understand.

There are three categories of work needed: (1) fixing the actual blocking issues in the continue pipeline so it can handle abandoned builds, (2) adding recovery tooling so colonies can be unblocked without manual state surgery, and (3) ensuring the wrapper-level continue flow (the playbooks) aligns with the runtime's actual capabilities.

**Primary recommendation:** Make continue handle abandoned builds gracefully -- detect them, explain what happened, and offer a clear unblock path. Then fix the specific code issues that prevent the verify/learn/advance pipeline from completing on a real colony.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Verification must be honest -- all verification paths must produce truthful results
- Continue orchestration must be complete -- verify, learn, advance end-to-end
- Colony advancement must work -- phase advancement is atomic (UpdateJSONAtomically from Phase 31)
- Error recovery must be graceful -- failed verification produces actionable error messages

### Claude's Discretion
- How to detect and recover from abandoned builds
- Whether to add a new recovery subcommand or extend existing continue
- How much of the wrapper-level playbook to align with runtime in this phase

### Deferred Ideas (OUT OF SCOPE)
- Continue performance optimization
- Continue UX improvements (wrapper-level, not runtime)
- Multi-phase continue (advance multiple phases at once)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-1 | Active colony stuck in phase 2 -- continue orchestration blocked, cannot advance | Colony state verified: EXECUTING at phase 2 with "spawned" dispatches, empty claims, failed tests. Manifest at `.aether/data/build/phase-2/manifest.json` |
| REQ-2 | Continue verification -- must verify all tasks honestly, no bypass paths | Phase 31 closed bypass paths. Continue correctly blocks when evidence is missing. Issue is recovery, not honesty. |
| REQ-3 | Continue learning -- must extract learnings from completed phase | Learning pipeline exists in `codex_continue.go` atomic commit block. Works when advancement succeeds. |
| REQ-4 | Continue advance -- must advance to next phase atomically | `UpdateJSONAtomically` at line 383 of `codex_continue.go` confirmed working. Tests prove it. |
| REQ-5 | Error handling -- failures produce clear messages, not silent skips | Continue currently produces detailed blocking messages but offers no actionable unblock path for abandoned builds. |
| REQ-6 | End-to-end flow -- verify/learn/advance must work as a complete pipeline | Pipeline works when build evidence is present. Fails when build was abandoned mid-execution. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Continue orchestration | `cmd/codex_continue.go` (Go runtime) | — | Runtime owns verify/gate/review/advance decision flow |
| Build evidence collection | `cmd/codex_build.go` (Go runtime) | `cmd/codex_build_worktree.go` | Build must produce manifest + claims for continue to consume |
| Colony state management | `pkg/storage/storage.go` + `pkg/colony/` | `cmd/codex_continue.go` | Storage owns atomicity; continue owns transaction scope |
| Worker dispatch | `pkg/codex/dispatch.go` + `pkg/codex/worker.go` | — | DispatchBatch and WorkerInvoker own execution |
| Wrapper presentation | `.claude/commands/ant/continue.md` | `.aether/docs/command-playbooks/continue-*.md` | Wrappers present runtime output; playbooks define Claude-side steps |
| Recovery tooling | `cmd/codex_continue.go` (Go runtime) | — | Continue should detect and explain abandoned builds |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.24 | `context`, `os`, `os/exec`, `path/filepath`, `encoding/json` | Native runtime, no dependencies |
| `pkg/storage` | in-tree | Atomic file operations (`UpdateJSONAtomically`, `SaveJSON`, `LoadJSON`) | Already implements temp-file + rename pattern |
| `pkg/codex` | in-tree | Worker dispatch, invocation, claims extraction | Core abstraction layer for all worker execution |
| `pkg/colony` | in-tree | State types (`ColonyState`, `Phase`, `Task`) | State mutations throughout |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pkg/agent` | in-tree | Spawn tree recording | Worker lifecycle tracking |

## Architecture Patterns

### System Architecture Diagram

```
[Colony State (EXECUTING/BUILT)]
         |
         v
[loadActiveColonyState] -----> validate state, phase, manifest
         |
         v
[runCodexContinueVerification]
  |-- resolveCodexVerificationCommands (build/test/type/lint)
  |-- runVerificationStep x4
  |-- verifyCodexBuildClaims
  |-- evaluateContinueWatcherVerification (from manifest)
  +-- runCodexContinueWatcherVerification (fresh dispatch)
         |
         v
[assessCodexContinue] -----> classify each task outcome
         |                      (verified/missing/simulated/needs_redispatch/...)
         v
[attachContinueClaimVerification]
         |
         v
[runCodexContinueGates] -----> manifest_present, verification_steps_passed,
         |                      implementation_evidence, operational_evidence,
         |                      critical_flags
         v
   gates.Passed?
   |-- NO --> record blocked flow, return "blocked" result
   +-- YES --> runCodexContinueReview (gatekeeper/auditor/probe)
         |
         v
   review.Passed?
   |-- NO --> record blocked flow, return "blocked" result
   +-- YES --> UpdateJSONAtomically (atomic state commit)
                  |
                  +-- Side effects: signal housekeeping, context update,
                      worker closures, report saves
```

### Recommended Project Structure

The continue command is already well-structured. Changes should be localized:

```
cmd/
  codex_continue.go         -- Primary: add abandoned-build detection + recovery
  codex_continue_test.go    -- Add tests for abandoned build scenarios
  codex_dispatch_contract.go -- Timeout constants
  dispatch_platform_helpers.go -- Platform dispatch utilities
pkg/
  codex/
    dispatch.go              -- DispatchBatch (no changes needed)
    worker.go                -- WorkerInvoker (no changes needed)
    platform_dispatch.go     -- Platform selection (no changes needed)
  storage/
    storage.go               -- UpdateJSONAtomically (no changes needed)
```

### Pattern 1: Abandoned Build Detection

**What:** Detect when a colony's build was abandoned (dispatches stuck at "spawned", no completed workers, stale `build_started_at`).

**When to use:** At the start of `runCodexContinue`, before verification runs.

**Example:**
```go
// Detect abandoned build: dispatches still "spawned" after significant time
func detectAbandonedBuild(manifest codexContinueManifest, state colony.ColonyState) (abandoned bool, staleDuration time.Duration, summary string) {
    if !manifest.Present || len(manifest.Data.Dispatches) == 0 {
        return false, 0, ""
    }
    if state.BuildStartedAt == nil {
        return false, 0, ""
    }
    allSpawned := true
    for _, d := range manifest.Data.Dispatches {
        if strings.TrimSpace(d.Status) != "spawned" {
            allSpawned = false
            break
        }
    }
    if !allSpawned {
        return false, 0, ""
    }
    elapsed := time.Since(*state.BuildStartedAt)
    if elapsed < 10*time.Minute {
        return false, 0, "" // Still possibly running
    }
    return true, elapsed, fmt.Sprintf("Build was abandoned %.0f minutes ago: all %d dispatches stuck at 'spawned'", elapsed.Minutes(), len(manifest.Data.Dispatches))
}
```

### Anti-Patterns to Avoid

- **Silent fallback to phase advancement:** Phase 31 closed bypass paths. Do not re-open them.
- **Auto-reconcile abandoned builds:** Do not automatically mark abandoned tasks as reconciled. The user must explicitly choose how to proceed.
- **State surgery in the wrapper:** The wrapper must never directly mutate COLONY_STATE.json. Recovery must go through the runtime.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| State atomicity | Custom lock + write patterns | `store.UpdateJSONAtomically` | Already handles locking, backup, validation |
| Worker dispatch | Custom subprocess spawning | `DispatchBatch` + `WorkerInvoker` | Already handles timeouts, progress, error propagation |
| Claim verification | Manual file existence checks | `verifyCodexBuildClaims` | Already handles synthetic detection, empty claims |
| Task classification | Custom outcome mapping | `classifyContinueTaskAssessment` | Already handles all outcome types honestly |

## Common Pitfalls

### Pitfall 1: Treating "blocked" as a Bug When It Is Correct Behavior
**What goes wrong:** The continue command correctly blocks advancement when evidence is missing. The "stuck" colony is not a bug in continue -- it is continue being honest about missing build evidence.
**Why it happens:** The build was abandoned mid-execution. Workers were dispatched but never completed. Continue has no evidence to verify.
**How to avoid:** Do not try to make continue "more lenient". Instead, add recovery paths that help the user produce real evidence (re-dispatch, reconcile, or reset).
**Warning signs:** Proposals to add bypass paths for "stale" builds.

### Pitfall 2: Stale Manifest Artifacts
**What goes wrong:** Previous continue runs produce verification.json, gates.json, and continue.json files that become stale. A new continue run may read these instead of fresh evidence.
**Why it happens:** The continue report files are written per-phase but not cleaned up between runs. The watcher reads the manifest, not the reports.
**How to avoid:** The continue watcher (`runCodexContinueWatcherVerification`) always dispatches fresh. The report files are informational, not authoritative. The `evaluateContinueWatcherVerification` function reads from the manifest, which is authoritative.
**Warning signs:** Reports from different timestamps in the same phase directory.

### Pitfall 3: Test Failures Block Advancement in the Repo Itself
**What goes wrong:** Continue runs `go test ./...` as part of verification. If the repo has failing tests (like this repo does when Phase 31 work is in progress), continue blocks advancement even if the build/claims evidence is perfect.
**Why it happens:** The verification loop runs ALL tests, not just the ones relevant to the current phase.
**How to avoid:** This is correct behavior for a real colony. The fix is to ensure tests pass, not to skip them. However, the recovery message should be specific about which tests failed and why.
**Warning signs:** Proposals to skip test verification for "known" failures.

### Pitfall 4: Empty Builder Claims
**What goes wrong:** The `last-build-claims.json` file exists but is empty (no files_created, files_modified, tests_written). Continue treats this as failed verification.
**Why it happens:** Workers were dispatched as "spawned" but never returned results. The build claims were never populated because the workers never completed.
**How to avoid:** This is correct behavior. The fix is to re-dispatch the build so workers actually run and produce claims.
**Warning signs:** Proposals to auto-populate claims from git diff.

### Pitfall 5: Wrapper-Runtime Misalignment
**What goes wrong:** The wrapper playbooks (continue-verify.md, continue-gates.md, etc.) describe an elaborate multi-step verification process with agent spawning, TDD gates, and runtime checks. The runtime implements a simpler, more focused pipeline. Users following the playbook may expect behavior the runtime does not provide.
**Why it happens:** The playbooks were written for Claude's agent-based continue flow. The runtime implements its own verification.
**How to avoid:** The runtime is authoritative. The wrapper delegates to the runtime and adds colony framing. The playbooks are reference material for the wrapper, not for the runtime.
**Warning signs:** Trying to make the runtime match the playbooks instead of vice versa.

## Code Examples

### Abandoned Build Recovery Message (Proposed)

```go
// Source: cmd/codex_continue.go (proposed addition)
if abandoned, duration, summary := detectAbandonedBuild(manifest, state); abandoned {
    result := map[string]interface{}{
        "advanced":       false,
        "blocked":        true,
        "abandoned":      true,
        "stale_duration": duration.String(),
        "current_phase":  state.CurrentPhase,
        "phase_name":     phase.Name,
        "state":          state.State,
        "recovery": map[string]interface{}{
            "option_redispatch": fmt.Sprintf("aether build %d", phase.ID),
            "option_reconcile":  fmt.Sprintf("aether continue --reconcile-task %s", strings.Join(taskIDs, " --reconcile-task ")),
            "description":       summary,
        },
    }
    runStatus = "blocked-abandoned"
    return result, state, phase, nil, nil, false, nil
}
```

### Current Blocked Flow (Existing Code)

The existing blocked flow in `runCodexContinue` (lines 252-306 of `codex_continue.go`) correctly:
1. Records the blocked worker flow
2. Saves the continue report with recovery commands
3. Updates session summary
4. Returns a detailed result map with `blocking_issues`

The recovery commands are already surfaced via `assessment.Recovery`:
- `ReverifyCommand`: `"aether continue"`
- `ReconcileCommand`: `"aether continue --reconcile-task 2.1 --reconcile-task 2.2"`
- `RedispatchCommand`: `"aether build 2 --task 2.1 --task 2.2"`

### Atomic State Commit (Existing, Phase 31)

```go
// Source: cmd/codex_continue.go lines 383-419
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
    updated = state
    // ... mutation logic ...
    updated.Plan.Phases[currentIdx].Status = colony.PhaseCompleted
    // ... advance phase, set READY, append events ...
    return nil
}); err != nil {
    return nil, state, phase, nil, nil, false, fmt.Errorf("failed to atomically advance phase: %w", err)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| FakeInvoker in production | `NewWorkerInvokerOrError` + platform dispatch | Phase 31 (2026-04-22) | Runtime fails honestly when no platform available |
| `SaveJSON` for state commit | `UpdateJSONAtomically` | Phase 31 (2026-04-22) | State is durable before side effects |
| 4 bypass paths in continue | All closed with honest blocking | Phase 31 (2026-04-22) | Continue only advances with real evidence |
| Silent dispatch errors | `DispatchBatch` error propagation | Phase 31 (2026-04-22) | Errors surface to callers |

**Deprecated/outdated:**
- `verified_partial` outcome: Removed in Phase 31. No longer allows advancement with failed workers.
- Environmental dismissal of test failures: Removed in Phase 31. Test failures are reported honestly.

## Active Colony Diagnosis

The colony in THIS repository (`/Users/callumcowie/repos/Aether`) is stuck at phase 2. Here is the exact diagnosis:

| Factor | Value | Blocking? |
|--------|-------|-----------|
| Colony state | EXECUTING | Yes -- continue expects EXECUTING or BUILT |
| Phase 2 status | in_progress | Yes -- continue checks `phase.Status != PhaseInProgress` |
| Manifest dispatches | All "spawned" (none completed) | Yes -- no worker evidence |
| Builder claims | Empty (no files created/modified/tests) | Yes -- claims verification fails |
| Test suite | Failing (cmd tests) | Yes -- verification step fails |
| Build started at | 2026-04-22T15:13:42Z (~8 hours ago) | Yes -- stale build |
| Watcher (from manifest) | "spawned" (never completed) | Yes -- no independent verification |

**Root cause:** The build for phase 2 was dispatched but never completed. Workers were launched but the session ended before they returned results. The manifest shows all 3 dispatches stuck at "spawned". Continue correctly blocks because there is no evidence of work.

**Recovery path:** The colony needs one of:
1. Re-dispatch: `aether build 2` (or `aether build 2 --task 2.1 --task 2.2`) to re-run the build
2. Manual reconciliation: `aether continue --reconcile-task 2.1 --reconcile-task 2.2` (only if work was actually done outside the build system)

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The test failures in the repo are from Phase 31 WIP code that needs to be committed/finished | Active Colony Diagnosis | Medium -- if tests are failing for other reasons, the colony may need different recovery |
| A2 | The wrapper playbooks (continue-verify.md etc.) do not need changes for this phase | Architecture Patterns | Low -- wrapper changes are deferred per CONTEXT.md |
| A3 | No new subcommands are needed -- extending continue and/or build is sufficient | Don't Hand-Roll | Low -- if a dedicated recovery command is needed, scope expands |

## Open Questions

1. **Should continue auto-detect abandoned builds and produce a special "abandoned" result?**
   - What we know: The current "blocked" result includes recovery commands but does not distinguish between "verification failed" and "build was never completed"
   - What's unclear: Whether the wrapper needs a different UX path for abandoned vs. failed
   - Recommendation: Add an `abandoned` field to the continue result that the wrapper can use for clearer messaging

2. **Should continue clear stale report files before re-running verification?**
   - What we know: Previous continue runs leave behind verification.json, gates.json, continue.json
   - What's unclear: Whether stale reports confuse the wrapper or the user
   - Recommendation: Clear stale reports at the start of continue (they are regenerated anyway)

3. **What is the minimum viable change to unblock the active colony?**
   - What we know: The colony needs `aether build 2` re-dispatched OR tests to pass + manual reconciliation
   - What's unclear: Whether the code changes from Phase 31 that are causing test failures need to be committed first
   - Recommendation: Commit Phase 31 changes, fix any test regressions, then re-dispatch build 2

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.24 | Build + test | Available | go1.24 | -- |
| `aether` binary | Continue runtime | Available | v1.0.19 | Build from source |
| Codex CLI | Worker dispatch | Not checked | -- | FakeInvoker in tests |
| Claude CLI | Worker dispatch | Available | -- | -- |
| `go test` | Verification | Available | -- | -- |

**Missing dependencies with no fallback:**
- None identified

**Missing dependencies with fallback:**
- Codex CLI: Colony uses Claude as primary platform per CLAUDE.md

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None |
| Quick run command | `go test ./cmd/... -run TestContinue -count=1 -timeout 60s` |
| Full suite command | `go test ./... -race -count=1 -timeout 300s` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-1 | Abandoned build detection | unit | `go test ./cmd/... -run TestContinueAbandonedBuild -count=1` | No -- Wave 0 |
| REQ-2 | Honest verification | unit | `go test ./cmd/... -run TestContinueBlocksWhen -count=1` | Yes (Phase 31) |
| REQ-3 | Learning extraction | unit | `go test ./cmd/... -run TestContinueConsumes -count=1` | Yes (existing) |
| REQ-4 | Atomic advancement | unit | `go test ./cmd/... -run TestContinueStateCommitted -count=1` | Yes (Phase 31) |
| REQ-5 | Actionable error messages | unit | `go test ./cmd/... -run TestContinueAbandoned -count=1` | No -- Wave 0 |
| REQ-6 | End-to-end pipeline | integration | `go test ./cmd/... -run TestContinueConsumesBuildPacket -count=1` | Yes (existing) |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run TestContinue -count=1 -timeout 60s`
- **Per wave merge:** `go test ./... -race -count=1 -timeout 300s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] Test for abandoned build detection (REQ-1)
- [ ] Test for abandoned build recovery messaging (REQ-5)
- [ ] Integration test: continue on a colony with stale manifest + empty claims

*(Existing tests cover: bypass closure, atomic commit, worker flow, housekeeping, final phase, watcher blocking)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A (no auth in continue) |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | yes | JSON parsing via `encoding/json`; file path validation via `filepath.Clean` |
| V6 Cryptography | no | N/A |

### Known Threat Patterns for Go Runtime

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal in claims | Tampering | `filepath.Clean` + `filepath.Rel` normalization in `normalizeClaimPaths` |
| JSON injection in state | Tampering | `encoding/json` strict parsing; no `json.RawMessage` mutation |
| Race condition in state writes | Tampering | `UpdateJSONAtomically` uses file locking |

## Sources

### Primary (HIGH confidence)
- `cmd/codex_continue.go` -- Full source read, 2145 lines, verified line-by-line
- `cmd/codex_continue_test.go` -- Full test suite read, 15 tests verified passing
- `cmd/codex_build.go` -- Worker invoker factory at line 86
- `pkg/codex/dispatch.go` -- DispatchBatch error propagation verified
- `pkg/codex/worker.go` -- WorkerInvoker interface, FakeInvoker, RealInvoker
- `pkg/codex/platform_dispatch.go` -- Platform selection and availability
- `pkg/storage/storage.go` -- UpdateJSONAtomically implementation
- `.aether/data/COLONY_STATE.json` -- Active colony state verified stuck at phase 2
- `.aether/data/build/phase-2/` -- Build artifacts verified: manifest (spawned), empty claims, failed tests

### Secondary (MEDIUM confidence)
- `.aether/docs/command-playbooks/continue-verify.md` -- Wrapper playbook (reference only, runtime is authoritative)
- `.aether/docs/command-playbooks/continue-gates.md` -- Gate checks (reference only)
- `.aether/docs/command-playbooks/continue-advance.md` -- Advance logic (reference only)
- `.aether/docs/command-playbooks/continue-finalize.md` -- Finalization (reference only)
- `.planning/phases/31-p0-runtime-truth-fixes/31-RESEARCH.md` -- Phase 31 research context

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all in-tree Go code, directly inspected
- Architecture: HIGH -- full pipeline traced from entry to exit with concrete data from active colony
- Pitfalls: HIGH -- identified from real colony state and code inspection
- Recovery path: MEDIUM -- depends on whether tests pass after Phase 31 work is finalized

**Research date:** 2026-04-22
**Valid until:** 2026-05-22 (stable -- Go runtime code changes slowly)
