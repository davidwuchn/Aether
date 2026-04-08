# Phase 3: Build Depth Controls - Research

**Researched:** 2026-04-07
**Domain:** Go CLI enum type, token budget scaling, colony depth gating
**Confidence:** HIGH

## Summary

This phase adds compile-time depth safety via a Go enum type, wires progressive token budget scaling into the build playbooks, adds `--depth` to the init command, and fixes broken depth labels in status output. The existing infrastructure is mature: `colony-depth get/set` commands already work, build playbooks already have depth-gating for archaeologist, oracle, architect, and chaos, and the `State` type in `pkg/colony/colony.go` provides an exact template for the enum pattern.

The main work is: (1) creating the `ColonyDepth` enum type and migrating the `string` field, (2) adding validation to `state-mutate --field colony_depth` (currently unvalidated at line 117-118 of `cmd/state_cmds.go`), (3) creating a `context-budget` subcommand so playbooks can read depth-based budget values at build time, (4) wiring those budgets into `build-context.md` and `build-wave.md`, (5) adding `--depth` to `aether init`, and (6) fixing the incorrect `depthLabel()` descriptions in `cmd/status.go`.

**Primary recommendation:** Follow the exact `State` type pattern in `pkg/colony/colony.go` for the `ColonyDepth` enum, create a simple budget lookup function in the `colony` package, expose it via a `context-budget` subcommand, and update the three playbook files to call it.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 4 depth levels: light, standard, deep, full. The "full" level is kept beyond the 3 defined in DEPTH-01 through DEPTH-06.
- **D-02:** Existing `depthLabel()` descriptions in `cmd/status.go` are wrong and must be corrected to match actual depth behavior per DEPTH-02/03/04 plus the full level's existing chaos gating.
- **D-03:** Progressive (non-linear) token budget scaling: Light 4K context + 4K skills, Standard 8K + 8K (current default), Deep 16K + 12K, Full 24K + 16K. Deeper builds get disproportionately more context.
- **D-04:** The budget values must be accessible via a Go subcommand (e.g., `aether context-budget --depth standard`) so the build playbooks can read them at build time.
- **D-05:** `/ant:build --depth <level>` persists the setting (current behavior via `colony-depth set`). Once set, all future builds use that depth until changed.
- **D-06:** `/ant:init "goal" --depth light` sets depth at colony creation time (new flag on init command). Depth defaults to "standard" if not specified.
- **D-07:** Create a proper Go enum type (`ColonyDepth`) with constants for each valid level (like `State` has `StateREADY`). Replace the bare `string` field on `ColonyState` with this typed field.
- **D-08:** `state-mutate` must validate depth values against the enum -- invalid values produce an error, not a silent fallback to "standard".

### Claude's Discretion
- Exact budget implementation (constant map, function, or config)
- Whether the enum type uses `type ColonyDepth string` with constants or an iota-based approach
- How to handle backward compatibility with existing COLONY_STATE.json files that have string depth values
- Whether `colony-depth set` command needs changes beyond the enum migration
- Exact error message when state-mutate receives an invalid depth value
- Whether the `--depth` flag on `/ant:init` should accept the same 4 values

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DEPTH-01 | User can set build depth via `/ant:init` or `/ant:build --depth` with three levels: light, standard, deep | D-06 adds `--depth` to init; `--depth` on build already exists via playbooks; enum (D-07) enforces valid values. 4th level "full" is additive per D-01. |
| DEPTH-02 | Light depth skips archaeologist, limits to 1 builder, skips measurer and ambassador | Build playbooks already gate archaeologist on depth in build-context.md:132. Builder count and measurer/ambassador gating must be added to build-wave.md spawn logic. |
| DEPTH-03 | Standard depth runs full build playbook with balanced spawn counts (current default) | Already the default behavior. No changes needed beyond ensuring enum defaults to standard. |
| DEPTH-04 | Deep depth runs all specialists including measurer, ambassador, increased chaos iterations, and extended verification | Oracle/Architect already gated on deep/full. Measurer gating needs depth check. Ambassador needs depth check. Chaos iterations need depth-based scaling. |
| DEPTH-05 | Colony-prime respects depth level when assembling worker context (adjusts token budget per depth) | D-03 defines budget values. D-04 requires `context-budget` subcommand. Playbooks must call it instead of hardcoded 8000. |
| DEPTH-06 | Depth setting persists in COLONY_STATE.json and is visible in `/ant:status` | Already persists via `colony-depth set`. Status display exists but labels are wrong (D-02). Enum migration preserves JSON field name. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.23+ (project Go version) | Enum types, fmt, errors | Project is pure Go, no external deps for this phase |
| spf13/cobra | (project version) | CLI flag registration | Already used by all commands; `--depth` flag follows same pattern |
| github.com/calcosmic/Aether/pkg/colony | local | ColonyDepth enum, budget lookup | New code goes in existing colony package |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| tidwall/gjson + sjson | (project version) | JSON path manipulation in state-mutate | Already used in state_cmds.go for expression evaluation |

