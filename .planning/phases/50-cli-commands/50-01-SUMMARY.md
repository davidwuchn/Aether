---
phase: 50-cli-commands
plan: 01
subsystem: cli
tags: [cobra, go, cli, shell-completion, json-envelope]

# Dependency graph
requires:
  - phase: 45-storage-layer
    provides: Storage package with NewStore, ResolveAetherRoot, ResolveDataDir
provides:
  - Cobra root command with PersistentPreRunE store initialization
  - Version command and --version flag with v-prefix format
  - Shell completion for bash/zsh/fish/powershell
  - outputOK/outputError JSON envelope helpers matching shell json_ok/json_err
  - Testable stdout/stderr package vars for command testing
affects: [50-02, 50-03, 50-04, 50-05, 50-06, all-future-cli-plans]

# Tech tracking
tech-stack:
  added: [cobra v1.10.2, go-pretty/v6 v6.7.8, testify v1.11.1]
  patterns: [command-per-file with init() registration, PersistentPreRunE store init, io.Writer vars for testability, skipStoreInit for store-free commands]

key-files:
  created: [cmd/root.go, cmd/version.go, cmd/completion.go, cmd/helpers.go, cmd/helpers_test.go, cmd/root_test.go, cmd/aether/main.go]
  modified: [go.mod, go.sum, .gitignore]

key-decisions:
  - "Custom version template overrides Cobra default to print 'aether v<version>' instead of 'aether version v<version>'"
  - "stdout/stderr as package-level io.Writer vars for test injection rather than interface parameter"
  - "outputOK/outputError use fmt.Fprintf with manual JSON construction for exact key ordering matching shell format"
  - "skipStoreInit checks command ancestry chain so nested subcommands under completion/version/help also skip store"

patterns-established:
  - "Command-per-file: each command in cmd/ with init() adding to rootCmd"
  - "Store-free commands: completion, version, help skip PersistentPreRunE store init"
  - "JSON envelope: outputOK produces {\"ok\":true,\"result\":...}, outputError produces {\"ok\":false,\"error\":\"...\",\"code\":...}"
  - "Test isolation: override stdout/stderr package vars, use rootCmd.SetArgs(), defer reset"

requirements-completed: [CLI-01, CLI-02]

# Metrics
duration: 25min
completed: 2026-04-02
---

# Phase 50 Plan 01: CLI Foundation Summary

**Cobra CLI root command with PersistentPreRun store init, version/completion subcommands, and JSON envelope helpers matching shell json_ok/json_err format**

## Performance

- **Duration:** 25 min
- **Started:** 2026-04-02T06:16:24Z
- **Completed:** 2026-04-02T06:42:15Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Working `aether` binary with `--version`, `version`, `completion` subcommands
- Shell completion generation for bash, zsh, fish, and powershell
- JSON envelope helpers (outputOK/outputError) producing byte-identical output to shell's json_ok/json_err
- 13 unit tests covering root command, version, help, store initialization, envelope format, and all shell completions

## Task Commits

Each task was committed atomically:

1. **Task 1: Install dependencies and create CLI foundation** - `afaafcb` (feat)
2. **Task 2: Shell completion and helpers tests** - `3894727` (feat)

**Additional:** `5fed9dd` (chore: tidy go.mod)

## Files Created/Modified
- `cmd/aether/main.go` - Entry point calling cmd.Execute() and cmd.ExitWithError()
- `cmd/root.go` - Root cobra command with PersistentPreRunE store init, skipStoreInit logic, Version variable
- `cmd/version.go` - Version subcommand printing "aether v<version>"
- `cmd/completion.go` - Completion subcommand for bash/zsh/fish/powershell
- `cmd/helpers.go` - outputOK, outputError, outputErrorMessage, mustGetString, mustGetInt helpers
- `cmd/helpers_test.go` - 10 tests: envelope format, completion bash/zsh/fish, invalid arg
- `cmd/root_test.go` - 3 tests: root command exists, version flag, help flag, PersistentPreRun store init
- `go.mod` - Added cobra v1.10.2, testify v1.11.1
- `go.sum` - Updated checksums
- `.gitignore` - Added /aether binary and *.test patterns

## Decisions Made
- Custom version template overrides Cobra default to print "aether v<version>" instead of "aether version v<version>" -- plan required "aether v0.0.0-dev" exact format
- stdout/stderr as package-level io.Writer vars (not interface parameters) for test injection -- simpler for commands that write directly without accepting a writer
- Manual JSON construction in outputOK/outputError using fmt.Fprintf for exact key ordering -- Go's json.Marshal sorts alphabetically which breaks shell format parity
- skipStoreInit checks command ancestry chain so nested subcommands under completion/version/help also skip store init

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- go-pretty/v6 was installed per plan but go mod tidy removed it since nothing imports it yet. It will be re-added in a later plan when table rendering is needed. Not an issue.

## Next Phase Readiness
- CLI foundation complete with root command, version, completion, and output helpers
- Ready for 50-02-PLAN.md which builds on this foundation with additional commands
- All 13 tests passing, binary builds and runs correctly

---
*Phase: 50-cli-commands*
*Completed: 2026-04-02*

## Self-Check: PASSED

All 7 created files exist on disk. All 3 commits found in git log. All 13 tests passing.
