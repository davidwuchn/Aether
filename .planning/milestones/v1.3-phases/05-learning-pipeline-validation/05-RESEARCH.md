# Phase 5: Learning Pipeline Validation - Research

**Researched:** 2026-03-19
**Domain:** Learning pipeline end-to-end validation (observation -> promotion -> instinct -> worker prompt influence)
**Confidence:** HIGH

## Summary

Phase 5 validates that the observation-to-instinct learning pipeline works end-to-end with real data, and that promoted instincts actually influence worker behavior through the colony-prime prompt_section. The pipeline consists of four subcommands chained together: `memory-capture` (entry point) -> `learning-observe` (deduplication + counting) -> `learning-promote-auto` (threshold check + promotion) -> `instinct-create` (behavioral encoding). When promotion succeeds, two things happen: the learning is written to QUEEN.md (via `queen-promote`) and an instinct is created in COLONY_STATE.json. Both are then assembled into the `prompt_section` by `colony-prime`, which calls `pheromone-prime` to format instincts and extracts QUEEN wisdom directly.

The infrastructure is fully built and already has extensive test coverage in `tests/integration/learning-pipeline.test.js` (9 tests), `tests/integration/instinct-pipeline.test.js` (8 tests), and `tests/integration/wisdom-promotion.test.js` (8 tests). However, all existing tests use synthetic/test data (hardcoded strings like "Test observation for pipeline verification"). The key gap this phase addresses is: (a) validating the pipeline with non-synthetic, realistic data that represents actual colony learning patterns, (b) proving end-to-end that `memory-capture` all the way through to `instinct-create` produces instincts visible in `colony-prime`, and (c) confirming that the pheromone_protocol in agent definitions (added in Phase 4) ensures promoted instincts reach worker prompts where they can influence behavior.

**Primary recommendation:** Write a focused set of integration tests that exercise the complete `memory-capture` -> `instinct-create` -> `colony-prime` pipeline using realistic observation content (actual patterns discovered during Aether development), verify instincts appear in `prompt_section` with correct domain grouping, and confirm agent definitions contain pheromone_protocol sections that instruct workers to act on instincts. No new subcommands or plumbing changes are needed -- this phase is purely validation and testing.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LRNG-01 | Observation -> learning -> instinct pipeline validated end-to-end with real (non-test) data | The pipeline chain is: `memory-capture` -> `learning-observe` -> `learning-promote-auto` -> `queen-promote` + `instinct-create`. All subcommands exist and work (verified by existing tests). The gap is using "real" data -- observations that represent actual patterns discovered during Aether development, not synthetic test strings. Research documents the exact thresholds: pattern auto=2, philosophy auto=3, redirect auto=2, failure auto=2, decree auto=0. Tests must call `memory-capture` enough times with the same content to meet threshold, then verify the instinct was created in COLONY_STATE.json. |
| LRNG-02 | Promoted instincts appear in colony-prime output and influence worker prompts | `colony-prime` calls `pheromone-prime` which reads instincts from COLONY_STATE.json memory.instincts, filters by confidence >= 0.5 and status != "disproven", groups by domain, and formats them as `--- INSTINCTS (Learned Behaviors) ---` in the prompt_section. Phase 4 added `<pheromone_protocol>` sections to builder, watcher, and scout agent definitions instructing workers to act on injected signals including instincts. The chain is complete. Tests must verify: (1) instinct appears in colony-prime prompt_section, (2) agent definitions contain pheromone_protocol, and (3) the instinct format includes both trigger and action text. |
| LRNG-03 | Integration test covers full pipeline: memory-capture -> learning-observe -> threshold met -> learning-promote-auto -> instinct-create | This is the end-to-end integration test requirement. Research confirms `memory-capture` internally calls both `learning-observe` AND `learning-promote-auto`, so calling `memory-capture` twice with the same content (for pattern type, auto threshold=2) is sufficient to trigger the entire chain. The test must then verify: (a) the instinct exists in COLONY_STATE.json, (b) QUEEN.md contains the promoted content, (c) colony-prime includes the instinct in prompt_section. The existing test `memory-capture failure emits redirect pheromone and auto-promotes on recurrence` in learning-pipeline.test.js is close but uses synthetic data ("Repeated null dereference in parser") and doesn't verify colony-prime output. |
</phase_requirements>

