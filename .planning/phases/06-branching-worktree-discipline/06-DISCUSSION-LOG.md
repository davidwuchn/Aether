# Phase 6: Branching & Worktree Discipline - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 06-branching-worktree-discipline
**Areas discussed:** Branch naming convention, Merge protocol gates, Worktree lifecycle tracking, Orchestrator integration

---

## Branch naming convention

| Option | Description | Selected |
|--------|-------------|----------|
| Two-track naming | Agents get phase-N/caste-name, humans get feature/fix/experiment/colony prefixes | ✓ |
| Unified prefix scheme | Everyone uses same prefixes: feature/, fix/, experiment/, colony/ | |
| Validate agents only | Only validate agent-spawned worktrees, human branches unrestricted | |

**User's choice:** Two-track naming
**Notes:** Separates agent automation from human workflows cleanly

### Enforcement level

| Option | Description | Selected |
|--------|-------------|----------|
| Reject invalid names | worktree-create rejects unrecognized names outright | |
| Warn but allow | Warning on invalid names, but creation proceeds | |

**User's choice:** Claude's discretion ("you decide")

---

## Merge protocol gates

| Option | Description | Selected |
|--------|-------------|----------|
| Tests + clash-check | Two gates: go test passes AND no file conflicts. Fast, catches biggest risks. | ✓ |
| Full gate | Tests + go vet + go fmt + clash-check. Stricter but slower. | |
| Tests only | Only tests must pass. Clash and formatting are warnings. | |

**User's choice:** Tests + clash-check

### Merge failure behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Block and report | Stop and create a flag/blocker. User decides what to do. | ✓ |
| Keep worktree, flag for retry | Keep alive with 'failed-merge' status for later retry. | |

**User's choice:** Block and report

---

## Worktree lifecycle tracking

| Option | Description | Selected |
|--------|-------------|----------|
| COLONY_STATE.json | Add worktree registry to ColonyState. Matches Phase 5 pattern. | ✓ |
| Separate worktrees.json | Lighter weight, doesn't bloat colony state. | |

**User's choice:** COLONY_STATE.json

### Stale threshold

| Option | Description | Selected |
|--------|-------------|----------|
| 24 hours | Flag after 1 day of no activity | |
| 48 hours | Flag after 2 days. More breathing room for multi-day features. | ✓ |
| Configurable threshold | Default 24h, adjustable per colony. | |

**User's choice:** 48 hours

---

## Orchestrator integration

| Option | Description | Selected |
|--------|-------------|----------|
| Through worktree-allocate | Orchestrator calls worktree-allocate directly. Single entry point. | ✓ |
| Orchestrator manages its own | Orchestrator manages internal worktree list, calls git directly. | |

**User's choice:** Through worktree-allocate

### Post-merge cleanup

| Option | Description | Selected |
|--------|-------------|----------|
| Full auto-cleanup | Remove worktree dir, delete branch, update state to 'merged'. | ✓ |
| Keep branch briefly | Mark as 'merged', remove dir, keep branch for a while. | |

**User's choice:** Full auto-cleanup

---

## Claude's Discretion

- Branch name enforcement level (reject vs warn)
- WorktreeEntry struct fields
- Orphan scan output format
- Whether to add individual worktree-status command
- Pre-commit hook integration for go vet/fmt

## Deferred Ideas

None — discussion stayed within phase scope.
