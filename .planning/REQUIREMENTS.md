# Requirements: Aether v1.0 Final Sweep

**Defined:** 2026-04-07
**Core Value:** Developers can describe a goal and have a self-organizing colony of agents build, test, and deliver it with structured orchestration, state protection, and accumulated wisdom.

## v1 Requirements

### State Protection (STATE)

- [ ] **STATE-01**: Every mutation to COLONY_STATE.json is recorded in an append-only audit log (state-changelog.jsonl)
- [ ] **STATE-02**: Audit log entries contain before/after diffs, timestamp, source command, and SHA-256 checksum
- [ ] **STATE-03**: Planning history is append-only — past phase plans cannot be overwritten, only extended
- [ ] **STATE-04**: User can view full mutation history via `aether state-history` command
- [ ] **STATE-05**: State corruption (known jq-expression bug) is detected and rejected at write time with clear error message
- [ ] **STATE-06**: Checkpoint snapshots of COLONY_STATE.json are created automatically before destructive operations
- [ ] **STATE-07**: BoundaryGuard protects sensitive paths from unauthorized writes during colony operations

### Build Depth (DEPTH)

- [x] **DEPTH-01**: User can set build depth via `/ant:init` or `/ant:build --depth` with three levels: light, standard, deep
- [ ] **DEPTH-02**: Light depth skips archaeologist pre-build scan, limits to 1 builder, skips measurer and ambassador
- [ ] **DEPTH-03**: Standard depth runs full build playbook with balanced spawn counts (current default behavior)
- [ ] **DEPTH-04**: Deep depth runs all specialists including measurer, ambassador, increased chaos iterations, and extended verification
- [ ] **DEPTH-05**: Colony-prime respects depth level when assembling worker context (adjusts token budget per depth)
- [ ] **DEPTH-06**: Depth setting persists in COLONY_STATE.json and is visible in `/ant:status`

### Planning Granularity (PLAN)

- [ ] **PLAN-01**: User can select planning granularity when running `/ant:plan` with four ranges: sprint (1-3), milestone (4-7), quarter (8-12), major (13-20)
- [ ] **PLAN-02**: Route-setter receives min/max phase constraints from selected granularity and generates plans within bounds
- [ ] **PLAN-03**: If generated plan exceeds selected range, user is warned and asked to approve or adjust
- [ ] **PLAN-04**: Planning granularity persists in COLONY_STATE.json and is visible in `/ant:status`
- [ ] **PLAN-05**: Autopilot (`/ant:run`) respects the selected planning granularity across all phases

### Orchestration (ORCH)

- [ ] **ORCH-01**: PhaseOrchestrator decomposes each phase into tasks and assigns specialist agents based on task type
- [ ] **ORCH-02**: TaskRouter maps task descriptions to agent castes (builder, watcher, scout, chaos, etc.)
- [ ] **ORCH-03**: Agents are isolated per task — no agent acts outside its assigned phase or skill scope
- [ ] **ORCH-04**: All agent outputs are handed back to the Orchestrator before progressing to next phase
- [ ] **ORCH-05**: Orchestrator validates outputs against success criteria before marking tasks complete
- [ ] **ORCH-06**: Agent-role contracts are explicit, versioned, and reusable across colonies
- [ ] **ORCH-07**: Orchestrator maintains full visibility of system state across all active agents

### Branching & Worktree Discipline (BRAN)

- [ ] **BRAN-01**: Branch naming convention is enforced: feature/, fix/, experiment/, colony/ prefixes required
- [ ] **BRAN-02**: Worktree lifecycle is tracked — creation, assignment, merge status, and cleanup are recorded
- [ ] **BRAN-03**: Stale worktree detection identifies worktrees with no recent activity and flags for cleanup
- [ ] **BRAN-04**: Merge protocol requires passing tests and verification before merge to main
- [ ] **BRAN-05**: Auto-cleanup runs after merge — worktree branch removed, worktree directory cleaned
- [ ] **BRAN-06**: No orphaned branches remain — Queen tracks all spawned worktrees and ensures cleanup

### System Integrity (INTG)

- [ ] **INTG-01**: All primary `aether` commands run without error on a clean installation
- [ ] **INTG-02**: No orphaned shell scripts remain in active code paths (deprecated scripts clearly marked)
- [ ] **INTG-03**: Error handling across all Go modules produces consistent, readable, actionable log output
- [ ] **INTG-04**: The `isTestArtifact` function in data-clean no longer false-positives on legitimate user data
- [ ] **INTG-05**: `backup-prune-global` and `temp-clean` require confirmation before deleting files or create audit trail
- [ ] **INTG-06**: All 524+ existing tests continue passing with no regressions

### Repository Hygiene (HYGN)

