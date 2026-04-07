# Phase 4: Planning Granularity Controls - Research

**Researched:** 2026-04-07
**Domain:** Go enum types, LLM prompt engineering for plan constraints, colony state persistence
**Confidence:** HIGH

## Summary

This phase adds a `PlanGranularity` enum type (following the exact `ColonyDepth` pattern established in Phase 3) with 4 presets (Sprint 1-3, Milestone 4-7, Quarter 8-12, Major 13-20). The implementation spans 5 integration points: the Go enum + state field, the plan command's `--granularity` flag + prompt injection, the route-setter agent's dynamic phase bounds, the status display, and the autopilot's COLONY_STATE.json read.

Every pattern needed already exists in the codebase from Phase 3's depth work. The `PlanGranularity` type, `Valid()` method, state field, `plan-granularity get/set` commands, and validation in `state-mutate` all have direct templates in `ColonyDepth`. The plan command already has a preset selection mechanism (planning depth: fast/balanced/deep/exhaustive) that granularity selection can follow. The route-setter agent currently hardcodes "3-6 phases" and "Maximum 6 phases" in two locations each across two agent definition files -- these become dynamic values injected from the selected granularity.

**Primary recommendation:** Follow the Phase 3 depth pattern exactly. Mirror every integration point (enum, state field, get/set commands, status display, state-mutate validation) and add the plan-command + route-setter + autopilot wiring on top.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 4 granularity presets: Sprint (1-3 phases), Milestone (4-7), Quarter (8-12), Major (13-20). Matches PLAN-01 exactly.
- **D-02:** Follow the same Go enum pattern as ColonyDepth -- `type PlanGranularity string` with const declarations and a `Valid()` method.
- **D-03:** `/ant:plan` always asks for granularity if none is set -- no silent default. The user picks from the 4 presets each time (unless one is already persisted from a previous `/ant:plan` or `/ant:init` call).
- **D-04:** Once selected, granularity persists in COLONY_STATE.json. Subsequent `/ant:plan` calls use the persisted value unless overridden with `--granularity`.
- **D-05:** If the route-setter generates a plan outside the selected range, show a clear warning (actual count vs. chosen range) and let the user decide: accept as-is, adjust the range to fit, or replan. No silent auto-trimming or hard rejection.
- **D-06:** Granularity and depth are fully independent. Granularity controls phase count (breadth), depth controls build thoroughness per phase. No cross-influence or soft recommendations.
- **D-07:** The `--granularity` bounds (min/max phases) must be injected into the route-setter prompt, replacing the current hardcoded "Maximum 6 phases" constraint. The plan command reads the persisted or selected granularity and passes min/max to the route-setter.
- **D-08:** The plan command's output constraint line (`Maximum 6 phases`) must be dynamically set based on the selected granularity range's max value.
- **D-09:** `/ant:run` reads the persisted granularity from COLONY_STATE.json and respects the phase count. If the plan has more phases than the range allows, the autopilot warns but continues (the plan was already accepted by the user during `/ant:plan`).

### Claude's Discretion
- Exact enum implementation details (iota vs string constants)
- Whether to add `--granularity` flag to `/ant:init` (like depth has)
- How the "always ask" prompt appears in `/ant:plan` (inline question vs separate step)
- Exact warning message format for out-of-range plans
- Whether `state-mutate` should validate granularity values (like depth)
- Whether to add a `plan-granularity get/set` command pair (like `colony-depth`)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLAN-01 | User can select planning granularity with 4 ranges: sprint (1-3), milestone (4-7), quarter (8-12), major (13-20) | D-01 defines exact ranges; ColonyDepth pattern provides enum template |
| PLAN-02 | Route-setter receives min/max phase constraints from selected granularity | D-07 specifies prompt injection; hardcoded "3-6 phases" found at 4 locations |
| PLAN-03 | If plan exceeds selected range, user warned and asked to approve or adjust | D-05 defines warn+choose behavior; plan.md Step 5 is insertion point |
| PLAN-04 | Planning granularity persists in COLONY_STATE.json and visible in `/ant:status` | D-04 + ColonyState struct has exact pattern; status.go has depth display to follow |
| PLAN-05 | Autopilot respects selected planning granularity across all phases | D-09; run.md Step 0 reads COLONY_STATE.json -- add granularity check |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | project Go version | Enum type, JSON marshaling, string constants | Already used for ColonyDepth -- same pattern |
| spf13/cobra | project version | CLI flag parsing for `--granularity` | Already used for `--depth` flag on init and colony-depth commands |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/calcosmic/Aether/pkg/colony | local | ColonyState struct, enum types | Adding PlanGranularity field and type |
| github.com/calcosmic/Aether/pkg/storage | local | JSON persistence, FileLocker | Saving granularity to COLONY_STATE.json |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `type PlanGranularity string` | `type PlanGranularity int` (iota) | iota gives compile-time exhaustiveness checking but produces opaque JSON values (0,1,2,3 vs "sprint","milestone"). String constants are consistent with ColonyDepth and produce readable JSON. |

