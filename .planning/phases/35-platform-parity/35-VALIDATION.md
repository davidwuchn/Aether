---
phase: 35
slug: platform-parity
status: passed
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-23
---

# Phase 35 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/... -run "Drift\|Parity\|Completeness"` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run "Drift\|Parity\|Completeness"`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 35-01-01 | 01 | 1 | Drift detection | — | Drift detection infrastructure with Claude/OpenCode comparison | unit | `go test ./cmd/... -run "TestDrift"` | ✅ | ✅ green |
| 35-01-02 | 01 | 1 | Mirror tests | — | Packaging mirror byte-identity tests | unit | `go test ./cmd/... -run "TestAgentMirror"` | ✅ | ✅ green |
| 35-02-01 | 02 | 1 | OpenCode sync | — | All 25 OpenCode agents synced from Claude masters | unit | `go test ./cmd/... -run "TestOpenCodeDrift"` | ✅ | ✅ green |
| 35-03-01 | 03 | 1 | Codex completeness | — | Codex agents updated with Phase 31-33 concepts | unit | `go test ./cmd/... -run "TestCodexCompleteness"` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Drift detection tests in cmd/codex_e2e_test.go
- [x] Packaging mirror tests in cmd/agent_mirror_test.go
- [x] OpenCode agent parity (zero drift from Claude masters)
- [x] Codex agent completeness (Phase 31-33 runtime concepts present)

---

## Notes

Phase 35 was verified inline during execute-phase (no formal VERIFICATION.md created).
Validation based on passing test suite and SUMMARY.md evidence from all 3 plans.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** passed 2026-04-23 (backfilled)
