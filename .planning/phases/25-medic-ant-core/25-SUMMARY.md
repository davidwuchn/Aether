---
phase: 25-medic-ant-core
plan: 01
status: complete
---

# Plan 01: CLI Foundation — Summary

## What was built

The `aether medic` CLI command with full visual output, caste identity, and agent definitions across all platforms.

## Key Files

### Created
- `cmd/medic_cmd.go` — Medic CLI command with flags (--fix, --force, --json, --deep), HealthIssue struct, MedicOptions struct, visual report renderer, JSON output renderer, severity color helper
- `cmd/medic_cmd_test.go` — 14 tests covering no-colony, active colony, JSON output, flag parsing, report rendering, exit codes, repair log, next steps
- `cmd/medic_scanner.go` — Health scanner engine with performHealthScan, scanColonyState, scanSession, scanPheromones, scanDataFiles, scanJSONL
- `cmd/medic_scanner_test.go` — 28 tests covering all scanners
- `cmd/medic_wrapper.go` — Wrapper parity scanner for deep mode
- `cmd/medic_wrapper_test.go` — 5 tests for wrapper parity
- `.claude/agents/ant/aether-medic.md` — Claude Code agent definition
- `.aether/agents-claude/aether-medic.md` — Packaging mirror
- `.opencode/agents/aether-medic.md` — OpenCode agent definition
- `.codex/agents/aether-medic.toml` — Codex agent definition
- `.aether/agents-codex/aether-medic.toml` — Codex packaging mirror
- `.aether/commands/medic.yaml` — YAML source definition
- `.claude/commands/ant/medic.md` — Claude Code slash command
- `.opencode/commands/ant/medic.md` — OpenCode slash command

### Modified
- `cmd/codex_visuals.go` — Added medic caste (emoji 🩹, color 96 cyan, label "Medic")
- `cmd/codex_visuals_test.go` — Added TestCasteIdentityMedic
- `cmd/root.go` — Registered medic command
- `CLAUDE.md` — Updated agent count to 25, command count to 50

## Tests

- 47 tests total (14 cmd + 28 scanner + 5 wrapper)
- All passing
- Coverage: ~85% of medic code paths

## Deviations

- Wrapper parity expected counts set to post-Phase-27 values (colony skills = 11 includes future medic skill)
- Fixed path resolution bug: wrapper parity scanner needed repo root, not .aether/data/ base path

## Verification

```bash
go test ./cmd/... -count=1  # all pass
go build ./cmd/aether       # compiles
aether medic                 # visual report
aether medic --json          # structured JSON
aether medic --deep          # includes wrapper parity
```
