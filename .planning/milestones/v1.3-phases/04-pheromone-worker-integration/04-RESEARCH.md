# Phase 4: Pheromone Worker Integration - Research

**Researched:** 2026-03-19
**Domain:** Agent definition modification, pheromone signal behavioral integration, midden threshold auto-REDIRECT
**Confidence:** HIGH

## Summary

Phase 4 bridges the gap between signals flowing through the system (Phase 3 proved this) and workers actually responding to those signals. Currently, the prompt_section containing pheromone signals is injected into worker spawn context via build-wave.md, but the agent definitions themselves (aether-builder.md, aether-watcher.md, aether-scout.md) contain ZERO references to pheromones, signals, FOCUS, REDIRECT, or FEEDBACK. Workers receive the signal text but have no instructions to acknowledge or act on it. This is the core problem this phase solves.

The phase has three distinct deliverables: (1) update agent definitions to contain explicit instructions for interpreting and acting on injected pheromone context, (2) verify that auto-emitted signals from one build phase influence subsequent builds, and (3) wire midden failure threshold detection to auto-REDIRECT creation and verify workers avoid flagged patterns. The infrastructure already exists -- memory-capture auto-emits pheromones, the midden threshold check in build-wave.md creates REDIRECTs, and colony-prime assembles prompt_section -- but the behavioral loop is broken at the agent definition level.

**Primary recommendation:** Add a dedicated `<pheromone_protocol>` section to aether-builder.md, aether-watcher.md, and aether-scout.md that explicitly instructs each agent how to interpret and respond to FOCUS, REDIRECT, and FEEDBACK signals injected via prompt_section. Simultaneously update the `.aether/agents-claude/` mirror copies. Write integration tests that prove signal-influenced behavior.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PHER-03 | Agent definitions updated to acknowledge and act on injected pheromone context (at minimum: builder, watcher, scout) | Agent definitions currently have ZERO pheromone references. Research identifies the exact insertion points, the `<pheromone_protocol>` section format, and the sync requirement with `.aether/agents-claude/` mirror. |
| PHER-04 | Auto-emitted signals during builds verified to influence subsequent build phases | memory-capture already auto-emits pheromones (failure->REDIRECT, learning->FEEDBACK). colony-prime already includes them in prompt_section. Research confirms the injection chain works end-to-end. Verifying influence requires agent definitions to have pheromone instructions (PHER-03 is prerequisite). |
| PHER-05 | Midden threshold auto-REDIRECT verified with real failure data | The midden threshold check exists in build-wave.md (Step 5.2 MID-03) and creates REDIRECT signals for categories with 3+ failures. Research confirms the bash code, the pheromone-write call with `auto:error` source, and the deduplication check against existing signals. Existing tests verify the primitive operations but not the full behavioral loop. |
</phase_requirements>

## Standard Stack

### Core

| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| Agent definitions (Claude) | `.claude/agents/ant/aether-{builder,watcher,scout}.md` | Agent behavior specifications consumed by Claude Code Task tool | These are the canonical agent definitions -- the Task tool loads them as subagent instructions |
| Agent mirror (packaging) | `.aether/agents-claude/aether-{builder,watcher,scout}.md` | Byte-identical copies for npm packaging | Required by `npm run lint:sync` -- must stay in sync with `.claude/agents/ant/` |
| aether-utils.sh | `.aether/aether-utils.sh` | 150 subcommands including all pheromone operations | Single bash entry point for colony operations |
| build-wave.md | `.aether/docs/command-playbooks/build-wave.md` | Worker spawn orchestration with prompt_section injection | Where the prompt_section (with pheromone signals) is injected into worker prompts |
| colony-prime | `aether-utils.sh colony-prime --compact` | Assembles unified prompt_section from QUEEN wisdom + context capsule + pheromone signals + instincts | Called at Step 4 of build-context.md before workers are spawned |

### Supporting

| Component | Location | Purpose | When Used |
|-----------|----------|---------|-----------|
| memory-capture | `aether-utils.sh memory-capture` | Auto-emits pheromones for failure/success/learning events | Called in build-wave.md when workers fail (Step 5.2) and approach changes |
| midden-write | `aether-utils.sh midden-write` | Records failures to structured midden for threshold detection | Called by build-wave.md for each failed worker |
| midden-recent-failures | `aether-utils.sh midden-recent-failures [limit]` | Reads recent failure entries from midden.json | Used by midden threshold check (MID-03) in build-wave.md Step 5.2 |
| pheromone-write | `aether-utils.sh pheromone-write` | Creates new pheromone signals in pheromones.json | Called by memory-capture and midden threshold code |
| generate-commands.sh | `bin/generate-commands.sh` | Sync validation between `.claude/agents/ant/` and `.aether/agents-claude/` | Must pass after agent definition changes |