**Installation:** No new dependencies required. This phase uses only existing project packages.

**Version verification:** N/A -- no new packages.

## Architecture Patterns

### Recommended File Changes

```
pkg/colony/
  colony.go           -- Add PlanGranularity type, constants, Valid(), ErrInvalidGranularity, PlanGranularity field on ColonyState
  granularity.go      -- New file: GranularityRange() function returning (min, max int) per preset [VERIFIED: follows depth.go pattern]
  granularity_test.go -- New file: Tests for Valid(), GranularityRange()

cmd/
  colony_cmds.go      -- Add plan-granularity get/set commands [VERIFIED: follows colony-depth get/set pattern]
  status.go           -- Add granularity display below depth line, add granularityLabel() function
  status_test.go      -- Update test expectations for new granularity line
  state_cmds.go       -- Add "plan_granularity" case in state-mutate switch + validate-state + state-read-field

.aether/commands/claude/plan.md      -- Add granularity selection step, inject min/max into route-setter prompt
.aether/agents/aether-route-setter.md   -- Replace hardcoded "3-6 phases" with dynamic bounds
.aether/agents-claude/aether-route-setter.md -- Mirror changes to agents-claude
.claude/agents/ant/aether-route-setter.md -- Mirror changes to Claude agents
.opencode/agents/aether-route-setter.md -- Mirror changes to OpenCode agents
.aether/commands/claude/run.md      -- Read persisted granularity in Step 0
```

### Pattern 1: Go Enum Type (from Phase 3 ColonyDepth)
**What:** String-based enum with const declarations, `Valid()` method, and sentinel error.
**When to use:** Every typed field on ColonyState that has a fixed set of valid values.
**Example:**
```go
// Source: pkg/colony/colony.go:25-45 (verified)
type ColonyDepth string

const (
    DepthLight    ColonyDepth = "light"
    DepthStandard ColonyDepth = "standard"
    DepthDeep     ColonyDepth = "deep"
    DepthFull     ColonyDepth = "full"
)

func (d ColonyDepth) Valid() bool {
    switch d {
    case DepthLight, DepthStandard, DepthDeep, DepthFull:
        return true
    }
    return false
}

var ErrInvalidDepth = fmt.Errorf("invalid colony depth")
```

### Pattern 2: Range Lookup Function (new, follows DepthBudget)
**What:** Function that maps enum value to (min, max) phase count.
**When to use:** The plan command and out-of-range validation need min/max bounds.
**Example:**
```go
// Source: pkg/colony/depth.go pattern (verified)
func GranularityRange(g PlanGranularity) (min int, max int) {
    switch g {
    case GranularitySprint:
        return 1, 3
    case GranularityMilestone:
        return 4, 7
    case GranularityQuarter:
        return 8, 12
    case GranularityMajor:
        return 13, 20
    default:
        return 1, 3
    }
}
```

### Pattern 3: get/set Command Pair (from colony-depth)
**What:** Parent command with get and set subcommands using cobra.
**When to use:** CLI access to read/write a persisted setting.
**Example:**
```go
// Source: cmd/colony_cmds.go:65-143 (verified)
var planGranularityCmd = &cobra.Command{
    Use:   "plan-granularity",
    Short: "Get or set planning granularity",
    Args:  cobra.NoArgs,
}

var planGranularityGetCmd = &cobra.Command{
    Use:   "get",
    Short: "Get current planning granularity",
    // ... load state, return PlanGranularity field or "none" if empty
}

var planGranularitySetCmd = &cobra.Command{
    Use:   "set",
    Short: "Set planning granularity",
    // ... validate via Valid(), save to COLONY_STATE.json
}
```

