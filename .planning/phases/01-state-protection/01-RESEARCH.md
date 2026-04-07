# Phase 1: State Protection - Research

**Researched:** 2026-04-07
**Domain:** Go CLI state management, audit logging, file protection
**Confidence:** HIGH

## Summary

This phase instruments all significant mutation paths to COLONY_STATE.json with an append-only audit log, adds a `state-history` command for viewing mutations, detects and rejects the known jq-expression corruption bug, auto-creates checkpoint snapshots before destructive operations, and introduces BoundaryGuard to protect sensitive paths from unauthorized writes.

The existing codebase already has all the building blocks: `Store.AppendJSONL()` and `Store.ReadJSONL()` handle JSONL append and read with locking; `Store.AtomicWrite()` does temp-file + rename with JSON validation; `FileLocker` provides cross-process locking via `syscall.Flock`; and `state-checkpoint` already saves named snapshots. The primary work is wiring these together into a coherent mutation audit pipeline, adding a guard layer, and building the `state-history` display command.

Five distinct mutation entry points must be instrumented: `state-mutate` (cmd/state_cmds.go), `state-write` (cmd/state_extra.go), `phase-insert` (cmd/state_extra.go), `update-progress` (cmd/build_flow_cmds.go), and phase advance (within executeFieldMode when `--field current_phase`). The `context-update` command is explicitly excluded from audit per D-01.

