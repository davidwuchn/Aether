---
phase: 2
slug: system-integrity
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — built into Go |
| **Quick run command** | `go test ./cmd/... ./pkg/storage/... -count=1 -timeout 60s` |
| **Full suite command** | `go test ./... -count=1 -timeout 300s` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... ./pkg/storage/... -count=1 -timeout 60s`
- **After every plan wave:** Run `go test ./... -count=1 -timeout 300s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | INTG-04 | — | User signals never flagged regardless of content | unit | `go test ./cmd/ -run TestIsTestArtifact -count=1` | ⬜ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | INTG-05 | T-02-01 | Destructive commands require confirmation | unit | `go test ./cmd/ -run TestBackupPrune\|TestTempClean -count=1` | ⬜ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | INTG-01 | — | All subcommands run without panic on fresh install | smoke | `go test ./cmd/ -run TestSmoke -count=1` | ⬜ W0 | ⬜ pending |
| 02-02-02 | 02 | 1 | INTG-03 | — | Error messages follow consistent format | unit | `go test ./cmd/ -run TestErrorFormat -count=1` | ⬜ W0 | ⬜ pending |
| 02-03-01 | 03 | 2 | INTG-02 | — | Deprecated code fully removed | unit | `grep -r deprecated ./cmd/ ./aether/utils/` | ⬜ W0 | ⬜ pending |
| 02-03-02 | 03 | 2 | INTG-06 | — | All 524+ tests pass with no regressions | full | `go test ./... -count=1` | ⬜ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/maintenance_test.go` — test cases for backup-prune-global and temp-clean confirmation gates
- [ ] `cmd/suggest_test.go` — test cases for isTestArtifact fix (if not already covered)
- [ ] `cmd/smoke_test.go` — smoke test suite for fresh install validation

*Existing infrastructure covers most phase requirements. New test files needed for smoke tests and confirmation gates.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| None | — | — | — |

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
