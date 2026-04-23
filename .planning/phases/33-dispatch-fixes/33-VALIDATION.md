---
phase: 33
slug: dispatch-fixes
status: passed
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-23
---

# Phase 33 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/... -run "Dispatch\|Build\|Visual"` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run "Dispatch\|Build\|Visual"`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 33-01-01 | 01 | 1 | R052 | — | Context capsule threshold (128 chars) wired | unit | `go test ./cmd/... -run "TestColonyPrime"` | ✅ | ✅ green |
| 33-01-02 | 01 | 1 | R053 | — | Builder-keyword priority in caste routing | unit | `go test ./cmd/... -run "TestVisual"` | ✅ | ✅ green |
| 33-01-03 | 01 | 1 | R054 | — | Autopilot single retry | unit | `go test ./cmd/... -run "TestAutopilot"` | ✅ | ✅ green |
| 33-02-01 | 02 | 1 | R055 | — | Codex CLI fallback in continue | unit | `go test ./cmd/... -run "TestContinue.*Codex"` | ✅ | ✅ green |
| 33-02-02 | 02 | 1 | R056 | — | SpawnTree nil store error handling | unit | `go test ./pkg/agent/... -run "TestSpawnTree"` | ✅ | ✅ green |
| 33-02-03 | 02 | 1 | — | — | Archaeologist in full-depth build | integration | `go test ./cmd/... -run "TestBuild.*Full"` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Context capsule threshold (128 chars) in colony_prime_context.go
- [x] Builder-keyword priority in codex_visuals.go
- [x] Autopilot single retry in compatibility_cmds.go
- [x] Codex CLI fallback in codex_continue.go
- [x] SpawnTree nil store error in pkg/agent/spawn_tree.go
- [x] Archaeologist dispatch gated by depth=="full"

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** passed 2026-04-23 (backfilled)