**Primary recommendation:** Create a centralized `pkg/storage/audit.go` package that all mutation commands call, rather than scattering audit logic across individual command handlers.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Only significant mutations are logged -- state-mutate, state-write, phase-insert, phase advance, and plan changes. High-frequency build internals (context-update, worker-spawn) are excluded to avoid log noise and performance overhead.
- **D-02:** Each audit entry must contain: before/after diffs, timestamp, source command, and SHA-256 checksum (per STATE-02).
- **D-03:** Default output is compact -- one line per mutation showing timestamp, command, field changed, and summary (like `git log --oneline`).
- **D-04:** A `--diff` flag shows full before/after JSON diffs for a specific entry or the last N entries.
- **D-05:** A `--tail N` flag limits output to the last N entries (default: 20).
- **D-06:** `state-write` requires a `--force` flag to execute. Without `--force`, it prints an error suggesting `state-mutate` instead.
- **D-07:** When `state-write --force` is used, the mutation IS recorded in the audit log (it's a significant mutation).
- **D-08:** Auto-checkpoint snapshots are created before destructive operations only: phase advance, plan overwrite, and `state-write --force`.
- **D-09:** Non-destructive mutations (goal update, phase status change, field set) are recorded in the audit log but do NOT trigger checkpoints.
- **D-10:** Checkpoints are stored in `.aether/data/checkpoints/` with timestamp-based naming.

### Claude's Discretion

- Exact audit log JSONL schema design (field names, nesting)
- Checkpoint rotation/retention policy (how many to keep, when to prune)
- `state-history` output formatting details (column widths, color if TTY)
- BoundaryGuard implementation approach (file watcher, hook-based, or explicit check)
- Corruption detection heuristics beyond the known jq-expression bug
- Whether the audit log itself should be checksummed or signed

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| STATE-01 | Every mutation to COLONY_STATE.json is recorded in an append-only audit log (state-changelog.jsonl) | `Store.AppendJSONL()` exists and handles append-only JSONL with locking; audit entry struct design needed |
| STATE-02 | Audit log entries contain before/after diffs, timestamp, source command, and SHA-256 checksum | `crypto/sha256` already imported in cmd; before/after diff via gjson path extraction |
| STATE-03 | Planning history is append-only -- past phase plans cannot be overwritten, only extended | `phase-insert` already appends; need guard on plan overwrite paths |
| STATE-04 | User can view full mutation history via `aether state-history` command | `Store.ReadJSONL()` exists; `go-pretty/v6/table` already in go.mod for formatted output |
| STATE-05 | State corruption (known jq-expression bug) is detected and rejected at write time with clear error message | Bracket notation in `reFieldSet` regex (`^\.([\w.\[\]]+)\s*=\s*(.+)$`) allows jq-like expressions through `applyFieldSet` which calls `sjson.SetRawBytes`; `normalizeBracketPath` converts `[N]` to `.N` but unquoted values in bracket paths cause sjson to produce invalid JSON |
| STATE-06 | Checkpoint snapshots of COLONY_STATE.json are created automatically before destructive operations | `state-checkpoint` command already saves named snapshots to `checkpoints/` dir; needs auto-trigger wiring |
| STATE-07 | BoundaryGuard protects sensitive paths from unauthorized writes during colony operations | No existing implementation; needs new code in `pkg/storage/` |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `crypto/sha256` | go1.26.1 | Checksum for audit entries | Already imported in cmd; no external dependency needed |
| Go stdlib `encoding/json` | go1.26.1 | JSONL serialization | Standard Go marshaling |
| `github.com/tidwall/gjson` | v1.18.0 | Read JSON paths for diff extraction | Already in go.mod; used by state_cmds.go |
| `github.com/tidwall/sjson` | v1.2.5 | Set JSON paths | Already in go.mod; used by state_cmds.go |
| `github.com/jedib0t/go-pretty/v6` | v6.7.8 | Table rendering for state-history | Already in go.mod; used by history.go, status.go, phase.go |
| `github.com/spf13/cobra` | v1.10.2 | CLI command framework | Already in go.mod; all commands use cobra |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/spf13/pflag` | v1.0.9 | Flag parsing (already pulled by cobra) | `--diff`, `--tail` flags on state-history |
| Go stdlib `os` | go1.26.1 | File operations for BoundaryGuard | Path validation |
| Go stdlib `path/filepath` | go1.26.1 | Path cleaning for BoundaryGuard | Prevent traversal attacks |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-pretty tables | Custom tabwriter formatting | go-pretty is already a dependency with consistent styling across the project |
| gjson path extraction for diffs | Full JSON marshal/compare | gjson is faster for targeted path reads and already available |
| JSONL audit log | SQLite WAL | JSONL is append-only by nature, human-readable, and the project explicitly scoped out SQLite |

**Installation:** No new packages needed. All dependencies are already in go.mod.

**Version verification:** All versions confirmed from go.mod [VERIFIED: go.mod].

## Architecture Patterns

### Recommended Project Structure
```
pkg/storage/
├── storage.go          # Existing: AtomicWrite, AppendJSONL, ReadJSONL
├── lock.go             # Existing: FileLocker
├── audit.go            # NEW: AuditLogger, AuditEntry, WriteBoundary
├── audit_test.go       # NEW: Tests for audit log
├── boundary.go         # NEW: BoundaryGuard
├── boundary_test.go    # NEW: Tests for BoundaryGuard
├── corruption.go       # NEW: CorruptionDetector
├── corruption_test.go  # NEW: Tests for corruption detection
├── checkpoint.go       # NEW: AutoCheckpoint (wraps existing state-checkpoint)
└── checkpoint_test.go  # NEW: Tests for auto-checkpoint

cmd/
├── state_cmds.go       # MODIFY: Instrument state-mutate with audit
├── state_extra.go      # MODIFY: Add --force to state-write, audit logging
├── state_history.go    # NEW: state-history command
├── state_history_test.go # NEW: Tests for state-history
├── build_flow_cmds.go  # MODIFY: Instrument update-progress with audit
└── ...
```

### Pattern 1: Centralized Audit Logger

**What:** A single `AuditLogger` struct in `pkg/storage/audit.go` that all mutation commands call before writing state. This avoids scattering audit logic across 5+ command files.

**When to use:** Every significant mutation to COLONY_STATE.json.

**Example:**
```go
// pkg/storage/audit.go

type AuditEntry struct {
    Timestamp string          `json:"timestamp"`
    Command   string          `json:"command"`
    Path      string          `json:"path"`       // e.g., "plan.phases.0.status"
    Before    json.RawMessage `json:"before,omitempty"`
    After     json.RawMessage `json:"after,omitempty"`
    Checksum  string          `json:"checksum"`   // SHA-256 of after-state
    Destructive bool          `json:"destructive"`
}

type AuditLogger struct {
    store *Store
}

// Record reads the current state, invokes the mutation callback, and
// appends an audit entry. The callback receives the raw state bytes
// and returns the mutated bytes.
func (al *AuditLogger) Record(cmd string, mutator func(before []byte) ([]byte, error)) error {
    before, err := al.store.ReadFile("COLONY_STATE.json")
    if err != nil {
        return err
    }
    after, err := mutator(before)
    if err != nil {
        return err
    }
    // Compute checksum of new state
    checksum := sha256.Sum256(after)
    // Extract diff info from the mutation
    entry := AuditEntry{
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Command:   cmd,
        Checksum:  hex.EncodeToString(checksum[:]),
    }
    // Write audit entry BEFORE writing state (append-only, never blocks)
    return al.store.AppendJSONL("state-changelog.jsonl", entry)
}
```

### Pattern 2: Write Boundary

**What:** A write boundary groups a read-modify-write sequence into an atomic unit. The audit logger holds the file lock for the duration of the read, mutation, and write -- preventing TOCTOU races.

**When to use:** Every mutation command that does load -> modify -> save.

**Example:**
```go
// WriteBoundary reads state, runs mutator, validates, audits, and writes.
// All within a single FileLocker hold.
func (al *AuditLogger) WriteBoundary(cmd string, destructive bool, mutator func(state *colony.ColonyState) error) error {
    before, err := al.store.ReadFile("COLONY_STATE.json")
    if err != nil {
        return err
    }

    var state colony.ColonyState
    if err := json.Unmarshal(before, &state); err != nil {
        return fmt.Errorf("audit: unmarshal state: %w", err)
    }

    if err := mutator(&state); err != nil {
        return err
    }

    // Corruption detection
    if err := DetectCorruption(&state); err != nil {
        return fmt.Errorf("audit: corruption detected: %w", err)
    }

    // Auto-checkpoint for destructive operations
    if destructive {
        ts := time.Now().UTC().Format("20060102-150405")
        checkpointPath := fmt.Sprintf("checkpoints/auto-%s.json", ts)
        al.store.AtomicWrite(checkpointPath, before)
    }

    // Write new state
    after, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return fmt.Errorf("audit: marshal state: %w", err)
    }
    after = append(after, '\n')

    if err := al.store.AtomicWrite("COLONY_STATE.json", after); err != nil {
        return fmt.Errorf("audit: write state: %w", err)
    }

    // Append audit entry
    checksum := sha256.Sum256(after)
    entry := AuditEntry{
        Timestamp:   time.Now().UTC().Format(time.RFC3339),
        Command:     cmd,
        Before:      before,
        After:       after,
        Checksum:    hex.EncodeToString(checksum[:]),
        Destructive: destructive,
    }
    return al.store.AppendJSONL("state-changelog.jsonl", entry)
}
```

### Pattern 3: BoundaryGuard

**What:** Explicit path validation before any write operation. Check if the target path is a protected path and reject unauthorized writes.

**When to use:** All file writes that go through the Store during colony operations.

**Example:**
```go
// pkg/storage/boundary.go

