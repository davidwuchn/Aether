# Phase 19: Visual UX Restoration — Stage Separators and Ceremony - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-21
**Phase:** 19-visual-ux-restoration-stage-separators-and-ceremony
**Areas discussed:** Missing build stage markers, Which commands get ceremony, Runtime vs wrapper responsibility, Context-clear guidance placement

---

## Missing build stage markers

| Option | Description | Selected |
|--------|-------------|----------|
| Add full stage markers to build | Use `── Context ──`, `── Tasks ──`, `── Dispatch ──` in the build visual, matching the continue pattern. | ✓ |
| Add only Tasks marker | Just add `── Tasks ──` since that's the clearest gap. Keep the banner for the top section. | |
| Keep current layout | The banner + divider already sections the build well. Stage markers aren't needed in build. | |
| You decide | Let the implementation agent choose based on what looks best. | |

**User's choice:** Add full stage markers to build
**Notes:** User wants the build visual to match continue's stage marker pattern consistently.

---

## Which commands get ceremony

| Option | Description | Selected |
|--------|-------------|----------|
| Build and continue only | These are the main colony lifecycle commands that advance state. Other commands are inspection-only. | |
| Build, continue, init, seal, entomb | Any command that creates, advances, or completes colony state. | |
| All 49 commands | Every command should tell the user what to do next, even if it's just 'run /ant:status'. | ✓ |
| You decide | Let the implementation agent choose based on which commands actually mutate state. | |

**User's choice:** All 49 commands
**Notes:** User wants consistent ceremony everywhere. Build and continue are the priority.

---

## Runtime vs wrapper responsibility

| Option | Description | Selected |
|--------|-------------|----------|
| Go runtime only | All stage separators and post-command guidance come from the Go CLI. Wrappers stay thin. | ✓ |
| Both runtime and wrappers | Runtime handles build/continue markers; wrappers add guidance for other commands. More flexible. | |
| Wrappers for guidance, runtime for markers | Hybrid: runtime owns visual separators, wrappers own next-step routing for non-build commands. | |

**User's choice:** Go runtime only (after explanation)
**Notes:** User initially asked for recommendation. Explained via theatre analogy: runtime is the stage, wrappers are program notes. User chose runtime-only after the explanation.

---

## Context-clear guidance placement

| Option | Description | Selected |
|--------|-------------|----------|
| Move all to runtime | Every state-changing command in the Go CLI outputs whether /clear is safe. | ✓ |
| Keep in wrappers for now | Don't change what already works. Only move when we overhaul each command. | |
| Build/continue only | Move context-clear guidance into the runtime for build and continue, since those are the focus of this phase. | |

**User's choice:** Move all to runtime
**Notes:** User initially asked for recommendation. Recommended build/continue in Phase 19 and rest later. User rejected and chose to do all 49 commands now.

---

## Claude's Discretion

- Exact stage marker names
- Batching strategy for 49 commands
- Exact wording of context-clear guidance
- Classification of commands as "state-changing" vs "read-only"

## Deferred Ideas

- None
