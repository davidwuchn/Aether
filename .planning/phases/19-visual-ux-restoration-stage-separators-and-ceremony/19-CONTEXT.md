# Phase 19: Visual UX Restoration — Stage Separators and Ceremony - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 19 restores and standardizes stage separators (`── Stage Name ──`) and post-command ceremony across all Aether commands. The runtime `aether` CLI remains the authoritative source for all visual output.

The goal is to make every command tell the user what happened, what to run next, and whether it is safe to `/clear` context — driven by runtime truth, not wrapper invention.

This phase touches all 49 commands because the user wants consistent ceremony everywhere. Build and continue are the priority since they already have partial ceremony from Phases 3 and 4.

</domain>

<decisions>
## Implementation Decisions

### Stage separator coverage
- **D-01:** Add full stage markers to build output: `── Context ──`, `── Tasks ──`, `── Dispatch ──`, matching the continue pattern that already uses `── Verification ──`, `── Housekeeping ──`, `── Next Phase ──`, `── Colony Complete ──`.
- **D-02:** Continue output already has stage markers for Verification, Housekeeping, Colony Complete, and Next Phase. Verify they are consistent and complete.

### Command ceremony scope
- **D-03:** All 49 commands get post-command ceremony guidance. This is a large scope but the user explicitly requested it.
- **D-04:** Build and continue are the priority commands. Other commands can be done in waves if needed.

### Runtime vs wrapper ownership
- **D-05:** Go runtime owns ALL stage separators and post-command ceremony. Wrappers remain thin pass-throughs that frame the moment but do not invent UI.
- **D-06:** The runtime already renders stage markers in `renderStageMarker()` — expand this to cover all relevant commands.
- **D-07:** Wrappers must NOT duplicate or contradict runtime ceremony. If wrapper docs and runtime output disagree, runtime wins (established in Phase 3).

### Context-clear guidance
- **D-08:** Move context-clear guidance ('It's safe to clear your context now') from wrappers into the Go runtime for all state-changing commands.
- **D-09:** Only the runtime can truthfully say whether it's safe to clear context, because only the runtime knows whether the command completed successfully.
- **D-10:** Context-clear guidance appears after successful completion of state-changing commands, not after blocked or failed commands.

### Codex parity
- **D-11:** Since ceremony is runtime-only, Codex CLI automatically gets the same stage separators and guidance as Claude Code. No extra Codex work needed.
- **D-12:** Codex remains out of scope for wrapper ceremony, as established in Phase 3.

### Implementation approach
- **D-13:** For the 49 commands, identify which ones are state-changing vs read-only. State-changing commands get full ceremony (stage markers + next-step + context-clear). Read-only commands get minimal ceremony (next-step suggestion only).
- **D-14:** Use `renderStageMarker()` consistently. Do not hand-write separator strings.
- **D-15:** The `visualDivider` constant (`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`) is for banners, not stage sections. Stage sections use `renderStageMarker()`.

### Claude's Discretion
- Choose the exact stage marker names as long as they are consistent across build and continue.
- Decide how to batch the 49 commands if full implementation exceeds phase capacity.
- Choose the exact wording of context-clear guidance and next-step prompts.
- Determine which commands are "state-changing" vs "read-only" for ceremony intensity.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and success criteria
- `.planning/PROJECT.md` — Core value: Aether should feel alive and truthful at runtime.
- `.planning/REQUIREMENTS.md` — R028 defines the stage separator and ceremony contract.
- `.planning/ROADMAP.md` — Phase 19 goal, dependency boundary (depends on Phase 18), and success criteria.
- `.planning/STATE.md` — Carry-forward decisions from Phases 3 and 4 (wrapper/runtime authority, Codex out-of-scope for wrapper ceremony).

### Source-of-truth implementation surfaces
- `cmd/codex_visuals.go` — `renderStageMarker()`, `renderBanner()`, `visualDivider`, caste identity maps. The authoritative renderer for all visual output.
- `cmd/codex_build_progress.go` — Build wave progress with `renderStageMarker("Dispatch")`.
- `cmd/codex_visuals_test.go` — Regression tests for stage marker rendering.

### Wrapper surfaces (reference only — do not edit as primary)
- `.aether/commands/build.yaml` — Build wrapper source.
- `.aether/commands/continue.yaml` — Continue wrapper source.
- `.claude/commands/ant/build.md` — Generated Claude build wrapper.
- `.claude/commands/ant/continue.md` — Generated Claude continue wrapper.
- `.opencode/commands/ant/build.md` — Generated OpenCode build wrapper.
- `.opencode/commands/ant/continue.md` — Generated OpenCode continue wrapper.

### Prior phase context
- `.planning/phases/03-restore-build-ceremony/03-CONTEXT.md` — Build ceremony decisions from Phase 3.
- `.planning/phases/04-restore-continue-ceremony/04-CONTEXT.md` — Continue ceremony decisions from Phase 4.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/codex_visuals.go`:
  - `renderStageMarker(title string) string` — Already exists and used for Verification, Housekeeping, Colony Complete, Next Phase, Dispatch.
  - `renderBanner(emoji, title string) string` — Used for command banners (e.g., `━━ 🔨 Build Phase N ━━`).
  - `visualDivider` constant — Full-width divider used after banners.
  - `renderNextUp(primary, secondary string) string` — Already renders next-step guidance.
- `cmd/codex_build_progress.go` — Already uses `renderStageMarker("Dispatch")` for wave dispatch output.
- `cmd/codex_visuals_test.go` — Already tests stage marker rendering.

### Established Patterns
- Runtime commands that render visuals follow this pattern: `renderBanner()` → `visualDivider` → content sections → `renderStageMarker()` for subsections → `renderNextUp()` for next steps.
- Commands that don't render visuals (JSON mode, non-TTY) skip all of this via `shouldRenderVisualOutput()`.
- The `emitVisualProgress()` helper handles conditional visual emission.

### Integration Points
- Build visual is rendered by `renderBuildVisualWithDispatches()` in `cmd/codex_visuals.go`.
- Continue visual is rendered by `renderContinueVisual()` in `cmd/codex_visuals.go`.
- Other commands (status, seal, entomb, etc.) have their own render functions.
- All visual output funnels through `stdout` when `AETHER_OUTPUT_MODE=visual` or TTY detected.

### Gaps to fill
- `renderBuildVisualWithDispatches()` does NOT use `renderStageMarker()` for Tasks section — it uses plain "Tasks\n".
- `renderBuildVisualWithDispatches()` does NOT have a Context section with stage marker.
- Context-clear guidance currently lives in `.claude/commands/ant/continue.md` wrapper, not in the runtime.
- Most commands outside build/continue do not have `renderStageMarker()` calls at all.

</code_context>

<specifics>
## Specific Ideas

- The v1.0.5 experience that felt "alive" was runtime-driven. Wrappers were thin. This phase returns to that model.
- Stage markers should feel like a natural breathing rhythm in the output: banner → divider → stage marker → content → stage marker → content → next steps.
- Context-clear guidance should be explicit but brief: "It's safe to clear your context now. Run `/ant:resume` to restore."
- For read-only commands (status, watch, history), a simple next-step suggestion is enough: "Run `/ant:build N` to continue."

</specifics>

<deferred>
## Deferred Ideas

- None — all discussed ideas fit within Phase 19 scope.

</deferred>

---

*Phase: 19-visual-ux-restoration-stage-separators-and-ceremony*
*Context gathered: 2026-04-21*