var protectedPaths = []string{
    "COLONY_STATE.json",
    "session.json",
    "checkpoints/",
    "midden/",
}

type BoundaryGuard struct {
    store  *Store
    allowed map[string]bool // paths explicitly allowed in current context
}

func (bg *BoundaryGuard) Allow(path string) {
    bg.allowed[path] = true
}

func (bg *BoundaryGuard) CheckWrite(path string) error {
    for _, pp := range protectedPaths {
        if path == pp || strings.HasPrefix(path, pp) {
            if !bg.allowed[path] {
                return fmt.Errorf("boundary: write to protected path %q is not allowed", path)
            }
        }
    }
    return nil
}
```

**Implementation recommendation (Claude's Discretion):** Use an explicit check-based approach rather than file watchers or hooks. File watchers add complexity and platform-specific behavior (inotify vs kqueue). Hooks would require modifying every call site. An explicit `BoundaryGuard.CheckWrite()` that gets called from `Store.AtomicWrite()` or as a wrapper is the simplest and most testable approach. The guard should be opt-in per operation scope (a colony operation sets up a guard session, individual writes check against it).

### Pattern 4: Corruption Detection

**What:** Validate state before writing to catch the jq-expression bug and other corruption patterns.

**When to use:** Every write through WriteBoundary.

**Known corruption pattern:** The `Events` field in ColonyState is `[]string`. When an LLM reconstructs full JSON (the "Frankenstein state" bug from memory note), jq expressions can end up stored literally in the events array. STATE-05 requires detecting and rejecting this at write time.

**Example:**
```go
// pkg/storage/corruption.go

