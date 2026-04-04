---
phase: 03-pheromone-signal-plumbing
verified: 2026-04-03T12:00:00Z
status: gaps_found
score: 0/5 must-haves verified
re_verification:
  previous_status: passed
  previous_score: 4/4
  gaps_closed: []
  gaps_remaining: []
  regressions:
    - "Previous verification verified OLD Phase 3 (pheromone plumbing, PHER-01/02/06/07). ROADMAP has been renumbered. Current Phase 3 is now 'Build Utility Commands' with CMD-01 through CMD-08 and DIFF-01."
gaps:
  - truth: "`aether generate-ant-name` produces a random ant name as a Go subcommand"
    status: failed
    reason: "No Go subcommand 'generate-ant-name' exists. The command only exists as a shell function in .aether/aether-utils.sh. The Go binary has 150+ subcommands but this is not among them."
    artifacts:
      - path: "cmd/"
        issue: "No file containing a 'generate-ant-name' cobra command"
    missing:
      - "Go subcommand file implementing generate-ant-name with cobra"
      - "Registration via rootCmd.AddCommand in the cmd/ package"
  - truth: "`aether generate-ant-name --seed N` produces deterministic output (DIFF-01)"
    status: failed
    reason: "No --seed flag exists in the shell implementation (treated as caste name). Shell output is non-deterministic even with same seed. No Go implementation exists at all."
    artifacts:
      - path: ".aether/aether-utils.sh"
        issue: "Line 2210: --seed is not parsed as a flag; 'bash .aether/aether-utils.sh generate-ant-name --seed 42 builder' treats '--seed' as the caste argument"
    missing:
      - "Flag parsing for --seed in either shell or Go implementation"
      - "Seeded PRNG logic for deterministic output"
  - truth: "`aether generate-commit-message` produces a commit message from git diff as a Go subcommand"
    status: failed
    reason: "No Go subcommand 'generate-commit-message' exists. Shell implementation only in .aether/aether-utils.sh."
    artifacts:
      - path: "cmd/"
        issue: "No file containing a 'generate-commit-message' cobra command"
    missing:
      - "Go subcommand file implementing generate-commit-message"
  - truth: "`aether version-check-cached` displays cached version information as a Go subcommand"
    status: failed
    reason: "No Go subcommand 'version-check-cached' exists. Shell implementation only."
    artifacts:
      - path: "cmd/"
        issue: "No file containing a 'version-check-cached' cobra command"
    missing:
      - "Go subcommand file implementing version-check-cached"
  - truth: "`aether milestone-detect` identifies current milestone as a Go subcommand"
    status: failed
    reason: "No Go subcommand 'milestone-detect' exists. Shell implementation only. Go status command reads a Milestone field from state but does not implement milestone-detect logic."
    artifacts:
      - path: "cmd/"
        issue: "No file containing a 'milestone-detect' cobra command"
    missing:
      - "Go subcommand file implementing milestone-detect"
  - truth: "`aether update-progress`, `aether print-next-up`, `aether generate-progress-bar`, `aether data-safety-stats` all produce correct output as Go subcommands"
    status: failed
    reason: "None of these four commands exist as Go subcommands. generateProgressBar exists as a private helper in cmd/status.go but is not a registered cobra command. All four only exist as shell functions."
    artifacts:
      - path: "cmd/"
        issue: "No cobra commands registered for update-progress, print-next-up, generate-progress-bar, or data-safety-stats"
    missing:
      - "4 Go subcommand files implementing these commands"
---

# Phase 3: Build Utility Commands Verification Report

**Phase Goal:** All 8 critical build utility commands work as Go subcommands, enabling worker spawns and status display
**Verified:** 2026-04-03T12:00:00Z
**Status:** GAPS FOUND
**Re-verification:** Yes -- ROADMAP renumbered since last verification. Previous verification (2026-03-19) verified the OLD Phase 3 (pheromone plumbing) which is no longer the current Phase 3.

## Context: ROADMAP Renumbering

The previous verification (score 4/4, passed) verified the OLD Phase 3 "Pheromone Signal Plumbing" with requirements PHER-01, PHER-02, PHER-06, PHER-07. That work is confirmed complete in the codebase (pheromone decay, injection chain, resume.md fix all present).

However, the ROADMAP has since been renumbered under v6.0. The current Phase 3 is "Build Utility Commands" with requirements CMD-01 through CMD-08 and DIFF-01. The phase directory name `03-pheromone-signal-plumbing` is a legacy name from the old numbering.

**No plans have been created for the current Phase 3** (ROADMAP shows "Plans: TBD").

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `aether generate-ant-name` produces a random ant name (with optional `--seed` for deterministic output) | FAILED | No Go subcommand exists. Shell version works (`bash .aether/aether-utils.sh generate-ant-name builder` returns valid JSON) but `--seed` is not parsed as a flag. Go binary `aether --help` lists 150+ commands; generate-ant-name is absent. |
| 2 | `aether generate-commit-message` produces a commit message from git diff | FAILED | No Go subcommand exists. Shell version works but no Go implementation. |
| 3 | `aether version-check-cached` displays cached version information | FAILED | No Go subcommand exists. Shell version works but no Go implementation. |
| 4 | `aether milestone-detect` identifies the current milestone from colony state | FAILED | No Go subcommand exists. Shell version works. Go `status` command reads a milestone field but does not implement milestone-detect detection logic. |
| 5 | `aether update-progress`, `aether print-next-up`, `aether generate-progress-bar`, and `aether data-safety-stats` all produce correct output | FAILED | None exist as Go subcommands. `generateProgressBar` is a private Go function in cmd/status.go (used internally by status command), not a registered cobra subcommand. All four exist as shell functions only. |

