---
phase: 32
slug: continue-unblock
status: passed
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-23
---

# Phase 32 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/... -run "Continue"` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run "Continue"`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 32-01-01 | 01 | 1 | Abandoned build detection | — | Detects all dispatches stuck at "spawned" >10 min | unit | `go test ./cmd/... -run "TestContinue.*Abandoned"` | ✅ | ✅ green |
| 32-01-02 | 01 | 1 | Recovery commands | — | Returns blocked=true with actionable recovery commands | unit | `go test ./cmd/... -run "TestContinue.*Recovery"` | ✅ | ✅ green |
| 32-02-01 | 02 | 1 | Stale report cleanup | — | Clears stale verification.json, gates.json, continue.json | unit | `go test ./cmd/... -run "TestContinue.*StaleReport"` | ✅ | ✅ green |
| 32-02-02 | 02 | 1 | E2E pipeline recovery | — | Full pipeline: abandoned detection → re-dispatch → verify → advance | integration | `go test ./cmd/... -run "TestContinueEndToEndAfterAbandonedRecovery"` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Abandoned build detection with 10-minute threshold
- [x] Blocked result with recovery commands (redispatch + reconcile)
- [x] Stale report file cleanup before verification
- [x] E2E recovery pipeline: abandoned → re-dispatch → verify → advance

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** passed 2026-04-23 (backfilled)
