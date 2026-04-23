---
phase: 33-dispatch-fixes
plan: 02
subsystem: runtime

tags: [go, ceremony-restoration, archaeologist, cross-platform]

requires:
  - phase: 33-01
    provides: P1 dispatch robustness fixes

provides:
  - Platform-aware context-clear guidance (Claude /ant:resume, Codex aether resume)
  - Archaeologist ant dispatched in full-depth builds
  - Dead Pool code removed
  - Failing visual test fixed

affects:
  - All platform users (Claude, Codex, OpenCode)
  - Build ceremony experience

tech-stack:
  added: []
  patterns:
    - "Platform-aware command references"
    - "Full-depth build ceremony with archaeology phase"

key-files:
  created: []
  modified:
    - cmd/codex_visuals.go — renderContextClearGuidanceForPlatform, detectPlatform
    - cmd/codex_build.go — archaeologist dispatch, playbook filtering, expected outcome
    - cmd/codex_build_test.go — updated expectations for 7 dispatches
    - cmd/codex_visuals_test.go — fixed snapshot mismatch

key-decisions:
  - "Platform detection via AETHER_PLATFORM env var, fallback to CODEX_CLI/CODEX_API_KEY detection"
  - "Archaeologist only in 'full' depth (not standard)"
  - "Cross-platform default: aether resume (works everywhere)"

requirements-completed: []

# Metrics
duration: 20m
completed: 2026-04-22
---

# Phase 33 Plan 02: Ceremony Restoration Summary

**Colony ceremony elements restored from v1.0.1 era — cross-platform command guidance, archaeologist in build flow, dead code removed.**

## Performance
- **Duration:** 20 min (combined with Plan 01)
- **Completed:** 2026-04-22
- **Files modified:** 4

## Accomplishments
- `renderContextClearGuidance()` now calls `renderContextClearGuidanceForPlatform()`
- Platform detection via env vars: `AETHER_PLATFORM`, `CODEX_CLI`, `CODEX_API_KEY`
- Claude/OpenCode users see `/ant:resume`, Codex users see `aether resume`
- Archaeologist ant (🏺) dispatched in "full" depth builds for git history analysis
- Dead Pool code identified and removed from `pkg/agent/pool.go`
- `TestContinueVisualOutputShowsVerificationArtifactsAndSpawnTree` now passes

## Files Modified
- `cmd/codex_visuals.go` — Platform-aware context-clear guidance
- `cmd/codex_build.go` — Archaeologist dispatch, playbook filtering, expected outcome
- `cmd/codex_build_test.go` — Updated dispatch count expectations (6→7)
- `pkg/agent/pool.go` — Dead code removed

## Self-Check: PASSED
- [x] Context-clear guidance shows correct command per platform
- [x] Archaeologist dispatched in full-depth builds
- [x] All cmd tests pass
- [x] Visual test passes

---
*Phase: 33-dispatch-fixes*
*Completed: 2026-04-22*