**Installation:** No new dependencies needed. All code uses existing project dependencies.

**Version verification:** Not applicable -- no new packages to install.

## Architecture Patterns

### Recommended Pattern: String-Based Enum (follow State type)

**What:** `type ColonyDepth string` with const declarations, a `Valid()` method, and a sentinel error.

**Why:** The `State` type in `pkg/colony/colony.go:16-23` uses this exact pattern. The `Transition()` function in `state_machine.go` validates via a map lookup. For depth, validation is simpler (no state machine -- just set membership) but the same type pattern ensures compile-time safety.

**Example:**
```go
// In pkg/colony/colony.go (near State type, line ~16)

// ColonyDepth represents the build thoroughness level.
type ColonyDepth string

const (
	DepthLight    ColonyDepth = "light"
	DepthStandard ColonyDepth = "standard"
	DepthDeep     ColonyDepth = "deep"
	DepthFull     ColonyDepth = "full"
)

// Valid reports whether d is a recognized depth level.
func (d ColonyDepth) Valid() bool {
	switch d {
	case DepthLight, DepthStandard, DepthDeep, DepthFull:
		return true
	}
	return false
}

// ErrInvalidDepth is returned when a depth value is not recognized.
var ErrInvalidDepth = fmt.Errorf("invalid colony depth")
```

### Pattern: Struct Field Migration (string -> ColonyDepth)

**What:** Change `ColonyState.ColonyDepth` from `string` to `ColonyDepth`.

**Backward compatibility:** JSON deserialization of `"colony_depth": "standard"` into a `ColonyDepth` typed field works automatically in Go because `ColonyDepth` is a `type string` alias. The `json:"colony_depth,omitempty"` tag stays the same. Old JSON files with string values deserialize correctly. [VERIFIED: Go encoding/json behavior for string-typed aliases]

**Implementation:**
```go
// ColonyState struct (line 45-64 of colony.go)
type ColonyState struct {
	// ... existing fields ...
	ColonyDepth ColonyDepth `json:"colony_depth,omitempty"` // was: string
	// ...
}
```

### Pattern: Budget Lookup Function

**What:** A function in `pkg/colony/` that returns context and skills budget for a given depth.

**Where:** Place in `pkg/colony/colony.go` or a new `pkg/colony/depth.go`.

```go
// DepthBudget returns (contextChars, skillsChars) for the given depth level.
func DepthBudget(d ColonyDepth) (context int, skills int) {
	switch d {
	case DepthLight:
		return 4000, 4000
	case DepthStandard:
		return 8000, 8000
	case DepthDeep:
		return 16000, 12000
	case DepthFull:
		return 24000, 16000
	default:
		return 8000, 8000 // safe fallback
	}
}
```

### Pattern: context-budget Subcommand

**What:** A new Cobra command `aether context-budget --depth <level>` that outputs the budget values as JSON.

**Where:** Add to `cmd/context.go` (alongside existing context commands).

```go
var contextBudgetCmd = &cobra.Command{
    Use:   "context-budget",
    Short: "Return context and skills budget for a depth level",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        depthStr, _ := cmd.Flags().GetString("depth")
        d := colony.ColonyDepth(depthStr)
        if !d.Valid() {
            outputError(1, fmt.Sprintf("invalid depth %q: must be light, standard, deep, or full", depthStr), nil)
            return nil
        }
        ctx, skills := colony.DepthBudget(d)
        outputOK(map[string]interface{}{
            "depth":   string(d),
            "context": ctx,
            "skills":  skills,
        })
        return nil
    },
}
```

