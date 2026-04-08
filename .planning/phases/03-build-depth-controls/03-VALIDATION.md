---
phase: 3
slug: build-depth-controls
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./cmd/... -run Depth -count=1` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run Depth -count=1`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | DEPTH-01 | — | N/A | unit | `go test ./pkg/colony/... -run Valid -count=1` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | DEPTH-01 | — | N/A | unit | `go test ./pkg/colony/... -run DepthBudget -count=1` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 1 | DEPTH-02 | — | N/A | unit | `go test ./cmd/... -run TestColonyDepthCmd -count=1` | ✅ | ⬜ pending |
| 03-02-02 | 02 | 1 | DEPTH-03 | — | N/A | unit | `go test ./cmd/... -run TestStateMutateDepth -count=1` | ❌ W0 | ⬜ pending |
| 03-03-01 | 03 | 1 | DEPTH-04 | — | N/A | unit | `go test ./cmd/... -run TestInitDepthFlag -count=1` | ❌ W0 | ⬜ pending |
| 03-04-01 | 04 | 2 | DEPTH-05 | — | N/A | unit | `go test ./cmd/... -run TestContextBudget -count=1` | ❌ W0 | ⬜ pending |
| 03-05-01 | 05 | 2 | DEPTH-06 | — | N/A | unit | `go test ./cmd/... -run TestBuildDepthFlag -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/colony/colony_test.go` — stubs for DEPTH-01 (enum validation, budget mapping)
- [ ] `cmd/write_cmds_test.go` — extend existing depth tests for DEPTH-02/03

*Existing test infrastructure covers Go testing framework and build tooling.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| End-to-end depth flow (init → build → verify context budget) | DEPTH-04, DEPTH-05, DEPTH-06 | Requires running full CLI with colony state | Run `aether init "test" --depth light && aether build 1 --depth light` and verify spawn count |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