## Standard Stack

### Core

| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| aether-utils.sh | `.aether/aether-utils.sh` | All pipeline subcommands (memory-capture, learning-observe, learning-promote-auto, instinct-create, colony-prime) | Single entry point for all colony operations; 150 subcommands |
| COLONY_STATE.json | `.aether/data/COLONY_STATE.json` | Stores instincts in `memory.instincts` array | Canonical state file for colony memory |
| QUEEN.md | `.aether/QUEEN.md` | Stores promoted wisdom in typed sections (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees) | Canonical wisdom file; colony-prime extracts from this |
| learning-observations.json | `.aether/data/learning-observations.json` | Tracks observation counts and deduplication via SHA256 content hashing | Used by learning-observe for threshold tracking |
| pheromones.json | `.aether/data/pheromones.json` | Stores pheromone signals (memory-capture auto-emits these) | Required by colony-prime for pheromone-prime call |

### Test Infrastructure

| Component | Location | Purpose | When Used |
|-----------|----------|---------|-----------|
| AVA test runner | `npm test` (runs `ava`) | Unit and integration test execution | All tests; configured with 30s timeout, files in `tests/unit/**/*.test.js` |
| setupTestColony helper | Used in all integration tests | Creates temp directory with QUEEN.md, COLONY_STATE.json, pheromones.json, learning-observations.json | Every test that exercises aether-utils.sh subcommands |
| runAetherUtil helper | Used in all integration tests | Calls `bash aether-utils.sh <subcommand>` with correct AETHER_ROOT and DATA_DIR env vars | Every subcommand invocation in tests |
| parseLastJson helper | `tests/integration/wisdom-promotion.test.js` | Extracts last JSON line from multi-line output (learning-promote-auto emits instinct-create output on separate line) | Required when calling learning-promote-auto or memory-capture which call instinct-create internally |

### Existing Test Coverage

| Test File | Tests | What It Covers | Gap |
|-----------|-------|----------------|-----|
| learning-pipeline.test.js | 9 | learning-observe, learning-check-promotion, queen-promote, colony-prime reads wisdom, complete pipeline, decree threshold, failure mapping, learning-promote-auto recurrence, memory-capture with auto-promotion | Uses synthetic data; doesn't verify instincts in colony-prime prompt_section |
| instinct-pipeline.test.js | 8 | instinct-create, confidence boost, instinct-read filters, pheromone-prime domain grouping, colony-prime includes instincts, complete pipeline | Uses pre-seeded instincts; doesn't exercise memory-capture -> instinct-create chain |
| wisdom-promotion.test.js | 8 | learning-promote-auto threshold, skip below threshold, idempotency, memory-capture e2e, batch sweep, colony-prime includes wisdom | Comprehensive but synthetic data; doesn't check instinct creation as part of wisdom promotion |

## Architecture Patterns

### Complete Pipeline Flow

```
User/System triggers memory-capture
    |
    v
memory-capture <event_type> <content> [wisdom_type] [source]
    |
    +---> learning-observe <content> <wisdom_type> [colony_name]
    |       |
    |       +---> Deduplicates via SHA256 hash
    |       +---> Increments observation_count
    |       +---> Returns: observation_count, threshold_met
    |
    +---> pheromone-write (auto-emits based on event_type)
    |       failure -> REDIRECT, learning/success -> FEEDBACK
    |
    +---> learning-promote-auto <wisdom_type> <content> [colony_name] [event_type]
            |
            +---> Checks observation_count >= auto threshold
            |       pattern=2, philosophy=3, redirect=2, stack=2, decree=0, failure=2
            |
            +---> If threshold met:
                    +---> queen-promote -> writes to QUEEN.md section
                    +---> instinct-create -> writes to COLONY_STATE.json memory.instincts
                            trigger: "When working on {wisdom_type} patterns"
                            action: "{content}"
                            confidence: min(0.7 + (obs_count - 1) * 0.05, 0.9)
                            domain: "{wisdom_type}"
                            source: "promoted_from_learning"
```