### Anti-Patterns to Avoid

- **Bare string comparison for depth in Go code:** After the enum migration, use `state.ColonyDepth == colony.DepthLight` not `state.ColonyDepth == "light"`. In playbook markdown, string comparison is fine (they read JSON output).
- **Silent fallback to "standard" in state-mutate:** Line 117-118 of `cmd/state_cmds.go` currently does `state.ColonyDepth = value` with no validation. This must validate first.
- **Hardcoding budget values in playbooks:** The whole point of D-04 is a single source of truth. Playbooks must call `aether context-budget --depth $colony_depth`.
- **Using iota for enum values:** Depth values must serialize to "light", "standard", etc. in JSON. iota produces 0, 1, 2, 3 which breaks JSON compatibility. Use `type ColonyDepth string` with const string values.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Depth validation in state-mutate | Ad-hoc string check | `ColonyDepth.Valid()` method | Centralized validation, consistent error messages |
| Budget lookup | Hardcoded if/else in playbooks | `aether context-budget --depth <level>` | Single source of truth, callable from any playbook |
| Depth label mapping | Scattered descriptions | `depthLabel()` function (already exists, needs fixing) | One place to update descriptions |

**Key insight:** The depth system is mostly playbook-driven (markdown instructions for LLMs). The Go code provides the data layer (persist, validate, expose). Don't try to put gating logic in Go -- the playbooks handle that via string comparisons against JSON output.

## Runtime State Inventory

