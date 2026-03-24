# Aether Colony Orchestration System

## What This Is

Aether is a multi-agent colony orchestration system for AI-assisted development. It provides 44 slash commands, 22 specialized worker agents, 28 skills (10 colony + 18 domain), and a living pheromone signaling system. Distributed as `aether-colony` on npm, it works with Claude Code and OpenCode.

## Core Value

The system must reliably interpret a user request, decompose it into executable work, verify outputs, and ship correct work with minimal user back-and-forth — not just look autonomous, but actually deliver.

## Current Milestone: v2.1 Production Hardening

**Goal:** Address Oracle audit findings, make Aether genuinely production-ready — deeper planning, verified features, accurate docs, great first-user experience.

**Target areas:**
- Fix reliability gaps found by Oracle (silent failures, state desync, dead code)
- Deepen planning quality (per-phase research, not just quick decomposition)
- Verify every feature works end-to-end (skills, oracle, hive, pheromones)
- Update all documentation to match reality (README, CLAUDE.md, docs/)
- Modularize aether-utils.sh (extract unused code into optional modules)
- Publish and verify clean install experience

## Requirements

### Validated

- Colony lifecycle works (init, plan, build, continue, seal, entomb) — existing
- 22 worker agents defined with caste roles — existing
- State management with file locking and atomic writes — existing
- Pheromone signal storage (FOCUS/REDIRECT/FEEDBACK) — existing
- NPM distribution via `aether-colony` package — existing
- Multi-provider support (Claude Code + OpenCode) — existing
- Midden failure tracking system — existing
- QUEEN.md wisdom promotion pipeline — existing
- ✓ Clean colony state — all test artifacts purged — v1.3
- ✓ Pheromone injection chain — signals flow emit → store → inject → worker — v1.3
- ✓ Worker pheromone protocol — builder/watcher/scout act on signals — v1.3
- ✓ Learning pipeline — observations auto-promote to instincts in worker prompts — v1.3
- ✓ XML exchange — /ant:export-signals, /ant:import-signals, seal auto-export — v1.3
- ✓ Fresh install hardened — lifecycle smoke test + content-aware validate-package.sh — v1.3
- ✓ Documentation accuracy — all docs match verified behavior — v1.3
- ✓ 537+ tests passing (AVA + bash) — v1.3

### Active

(Defined in REQUIREMENTS.md for v2.1)

### Out of Scope

- Full rewrite of aether-utils.sh — extract modules, don't rewrite from scratch
- Web/TUI dashboard — CLI tool, ASCII dashboards work in terminal
- Multi-repo colony coordination — future architecture work
- Performance optimization (state caching, lock backoff) — defer unless blocking
- Agent Teams inter-worker communication — subagents can't communicate mid-execution
- Per-worker model routing — Claude Code Task tool doesn't support per-subagent env vars

## Context

- Aether is at v2.0.0, published on npm as `aether-colony`
- v2.0 shipped: Skills system (28 skills), oracle distribution fix, v2 release
- v1.3 shipped: Pheromone integration, learning pipeline, XML exchange, install hardening
- 44 Claude commands, 44 OpenCode commands, 22 agents, 178 subcommands (76 unused per Oracle audit)
- 11,272 lines in aether-utils.sh with rising bug-fix ratio (33.8% → 45.8%)
- Oracle audit (82% confidence, 55 findings) identified: silent error suppression (338 instances), state desync risks, 43% dead code, documentation drift
- Bug-fix ratio improving overall (17.5% Feb → 11.6% Mar) but spikes late in each period
- 572+ tests passing (1 pre-existing failure in context-continuity)

## Constraints

- **Testing**: All changes must maintain 537+ passing tests; new features need tests
- **Compatibility**: Must work with bash 4+, Node 16+, jq 1.6+
- **Distribution**: Changes must pass `bin/validate-package.sh` (content-aware) before publish
- **No breaking changes**: Existing colonies using Aether must not break on update

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Activate XML system (don't archive) | Cross-colony signal transfer has clear value | ✓ Good — /ant:export-signals and /ant:import-signals created |
| Pheromone integration is top priority | User's primary pain point; signals exist but don't influence behavior | ✓ Good — full injection chain + worker protocols |
| Fresh install as "done" test | If someone can install and run a colony without issues, maintenance is complete | ✓ Good — 430-line smoke test validates full lifecycle |
| Clean before integrating | Test data must be purged before pheromone integration can be validated | ✓ Good — Phase 1 first, integration phases after |
| Principle-based agent protocols | Workers are LLMs — they understand intent, don't need 100-line rule sets | ✓ Good — 35 lines per agent, all effective |
| Define "influence" structurally | Signal in prompt + agent has protocol = maximum testable without live LLM | ✓ Good — pragmatic definition, fully tested |

| Oracle distribution fix | oracle.sh excluded from npm by .npmignore blanket rule | ✓ Good — moved to .aether/utils/oracle/, position-aware HUB_EXCLUDE |
| Extract modules not rewrite | 76 unused subcommands should be modularized, not deleted | — Pending |
| Deepen planning quality | Aether plans too quickly vs GSD's per-phase research depth | — Pending |

---
*Last updated: 2026-03-23 after v2.1 milestone start*
