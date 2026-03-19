# Aether Maintenance & Pheromone Integration

## What This Is

Aether is a multi-agent colony orchestration system for AI-assisted development. It provides 36 slash commands, 22 specialized worker agents, and a pheromone signaling system for guiding colony behavior. This project is a maintenance push to clean up test artifacts, fix broken commands, make the pheromone system actually work end-to-end, update outdated docs, and ensure a clean fresh-install experience.

## Core Value

The pheromone system should be a living system — auto-emitting signals during builds, carrying context across sessions, and actually changing worker behavior — not just a storage format that nobody reads.

## Requirements

### Validated

- Colony lifecycle works (init, plan, build, continue, seal, entomb) — existing
- 22 worker agents defined with caste roles — existing
- State management with file locking and atomic writes — existing
- Pheromone signal storage (FOCUS/REDIRECT/FEEDBACK) — existing
- XML exchange system for cross-colony knowledge transfer — existing
- 490+ tests passing (AVA + bash) — existing
- NPM distribution via `aether-colony` package — existing
- Multi-provider support (Claude Code + OpenCode) — existing
- Midden failure tracking system — existing
- QUEEN.md wisdom promotion pipeline — existing

### Active

- [ ] Clean test artifacts from colony memory (QUEEN.md, pheromones.json, constraints.json)
- [ ] Reset stale colony state (COLONY_STATE.json from wrong project)
- [ ] Archive unused XML exchange system (built but never integrated into commands)
- [ ] Make pheromones auto-emit during builds based on discovered patterns
- [ ] Make pheromones carry across sessions (survive /clear and resume)
- [ ] Make workers actually read and act on pheromone signals
- [ ] Fix broken or unreliable slash commands
- [ ] Update outdated documentation (CLAUDE.md, README, docs/)
- [ ] Ensure fresh install works cleanly (npm install -g, aether update in a new repo)

### Out of Scope

- Splitting aether-utils.sh into modules — large refactor, separate initiative
- Web/TUI dashboard — nice to have, not this round
- Multi-repo colony coordination — future architecture work
- Performance optimization (state caching, lock backoff) — defer unless blocking

## Context

- Aether is at v1.1.0, published on npm as `aether-colony`
- The codebase has accumulated test data in QUEEN.md (25+ junk entries), pheromones.json (5 test signals), and constraints.json (entire focus array is test data)
- COLONY_STATE.json contains a stale goal from a different project (Electron-to-Xcode migration from February)
- The XML exchange system (.aether/exchange/, .aether/schemas/) was fully built but never wired into actual commands
- The CONCERNS.md audit identified pheromone integration as a key gap: signals are stored but don't influence worker behavior
- Commands are split into playbooks (.aether/docs/command-playbooks/) for reliability
- The codebase mapper identified 390 lines of concerns including security, tech debt, and test coverage gaps

## Constraints

- **Testing**: All changes must maintain 490+ passing tests; new features need tests
- **Compatibility**: Must work with bash 4+, Node 16+, jq 1.6+
- **Distribution**: Changes must pass `bin/validate-package.sh` before publish
- **No breaking changes**: Existing colonies using Aether must not break on update

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Archive XML system (don't integrate) | Built but unused; JSON system works; integration effort unclear payoff | -- Pending |
| Pheromone integration is top priority | User's primary pain point; signals exist but don't influence behavior | -- Pending |
| Fresh install as "done" test | If someone can install and run a colony without issues, maintenance is complete | -- Pending |

---
*Last updated: 2026-03-19 after initialization*