### Pattern 4: State Mutation Validation (from state-mutate depth case)
**What:** Add a case in `state-mutate` switch statement for the new field, validate against enum.
**When to use:** Any new typed field on ColonyState needs validation in state-mutate and validate-state.
**Example:**
```go
// Source: cmd/state_cmds.go:117-123 (verified)
case "plan_granularity":
    g := colony.PlanGranularity(value)
    if !g.Valid() {
        outputError(1, fmt.Sprintf("invalid granularity %q: must be sprint, milestone, quarter, or major", value), nil)
        return nil
    }
    state.PlanGranularity = g
```

And in the expression validation block:
```go
// Source: cmd/state_cmds.go:172-178 (verified)
if validateState.PlanGranularity != "" && !validateState.PlanGranularity.Valid() {
    outputError(1, fmt.Sprintf("invalid plan_granularity %q: must be sprint, milestone, quarter, or major", validateState.PlanGranularity), nil)
    return nil
}
```

### Pattern 5: Status Display (from depth display)
**What:** Add a line below the existing depth line in `renderDashboard()`.
**When to use:** Any persisted setting that users need to see at a glance.
**Example:**
```go
// Source: cmd/status.go:114-120 (verified)
// After depth display:
granularity := string(state.PlanGranularity)
if granularity == "" {
    granularity = "not set"
}
granLbl := granularityLabel(granularity)
fmt.Fprintf(&b, "Granularity: %s\n\n", granLbl)
```

### Anti-Patterns to Avoid
- **Hardcoding phase bounds in agent definitions:** The route-setter currently has "3-6 phases" baked in at 4 locations. These must become dynamic. Leaving stale hardcoded values creates a dual-source-of-truth where the agent ignores the granularity setting.
- **Silent defaults:** D-03 explicitly requires asking the user if no granularity is persisted. Silently defaulting to "milestone" violates this.
- **Auto-trimming out-of-range plans:** D-05 requires warn+choose, not auto-adjustment. The LLM-generated plan might legitimately need 5 phases for a "sprint" goal -- the user should decide.
- **Separate state file for granularity:** D-09 warns against this. Granularity MUST live in COLONY_STATE.json alongside depth.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Enum type with validation | Custom string matching everywhere | `type PlanGranularity string` + `Valid()` method | Consistent with ColonyDepth; compile-time type safety; single source of truth for valid values |
| Range lookup | Inline if/else in plan.md | `GranularityRange()` Go function exposed via `plan-granularity get` | Plan command can call `aether plan-granularity get` to read min/max; single source of truth |
| State persistence | Direct jq manipulation in plan.md | `aether plan-granularity set <value>` or `aether state-mutate` | Uses FileLocker, atomic writes, and validation already built into the Go command layer |

**Key insight:** The depth controls in Phase 3 already solved every Go-side problem this phase has. The new work is the LLM prompt engineering (injecting bounds into the route-setter) and the out-of-range validation UX.

## Common Pitfalls

### Pitfall 1: Stale Hardcoded Phase Limits in Agent Definitions
**What goes wrong:** The route-setter agent still says "3-6 phases" and "Maximum 6 phases" after the granularity feature is added. The agent ignores the granularity bounds and always generates 3-6 phases.
**Why it happens:** There are 4 files containing route-setter definitions, each with hardcoded phase limits in 2 locations. Missing any of them creates an inconsistency.
**How to avoid:** Grep for all instances of "3-6 phases", "Maximum 6 phases", and "3-6 for most goals" before starting. Replace every one. The canonical refs in CONTEXT.md list all 4 files.
**Warning signs:** Running `/ant:plan --granularity sprint` still produces 3+ phases.