### Instinct Visibility in colony-prime

```
colony-prime [--compact]
    |
    +---> Extracts QUEEN wisdom (Philosophies, Patterns, Redirects, Stack, Decrees)
    |       -> "--- QUEEN WISDOM (Eternal Guidance) ---" in prompt_section
    |
    +---> Calls pheromone-prime -> reads instincts from COLONY_STATE.json
    |       -> Filters: confidence >= 0.5, status != "disproven"
    |       -> Sorts by confidence descending
    |       -> Caps at max (default 5, compact 3)
    |       -> Groups by domain, formats as:
    |           "--- INSTINCTS (Learned Behaviors) ---"
    |           "Domain:"
    |           "  [0.7] When trigger -> action"
    |       -> Appended to prompt_section
    |
    +---> Extracts phase learnings, decisions, blockers, rolling summary
    |
    +---> Returns: { wisdom, prompt_section, log_line, signals }
```

### Wisdom Threshold Table (Verified from source)

| Wisdom Type | Propose Threshold | Auto Threshold | Notes |
|-------------|------------------|----------------|-------|
| philosophy | 1 | 3 | Highest bar for auto-promotion |
| pattern | 1 | 2 | Most common type |
| redirect | 1 | 2 | Anti-patterns |
| stack | 1 | 2 | Technology-specific |
| decree | 0 | 0 | Immediate promotion (user mandates) |
| failure | 1 | 2 | Maps to Patterns section when promoted |

Source: `get_wisdom_threshold()` at lines 938-959 of aether-utils.sh

### Recurrence-Calibrated Confidence Formula

When `learning-promote-auto` creates an instinct, the confidence is calculated as:

```
confidence = min(0.7 + (observation_count - 1) * 0.05, 0.9)
```

- At 2 observations: 0.75
- At 3 observations: 0.80
- At 5 observations: 0.90 (cap)

Source: lines 5367-5372 of aether-utils.sh

### Test Colony Setup Pattern

All integration tests follow this pattern:

```javascript
const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

async function createTempDir() {
  return await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-learning-'));
}

function runAetherUtil(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

function parseLastJson(output) {
  const lines = output.trim().split('\n');
  return JSON.parse(lines[lines.length - 1]);
}

async function setupTestColony(tmpDir, opts = {}) {
  // Creates: .aether/QUEEN.md, .aether/data/COLONY_STATE.json,
  //          .aether/data/pheromones.json, .aether/data/learning-observations.json
  // Accepts: opts.instincts, opts.phaseLearnings, opts.currentPhase, opts.sessionId
}

test.serial('test name', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    // ... test body ...
  } finally {
    await fs.promises.rm(tmpDir, { recursive: true, force: true }).catch(() => {});
  }
});
```

### Anti-Patterns to Avoid