> Not applicable -- this is a greenfield feature phase (adding depth controls), not a rename/refactor/migration. The enum migration is additive (backward compatible due to Go's string alias behavior).

## Common Pitfalls

### Pitfall 1: Forgetting to update all depth comparison sites in playbooks
**What goes wrong:** Some playbook files check for "light"/"standard"/"deep"/"full" as strings. If a new depth level is added later, every comparison site must be found and updated.
**Why it happens:** Playbooks are markdown, not Go code -- no compiler to catch missing cases.
**How to avoid:** During this phase, grep for all `colony_depth` references in playbooks and verify each one handles all 4 levels correctly. Document the locations in the plan.
**Warning signs:** A depth level is introduced but some build steps still fire that shouldn't.

### Pitfall 2: state-mutate expression path bypasses field validation
**What goes wrong:** `state-mutate --field colony_depth --value "invalid"` would be caught (if we add validation to `executeFieldMode`), but `state-mutate '.colony_depth = "invalid"'` (expression mode) bypasses the field-mode switch entirely and goes through `applyFieldSet` -> `sjson.SetRawBytes` with no validation.
**Why it happens:** Expression mode in `state_cmds.go:145-171` works on raw bytes, not typed structs. The field-mode validation (lines 95-136) is a separate code path.
**How to avoid:** Either (a) add post-mutation validation in `executeExpression` that deserializes the modified JSON back into `ColonyState` and validates the depth field, or (b) add a validation hook after any state-mutate operation. Option (a) is simpler and sufficient.
**Warning signs:** Setting depth to "banana" via expression mode succeeds silently.

### Pitfall 3: Breaking existing COLONY_STATE.json files
**What goes wrong:** After changing `ColonyDepth` from `string` to `ColonyDepth`, existing JSON files with `"colony_depth": "standard"` might fail to deserialize.
**Why it happens:** If the type were `int` or used custom unmarshaling, this would be an issue.
**How to avoid:** Not actually a pitfall for string-based aliases. Go's `encoding/json` handles `type X string` transparently -- the underlying representation is still a JSON string. [VERIFIED: Go encoding/json spec]
**Warning signs:** Existing tests break after the type change. Run `go test ./...` immediately after the migration.

### Pitfall 4: Playbook budget values not matching Go subcommand output
**What goes wrong:** Playbooks hardcode 8000 while the `context-budget` command returns different values for non-standard depths.
**Why it happens:** Forgetting to update all hardcoded budget references in playbooks.
**How to avoid:** After creating `context-budget`, grep playbooks for all hardcoded budget values (8000, 4000, 2000) and replace with `aether context-budget` calls.
**Warning signs:** Research content gets the same budget at light depth as standard depth.

## Code Examples

### Existing State Type Pattern (template to follow)

```go
// Source: pkg/colony/colony.go:16-23
type State string

const (
	StateREADY     State = "READY"
	StateEXECUTING State = "EXECUTING"
	StateBUILT     State = "BUILT"
	StateCOMPLETED State = "COMPLETED"
)
```

### Existing state-mutate field validation (needs depth validation added)

```go
// Source: cmd/state_cmds.go:88-143
func executeFieldMode(cmd *cobra.Command, field string) error {
    // ...
    switch field {
    case "colony_depth":
        state.ColonyDepth = value  // LINE 118: NO VALIDATION -- this is the bug
    // ...
    }
    // ...
}
```

### Existing depth set validation (already correct, but uses raw strings)

```go
// Source: cmd/colony_cmds.go:120-125
switch depth {
case "light", "standard", "deep", "full":
default:
    outputError(1, fmt.Sprintf("invalid depth %q: must be light, standard, deep, or full", depth), nil)
    return nil
}
```

After enum migration, this should use `colony.ColonyDepth(value).Valid()` instead.

### Existing init command (needs --depth flag)

```go
// Source: cmd/init_cmd.go:89-110
state := colony.ColonyState{
    Version:       "3.0",
    Goal:          &goal,
    ColonyVersion: 0,
    State:         colony.StateREADY,
    // ColonyDepth not set -- defaults to empty, which colony-depth get treats as "standard"
    // After this phase: ColonyDepth: colony.DepthStandard (explicit default)
}
```

### Existing budget hardcode in playbook (needs to call subcommand)

```bash
# Source: .aether/docs/command-playbooks/build-context.md:95-99
# Apply 8K character budget (same size as colony-prime's 8K; skills has its own 8K)
research_budget=8000
if [[ ${#research_content} -gt $research_budget ]]; then
    research_content="${research_content:0:$research_budget}"
fi
```

After this phase, should be:
```bash
budget_result=$(aether context-budget --depth "$colony_depth")
research_budget=$(echo "$budget_result" | jq -r '.result.context')
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bare string for ColonyDepth | Typed enum `ColonyDepth` | This phase | Compile-time safety, centralized validation |
| Hardcoded 8000 budget in playbooks | `aether context-budget` subcommand | This phase | Single source of truth, depth-aware budgets |
| No init depth flag | `--depth` on `aether init` | This phase | Depth set at colony creation, not just build time |
| Silent fallback on invalid depth | Explicit error with `ErrInvalidDepth` | This phase | Catches configuration mistakes early |

**Deprecated/outdated:**
- The current `depthLabel()` descriptions in `cmd/status.go:188-201` are inaccurate and must be corrected per D-02.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go's encoding/json handles `type ColonyDepth string` transparently with existing JSON files | Architecture Patterns | Existing COLONY_STATE.json files fail to load; requires data migration |
| A2 | No other Go code references `state.ColonyDepth` as a bare string comparison (outside of colony_cmds.go and state_cmds.go) | Common Pitfalls | Missed comparison sites would need updating |
| A3 | The build playbooks are the only consumers of budget values (no other code paths read the 8K constant) | Code Examples | If colony-prime or context-capsule also hardcodes budgets, they'd need updating too |

## Open Questions

1. **Should the expression-mode state-mutate also validate depth?**
   - What we know: Field mode (`--field colony_depth`) can be validated easily. Expression mode (`.colony_depth = "invalid"`) bypasses the struct.
   - What's unclear: Whether to add post-expression validation (deserialize + validate) or document that expression mode is advanced/unsafe for typed fields.
   - Recommendation: Add post-mutation validation in `executeExpression` -- deserialize the modified JSON back to `ColonyState` and call `ColonyDepth.Valid()`. Simple, catches the bug, minimal performance cost.

2. **Should `colony-depth set` be updated to use the enum type?**
   - What we know: It already validates via a switch statement (lines 120-125).
   - What's unclear: Whether to refactor to use `ColonyDepth.Valid()` or keep the existing switch.
   - Recommendation: Refactor to use `ColonyDepth.Valid()` for consistency. It's a one-line change.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- all changes are Go code and markdown playbook updates within the existing repo).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib `testing` package) |
| Config file | None -- uses Go's built-in test runner |
| Quick run command | `go test ./pkg/colony/ -run TestDepth -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DEPTH-01 | `aether init "goal" --depth light` sets depth in state | unit | `go test ./cmd/ -run TestInitDepth -v` | No -- Wave 0 |
| DEPTH-01 | `aether colony-depth set` validates 4 values | unit | `go test ./cmd/ -run TestColonyDepth -v` | Yes (cmd/write_cmds_test.go:862-935) |
| DEPTH-02 | Light depth skips archaeologist/scout in playbook | manual | N/A (playbook behavior) | N/A |
| DEPTH-03 | Standard is default | unit | `go test ./cmd/ -run TestColonyDepthGet -v` | Yes (cmd/write_cmds_test.go:862) |
| DEPTH-04 | Deep runs all specialists | manual | N/A (playbook behavior) | N/A |
| DEPTH-05 | Budget scales with depth | unit | `go test ./cmd/ -run TestContextBudget -v` | No -- Wave 0 |
| DEPTH-06 | Depth visible in status | unit | `go test ./cmd/ -run TestStatus -v` | Yes (cmd/status_test.go) |
| D-07 | ColonyDepth enum Valid() method | unit | `go test ./pkg/colony/ -run TestDepthValid -v` | No -- Wave 0 |
| D-08 | state-mutate rejects invalid depth | unit | `go test ./cmd/ -run TestMutateDepth -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/colony/ -run TestDepth -v && go test ./cmd/ -run TestColonyDepth -v -timeout 30s`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `pkg/colony/depth_test.go` -- ColonyDepth.Valid(), DepthBudget() unit tests
- [ ] `cmd/context_test.go` -- context-budget subcommand test
- [ ] `cmd/init_cmd_test.go` -- init --depth flag test (file exists, may need new test case)
- [ ] `cmd/state_cmds_test.go` -- state-mutate depth validation test (file exists, may need new test case)

## Security Domain

> This phase has minimal security surface. Depth controls are a UX/configuration feature, not auth/crypto/network.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | `ColonyDepth.Valid()` validates enum values; no injection risk since values are string literals |
| V2 Authentication | no | N/A |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V6 Cryptography | no | N/A |

### Known Threat Patterns for Go CLI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Invalid depth via state-mutate expression | Tampering | Post-mutation validation (see Open Question 1) |
| Path traversal via --depth flag | Tampering | Cobra string flag is just a string -- no path involved |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: source code] `pkg/colony/colony.go:16-23` -- State type pattern (template for ColonyDepth enum)
- [VERIFIED: source code] `pkg/colony/colony.go:45-64` -- ColonyState struct with ColonyDepth field
- [VERIFIED: source code] `pkg/colony/state_machine.go:7-18` -- Transition() validation pattern
- [VERIFIED: source code] `cmd/state_cmds.go:88-143` -- executeFieldMode with unvalidated colony_depth
- [VERIFIED: source code] `cmd/colony_cmds.go:65-145` -- colony-depth get/set commands
- [VERIFIED: source code] `cmd/status.go:114-120, 187-201` -- depth display and depthLabel()
- [VERIFIED: source code] `cmd/init_cmd.go:22-177` -- init command (no --depth flag yet)
- [VERIFIED: source code] `.aether/docs/command-playbooks/build-prep.md:119-178` -- --depth flag parsing
- [VERIFIED: source code] `.aether/docs/command-playbooks/build-context.md:95-126` -- 8K research budget hardcode
- [VERIFIED: source code] `.aether/docs/command-playbooks/build-wave.md:36-62, 79-85, 153-159, 470-472` -- depth gating sites
- [VERIFIED: source code] `.aether/docs/command-playbooks/build-verify.md:106-256, 260-263` -- Measurer and Chaos depth checks

### Secondary (MEDIUM confidence)
- [VERIFIED: Go encoding/json] String type aliases deserialize transparently from JSON string values -- Go spec and standard library behavior

### Tertiary (LOW confidence)
- None -- all claims verified against source code.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies, all existing project code
- Architecture: HIGH - exact template pattern exists in codebase (State type)
- Pitfalls: HIGH - identified via code reading, expression-mode bypass is a real gap
- Validation: HIGH - existing test infrastructure is comprehensive (524+ tests)

**Research date:** 2026-04-07
**Valid until:** 60 days (stable domain -- Go enum pattern, no external dependencies)