**Score:** 0/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/generate_ant_name.go` (or equivalent) | Go cobra command for generate-ant-name | MISSING | No file in cmd/ registers a "generate-ant-name" subcommand |
| `cmd/generate_commit_message.go` (or equivalent) | Go cobra command for generate-commit-message | MISSING | No file in cmd/ registers this subcommand |
| `cmd/version_check_cached.go` (or equivalent) | Go cobra command for version-check-cached | MISSING | No file in cmd/ registers this subcommand |
| `cmd/milestone_detect.go` (or equivalent) | Go cobra command for milestone-detect | MISSING | No file in cmd/ registers this subcommand |
| `cmd/update_progress.go` (or equivalent) | Go cobra command for update-progress | MISSING | No file in cmd/ registers this subcommand |
| `cmd/print_next_up.go` (or equivalent) | Go cobra command for print-next-up | MISSING | No file in cmd/ registers this subcommand |
| `cmd/generate_progress_bar.go` (or equivalent) | Go cobra command for generate-progress-bar | MISSING | Private function exists in status.go but no cobra command |
| `cmd/data_safety_stats.go` (or equivalent) | Go cobra command for data-safety-stats | MISSING | No file in cmd/ registers this subcommand |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| N/A | N/A | N/A | N/A | No Go commands exist to verify wiring for |

### Data-Flow Trace (Level 4)

Skipped -- no Go artifacts to trace.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Shell generate-ant-name works | `bash .aether/aether-utils.sh generate-ant-name builder` | `{"ok":true,"result":"Hammer-88"}` | PASS (shell only) |
| Shell --seed is NOT deterministic | `bash .aether/aether-utils.sh generate-ant-name --seed 42 builder` (twice) | "Drone-83" then "Marcher-69" | FAIL -- DIFF-01 not met |
| Go generate-ant-name exists | `go run cmd/aether/*.go generate-ant-name --help` | "unknown command" | FAIL |
| Go generate-commit-message exists | `go run cmd/aether/*.go generate-commit-message --help` | "unknown command" | FAIL |
| Go version-check-cached exists | `go run cmd/aether/*.go version-check-cached --help` | "unknown command" | FAIL |
| Go milestone-detect exists | `go run cmd/aether/*.go milestone-detect --help` | "unknown command" | FAIL |
| Go update-progress exists | `go run cmd/aether/*.go update-progress --help` | "unknown command" | FAIL |
| Go print-next-up exists | `go run cmd/aether/*.go print-next-up --help` | "unknown command" | FAIL |
| Go generate-progress-bar exists | `go run cmd/aether/*.go generate-progress-bar --help` | "unknown command" | FAIL |
| Go data-safety-stats exists | `go run cmd/aether/*.go data-safety-stats --help` | "unknown command" | FAIL |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CMD-01 | None (no plans created) | `generate-ant-name` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 2209. No Go subcommand. |
| CMD-02 | None | `generate-commit-message` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 2427. No Go subcommand. |
| CMD-03 | None | `version-check-cached` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 2587. No Go subcommand. |
| CMD-04 | None | `milestone-detect` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 2918. No Go subcommand. |
| CMD-05 | None | `update-progress` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 1851. No Go subcommand. |
| CMD-06 | None | `print-next-up` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 396. No Go subcommand. |
| CMD-07 | None | `generate-progress-bar` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 358. Private Go helper in status.go but no cobra command. |
| CMD-08 | None | `data-safety-stats` Go subcommand | BLOCKED | Shell implementation exists in aether-utils.sh line 4935. No Go subcommand. |
| DIFF-01 | None | `generate-ant-name --seed` deterministic mode | BLOCKED | No --seed flag parsing in shell or Go. Shell treats --seed as caste name. |

**All 9 requirements are ORPHANED** -- claimed by ROADMAP Phase 3 but no plan has been created for any of them.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| N/A | N/A | N/A | N/A | Phase has not been started -- no code to scan |

### Human Verification Required

None required. The gap is clear: no Go subcommands exist and no plans have been created.

### Gaps Summary

Phase 3 ("Build Utility Commands") has not been started. The ROADMAP lists 9 requirements (CMD-01 through CMD-08, DIFF-01) but shows "Plans: TBD". No plan files exist for this phase under the current ROADMAP numbering.

All 8 shell commands exist and work in `.aether/aether-utils.sh`, providing clear reference implementations for the Go ports. The `generateProgressBar` private function in `cmd/status.go` demonstrates the Go pattern to follow.

The previous verification (2026-03-19) confirmed completion of the OLD Phase 3 (pheromone signal plumbing) which was a different phase entirely under the old roadmap numbering. That work remains verified and intact in the codebase.

**Root cause:** ROADMAP renumbering moved pheromone plumbing out of Phase 3 and replaced it with Build Utility Commands, but no planning or execution has occurred for the new Phase 3.

---

_Verified: 2026-04-03T12:00:00Z_
_Verifier: Claude (gsd-verifier)_