func DetectCorruption(state *colony.ColonyState) error {
    for i, evt := range state.Events {
        if looksLikeJQExpression(evt) {
            return fmt.Errorf("corruption detected in events[%d]: jq expression stored as literal: %q", i, evt)
        }
    }
    return nil
}

var jqPattern = regexp.MustCompile(`^\.[\w.\[\]]+\s*[|=]`) // matches .path = value or .path |= expr

func looksLikeJQExpression(s string) bool {
    return jqPattern.MatchString(s)
}
```

### Anti-Patterns to Avoid

- **Scattering audit calls across commands:** Don't add `store.AppendJSONL("state-changelog.jsonl", ...)` directly in each command handler. Use the centralized `WriteBoundary` or `AuditLogger.Record()` to ensure consistency.
- **Writing audit AFTER state write:** Always write audit entry after the state write succeeds. If the audit write fails, the state is already persisted -- log the audit failure to stderr but don't roll back the state write.
- **Locking audit log and state separately:** The FileLocker serializes by file path. `COLONY_STATE.json` and `state-changelog.jsonl` have separate locks. This is fine -- audit entries are append-only and never conflict. But do NOT try to hold both locks simultaneously (deadlock risk with the same FileLocker instance).
- **Storing full before/after for every mutation:** For non-destructive mutations, consider storing only the changed path and the diff, not the entire state. This keeps the audit log manageable.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON path extraction for diffs | Custom string parsing | `gjson.GetBytes()` | Already in go.mod, handles dot/bracket notation, battle-tested |
| JSON path setting | Custom string manipulation | `sjson.SetRawBytes()` | Already in go.mod, handles nested paths correctly |
| Table rendering for state-history | Manual column formatting | `go-pretty/v6/table` | Already in go.mod, consistent with existing history.go/phase.go output |
| SHA-256 checksums | Custom hash implementation | `crypto/sha256` from stdlib | Standard library, no dependencies |
| JSONL append | Custom file append with lock | `Store.AppendJSONL()` | Already exists with proper locking and error handling |
| Atomic file writes | Manual temp file + rename | `Store.AtomicWrite()` | Already exists with JSON validation and cleanup |

**Key insight:** The storage layer already provides all the primitives needed. This phase is primarily about orchestration -- wiring existing primitives together through a new audit pipeline -- not building new infrastructure.

## Common Pitfalls

### Pitfall 1: Audit Log as Bottleneck
**What goes wrong:** Every mutation now does state write + audit append, doubling I/O. Under rapid mutation (e.g., phase advance followed by multiple task updates), this slows things down.
**Why it happens:** Each `AppendJSONL` acquires its own file lock.
**How to avoid:** The audit log file lock is separate from the state file lock, so there's no contention. JSONL append is O(1) -- just appending a line to a file. Performance impact is negligible.
**Warning signs:** `time go test ./...` showing regression after instrumentation.

### Pitfall 2: Before/After Diff Too Large
**What goes wrong:** Storing full before/after state in every audit entry makes the JSONL file grow quickly (COLONY_STATE.json can be 10KB+ with many phases/tasks).
**Why it happens:** Naive approach stores entire JSON document as `before` and `after`.
**How to avoid:** Store only the changed path and the diff (before value and after value at that path). For operations that change the entire state (like `state-write --force`), store a summary instead.
**Warning signs:** `state-changelog.jsonl` growing beyond 1MB in a typical session.

### Pitfall 3: Checkpoint Accumulation
**What goes wrong:** Auto-checkpoints for every destructive operation accumulate indefinitely in `.aether/data/checkpoints/`.
**Why it happens:** No rotation policy.
**How to avoid:** Implement a simple retention policy -- keep the last N checkpoints (e.g., 10) and delete older ones. This is Claude's Discretion per CONTEXT.md.
**Warning signs:** `ls -la .aether/data/checkpoints/` showing dozens of auto-* files.

### Pitfall 4: Bracket Notation Bug Surface Area
**What goes wrong:** The `normalizeBracketPath` function in state_cmds.go converts `[N]` to `.N` for sjson, but the regex `reFieldSet` (`^\.([\w.\[\]]+)\s*=\s*(.+)$`) accepts bracket notation in the field path. If the value expression contains unquoted strings or special characters, sjson produces invalid JSON that gets written via `AtomicWrite` -- which validates JSON and rejects it. BUT if the value happens to be valid JSON (e.g., a number or a quoted string), the write succeeds but may produce unintended state.
**Why it happens:** The regex-based expression parser is not a full jq implementation. It handles common patterns but has edge cases.
**How to avoid:** Add explicit validation in `applyFieldSet` that rejects value expressions containing jq operators (`|`, `//`, `?`, `select`, etc.). Add corruption detection in the WriteBoundary that validates the resulting state struct before writing.
**Warning signs:** State containing literal jq expressions in string fields.