### Pitfall 2: Plan Command's "Maximum N Phases" Not Dynamic
**What goes wrong:** The plan command injects "Maximum 6 phases" into the route-setter prompt regardless of selected granularity.
**Why it happens:** The constraint is in the Route-Setter prompt template in plan.md Step 4, not read from state. It needs to be dynamically set from `GranularityRange()`.
**How to avoid:** The plan command must call `aether plan-granularity get` (or read from state directly) and inject the max value into the prompt template.
**Warning signs:** Quarter granularity (8-12) still produces at most 6 phases.

### Pitfall 3: JSON Field Name Mismatch Between Go and jq
**What goes wrong:** Go struct field `PlanGranularity` serializes to `plan_granularity` (snake_case via json tag) but plan.md or playbooks reference `planGranularity` (camelCase).
**Why it happens:** Go struct tags use `json:"plan_granularity"` to match COLONY_STATE.json convention, but internal references might use the Go field name.
**How to avoid:** Use the exact JSON key name `plan_granularity` everywhere outside Go code. The `state-mutate` command and `state-read-field` command both use the snake_case key.
**Warning signs:** `aether state-mutate 'plan_granularity = "sprint"'` fails with "unknown field".

### Pitfall 4: Forgetting the "Always Ask" Behavior
**What goes wrong:** `/ant:plan` runs without asking for granularity when none is persisted, silently skipping the selection step.
**Why it happens:** The plan command jumps straight to the planning depth selection (Step 2) without checking for persisted granularity first.
**How to avoid:** Add a granularity check before the planning depth selection. If `state.plan_granularity` is empty and `--granularity` flag is not provided, prompt the user.
**Warning signs:** First-time `/ant:plan` users never see a granularity prompt.

### Pitfall 5: Agents-CLAUDE Mirror Drift
**What goes wrong:** Changes to `.aether/agents/aether-route-setter.md` are not mirrored to `.aether/agents-claude/aether-route-setter.md`.
**Why it happens:** The agents-claude directory is a packaging mirror that must stay byte-identical. Manual edits to one without the other create drift.
**How to avoid:** After editing route-setter, immediately copy the change to agents-claude. Better: edit both in the same task.
**Warning signs:** `diff .aether/agents/aether-route-setter.md .aether/agents-claude/aether-route-setter.md` shows differences.

### Pitfall 6: OpenCode Agent Parity
**What goes wrong:** Changes to the Claude route-setter are not reflected in the OpenCode agent definition at `.opencode/agents/aether-route-setter.md`.
**Why it happens:** OpenCode maintains "structural parity" (same filenames/count) but the content must also be updated.
**How to avoid:** After updating the Claude agent, update the OpenCode agent with the same dynamic bounds pattern.
**Warning signs:** OpenCode's route-setter still says "3-6 phases".

## Code Examples

Verified patterns from the existing codebase:

### PlanGranularity Enum Type (to add to colony.go)
```go
// Source: Pattern from pkg/colony/colony.go:25-45
type PlanGranularity string

const (
    GranularitySprint    PlanGranularity = "sprint"
    GranularityMilestone PlanGranularity = "milestone"
    GranularityQuarter   PlanGranularity = "quarter"
    GranularityMajor     PlanGranularity = "major"
)

func (g PlanGranularity) Valid() bool {
    switch g {
    case GranularitySprint, GranularityMilestone, GranularityQuarter, GranularityMajor:
        return true
    }
    return false
}

var ErrInvalidGranularity = fmt.Errorf("invalid plan granularity")
```

### ColonyState Field Addition
```go
// Source: pkg/colony/colony.go:67-86 (add after ColonyDepth field)
type ColonyState struct {
    // ... existing fields ...
    ColonyDepth        ColonyDepth       `json:"colony_depth,omitempty"`
    PlanGranularity    PlanGranularity   `json:"plan_granularity,omitempty"`
    // ... remaining fields ...
}
```

### GranularityRange Function (new file: granularity.go)
```go
// Source: Pattern from pkg/colony/depth.go
package colony

func GranularityRange(g PlanGranularity) (min int, max int) {
    switch g {
    case GranularitySprint:
        return 1, 3
    case GranularityMilestone:
        return 4, 7
    case GranularityQuarter:
        return 8, 12
    case GranularityMajor:
        return 13, 20
    default:
        return 1, 3
    }
}
```

