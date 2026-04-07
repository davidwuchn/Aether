# Phase 1: State Protection - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 01-state-protection
**Areas discussed:** Audit scope, state-history UX, state-write safety, Checkpoint policy

---

## Audit Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Significant mutations only | Log state-mutate, state-write, phase-insert, phase advance, plan changes. Skip context-update and build internals. | ✓ |
| Everything | Log every single state touch including context-update during builds. | |
| Significant + opt-in verbose | Significant by default, --verbose flag for build internals. | |

**User's choice:** Significant mutations only
**Notes:** User wants meaningful changes tracked without log spam from high-frequency build operations.

---

## state-history UX

| Option | Description | Selected |
|--------|-------------|----------|
| Compact summary | One line per mutation: timestamp, command, field changed, summary. Like `git log --oneline`. | |
| Verbose with diffs | Full before/after diffs for every entry. Like `git log -p`. | |
| Compact + opt-in diffs | Default compact, --diff flag for full before/after on specific entries. | ✓ |

**User's choice:** Compact + opt-in diffs
**Notes:** User wants quick scanning by default with the ability to drill into specific changes.

---

## state-write Safety

| Option | Description | Selected |
|--------|-------------|----------|
| --force guard | Keep state-write but require --force. Without it, refuse and suggest state-mutate. | ✓ |
| Deprecate/remove | Remove state-write entirely. All mutations go through state-mutate. | |
| Keep unchanged | Keep as-is. Audit log catches unexpected writes. | |

**User's choice:** --force guard
**Notes:** User wants safety without breaking agents that need direct writes. Explicit opt-in via --force.

---

## Checkpoint Policy

| Option | Description | Selected |
|--------|-------------|----------|
| Destructive ops only | Auto-checkpoint before phase advance, plan overwrite, state-write --force. | ✓ |
| Every mutation | Auto-checkpoint before any state change. Needs rotation. | |
| Manual only | No auto-checkpoints. Audit log is enough. | |

**User's choice:** Destructive ops only
**Notes:** User wants snapshots for hard-to-undo operations, not for every minor change.

---

## Claude's Discretion

Areas where Claude has flexibility during planning/implementation:
- Audit log JSONL schema design
- Checkpoint rotation/retention policy
- state-history output formatting details
- BoundaryGuard implementation approach
- Corruption detection heuristics beyond the known jq-expression bug
- Whether the audit log itself should be checksummed or signed

## Deferred Ideas

None — discussion stayed within phase scope.