### Test Infrastructure

| Component | Location | Purpose |
|-----------|----------|---------|
| pheromone-auto-emission.test.js | `tests/integration/` | Tests auto-emission primitives (auto:decision, auto:error, auto:success) |
| pheromone-injection-chain.test.js | `tests/integration/` | Tests signal flow: pheromone-write -> colony-prime -> prompt_section |
| ava test runner | `npm test` | Full test suite (537 tests as of Phase 3 completion) |

## Architecture Patterns

### Current Signal Flow (Working)

```
User emits signal (e.g., /ant:focus "security")
    |
    v
pheromone-write -> pheromones.json
    |
    v
colony-prime --compact -> pheromone-prime
    |
    v
prompt_section (formatted markdown with FOCUS/REDIRECT/FEEDBACK groups)
    |
    v
build-wave.md Step 5.1 injects { prompt_section } into Builder Worker Prompt
    |
    v
Worker receives signal text... but has NO INSTRUCTIONS to act on it  <-- GAP
```

### Auto-Emission Flow (Working)

```
Worker fails during build
    |
    v
memory-capture "failure" "Builder X failed on task Y: reason" "failure" "worker:builder"
    |
    v
pheromone-write REDIRECT "Avoid repeating failure: ..." --source "auto:error" --strength 0.7
    |
    v
pheromones.json now has REDIRECT signal
    |
    v
Next build phase: colony-prime includes this REDIRECT in prompt_section
    |
    v
Worker receives REDIRECT text... but has NO INSTRUCTIONS to honor it  <-- GAP
```

### Midden Threshold Flow (Partially Working)

```
build-wave.md Step 5.2 (after processing wave results):
    |
    v
midden-recent-failures 50 -> get all midden entries
    |
    v
jq groups by .category, selects groups with length >= 3
    |
    v
For each recurring category:
  - Check if REDIRECT already exists (source == "auto:error" && content contains category)
  - If not: pheromone-write REDIRECT "[error-pattern] Category X recurring (N occurrences)"
    |
    v
REDIRECT is created... but workers have no protocol to avoid the flagged pattern  <-- GAP
```

### Recommended Agent Definition Structure

Each of the three target agents (builder, watcher, scout) should receive a new `<pheromone_protocol>` section. The section should be placed AFTER the `<critical_rules>` section and BEFORE the `<return_format>` section, because pheromone awareness is a critical behavioral requirement but secondary to core rules like TDD.

```
<pheromone_protocol>
## Pheromone Signal Response Protocol

Your spawn context may include a `--- COMPACT SIGNALS ---` or `--- ACTIVE SIGNALS ---`
section containing colony guidance. These signals are injected by the Queen and represent
live colony intelligence.

### Signal Types and Required Response

**REDIRECT (HARD CONSTRAINTS - MUST follow):**
- These are non-negotiable avoidance instructions
- If a REDIRECT says "avoid pattern X", you MUST NOT use pattern X
- REDIRECTs marked [error-pattern] come from repeated colony failures -- treat as lessons learned
- Acknowledge each REDIRECT in your output summary

**FOCUS (Pay attention to):**
- These are attention directives -- prioritize the indicated area
- When choosing between approaches, prefer the one aligned with active FOCUS signals
- If a FOCUS says "security", apply extra scrutiny to auth, input validation, secrets

**FEEDBACK (Flexible guidance):**
- These are calibration signals from past experience
- Consider them when making judgment calls
- You may deviate if you have good reason, but note the deviation

### Acknowledgment Protocol

In your return JSON, if any signals were present in your context, include a brief note
in your summary field indicating which signals you observed and how they influenced your work.
</pheromone_protocol>
```

### Agent-Specific Adaptations

**Builder:** REDIRECT signals constrain implementation choices. FOCUS signals influence which areas get extra test coverage. FEEDBACK signals adjust coding patterns.

**Watcher:** REDIRECT signals become verification checkpoints (verify the avoided pattern was indeed avoided). FOCUS signals direct which areas receive deeper scrutiny. FEEDBACK signals influence quality scoring weights.

