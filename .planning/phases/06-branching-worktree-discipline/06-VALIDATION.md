---
phase: 06
slug: branching-worktree-discipline
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
---

# Phase 06 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./cmd/... ./pkg/...` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... ./pkg/...`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | BRAN-01 | T-06-01 | Reject invalid branch names, log rejection | unit | `go test ./cmd/... -run TestBranchName` | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | BRAN-01 | — | Allocate creates valid branch + worktree | unit | `go test ./cmd/... -run TestWorktreeAllocate` | ❌ W0 | ⬜ pending |
| 06-02-01 | 02 | 1 | BRAN-02 | — | List shows all tracked worktrees with status | unit | `go test ./cmd/... -run TestWorktreeList` | ❌ W0 | ⬜ pending |
| 06-03-01 | 03 | 2 | BRAN-03 | — | Orphan scan detects stale worktrees | unit | `go test ./cmd/... -run TestOrphanScan` | ❌ W0 | ⬜ pending |
| 06-04-01 | 04 | 2 | BRAN-04 | T-06-02 | Merge refuses if tests fail or conflicts | unit | `go test ./cmd/... -run TestMergeBack` | ❌ W0 | ⬜ pending |
| 06-04-02 | 04 | 2 | BRAN-05 | T-06-03 | Post-merge cleanup removes branch + directory | unit | `go test ./cmd/... -run TestMergeCleanup` | ❌ W0 | ⬜ pending |
| 06-05-01 | 05 | 3 | BRAN-06 | — | Integration: full lifecycle allocate→merge→cleanup | integration | `go test ./cmd/... -run TestLifecycle` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. Go test framework is already set up.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Worktree directory actually cleaned from filesystem | BRAN-05 | Requires real git worktree on disk | Run `aether worktree-merge-back` then verify directory removed with `ls` |
| Orphan scan detects real stale worktrees | BRAN-03 | Requires real worktree with no activity | Create worktree, wait, run `aether worktree-orphan-scan` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
