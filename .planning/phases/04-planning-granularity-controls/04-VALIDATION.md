---
phase: 04
slug: planning-granularity-controls
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
---

# Phase 04 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — built into Go |
| **Quick run command** | `go test ./cmd/... ./pkg/colony/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... ./pkg/colony/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | PLAN-01 | — | N/A | unit | `go test ./pkg/colony/... -run TestPlanGranularity -count=1` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | PLAN-01 | — | N/A | unit | `go test ./pkg/colony/... -run TestPlanGranularity_Valid -count=1` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | PLAN-02 | — | N/A | unit | `go test ./cmd/... -run TestPlanGranularityGet -count=1` | ❌ W0 | ⬜ pending |
| 04-01-04 | 01 | 1 | PLAN-02 | — | N/A | unit | `go test ./cmd/... -run TestPlanGranularitySet -count=1` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | PLAN-03 | — | N/A | integration | `go test ./cmd/... -run TestPlanCommand_Granularity -count=1` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | PLAN-04 | — | N/A | unit | `go test ./cmd/... -run TestStatusOutput_Granularity -count=1` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 2 | PLAN-03 | — | N/A | integration | `go test ./cmd/... -run TestPlanCommand_OutOfRange -count=1` | ❌ W0 | ⬜ pending |
| 04-04-01 | 04 | 2 | PLAN-05 | — | N/A | integration | `go test ./cmd/... -run TestAutopilot_Granularity -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `pkg/colony/colony_test.go` — add TestPlanGranularity, TestPlanGranularity_Valid stubs
- [ ] `cmd/colony_cmds_test.go` — add TestPlanGranularityGet, TestPlanGranularitySet stubs
- [ ] `cmd/status_test.go` — add TestStatusOutput_Granularity stub
- [ ] `cmd/plan_test.go` — add TestPlanCommand_Granularity, TestPlanCommand_OutOfRange stubs
- [ ] `cmd/run_test.go` — add TestAutopilot_Granularity stub

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Route-setter prompt contains dynamic bounds | PLAN-03 | Requires reading agent prompt output | Run `/ant:plan --granularity sprint` and verify route-setter receives "1-3 phases" constraint |
| Autopilot respects granularity across phases | PLAN-05 | Multi-phase execution | Run `/ant:run --max-phases 2` with quarter granularity set, verify warning if plan exceeds range |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