### Route-Setter Prompt Injection (plan.md modification)
The current hardcoded text in plan.md Step 4:
```
1. If no plan exists, create 3-6 phases with concrete tasks
```
Becomes:
```
1. If no plan exists, create {granularity_min}-{granularity_max} phases with concrete tasks
```

And the current output constraint:
```
Maximum 6 phases. Maximum 4 tasks per phase.
```
Becomes:
```
Maximum {granularity_max} phases. Maximum 4 tasks per phase.
```

### Out-of-Range Validation (plan.md Step 5 addition)
After the plan is finalized, before or during Step 7 (Display Plan), add validation:
```
granularity_min = {from persisted or selected granularity}
granularity_max = {from persisted or selected granularity}
actual_phases = {count of plan.phases}

if actual_phases < granularity_min OR actual_phases > granularity_max:
    Display warning:
    "Plan has {actual_phases} phases, but {granularity} granularity expects {granularity_min}-{granularity_max}.
     Options:
     1) Accept as-is (the plan may be better than the range suggests)
     2) Adjust granularity to fit
     3) Replan with current granularity"
```

### Autopilot Integration (run.md Step 0 addition)
```markdown
# Source: .aether/commands/claude/run.md:67-69
# After reading COLONY_STATE.json in Step 0:
granularity = state.plan_granularity  # from COLONY_STATE.json
if granularity is set:
    _, max_phases_granularity = GranularityRange(granularity)
    # Use min of --max-phases flag and granularity max as the effective cap
    # But warn (don't block) if the plan exceeds granularity range
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded "3-6 phases" in agent prompt | Dynamic {min}-{max} from PlanGranularity enum | This phase | Plans respect user's chosen scope |
| No granularity persistence | Persist in COLONY_STATE.json alongside depth | This phase | Granularity survives session resets |
| Silent phase count acceptance | Warn+choose on out-of-range plans | This phase | Users maintain control over plan scope |

**Deprecated/outdated:**
- The route-setter's "3-6 phases for most goals" guidance: will be replaced with dynamic bounds from the selected granularity.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The OpenCode route-setter at `.opencode/agents/aether-route-setter.md` also has hardcoded "3-6 phases" text that needs updating | Architecture Patterns | Low -- even if it doesn't, the Claude agent is the primary path; OpenCode can be updated separately |
| A2 | The `.aether/commands/claude/plan.md` and `.claude/commands/ant/plan.md` are the same file (plan.md says "Generated from .aether/commands/plan.yaml") | Architecture Patterns | Low -- CONTEXT.md lists both with different line numbers; if they diverge, both need updating |
| A3 | Adding `--granularity` to `/ant:init` is optional (Claude's discretion) | User Constraints | Low -- can be added later without breaking changes; init already has `--depth` as precedent |

## Open Questions

1. **Should `--granularity` be added to `/ant:init`?**
   - What we know: `/ant:init` already supports `--depth` (Phase 3). The pattern exists.
   - What's unclear: Whether users want to set granularity at colony creation time or only at plan time.
   - Recommendation: Add it. It's low effort (follows `--depth` pattern exactly) and users creating a colony with a known scope (sprint vs quarter) would benefit. Default to empty (not set) so the "always ask" behavior in `/ant:plan` still triggers if not specified at init time.

2. **How should the out-of-range warning interact with the auto-finalize flow?**
   - What we know: The plan command auto-finalizes without user confirmation (Step 4 exits loop, Step 5 writes state, Step 7 displays).
   - What's unclear: If the plan is out of range, should auto-finalize pause for user input, or should the warning appear after finalization?
   - Recommendation: Pause before finalization. The warn+choose decision (accept/adjust/replan) requires user input, so it must happen before the plan is written to COLONY_STATE.json. This means inserting a check between Step 4 (loop exit) and Step 5 (finalize).

## Environment Availability

> Step 2.6: SKIPPED (no external dependencies identified -- this phase is purely code/config changes within the existing Go project and markdown agent definitions).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (standard library) |
| Config file | none -- `go test ./...` |
| Quick run command | `go test ./pkg/colony/ -run TestPlanGranularity -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PLAN-01 | Granularity enum has 4 valid values, Valid() returns correct bools | unit | `go test ./pkg/colony/ -run TestPlanGranularityValid -v` | No -- Wave 0 |
| PLAN-01 | GranularityRange returns correct min/max for each preset | unit | `go test ./pkg/colony/ -run TestGranularityRange -v` | No -- Wave 0 |
| PLAN-02 | plan-granularity get/set commands work correctly | unit | `go test ./cmd/ -run TestPlanGranularity -v` | No -- Wave 0 |
| PLAN-04 | ColonyState JSON round-trip includes plan_granularity | unit | `go test ./pkg/colony/ -run TestRoundTrip -v` | Yes (existing, needs update) |
| PLAN-04 | Status output includes granularity line | unit | `go test ./cmd/ -run TestStatusOutput -v` | Yes (existing, needs update) |
| PLAN-03 | Out-of-range plan triggers warning | manual-only | -- | N/A (LLM prompt behavior) |
| PLAN-05 | Autopilot reads persisted granularity | manual-only | -- | N/A (agent behavior) |