### Pitfall 5: Test Compilation Issues
**What goes wrong:** New test files in `pkg/storage/` compile but tests in `cmd/` show "[no tests to run]" due to package-level globals and test helper patterns.
**Why it happens:** The `cmd` package uses package-level globals (`store`, `stdout`, `stderr`) that require `saveGlobals(t)` and `resetRootCmd(t)` in every test. Tests for `pkg/storage/` audit functions should use `t.TempDir()` and `storage.NewStore()` directly.
**How to avoid:** Write `pkg/storage/` tests as independent package tests using temp dirs. Write `cmd/` tests following the existing pattern with `saveGlobals`/`resetRootCmd`.
**Warning signs:** `go test ./...` showing "[no tests to run]" for packages that have test files.

## Code Examples

Verified patterns from the existing codebase:

### Existing AppendJSONL Usage (from storage.go)
```go
// Source: pkg/storage/storage.go (lines 139-167) [VERIFIED: codebase]
func (s *Store) AppendJSONL(path string, entry interface{}) error {
    if err := s.locker.Lock(path); err != nil {
        return fmt.Errorf("storage: acquire lock for %q: %w", path, err)
    }
    defer s.locker.Unlock(path)
    // ... opens file with O_APPEND|O_CREATE|O_WRONLY
    // ... marshals entry as JSON line, writes with newline
}
```

### Existing go-pretty Table Pattern (from history.go)
```go
// Source: cmd/history.go (lines 119-142) [VERIFIED: codebase]
func renderHistoryTable(events []string) {
    t := table.NewWriter()
    t.AppendHeader(table.Row{"Timestamp", "Type", "Source", "Message"})
    for i := len(events) - 1; i >= 0; i-- {
        t.AppendRow(table.Row{ts, eventType, source, message})
    }
    t.SetStyle(table.StyleRounded)
    fmt.Fprintln(stdout, t.Render())
}
```

### Existing Test Pattern (from write_cmds_test.go)
```go
// Source: cmd/write_cmds_test.go (lines 17-40) [VERIFIED: codebase]
func newTestStore(t *testing.T) (*storage.Store, string) {
    t.Helper()
    tmpDir := t.TempDir()
    dataDir := tmpDir + "/.aether/data"
    os.MkdirAll(dataDir, 0755)
    os.Setenv("COLONY_DATA_DIR", dataDir)
    s, err := storage.NewStore(dataDir)
    // ...
    return s, tmpDir
}
```