- **Using synthetic test strings:** LRNG-01 explicitly requires "real (non-test) data." Instead of "Test observation for pipeline verification", use realistic content like "Use jq explicit if/elif chains instead of // operator when checking boolean fields that can be false" (an actual pattern discovered during Phase 3).
- **Testing learning-promote-auto directly instead of through memory-capture:** LRNG-03 specifies the test must go through `memory-capture`, not call pipeline steps individually. The memory-capture subcommand is the real entry point.
- **Forgetting parseLastJson:** `memory-capture` calls `learning-promote-auto` which calls `instinct-create`. The instinct-create output is on a separate line before the memory-capture result. Using `JSON.parse(output)` will fail; use `parseLastJson(output)` to get the authoritative last-line result.
- **Not checking both QUEEN.md AND instincts:** The `learning-promote-auto` path writes to BOTH QUEEN.md (via queen-promote) AND COLONY_STATE.json (via instinct-create). Tests must verify both artifacts.
- **Assuming COLONY_STATE.json instincts appear without pheromones.json:** colony-prime calls pheromone-prime which requires pheromones.json to exist. The setupTestColony helper must create an empty pheromones.json even if no signals are needed.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Observation deduplication | Custom hash-based tracking | `learning-observe` subcommand (SHA256 content hashing, count tracking) | Already handles hash generation, count increment, colony tracking |
| Threshold-based promotion | Custom threshold checking logic | `learning-promote-auto` subcommand (reads from `get_wisdom_threshold`) | Already compares observation_count against auto threshold from centralized table |
| Instinct creation from learning | Custom instinct JSON building | `instinct-create` (called by learning-promote-auto automatically) | Handles dedup, confidence boost, 30-instinct cap, atomic writes |
| Prompt assembly | Custom prompt formatting | `colony-prime` (calls pheromone-prime for instinct formatting) | Already assembles QUEEN wisdom + pheromone signals + instincts + learnings + decisions |
| Test colony setup | Ad-hoc file creation | `setupTestColony` helper pattern (used by all existing integration tests) | Consistent structure: QUEEN.md with emoji headers, COLONY_STATE.json with memory, pheromones.json, learning-observations.json |

**Key insight:** Every piece of the pipeline already exists and works. This phase is 100% validation -- writing tests that prove the chain works end-to-end with realistic data. No new subcommands, no plumbing changes.

## Common Pitfalls