- [ ] **HYGN-01**: Directory structure is clean — no legacy clutter, no ambiguous old/new system files
- [ ] **HYGN-02**: README and all documentation reflect current Go architecture with zero shell references
- [ ] **HYGN-03`: Naming conventions are consistent across cmd/, pkg/, .aether/, .claude/, .opencode/
- [ ] **HYGN-04**: Dead code is removed — unused functions, unreachable paths, commented-out blocks
- [ ] **HYGN-05**: A new contributor can understand repo structure in under 5 minutes

### Final Validation (VALD)

- [ ] **VALD-01**: Full test suite passes with race detection (`go test ./... -race`)
- [ ] **VALD-02**: Simulated real workflow completes end-to-end: init → colonize → plan → build → continue → seal
- [ ] **VALD-03**: Edge cases are stress-tested: concurrent state mutations, large colony states, rapid phase cycling
- [ ] **VALD-04**: Orchestration flow is validated across multiple sequential phases with specialist handoffs
- [ ] **VALD-05**: Any remaining gaps are documented in known-issues.md with severity ratings

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced State

- **STATE-08**: Event sourcing with full state reconstruction from audit log
- **STATE-09**: Cross-colony state synchronization via hive

### Advanced Orchestration

- **ORCH-08**: Dynamic agent spawning based on task complexity analysis
- **ORCH-09**: Graph-based task routing with dependency resolution

### Advanced Branching

- **BRAN-07**: Branch protection rules enforced via git hooks
- **BRAN-08**: Stale branch cleanup automation with configurable TTL

## Out of Scope

| Feature | Reason |
|---------|--------|
| SQLite/BoltDB for state storage | JSON files with file locking sufficient for single-user CLI |
| go-git library | shelling out to git is simpler and already works |
| Version vectors | Single-user system — no concurrent writers to coordinate |
| WAL (Write-Ahead Log) | Append-only audit log covers the same need without complexity |
| GitFlow branching | Too heavy for solo/small-team development on this project |
| Mandatory PR reviews | No team size to justify — commit suggestion workflow sufficient |
| Merge queues | Not needed until multiple concurrent contributors |
| Viper for configuration | Cobra + JSON is simpler and already integrated |
| gRPC / WASM plugins | Massive complexity for no current use case |
| Peer-to-peer agent negotiation | Orchestrator pattern is simpler and more predictable |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| STATE-01 | Phase 1 | Pending |
| STATE-02 | Phase 1 | Pending |
| STATE-03 | Phase 1 | Pending |
| STATE-04 | Phase 1 | Pending |
| STATE-05 | Phase 1 | Pending |
| STATE-06 | Phase 1 | Pending |
| STATE-07 | Phase 1 | Pending |
| DEPTH-01 | Phase 2 | Complete |
| DEPTH-02 | Phase 2 | Pending |
| DEPTH-03 | Phase 2 | Pending |
| DEPTH-04 | Phase 2 | Pending |
| DEPTH-05 | Phase 2 | Pending |
| DEPTH-06 | Phase 2 | Pending |
| PLAN-01 | Phase 3 | Pending |
| PLAN-02 | Phase 3 | Pending |
| PLAN-03 | Phase 3 | Pending |
| PLAN-04 | Phase 3 | Pending |
| PLAN-05 | Phase 3 | Pending |
| ORCH-01 | Phase 4 | Pending |
| ORCH-02 | Phase 4 | Pending |
| ORCH-03 | Phase 4 | Pending |
| ORCH-04 | Phase 4 | Pending |
| ORCH-05 | Phase 4 | Pending |
| ORCH-06 | Phase 4 | Pending |
| ORCH-07 | Phase 4 | Pending |
| BRAN-01 | Phase 5 | Pending |
| BRAN-02 | Phase 5 | Pending |
| BRAN-03 | Phase 5 | Pending |
| BRAN-04 | Phase 5 | Pending |
| BRAN-05 | Phase 5 | Pending |
| BRAN-06 | Phase 5 | Pending |
| INTG-01 | Phase 6 | Pending |
| INTG-02 | Phase 6 | Pending |
| INTG-03 | Phase 6 | Pending |
| INTG-04 | Phase 6 | Pending |
| INTG-05 | Phase 6 | Pending |
| INTG-06 | Phase 6 | Pending |
| HYGN-01 | Phase 7 | Pending |
| HYGN-02 | Phase 7 | Pending |
| HYGN-03 | Phase 7 | Pending |
| HYGN-04 | Phase 7 | Pending |
| HYGN-05 | Phase 7 | Pending |
| VALD-01 | Phase 8 | Pending |
| VALD-02 | Phase 8 | Pending |
| VALD-03 | Phase 8 | Pending |
| VALD-04 | Phase 8 | Pending |
| VALD-05 | Phase 8 | Pending |

**Coverage:**
- v1 requirements: 46 total
- Mapped to phases: 46
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-07*
*Last updated: 2026-04-07 after initial definition*