### Existing Checkpoint Pattern (from state_extra.go)
```go
// Source: cmd/state_extra.go (lines 12-45) [VERIFIED: codebase]
checkpointPath := filepath.Join("checkpoints", name+".json")
if err := store.SaveJSON(checkpointPath, state); err != nil {
    outputError(2, fmt.Sprintf("failed to save checkpoint: %v", err), nil)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell-based atomic-write.sh | Go `Store.AtomicWrite()` with temp-file + rename | v5.0 architecture | All state writes go through Go, no shell dependency |
| No audit trail | Append-only JSONL audit log | This phase | Every mutation is traceable |
| state-write bypasses validation | state-write requires `--force` + audit | This phase | D-06/D-07 enforcement |
| Manual checkpoints | Auto-checkpoint before destructive ops | This phase | D-08 enforcement |
| No path protection | BoundaryGuard for sensitive paths | This phase | STATE-07 |

**Deprecated/outdated:**
- Shell-based state mutation scripts: Already removed in v5.0 Go migration. No action needed.
- `state-write` without `--force`: Still current behavior but will be changed in this phase.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The jq-expression corruption bug manifests as jq operators stored literally in the `Events` field (string array) | Architecture Patterns / Corruption Detection | If the bug manifests differently (e.g., in other string fields), the regex pattern needs broadening |
| A2 | `Store.AtomicWrite()` JSON validation is sufficient to catch most malformed state writes | Common Pitfalls / Pitfall 4 | If sjson produces technically valid JSON with wrong semantics, AtomicWrite won't catch it |
| A3 | The `context-update` command is the only high-frequency operation excluded from audit | User Constraints / D-01 | If other high-frequency commands exist (e.g., pheromone-write), they should also be excluded |
| A4 | Checkpoint retention of 10 auto-checkpoints is sufficient | Claude's Discretion | If users run many destructive operations between manual cleanups, older checkpoints may be needed |
| A5 | BoundaryGuard using explicit check-based approach is simpler than file watchers | Architecture Patterns / Pattern 3 | If colony operations need to protect against external processes (not just internal Go code), file watchers may be needed |

## Open Questions

1. **BoundaryGuard scope during colony operations**
   - What we know: STATE-07 requires protecting `.aether/data/` and `.aether/dreams/` from unauthorized writes during colony operations.
   - What's unclear: Whether "colony operations" means only when a colony is active (EXECUTING state), or always. Whether agents spawned by the colony should have different permissions than the `aether` CLI itself.
   - Recommendation: Start with a simple approach -- BoundaryGuard is activated when a colony is in EXECUTING state and checks all writes to protected paths. The CLI itself is always authorized. This can be refined later.

2. **Audit log checksum scope**
   - What we know: D-02 requires SHA-256 checksum per audit entry.
   - What's unclear: Whether the checksum covers just the `after` state, the entire audit entry itself, or both.
   - Recommendation: Checksum the `after` state bytes (the new COLONY_STATE.json content). This lets users verify state integrity by comparing the file's checksum against the latest audit entry.

3. **Checkpoint retention policy**
   - What we know: D-10 says checkpoints go to `.aether/data/checkpoints/` with timestamp naming. Retention is Claude's Discretion.
   - What's unclear: How many auto-checkpoints to keep, and whether manual checkpoints (via `state-checkpoint`) should also be subject to rotation.
   - Recommendation: Keep last 10 auto-checkpoints, never delete manual checkpoints (they have explicit names). Auto-checkpoint names use `auto-YYYYMMDD-HHMMSS.json` prefix to distinguish from manual ones.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | All implementation | Yes | 1.26.1 | -- |
| go-pretty/v6 | state-history table rendering | Yes | v6.7.8 (in go.mod) | -- |
| gjson | JSON path extraction for diffs | Yes | v1.18.0 (in go.mod) | -- |
| sjson | JSON path setting | Yes | v1.2.5 (in go.mod) | -- |
| cobra | CLI commands | Yes | v1.10.2 (in go.mod) | -- |

**Missing dependencies with no fallback:**
- None

**Missing dependencies with fallback:**
- None

Step 2.6 note: All dependencies are already in go.mod. No new external packages are needed for this phase.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (no external test framework) |
| Config file | None -- tests use `t.TempDir()` and `newTestStore()` |
| Quick run command | `go test ./pkg/storage/ -run "TestAudit" -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| STATE-01 | Mutation appends audit entry | unit | `go test ./pkg/storage/ -run TestAuditLogger_Record -count=1` | No -- Wave 0 |
| STATE-02 | Audit entry has timestamp, source, checksum, diff | unit | `go test ./pkg/storage/ -run TestAuditEntry_Fields -count=1` | No -- Wave 0 |
| STATE-03 | Plan overwrite rejected | unit | `go test ./cmd/ -run TestPlanOverwrite -count=1` | No -- Wave 0 |
| STATE-04 | state-history displays mutations | unit | `go test ./cmd/ -run TestStateHistory -count=1` | No -- Wave 0 |
| STATE-05 | JQ expression rejected at write time | unit | `go test ./pkg/storage/ -run TestDetectCorruption -count=1` | No -- Wave 0 |
| STATE-06 | Auto-checkpoint before destructive op | unit | `go test ./pkg/storage/ -run TestAutoCheckpoint -count=1` | No -- Wave 0 |
| STATE-07 | BoundaryGuard rejects unauthorized write | unit | `go test ./pkg/storage/ -run TestBoundaryGuard -count=1` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/storage/ -count=1` (quick run for audit/corruption/boundary)
- **Per wave merge:** `go test ./... -count=1` (full suite, catches cmd integration regressions)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `pkg/storage/audit_test.go` -- covers STATE-01, STATE-02 (AuditLogger, AuditEntry)
- [ ] `pkg/storage/corruption_test.go` -- covers STATE-05 (DetectCorruption)
- [ ] `pkg/storage/boundary_test.go` -- covers STATE-07 (BoundaryGuard)
- [ ] `pkg/storage/checkpoint_test.go` -- covers STATE-06 (AutoCheckpoint)
- [ ] `cmd/state_history_test.go` -- covers STATE-04 (state-history command)
- [ ] `cmd/state_cmds_test.go` -- extend existing file for STATE-03 (plan overwrite guard)
- [ ] `cmd/state_extra_test.go` -- extend existing file for D-06 (--force flag)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | N/A -- single-user CLI |
| V3 Session Management | No | N/A -- no sessions |
| V4 Access Control | No | N/A -- single-user local system |
| V5 Input Validation | Yes | Corruption detection validates state before write; regex validation rejects jq expressions |
| V6 Cryptography | Yes | SHA-256 checksums via stdlib `crypto/sha256` |

### Known Threat Patterns for Go CLI State Files

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal in file writes | Tampering | `filepath.Clean()` + BoundaryGuard path validation |
| JSON injection via mutation expressions | Tampering | Corruption detection + AtomicWrite JSON validation |
| Audit log tampering | Tampering | Append-only file; future: checksum the audit log itself |
| Checkpoint file deletion | Denial of Service | Checkpoint retention keeps backups; auto-checkpoints are copies |
| Concurrent write corruption | Tampering | FileLocker with syscall.Flock serializes writes |

## Sources

### Primary (HIGH confidence)
- `pkg/storage/storage.go` -- verified AppendJSONL, AtomicWrite, ReadJSONL implementations
- `pkg/storage/lock.go` -- verified FileLocker with syscall.Flock
- `pkg/colony/colony.go` -- verified ColonyState struct, Events field type
- `cmd/state_cmds.go` -- verified state-mutate expression parsing, bracket notation handling
- `cmd/state_extra.go` -- verified state-write, state-checkpoint, phase-insert
- `cmd/build_flow_cmds.go` -- verified update-progress mutation path
- `cmd/history.go` -- verified go-pretty table pattern for rendering
- `go.mod` -- verified all dependency versions

### Secondary (MEDIUM confidence)
- CONTEXT.md (01-CONTEXT.md) -- user decisions D-01 through D-10
- REQUIREMENTS.md -- STATE-01 through STATE-07 definitions
- ROADMAP.md -- Phase 1 success criteria and risk notes
- CLAUDE.md -- project architecture and conventions

### Tertiary (LOW confidence)
- Memory note about jq-expression corruption bug (from MEMORY.md) -- the exact manifestation needs runtime confirmation; the regex-based corruption detector is a hypothesis that should be validated against actual corrupted state files if any exist

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all dependencies verified in go.mod, no new packages needed
- Architecture: HIGH -- all patterns derived from existing codebase patterns (verified by reading source files)
- Pitfalls: MEDIUM -- corruption detection heuristics (A1, A2) are hypotheses that need validation against real corrupted state

**Research date:** 2026-04-07
**Valid until:** 2026-05-07 (stable -- no fast-moving dependencies)