### Pitfall 1: Synthetic Data Passing When Real Data Fails
**What goes wrong:** Tests pass with simple strings but fail with realistic content containing special characters, quotes, or multi-word phrases.
**Why it happens:** Shell argument quoting issues with content like `Use jq's // operator carefully when checking booleans` (the apostrophe and // can cause problems).
**How to avoid:** Use realistic content that exercises shell quoting edge cases. The `runAetherUtil` helper already wraps args in double quotes, but content with embedded double quotes, single quotes, or backslashes needs testing.
**Warning signs:** Tests pass with "simple test content" but the pipeline fails in real colony usage.

### Pitfall 2: Multi-Line JSON Output Parsing
**What goes wrong:** `JSON.parse()` fails because `memory-capture` (or `learning-promote-auto`) emits multiple JSON lines -- the instinct-create output followed by the memory-capture result.
**Why it happens:** `learning-promote-auto` calls `instinct-create` which outputs its own JSON to stdout. Then memory-capture captures the `learning-promote-auto` output using `tail -1` internally, but the raw output from memory-capture still has multiple lines in some code paths.
**How to avoid:** Always use `parseLastJson()` helper to extract the last JSON line. The existing `wisdom-promotion.test.js` already implements this pattern.
**Warning signs:** Tests failing with "Unexpected token" JSON parse errors.

### Pitfall 3: Threshold Not Met Because observation_count Is Wrong
**What goes wrong:** Test calls `memory-capture` twice but the learning isn't promoted because the observation_count doesn't reach the auto threshold.
**Why it happens:** `memory-capture` calls `learning-observe` internally. If the content differs even slightly between calls (trailing whitespace, different capitalization), SHA256 produces a different hash, so the observations aren't deduplicated -- each is counted as observation_count=1.
**How to avoid:** Use the EXACT same content string for repeated calls. Trim whitespace. Be consistent with capitalization. The deduplication is exact-match on SHA256 hash of the content.
**Warning signs:** `auto_promoted: false` when it should be true; `reason: "threshold_not_met"`.

### Pitfall 4: QUEEN.md Threshold vs Auto Threshold Confusion
**What goes wrong:** Test assumes one observation is enough for auto-promotion because the "propose" threshold is 1.
**Why it happens:** Two separate thresholds exist: `propose` (for manual review, typically 1) and `auto` (for automatic promotion, typically 2-3). `learning-promote-auto` uses the `auto` threshold.
**How to avoid:** Always check `get_wisdom_threshold <type> auto` for the correct threshold. For pattern type, auto threshold is 2, meaning memory-capture must be called at least twice with the same content.
**Warning signs:** Test expects promotion after 1 call but gets `threshold_not_met`.

### Pitfall 5: Colony State Missing memory.instincts Array
**What goes wrong:** `instinct-create` fails because COLONY_STATE.json doesn't have a `memory.instincts` field.
**Why it happens:** The setupTestColony helper might not include the `memory` object with an `instincts` array.
**How to avoid:** Ensure COLONY_STATE.json has `{"memory": {"instincts": [], "phase_learnings": [], "decisions": []}}`. The existing setupTestColony patterns all include this.
**Warning signs:** `instinct-create` returning error about missing state file or failing to read instincts.

## Code Examples

### Realistic Content for LRNG-01 (Non-Synthetic Data)

Based on actual patterns discovered during Aether v1.3 maintenance (prior phases):

```javascript
// Pattern: discovered in Phase 3 (03-01) -- jq boolean handling
const realPattern = 'Use explicit jq if/elif chains instead of the // operator when checking fields that can legitimately be false';

// Redirect: discovered during data purge (Phase 1) -- test data pollution
const realRedirect = 'Never commit test artifacts to colony state files that ship to users';

// Failure: recurring build issue pattern
const realFailure = 'Validate JSON structure before atomic_write to prevent corrupted state files';

// Philosophy: emerged from multi-phase development
const realPhilosophy = 'Test the pipeline end-to-end before testing individual components in isolation';
```

### Complete Pipeline Test (LRNG-03 Pattern)

```javascript
test.serial('memory-capture -> instinct-create -> colony-prime end-to-end', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);

    const content = 'Use explicit jq if/elif chains instead of the // operator when checking fields that can legitimately be false';

    // First memory-capture: observation recorded, not yet promoted (pattern auto=2)
    const first = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', content, 'pattern', 'worker:builder'
    ]));
    t.true(first.ok);
    t.false(first.result.auto_promoted, 'First call should not auto-promote');
    t.is(first.result.observation_count, 1);

    // Second memory-capture: threshold met, auto-promoted
    const second = parseLastJson(runAetherUtil(tmpDir, 'memory-capture', [
      'learning', content, 'pattern', 'worker:builder'
    ]));
    t.true(second.ok);
    t.true(second.result.auto_promoted, 'Second call should auto-promote');

    // Verify instinct created in COLONY_STATE.json
    const state = JSON.parse(fs.readFileSync(
      path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json'), 'utf8'
    ));
    t.true(state.memory.instincts.length >= 1, 'Should have at least 1 instinct');
    const instinct = state.memory.instincts.find(i => i.action === content);
    t.truthy(instinct, 'Instinct action should match promoted content');
    t.is(instinct.source, 'promoted_from_learning');
    t.is(instinct.domain, 'pattern');
    t.true(instinct.confidence >= 0.7);

    // Verify QUEEN.md contains the promoted wisdom
    const queen = fs.readFileSync(
      path.join(tmpDir, '.aether', 'QUEEN.md'), 'utf8'
    );
    t.true(queen.includes(content), 'QUEEN.md should contain promoted content');

    // Verify colony-prime includes instinct in prompt_section
    const prime = JSON.parse(runAetherUtil(tmpDir, 'colony-prime'));
    t.true(prime.ok);
    t.true(prime.result.prompt_section.includes(content),
      'colony-prime prompt_section should include instinct action');
    t.true(prime.result.prompt_section.includes('INSTINCTS'),
      'prompt_section should have INSTINCTS header');
  } finally {
    await fs.promises.rm(tmpDir, { recursive: true, force: true }).catch(() => {});
  }
});
```

### Verifying Agent Definitions Contain Pheromone Protocol (LRNG-02)

```javascript
test.serial('agent definitions contain pheromone_protocol for instinct influence', async (t) => {
  const agentDir = path.join(process.cwd(), '.claude', 'agents', 'ant');
  const agents = ['aether-builder.md', 'aether-watcher.md', 'aether-scout.md'];

  for (const agent of agents) {
    const content = fs.readFileSync(path.join(agentDir, agent), 'utf8');
    t.true(content.includes('<pheromone_protocol>'),
      `${agent} should contain <pheromone_protocol> section`);
    t.true(content.includes('INSTINCTS') || content.includes('instinct') || content.includes('Learned Behaviors'),
      `${agent} should reference instincts or learned behaviors`);
  }
});
```

### Instinct Auto-Generated Trigger/Action Format

When `learning-promote-auto` creates an instinct, it uses this exact format:

```bash
bash "$0" instinct-create \
  --trigger "When working on $wisdom_type patterns" \
  --action "$content" \
  --confidence "$lp_confidence" \
  --domain "$wisdom_type" \
  --source "promoted_from_learning" \
  --evidence "Auto-promoted after $observation_count observations (confidence: $lp_confidence)"
```

Source: lines 5393-5399 of aether-utils.sh

So for a pattern-type promotion with content "Use explicit jq if/elif chains...", the instinct will have:
- trigger: `"When working on pattern patterns"`
- action: `"Use explicit jq if/elif chains..."`
- domain: `"pattern"`
- source: `"promoted_from_learning"`

## State of the Art

| Current State | After Phase 5 | Impact |
|---------------|---------------|--------|
| 25 pipeline tests exist but all use synthetic data | New tests use realistic, non-synthetic observation content | LRNG-01: Validates pipeline works with real-world data |
| Existing tests verify individual pipeline segments | New end-to-end test covers memory-capture through colony-prime output | LRNG-03: Proves full chain works in single test |
| Instinct visibility in colony-prime is tested separately from learning pipeline | New test verifies learning-promoted instincts appear in colony-prime | LRNG-02: Closes the loop between learning and worker influence |
| No test verifies agent definitions acknowledge instincts | New test verifies pheromone_protocol exists in agent definitions | LRNG-02: Confirms influence mechanism is in place |
| learning-observations.json is empty (purged in Phase 1) | Remains empty after tests (tests use temp directories) | Clean state maintained |

## Implementation Strategy

### Two Natural Plans

**Plan 05-01: End-to-End Pipeline Validation Tests (LRNG-01, LRNG-03)**

Write integration tests in a new test file (e.g., `tests/integration/learning-pipeline-e2e.test.js`) that:
1. Exercise `memory-capture` with realistic content (not synthetic "test observation" strings)
2. Verify the full chain: memory-capture -> learning-observe -> threshold met -> learning-promote-auto -> queen-promote + instinct-create
3. Use different wisdom types: pattern (auto=2), failure (auto=2), philosophy (auto=3)
4. Verify edge cases: content with special characters, idempotency (calling past threshold)
5. Verify both artifacts: QUEEN.md has the promoted wisdom AND COLONY_STATE.json has the instinct

Estimated: 6-8 tests

**Plan 05-02: Instinct Influence and colony-prime Verification (LRNG-02)**

Write tests that verify promoted instincts reach worker prompts:
1. Exercise memory-capture until promotion, then call colony-prime, verify instinct in prompt_section
2. Verify instinct format in prompt_section (domain grouping, trigger/action text, confidence display)
3. Verify agent definitions contain pheromone_protocol sections (builder, watcher, scout)
4. Verify pheromone_protocol references instinct/learned behavior concepts
5. Test that colony-prime correctly assembles QUEEN wisdom + instincts in a single prompt_section

Estimated: 5-7 tests

### Why This Split

Plan 05-01 focuses on the data pipeline (observation -> promotion -> instinct creation). Plan 05-02 focuses on the visibility pipeline (instinct -> colony-prime -> prompt_section -> agent protocol). This mirrors how the requirements are naturally grouped: LRNG-01 and LRNG-03 are about the pipeline mechanics, LRNG-02 is about influence on workers.

## Open Questions

1. **What constitutes "non-synthetic" data for LRNG-01?**
   - What we know: The requirement says "real (non-test) data" and "non-synthetic data." Existing tests use strings like "Test observation for pipeline verification" and "Duplicate content test."
   - What's unclear: Whether "real data" means data from actual colony operation, or just realistic-looking content that represents genuine patterns.
   - Recommendation: Use content that represents actual patterns discovered during Aether development (e.g., the jq boolean handling fix from Phase 3, the test data pollution issue from Phase 1). This satisfies "non-synthetic" without requiring a live colony session. The tests still run in temp directories for isolation.

2. **Should Phase 5 tests run against the real colony state or temp directories?**
   - What we know: All existing integration tests use temp directories (createTempDir + setupTestColony). The real colony is IDLE with empty observations and zero instincts.
   - What's unclear: Whether LRNG-01's "real data" implies testing against the actual `.aether/data/` files.
   - Recommendation: Continue using temp directories for test isolation. "Real data" refers to the content being realistic, not the test environment being production. Running tests against actual state would be fragile and leave artifacts.

3. **The auto-generated instinct trigger format is generic**
   - What we know: `learning-promote-auto` creates instincts with trigger `"When working on {wisdom_type} patterns"` -- this is generic and may not be ideal.
   - What's unclear: Whether this generic trigger is considered a problem for Phase 5.
   - Recommendation: Out of scope. Phase 5 validates the pipeline works as-is. Improving the trigger quality is a future enhancement. The current trigger format is functional and verifiable.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 5151-5287 -- `learning-observe` subcommand (observation recording with SHA256 dedup)
- `.aether/aether-utils.sh` lines 5336-5413 -- `learning-promote-auto` subcommand (threshold check + queen-promote + instinct-create)
- `.aether/aether-utils.sh` lines 5414-5516 -- `memory-capture` subcommand (entry point, calls learning-observe + learning-promote-auto + pheromone-write)
- `.aether/aether-utils.sh` lines 7264-7379 -- `instinct-create` subcommand (dedup, confidence boost, 30-instinct cap)
- `.aether/aether-utils.sh` lines 7381-7558 -- `pheromone-prime` subcommand (instinct formatting with domain grouping)
- `.aether/aether-utils.sh` lines 7560-7964 -- `colony-prime` subcommand (unified prompt assembly)
- `.aether/aether-utils.sh` lines 938-972 -- `get_wisdom_threshold` and `get_wisdom_thresholds_json` (threshold table)
- `tests/integration/learning-pipeline.test.js` -- 9 existing pipeline tests (verified structure and patterns)
- `tests/integration/instinct-pipeline.test.js` -- 8 existing instinct tests (verified structure)
- `tests/integration/wisdom-promotion.test.js` -- 8 existing wisdom promotion tests (verified parseLastJson pattern)
- `.claude/agents/ant/aether-builder.md` -- pheromone_protocol section at lines 76-110 (added in Phase 4)
- `.claude/agents/ant/aether-watcher.md` -- pheromone_protocol section (added in Phase 4)
- `.claude/agents/ant/aether-scout.md` -- pheromone_protocol section (added in Phase 4)

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` -- LRNG-01, LRNG-02, LRNG-03 definitions
- `.planning/ROADMAP.md` -- Phase 5 description and success criteria
- `.planning/phases/04-pheromone-worker-integration/04-RESEARCH.md` -- Phase 4 research establishing pheromone_protocol pattern

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all subcommands verified by direct code reading, line numbers documented
- Architecture: HIGH -- complete pipeline flow mapped from source, thresholds verified, JSON output formats documented
- Pitfalls: HIGH -- identified from actual code analysis (multi-line JSON, SHA256 dedup sensitivity, threshold confusion, setupTestColony requirements)
- Implementation strategy: HIGH -- clear test patterns established by 8 prior integration test files; the code under test is stable and well-understood

**Research date:** 2026-03-19
**Valid until:** 2026-04-19 (30 days -- stable system, no external dependencies, all components already built)