**Scout:** REDIRECT signals constrain research scope (don't recommend patterns that are redirected). FOCUS signals prioritize research areas. FEEDBACK signals weight source credibility.

### File Sync Requirement

Agent definitions exist in THREE locations:

| Location | Role | Update Required |
|----------|------|-----------------|
| `.claude/agents/ant/aether-{agent}.md` | Canonical (Claude Code reads this) | YES - primary edit |
| `.aether/agents-claude/aether-{agent}.md` | Packaging mirror (npm publish) | YES - must be byte-identical copy |
| `.opencode/agents/aether-{agent}.md` | OpenCode agents | NO - structural parity only, different format |

After editing `.claude/agents/ant/`, copy to `.aether/agents-claude/` and run `npm run lint:sync` to verify.

### Anti-Patterns to Avoid

- **Over-prescriptive signal handling:** Do NOT add 50 lines of complex conditional logic. The pheromone_protocol should be short (under 40 lines) and principle-based, not rule-based. Workers are LLMs -- they understand intent.
- **Duplicating signal parsing logic:** Do NOT instruct workers to parse JSON from pheromones.json directly. The prompt_section is pre-formatted markdown. Workers read it as text, not structured data.
- **Breaking existing worker behavior:** The pheromone_protocol is ADDITIVE. It must not change any existing behavior (TDD discipline, Evidence Iron Law, No Write tools for Watcher/Scout, etc.).
- **Forgetting the mirror:** Editing `.claude/agents/ant/` without updating `.aether/agents-claude/` will break `npm run lint:sync`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Signal formatting for worker prompts | Custom formatting in agent definitions | colony-prime + pheromone-prime (already produces prompt_section) | The formatting is already done correctly with type grouping, strength display, etc. |
| Midden failure tracking | New failure tracking system | midden-write + midden-recent-failures (already exist) | The midden system with category-based grouping and threshold detection is built and tested |
| Auto-pheromone creation from failures | New auto-emission logic | memory-capture subcommand (already auto-emits) | memory-capture handles the observe -> pheromone-write -> auto-promote pipeline |
| Midden threshold -> REDIRECT | New threshold detection | build-wave.md MID-03 code block (already exists in Step 5.2) | The bash code that groups by category, checks for 3+ occurrences, and creates REDIRECT signals is already in build-wave.md |

**Key insight:** The infrastructure is 100% built. Phase 4 is about the agent definition layer (making workers understand signals) and the verification layer (proving the behavioral loop works). No new subcommands or plumbing are needed.

## Common Pitfalls

### Pitfall 1: Editing Only One Copy of Agent Definitions
**What goes wrong:** Agent definition edited in `.claude/agents/ant/` but not mirrored to `.aether/agents-claude/`. The `npm run lint:sync` check fails.
**Why it happens:** Developer forgets about the byte-identical mirror requirement.
**How to avoid:** Always copy the edited file to `.aether/agents-claude/` immediately after editing. Run `npm run lint:sync` as verification.
**Warning signs:** `lint:sync` failure in CI or pre-commit hook.

### Pitfall 2: Making Pheromone Protocol Too Verbose
**What goes wrong:** A 100-line pheromone protocol section bloats the agent definition, consuming precious context window space for every worker spawn.
**Why it happens:** Trying to cover every edge case in the protocol definition.
**How to avoid:** Keep the protocol under 40 lines. Workers are LLMs -- they understand intent-based instructions. Focus on the three signal types and their response expectations.
**Warning signs:** Worker spawn prompts exceeding useful length; workers ignoring protocol due to context saturation.

### Pitfall 3: Testing Signal "Influence" Without Measurable Criteria
**What goes wrong:** Claiming PHER-04 is satisfied because "the signal was in the prompt" without proving the signal changed behavior.
**Why it happens:** Confusing signal presence (Phase 3 already proved this) with behavioral influence.
**How to avoid:** Design tests with a clear before/after: emit a REDIRECT in Phase N, verify in Phase N+1 that the worker's output explicitly acknowledges the REDIRECT and avoids the flagged pattern.
**Warning signs:** Test assertions that only check prompt_section content (that's Phase 3), not worker behavioral output.

### Pitfall 4: Midden Threshold Test Without Real Failure Data
**What goes wrong:** Using synthetic/mock midden data for PHER-05 when the requirement says "verified with real failure data."
**Why it happens:** Real failure data is harder to generate in a test environment.
**How to avoid:** The test should either (a) use the real midden-write subcommand to create failures, then run the threshold check, or (b) seed midden.json with entries that look like real failures (correct schema, realistic categories and messages).
**Warning signs:** Tests that hardcode JSON strings instead of using midden-write.

### Pitfall 5: Assuming Workers Can Read Pheromones Independently
**What goes wrong:** Adding instructions for workers to call `pheromone-read` directly, which duplicates the colony-prime injection and wastes tool calls.
**Why it happens:** Confusion about the architecture. Workers receive pheromone context via prompt_section injection, not by independently reading pheromones.json.
**How to avoid:** The protocol should instruct workers to read the injected signal section in their prompt, not to make additional bash calls.
**Warning signs:** Workers making unnecessary `pheromone-read` calls at spawn time.

## Code Examples

### Current Worker Spawn Prompt (from build-wave.md)

The prompt_section is injected as `{ prompt_section }` in the builder worker prompt. The actual text looks like:

```markdown
--- COMPACT SIGNALS ---

FOCUS (Pay attention to):
[0.8] security

REDIRECT (HARD CONSTRAINTS - MUST follow):
[0.7] [error-pattern] Category "security" recurring (3 occurrences)

FEEDBACK (Flexible guidance):
[0.6] Learning captured: Builder approach-change logged for parsing

--- INSTINCTS (Learned Behaviors) ---

Architecture:
  [0.7] When modifying aether-utils.sh -> Run bash -n before committing

--- END COLONY CONTEXT ---
```

Source: pheromone-prime at lines 7488-7557 of aether-utils.sh

### Midden Threshold Check (from build-wave.md Step 5.2)

```bash
midden_result=$(bash .aether/aether-utils.sh midden-recent-failures 50 2>/dev/null || echo '{"count":0,"failures":[]}')
midden_count=$(echo "$midden_result" | jq '.count // 0')

if [[ "$midden_count" -gt 0 ]]; then
  recurring_categories=$(echo "$midden_result" | jq -r '
    [.failures[] | .category]
    | group_by(.)
    | map(select(length >= 3))
    | map({category: .[0], count: length})
    | .[]
    | @base64
  ' 2>/dev/null || echo "")

  for encoded in $recurring_categories; do
    # ... dedup check against existing auto:error signals ...
    bash .aether/aether-utils.sh pheromone-write REDIRECT \
      "[error-pattern] Category \"$category\" recurring ($count occurrences)" \
      --strength 0.7 \
      --source "auto:error" \
      --reason "Auto-emitted: midden error pattern recurred 3+ times mid-build" \
      --ttl "30d" 2>/dev/null || true
  done
fi
```

Source: build-wave.md lines 499-540

### memory-capture Auto-Emission (from aether-utils.sh)

```bash
# On failure event:
pheromone_type="REDIRECT"
pheromone_content="Avoid repeating failure: $mc_content"
pheromone_strength="0.7"
pheromone_reason="Auto-emitted from failure event"

# Creates via:
pheromone_result=$(bash "$0" pheromone-write "$pheromone_type" "$pheromone_content" \
  --strength "$pheromone_strength" --source "$mc_source" \
  --reason "$pheromone_reason" --ttl "$pheromone_ttl" 2>/dev/null || echo '{}')
```

Source: aether-utils.sh lines 5458-5498

## State of the Art

| Current State | After Phase 4 | Impact |
|---------------|---------------|--------|
| Agent definitions have 0 pheromone references | Builder, watcher, scout each have `<pheromone_protocol>` section | Workers explicitly acknowledge and respond to signals |
| prompt_section injected but not interpreted | Workers have behavioral rules for each signal type | REDIRECT signals actually constrain behavior |
| Midden threshold creates REDIRECTs but nobody reads them | Builder avoids flagged patterns, watcher verifies avoidance | Learning loop closes: fail -> track -> signal -> avoid |
| Auto-emitted pheromones exist but influence is unverified | Integration tests prove cross-phase signal influence | PHER-04 satisfied with evidence |

## Implementation Strategy

### Three Natural Work Streams

**Stream 1: Agent Definition Updates (PHER-03)**
- Add `<pheromone_protocol>` section to aether-builder.md, aether-watcher.md, aether-scout.md
- Each agent gets the same base protocol plus agent-specific adaptations
- Mirror updates to `.aether/agents-claude/`
- Verify with `npm run lint:sync`
- Write tests that grep agent definitions for required pheromone keywords

**Stream 2: Cross-Phase Signal Influence Verification (PHER-04)**
- Requires PHER-03 to be done first (agent definitions must have pheromone instructions)
- Create a test scenario: Phase N emits a REDIRECT via memory-capture -> Phase N+1 worker receives it via colony-prime -> worker output acknowledges the REDIRECT
- The test can be an integration test using the existing test infrastructure (setupTestColony, runAetherUtil)
- The key assertion: the prompt_section delivered to a subsequent phase contains the auto-emitted signal

**Stream 3: Midden Threshold Auto-REDIRECT Verification (PHER-05)**
- Write 3+ failures to midden via midden-write (same category)
- Run the midden threshold check logic (the bash code from build-wave.md)
- Verify REDIRECT signal is created in pheromones.json with source "auto:error"
- Verify the REDIRECT appears in colony-prime prompt_section
- Verify agent definition instructs builder to avoid flagged patterns (ties to PHER-03)

### Recommended Plan Split

**Plan 04-01:** Agent definition updates (PHER-03) + lint:sync verification
**Plan 04-02:** Cross-phase signal influence tests (PHER-04) + midden threshold auto-REDIRECT tests (PHER-05)

Rationale: PHER-03 is the prerequisite for both PHER-04 and PHER-05. Without pheromone protocol in agent definitions, there is no way to verify that signals "influence behavior." Plans should be sequential.

## Open Questions

1. **How to verify "influence" in an automated test?**
   - What we know: The prompt_section containing signals reaches the worker. Agent definitions will instruct workers to acknowledge signals.
   - What's unclear: In a test environment, we can verify the signal is in the prompt but not that a live LLM actually changes its behavior. The test can verify the structural chain (signal emitted -> signal in subsequent prompt_section) but not the cognitive response.
   - Recommendation: Define "influence" as: (a) the auto-emitted signal appears in the prompt_section of the subsequent build's colony-prime output, AND (b) the agent definition contains explicit instructions to respond to that signal type. This is the maximum we can verify without actually running a live build.

2. **Should OpenCode agent definitions also be updated?**
   - What we know: OpenCode agents in `.opencode/agents/` maintain "structural parity" (same filenames/count) but are NOT byte-identical. CLAUDE.md states they have "different format."
   - What's unclear: Whether the roadmap intends PHER-03 to cover OpenCode agents too.
   - Recommendation: Defer OpenCode agent updates. The requirement says "at minimum: builder, watcher, scout" which refers to the Claude Code agent definitions. OpenCode parity can be a follow-up.

## Sources

### Primary (HIGH confidence)
- `.claude/agents/ant/aether-builder.md` (188 lines, zero pheromone references) -- verified by grep
- `.claude/agents/ant/aether-watcher.md` (245 lines, zero pheromone references) -- verified by grep
- `.claude/agents/ant/aether-scout.md` (143 lines, zero pheromone references) -- verified by grep
- `.aether/docs/command-playbooks/build-wave.md` -- prompt_section injection at line 319, midden threshold at lines 495-540
- `.aether/docs/command-playbooks/build-context.md` -- colony-prime call at Step 4
- `.aether/aether-utils.sh` lines 5414-5516 -- memory-capture with auto-pheromone emission
- `.aether/aether-utils.sh` lines 7381-7558 -- pheromone-prime subcommand
- `.aether/aether-utils.sh` lines 7560-7760 -- colony-prime subcommand
- `.aether/aether-utils.sh` lines 8245-8312 -- midden-write subcommand
- `.aether/aether-utils.sh` lines 9615-9640 -- midden-recent-failures subcommand
- `tests/integration/pheromone-auto-emission.test.js` -- existing auto-emission tests
- `tests/integration/pheromone-injection-chain.test.js` -- existing injection chain tests
- `.planning/phases/03-pheromone-signal-plumbing/03-VERIFICATION.md` -- Phase 3 completion evidence
- `bin/generate-commands.sh` -- sync validation script

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` -- PHER-03, PHER-04, PHER-05 definitions
- `.planning/ROADMAP.md` -- Phase 4 description and success criteria

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all components verified by direct code reading
- Architecture: HIGH -- the signal flow, injection chain, and midden system are well-documented and tested
- Pitfalls: HIGH -- identified from actual code analysis (missing mirror sync, verbose protocols, testing challenges)
- Implementation strategy: HIGH -- based on concrete code analysis showing exact gaps

**Research date:** 2026-03-19
**Valid until:** 2026-04-19 (30 days -- stable system, no external dependencies)
