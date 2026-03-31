---
phase: 43-clash-detection-integration
plan: 02
status: complete
created: 2026-03-31
---

# Plan 43-02: Wire Clash Detection and Merge Driver into Init

## Objective
Wire clash detection hook installation and merge driver setup into `/ant:init` so that new colonies automatically get clash protection.

## Tasks Completed

### Task 1: Add Step 11 to init.md for clash detection and merge driver setup
- **Status:** Complete
- Added Step 11 (clash hook install + merge driver registration) to both `.claude/commands/ant/init.md` and `.opencode/commands/ant/init.md`
- Removed `.claude/settings.json` from init.md read-only list (resolves D-01 conflict)
- Renumbered subsequent steps (12→13)
- Both setup operations are non-blocking (`|| true`)

### Task 2: Verify clash-setup --install works end-to-end
- **Status:** Complete
- Verified hook installs correctly alongside existing gsd-prompt-guard PreToolUse entry
- Verified idempotency (no duplicate entries on re-run)
- Verified uninstall removes only clash hook, preserves gsd-prompt-guard
- Merge driver confirmed working (`merge-driver-lockfile.sh` exits 0)

## Verification

- `bash test/test-clash-detect.sh` — 7/7 pass
- `bash test/test-clash-pre-tool-use.sh` — 8/8 pass
- `bash test/test-clash-subcommands.sh` — 10/10 pass
- `bash test/test-clash-integration.sh` — 13/13 pass

## Key Files
- `.claude/commands/ant/init.md` — Step 11 added, read-only list updated
- `.opencode/commands/ant/init.md` — Step 11 added (provider parity)
- `test/test-clash-integration.sh` — 13-assertion integration test (new)

## Requirements Addressed
- CLASH-01: PreToolUse hook installed during /ant:init
- CLASH-03: Merge driver registered during /ant:init
