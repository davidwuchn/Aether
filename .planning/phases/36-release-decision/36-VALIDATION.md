---
phase: 36
slug: release-decision
status: passed
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-23
---

# Phase 36 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 36-01-01 | 01 | 1 | Release readiness | — | All tests pass (2900+) | automated | `go test ./...` | ✅ | ✅ green |
| 36-01-01a | 01 | 1 | Binary builds | — | Clean build with exit 0 | automated | `go build ./cmd/aether` | ✅ | ✅ green |
| 36-01-01b | 01 | 1 | Agent parity | — | Drift/parity tests pass with zero drift | automated | `go test ./... -run "Drift\|Parity\|Completeness"` | ✅ | ✅ green |
| 36-01-02 | 01 | 1 | Version consistency | T-36-02 | .aether/version.json and npm/package.json both say 1.0.20 | automated | `grep '"version.*1.0.20' .aether/version.json npm/package.json` | ✅ | ✅ green |
| 36-01-02a | 01 | 1 | Changelog entry | T-36-02 | CHANGELOG.md has v1.0.20 section with R045-R051 | automated | `grep '1.0.20' CHANGELOG.md` | ✅ | ✅ green |
| 36-01-02b | 01 | 1 | CLAUDE.md updated | — | Exactly 3 occurrences of v1.0.20 | automated | `grep -c 'v1.0.20' CLAUDE.md` | ✅ | ✅ green |
| 36-01-03 | 01 | 1 | Git tag | T-36-01 | Annotated tag v1.0.20 on release commit | automated | `git tag -l v1.0.20` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `go test ./...` — all packages green (2900+ tests)
- [x] `go build ./cmd/aether` — clean build
- [x] Drift/parity tests — zero agent drift
- [x] Version files consistent at 1.0.20
- [x] CHANGELOG.md has v1.0.20 entry with headline and key items
- [x] CLAUDE.md updated to v1.0.20 in all three locations
- [x] Annotated git tag v1.0.20 created on release commit

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** passed 2026-04-23 (backfilled)
