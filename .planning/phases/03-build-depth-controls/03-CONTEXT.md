# Phase 3: Build Depth Controls - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Let users control how thoroughly the colony builds by selecting light, standard, deep, or full depth. This phase covers: defining the 4 depth levels and what each controls, adding a Go enum type for compile-time safety, wiring token budget scaling per depth, ensuring depth persists across builds via `/ant:init` and `/ant:build --depth`, and making depth visible in `/ant:status`. It does NOT cover planning granularity or orchestration — those are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Depth levels
- **D-01:** 4 depth levels: light, standard, deep, full. The "full" level is kept beyond the 3 defined in DEPTH-01 through DEPTH-06 (requirements traceability updated to note full as an additional level).
- **D-02:** Existing `depthLabel()` descriptions in `cmd/status.go` are wrong and must be corrected to match actual depth behavior per DEPTH-02/03/04 plus the full level's existing chaos gating.

### Token budget scaling
- **D-03:** Progressive (non-linear) token budget scaling: Light 4K context + 4K skills, Standard 8K + 8K (current default), Deep 16K + 12K, Full 24K + 16K. Deeper builds get disproportionately more context to support extra specialists.
- **D-04:** The budget values must be accessible via a Go subcommand (e.g., `aether context-budget --depth standard`) so the build playbooks can read them at build time.

### Depth persistence model
- **D-05:** `/ant:build --depth <level>` persists the setting (current behavior via `colony-depth set`). Once set, all future builds use that depth until changed. Simple, predictable, already implemented in playbooks.
- **D-06:** `/ant:init "goal" --depth light` sets depth at colony creation time (new flag on init command). Depth defaults to "standard" if not specified.

### Depth validation enforcement
- **D-07:** Create a proper Go enum type (`ColonyDepth`) with constants for each valid level (like `State` has `StateREADY`). Replace the bare `string` field on `ColonyState` with this typed field.
- **D-08:** `state-mutate` must validate depth values against the enum — invalid values produce an error, not a silent fallback to "standard".

### Claude's Discretion
- Exact budget implementation (constant map, function, or config)
- Whether the enum type uses `type ColonyDepth string` with constants or an iota-based approach
- How to handle backward compatibility with existing COLONY_STATE.json files that have string depth values
- Whether `colony-depth set` command needs changes beyond the enum migration
- Exact error message when state-mutate receives an invalid depth value
- Whether the `--depth` flag on `/ant:init` should accept the same 4 values

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — DEPTH-01 through DEPTH-06 define the exact requirements for this phase

### Roadmap
- `.planning/ROADMAP.md` §Phase 3 — Success criteria, dependencies, and risk notes

### Colony state model
- `pkg/colony/colony.go:45-64` — `ColonyState` struct with `ColonyDepth string` field (needs enum migration)
- `pkg/colony/colony.go:16-23` — `State` type pattern to follow for the new `ColonyDepth` enum

### Existing depth commands
- `cmd/colony_cmds.go:65-143` — `colony-depth get/set` commands (validate against light/standard/deep/full)
- `cmd/status.go:114-120` — Depth display in status output
- `cmd/status.go:187-201` — `depthLabel()` function (descriptions are WRONG — need correction)

### Init command (needs --depth flag)
- `cmd/init_cmd.go:22-177` — `aether init` command (add `--depth` flag, persist in colony state)

### Build playbooks (depth gating already exists)
- `.aether/docs/command-playbooks/build-prep.md:119-178` — `--depth` flag parsing, colony-depth get/set calls
- `.aether/docs/command-playbooks/build-context.md:95-126` — 8K research budget (needs depth-based scaling)
- `.aether/docs/command-playbooks/build-wave.md:36-62` — Scout gating at light depth, Oracle/Architect gating, spawn count formula
- `.aether/docs/command-playbooks/build-wave.md:79-85` — Oracle DEPTH CHECK (skip at light/standard)
- `.aether/docs/command-playbooks/build-wave.md:153-159` — Architect DEPTH CHECK (skip at light/standard)
- `.aether/docs/command-playbooks/build-wave.md:470-472` — Context layer budget caps (need depth-based scaling)
- `.aether/docs/command-playbooks/build-verify.md:260-263` — Chaos DEPTH CHECK (skip if not full)

### State mutation (needs validation)
- `cmd/state_cmds.go` — `state-mutate` command (must validate depth values)
- `cmd/state_extra.go` — `state-write` command

### Phase 1 infrastructure (audit logging for depth changes)
- `.planning/phases/01-state-protection/01-CONTEXT.md` — Audit patterns established
- `pkg/storage/storage.go` — `AppendJSONL` for audit trail

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `colony-depth get/set` commands already work — validate and persist depth correctly
- Build playbooks already have DEPTH CHECK gates at archaeologist, oracle, architect, and chaos steps
- `/ant:build --depth <level>` flag parsing already exists in build-prep.md
- `State` type pattern in `pkg/colony/colony.go` provides the exact template for the new `ColonyDepth` enum
- Phase 1 audit infrastructure (`AppendJSONL`) available for logging depth changes

### Established Patterns
- Commands use `outputOK()` / `outputError()` for consistent JSON output
- State mutations go through `store.SaveJSON()` with `FileLocker`
- Enum types use `type X string` with const declarations (see `State` type)
- Build playbooks read depth via `aether colony-depth get` and store as cross-stage variable

### Integration Points
- `pkg/colony/colony.go` — `ColonyState.ColonyDepth` field (string → enum migration)
- `cmd/init_cmd.go` — Add `--depth` flag
- `cmd/state_cmds.go` — Add depth validation in `state-mutate`
- `cmd/status.go` — Fix `depthLabel()` descriptions
- `.aether/docs/command-playbooks/build-prep.md` — Wire budget subcommand
- `.aether/docs/command-playbooks/build-context.md` — Use depth-based budget cap
- `.aether/docs/command-playbooks/build-wave.md` — Use depth-based context budget caps

</code_context>

<specifics>
## Specific Ideas

- Progressive budget scaling gives deep/full builds disproportionately more context — this matters because deep builds spawn oracle + architect who need richer context
- Keeping 4 levels means the existing playbook depth checks (which gate on "full" for chaos) don't need restructuring
- The `full` level is the "no compromises" mode — everything runs, max budget, all specialists

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---
*Phase: 03-build-depth-controls*
*Context gathered: 2026-04-07*
