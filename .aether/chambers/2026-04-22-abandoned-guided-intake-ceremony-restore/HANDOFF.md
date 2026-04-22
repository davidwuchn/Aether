# Colony Session — Paused Colony

Updated: 2026-04-22T09:20:42Z

## Goal

- Restore guided colony intake so Aether synthesizes the best colony goal before init, asks for planning depth, clarification depth, verification strictness, and execution mode, persists those choices in colony state, gives explicit next-command guidance at every lifecycle step, and tells the user exactly when to clear context or resume across Claude, OpenCode, and Codex.

## Phase

- Current: 1/6 — Contract and gap mapping
- State: READY

## Signals

- None

## Blockers

- cmd/codex_command_contract_test.go requires .aether/docs/codex-command-surface-contract.md, but the file is absent in the repo. Even after the compile break ...
- Fresh 'AETHER_OUTPUT_MODE=json go test ./... -count=1' is not fully runnable in this environment. Port-binding tests panic with 'listen tcp6 [::1]:0: bind: o...
- Verification needed an explicit retry because ambient AETHER_OUTPUT_MODE=visual makes JSON-parsing tests fail. Example: 'AETHER_OUTPUT_MODE=visual go test ./...
- cmd/codex_continue.go:380-384 treats the whole tests step as passed whenever the output contains any environmental marker. A mixed failure run (real regressi...
- cmd/codex_continue.go:402-405 skips any failure line that matches isEnvironmentalConstraintText(), and the matcher at 777-800 treats the generic substring 'o...
- cmd/codex_worker_activity_test.go shows aether continue replaces a completed builder's honest summary with generic closure text, so the live worker record is...
- A live smoke repo in /tmp reached ━━ 🏃 R U N ━━
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Autopilot loop executed.
State: EXECUTING
Current Phase: 6

━━ 🐜...

## Next Step

- Run `aether resume`
- Quick restore: `aether resume`
- Full restore: `aether resume-colony`

## Tasks

- Compare the documented ant workflow with the current Codex command behavior
- Decide the observable ant-process outputs Codex must emit during each core command

## Session Summary

Paused at phase 1