### Sampling Rate
- **Per task commit:** `go test ./pkg/colony/ -run "TestPlanGranularity|TestGranularityRange" -v && go test ./cmd/ -run "TestPlanGranularity|TestStatus" -v`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `pkg/colony/granularity_test.go` -- Tests for `PlanGranularity.Valid()` and `GranularityRange()`
- [ ] `pkg/colony/colony.go` -- Update `TestRoundTripColonyState` golden test to include `PlanGranularity` field
- [ ] `pkg/colony/testdata/COLONY_STATE.golden.json` -- Add `plan_granularity` field to golden file
- [ ] `cmd/status_test.go` -- Update `TestStatusOutput` to expect granularity line in dashboard output

*(Existing test infrastructure covers most needs -- the main gaps are new test files for the new enum type and updates to golden fixtures.)*

## Security Domain

> Not applicable. This phase adds UI/UX controls for plan granularity -- no authentication, cryptography, access control, or input validation beyond the enum validation already described.

## Sources

### Primary (HIGH confidence)
- `pkg/colony/colony.go:25-45` -- ColonyDepth enum pattern (verified via Read tool)
- `pkg/colony/colony.go:66-86` -- ColonyState struct (verified via Read tool)
- `pkg/colony/depth.go` -- DepthBudget function pattern (verified via Read tool)
- `pkg/colony/depth_test.go` -- Test pattern for enum + range function (verified via Read tool)
- `cmd/colony_cmds.go:65-143` -- colony-depth get/set commands (verified via Read tool)
- `cmd/status.go:114-120` -- Depth display in status dashboard (verified via Read tool)
- `cmd/state_cmds.go:117-123` -- state-mutate depth validation (verified via Read tool)
- `cmd/state_cmds.go:172-178` -- Expression validation for depth (verified via Read tool)
- `.claude/commands/ant/plan.md:477-487` -- Route-setter output constraints with hardcoded "Maximum 6 phases" (verified via Read tool)
- `.aether/agents/aether-route-setter.md:36,106` -- Hardcoded "3-6 phases" references (verified via Read tool)
- `.aether/agents-claude/aether-route-setter.md` -- Mirror file (verified via Read tool)
- `.aether/commands/claude/run.md:67-69` -- Autopilot Step 0 COLONY_STATE.json read (verified via Read tool)
- `.planning/phases/03-build-depth-controls/03-CONTEXT.md` -- Phase 3 established patterns (verified via Read tool)
- `04-CONTEXT.md` -- User decisions and canonical references (verified via Read tool)

### Secondary (MEDIUM confidence)
- None -- all findings are from verified source code reads.

### Tertiary (LOW confidence)
- None -- no web search was needed for this phase.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- uses only existing project packages, no new dependencies
- Architecture: HIGH -- every pattern has a verified template in Phase 3's depth work
- Pitfalls: HIGH -- all pitfalls identified from reading the actual code with hardcoded values

**Research date:** 2026-04-07
**Valid until:** 60 days (stable domain -- enum types and prompt templates don't change frequently)
