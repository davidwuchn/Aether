---
phase: 08
slug: slash-command-wiring
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-04
---

# Phase 08 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./cmd/ -run TestSlash -count=1 -timeout 30s` |
| **Full suite command** | `go test ./cmd/... -count=1 -timeout 120s` |
| **Estimated runtime** | ~30 seconds (full), ~5s (quick) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run TestSlash -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./cmd/... -count=1 -timeout 120s`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | WIRE-01 | unit | `go test ./cmd/ -run TestFlagList -count=1` | inline TDD | ⬜ pending |
| 08-01-01 | 01 | 1 | WIRE-01 | unit | `go test ./cmd/ -run TestHistory -count=1` | inline TDD | ⬜ pending |
| 08-01-01 | 01 | 1 | WIRE-01 | unit | `go test ./cmd/ -run TestPhase -count=1` | inline TDD | ⬜ pending |
| 08-01-02 | 01 | 1 | WIRE-01 | unit | `go test ./cmd/ -run TestNormalizeArgs -count=1` | inline TDD | ⬜ pending |
| 08-02-01 | 02 | 1 | WIRE-02 | unit | `grep -c 'bash .aether/aether-utils.sh' .aether/commands/*.yaml \|\| grep -v ':0$' \|\| echo "CLEAN"` | created inline | ⬜ pending |
| 08-02-02 | 02 | 1 | WIRE-03 | unit | `grep -c 'bash .aether/aether-utils.sh' .aether/commands/*.yaml \|\| grep -v ':0$' \|\| echo "ALL YAML FILES CLEAN"` | created inline | ⬜ pending |
| 08-03-01 | 03 | 2 | WIRE-01 | integration | `node bin/generate-commands.js --check` | created inline | ⬜ pending |
| 08-03-02 | 03 | 2 | WIRE-02 | integration | `go build -o aether ./cmd/aether && ./aether status 2>&1 \| head -5` | created inline | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Test Coverage Strategy

Plan 01 uses inline TDD (`tdd="true"`) — test files are created as part of the RED-GREEN-REFACTOR cycle within each task, not as a separate Wave 0. This satisfies the Nyquist requirement because:

1. **Task 08-01-01** creates `cmd/flags_test.go`, `cmd/history_test.go`, `cmd/phase_test.go` inline
2. **Task 08-01-02** creates `cmd/normalize_args_test.go` inline
3. Both tasks write failing tests first (RED), then implement (GREEN), ensuring coverage exists before code

No separate Wave 0 tasks are needed because the TDD tasks produce their own test scaffolding.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `/ant:status` output matches Go binary | WIRE-01 | Requires Claude Code runtime | Run `/ant:status` in Claude Code, verify dashboard renders correctly |
| `/ant:pheromones` output identical | WIRE-01 | Requires Claude Code runtime | Compare output of `aether pheromone-display` vs `bash .aether/aether-utils.sh pheromone-display` |
| Deprecation notice on stderr | WIRE-02 | Visual inspection | Trigger a fallback command, check stderr for deprecation message |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or inline TDD coverage
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Test coverage provided inline via TDD tasks (no separate Wave 0 needed)
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter
