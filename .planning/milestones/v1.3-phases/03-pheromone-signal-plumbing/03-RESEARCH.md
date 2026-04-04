# Phase 3: Pheromone Signal Plumbing - Research

**Researched:** 2026-03-19
**Domain:** Pheromone signal injection chain, lifecycle management, decay math, session persistence
**Confidence:** HIGH

## Summary

Phase 3 addresses four requirements: verifying the end-to-end signal injection chain (PHER-01), fixing signal lifecycle and expiration (PHER-02), testing decay math with known timestamps (PHER-06), and ensuring pheromones survive `/clear` and are available on `/ant:resume` (PHER-07).

The architecture research confirmed: "The signal propagation pipeline is fundamentally sound. The gap is not architectural -- it is in initialization and lifecycle completeness." My deep-dive into the code confirms this assessment but reveals several specific bugs and inconsistencies that must be fixed:

1. **Two different epoch conversion functions** (`to_epoch` vs `approx_epoch`) produce different values for the same timestamp, causing inconsistent decay/expiry behavior across subcommands
2. **`/ant:resume` reads `constraints.json` for pheromone signals** instead of `pheromones.json`, meaning session recovery shows legacy data rather than actual signals (PHER-07 blocker)
3. **Existing signals lack the `strength` field** (2 of 3 real signals use an older format), so decay math silently falls back to 0.8
4. **No dedicated tests exist for decay math** -- zero test files match "decay" or "signal" patterns

**Primary recommendation:** Fix the concrete bugs (dual epoch functions, resume reading wrong file, old-format signals) and write focused tests for decay math edge cases. The plumbing itself works -- this is verification, consistency, and test coverage work.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PHER-01 | Signal injection chain verified end-to-end (user emits signal -> pheromone-write -> colony-prime -> prompt_section -> worker receives in spawn context) | The chain is confirmed WORKING via code trace: focus.md calls pheromone-write, colony-prime calls pheromone-prime which reads pheromones.json, build-context.md calls colony-prime, build-wave.md injects prompt_section into worker prompts. Verification test needed to exercise the full path. |
| PHER-02 | Signal lifecycle works correctly (expiration at phase_end, time-based decay, garbage collection of expired signals) | pheromone-expire exists with --phase-end-only mode; continue-advance.md Step 2.1e calls it. Time-based expiry done at read-time in pheromone-read. BUG: two inconsistent epoch functions. Garbage collection only runs during /ant:continue, not during builds. |
| PHER-06 | Pheromone decay math tested with known timestamps and edge cases | Zero tests exist for decay math. The `to_epoch` jq function uses approximate month/year constants (365 days/year, 30 days/month) which are deliberately approximate but must be tested for consistency. Edge cases: zero elapsed time, exactly at expiry boundary, past expiry, missing strength field, phase_end expires_at. |
| PHER-07 | Pheromones persist and carry across sessions (survive /clear, available on /ant:resume) | CRITICAL BUG: `/ant:resume` Step 3 reads `constraints.json`, NOT `pheromones.json`. It says "Pheromones persist until explicitly cleared -- no decay." This is wrong on two counts: (1) wrong file, (2) signals DO decay. pheromones.json is a file on disk that survives /clear, but resume does not read it. Fix: update resume.md to read pheromones.json via pheromone-display or pheromone-read. |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| bash + jq | system | Signal storage, lifecycle, decay calculation | Already used throughout aether-utils.sh; all pheromone subcommands are bash |
| AVA | ^6.4.1 | JavaScript integration tests | Already installed; used by existing pheromone-auto-emission tests |
| Node.js child_process | built-in | Running aether-utils.sh from JS tests | Pattern established in pheromone-auto-emission.test.js |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Bash test helpers | tests/bash/test-helpers.sh | Shell-level assertions for decay math tests | Testing pheromone-read, pheromone-expire, pheromone-display decay output |
| temp directory isolation | os.tmpdir() | Isolated test environments | All integration tests must use AETHER_ROOT=$tmpDir pattern (already established) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| jq epoch math | date command for epoch conversion | jq is self-contained within the pipeline; date command differs between macOS/Linux. Keep jq but unify the function. |
| Bash decay tests | JS-only tests | Bash tests can directly verify subcommand output; JS tests test the integration path. Use both. |

**Installation:**
```bash
# No new dependencies needed - everything exists
npm test  # Verify baseline before starting
```

## Architecture Patterns

### Recommended Project Structure
```
.aether/
  aether-utils.sh              # Pheromone subcommands (lines 6786-8120)
    pheromone-write  (6786)     # Create signals -> pheromones.json
    pheromone-count  (6971)     # Count active by type
    pheromone-display (6995)    # Human-readable table with decay
    pheromone-read   (7111)     # Read active with decay calculation
    pheromone-prime  (7381)     # Format for prompt injection
    colony-prime     (7560)     # Unified aggregation: wisdom + signals + instincts
    pheromone-expire (7966)     # Archive expired to midden

  data/
    pheromones.json             # SOURCE OF TRUTH for signals
    constraints.json            # DEPRECATED backward-compat (written by pheromone-write)
    COLONY_STATE.json           # Instincts (separate from signals)
    session.json                # Session state (survives /clear)
    midden/midden.json          # Archived expired signals

  docs/command-playbooks/
    build-context.md            # Step 4: calls colony-prime --compact
    build-wave.md               # Step 5.1: injects prompt_section into worker prompts
    continue-advance.md         # Step 2.1e: calls pheromone-expire --phase-end-only

.claude/commands/ant/
    focus.md                    # /ant:focus -> pheromone-write FOCUS
    redirect.md                 # /ant:redirect -> pheromone-write REDIRECT
    feedback.md                 # /ant:feedback -> pheromone-write FEEDBACK
    resume.md                   # /ant:resume -> READS constraints.json (BUG)
    pheromones.md               # /ant:pheromones -> pheromone-display

tests/integration/
    pheromone-auto-emission.test.js  # Existing: auto-emission chain tests
    (NEW) pheromone-lifecycle.test.js # Signal lifecycle, decay, session persistence
```

### Pattern 1: Signal Injection Chain (End-to-End)
**What:** User emits signal -> signal stored -> signal compiled into prompt -> worker receives it
**When to use:** Every build cycle
**Verified path:**
```
/ant:focus "area"
  -> focus.md calls: pheromone-write FOCUS "area" --strength 0.8
    -> pheromones.json: signal appended with {id, type, strength, content, created_at, expires_at}
  -> constraints.json: backward-compat write (FOCUS -> .focus[], REDIRECT -> .constraints[])

/ant:build N
  -> build-context.md Step 4: colony-prime --compact
    -> colony-prime calls: pheromone-prime --compact --max-signals 8 --max-instincts 3
      -> pheromone-prime reads pheromones.json
      -> filters: active==true AND effective_strength >= 0.1 AND not expired
      -> sorts: REDIRECT first, then by effective_strength desc
      -> takes top 8 signals
      -> formats as markdown: "FOCUS (Pay attention to):", "REDIRECT (HARD CONSTRAINTS):", etc.
    -> colony-prime combines: QUEEN wisdom + context capsule + phase learnings + signals
    -> result.prompt_section stored as variable

  -> build-wave.md Step 5.1: Before each wave, refreshes colony-prime --compact
    -> Injects { prompt_section } into Builder worker's Task tool prompt
    -> Worker sees signals as inline text, not structured API
```

### Pattern 2: Signal Decay Model (Linear)
**What:** Signals lose effective strength linearly over time
**Decay formula:** `effective_strength = original_strength * (1 - elapsed_days / decay_days)`
**Decay periods by type:**

| Signal Type | Decay Period | Half-Life | Inactive Threshold |
|-------------|-------------|-----------|-------------------|
| FOCUS | 30 days | 15 days | effective_strength < 0.1 |
| REDIRECT | 60 days | 30 days | effective_strength < 0.1 |
| FEEDBACK | 90 days | 45 days | effective_strength < 0.1 |

**Default strengths (when not specified):**

| Signal Type | Default Strength | Priority |
|-------------|-----------------|----------|
| REDIRECT | 0.9 | high |
| FOCUS | 0.8 | normal |
| FEEDBACK | 0.7 | low |

**When effective_strength < 0.1:** Signal treated as inactive on read. Not removed from file -- just filtered out.

### Pattern 3: Dual Expiration Mechanisms
**What:** Signals can expire two ways:
1. **phase_end:** expires_at == "phase_end" -> expired by pheromone-expire --phase-end-only (during /ant:continue)
2. **time-based:** expires_at == ISO-8601 timestamp -> filtered at read-time when past expiry OR when decay drops below 0.1

### Anti-Patterns to Avoid
- **Direct pheromones.json reading from worker prompts:** Workers receive signals through prompt_section injection, never by reading the file directly. The build-wave instruction to "check for new signals at natural breakpoints" is aspirational -- workers cannot reliably do this.
- **Writing to constraints.json as primary store:** constraints.json is a backward-compat write from pheromone-write. It is NOT the source of truth. Any code reading pheromone signals must read pheromones.json.
- **Using real date command for epoch in jq:** The to_epoch jq function is self-contained within jq pipelines. Using bash `date` for epoch conversion and passing it in works but introduces macOS/Linux differences. Keep epoch math inside jq where possible.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Epoch conversion in jq | Write another custom epoch function | Unify the existing `to_epoch` function and use it everywhere | There are already TWO variants causing bugs; adding a third makes it worse |
| Pheromone display formatting | Custom display logic | Use existing pheromone-display subcommand | Already handles decay calculation, filtering, and formatting |
| Signal write with locking | Direct jq write to file | Use pheromone-write subcommand | Handles locking, validation, sanitization, ID generation, backward-compat writes |
| Session persistence check | Custom file-exists checks | Use session-read subcommand | Already parses session.json with staleness detection |

**Key insight:** The pheromone subcommands are complete and well-structured. The work is in fixing bugs within them and writing verification tests, not building new infrastructure.

## Common Pitfalls

### Pitfall 1: Dual Epoch Functions Cause Inconsistent Expiry
**What goes wrong:** `pheromone-read`, `pheromone-display`, and `pheromone-prime` use `to_epoch()` which calculates `years * 365 * 86400 + months * 30 * 86400`. But `pheromone-expire` uses `approx_epoch()` which calculates `years * 31557600 + months * 2629800`. For the timestamp "2026-03-19T12:00:00Z", the two functions produce different epoch values. This means a signal that pheromone-read considers active could be expired by pheromone-expire, or vice versa.
**Why it happens:** The functions were written at different times with slightly different approximation constants.
**How to avoid:** Unify to a single epoch function. Use the `to_epoch` version (used by 4 of 5 consumers) and update pheromone-expire to use it too. Add a test that verifies both functions produce the same result for known timestamps.
**Warning signs:** A signal that appears active in pheromone-display but gets expired by pheromone-expire, or a signal that should have expired but remains visible.

### Pitfall 2: `/ant:resume` Reads constraints.json Instead of pheromones.json
**What goes wrong:** When a user runs `/ant:resume` after `/clear`, Step 3 reads `.aether/data/constraints.json` to display active signals. It states "Pheromones persist until explicitly cleared -- no decay." This is doubly wrong: (1) the canonical signal store is pheromones.json, not constraints.json; (2) signals DO decay. The user sees stale XML-era constraints instead of their actual pheromone signals.
**Why it happens:** resume.md was written before the pheromone system matured. constraints.json was the original signal store; pheromone-write still writes to it for backward compatibility.
**How to avoid:** Update resume.md Step 3 to call `pheromone-read` or `pheromone-display` instead of reading constraints.json directly.
**Warning signs:** After emitting FOCUS/REDIRECT/FEEDBACK signals, running `/clear` then `/ant:resume` shows XML-era constraints instead of the signals just emitted.

### Pitfall 3: Old-Format Signals Lack strength Field
**What goes wrong:** The 3 real signals in pheromones.json include 2 signals (sig_feedback_001, sig_redirect_001) in an older format with `tags` and `scope` fields but no `strength` or `reason` fields. The decay math uses `.strength // 0.8` as fallback, which works, but makes decay behavior implicit rather than explicit. The first signal also has no `expires_at` field at all.
**Why it happens:** The signal format evolved; older signals were not migrated.
**How to avoid:** As part of this phase, either migrate old-format signals to include the `strength` field explicitly, or accept the fallback and document it in pheromones.md. Also consider: the REDIRECT signal (sig_redirect_001) has `expires_at: "2026-03-16T08:00:00Z"` which is already past -- it should be showing as expired but may not be because the `active: true` flag is read before decay is checked in some code paths.
**Warning signs:** Signals appearing active in pheromone-count (which checks `.active == true` without decay) but inactive in pheromone-read (which applies decay).

### Pitfall 4: pheromone-count Does NOT Apply Decay
**What goes wrong:** `pheromone-count` (lines 6971-6993) simply counts signals where `.active == true` without computing decay. This means it can report a signal as active even when pheromone-read would filter it out due to decay below 0.1. The count is used by focus.md Step 3 to display "Active signals: N FOCUS, N REDIRECT, N FEEDBACK" after emitting a signal -- this count may be inflated.
**Why it happens:** pheromone-count was designed for quick reporting, not accuracy. Decay computation is expensive (requires epoch math in jq).
**How to avoid:** Either update pheromone-count to apply the same decay filter as pheromone-read, or document that it reports raw active count. For this phase, awareness is sufficient -- fixing this is a nice-to-have, not a blocker.

### Pitfall 5: Parallel Worker Lock Contention on pheromones.json
**What goes wrong:** During builds, multiple workers can call memory-capture which calls pheromone-write, which acquires a lock on pheromones.json. The lock retry is 100 attempts at 500ms intervals with no jitter. Under parallel load, this can cause workers to timeout waiting for the lock.
**Why it happens:** File-based locking was designed for single-threaded use.
**How to avoid:** This is out of scope for Phase 3 (no multi-worker tests needed here). But be aware when writing tests: run them with `test.serial()` if they share state files.

## Code Examples

Verified patterns from the codebase:

### Writing a Signal (pheromone-write)
```bash
# Source: .aether/aether-utils.sh lines 6786-6969
# From focus.md Step 2:
bash .aether/aether-utils.sh pheromone-write FOCUS "area" --strength 0.8 --reason "User directed colony attention" --ttl phase_end

# Returns JSON:
# {"ok":true,"result":{"signal_id":"sig_focus_1234567890_1234","type":"FOCUS","active_count":4}}
```

### Reading Signals with Decay (pheromone-read)
```bash
# Source: .aether/aether-utils.sh lines 7111-7190
bash .aether/aether-utils.sh pheromone-read all

# Returns JSON with effective_strength computed:
# {"ok":true,"result":{"version":"1.0.0","colony_id":"aether-dev","signals":[{...with effective_strength...}]}}
```

### The Decay Formula (jq)
```jq
# Source: .aether/aether-utils.sh lines 7020-7062 (pheromone-display)
# Also duplicated at: lines 7140-7180 (pheromone-read), lines 7412-7448 (pheromone-prime)

def to_epoch(ts):
  if ts == null or ts == "" or ts == "phase_end" then null
  else
    (ts | split("T")) as $parts |
    ($parts[0] | split("-")) as $d |
    ($parts[1] | rtrimstr("Z") | split(":")) as $t |
    (($d[0] | tonumber) - 1970) * 365 * 86400 +
    (($d[1] | tonumber) - 1) * 30 * 86400 +
    (($d[2] | tonumber) - 1) * 86400 +
    ($t[0] | tonumber) * 3600 +
    ($t[1] | tonumber) * 60 +
    ($t[2] | rtrimstr("Z") | tonumber)
  end;

def decay_days(t):
  if t == "FOCUS"    then 30
  elif t == "REDIRECT" then 60
  else 90
  end;

# For each signal:
(to_epoch(.created_at)) as $created_epoch |
(($now - $created_epoch) / 86400) as $elapsed_days |
(decay_days(.type)) as $dd |
((.strength // 0.8) * (1 - ($elapsed_days / $dd))) as $eff_raw |
(if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
# Signal is active if eff >= 0.1 AND not time-expired AND .active != false
```

### The DIFFERENT Epoch Function (pheromone-expire -- BUG)
```jq
# Source: .aether/aether-utils.sh lines 8024-8033 (pheromone-expire)
# NOTE: Uses DIFFERENT constants than to_epoch!

def approx_epoch(ts):
  # ... same parsing ...
  (($y - 1970) * 31557600) +    # 365.25 days/year vs 365 in to_epoch
  (($mo - 1) * 2629800) +       # 30.4375 days/month vs 30 in to_epoch
  (($day - 1) * 86400) + ($h * 3600) + ($m * 60) + $s;
```

### Integration Test Pattern (from existing tests)
```javascript
// Source: tests/integration/pheromone-auto-emission.test.js
// Key pattern: temp directory isolation with AETHER_ROOT override

async function setupTestColony(tmpDir, opts = {}) {
  const dataDir = path.join(tmpDir, '.aether', 'data');
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md (required by colony-prime)
  await fs.promises.writeFile(path.join(tmpDir, '.aether', 'QUEEN.md'), queenTemplate);

  // Create COLONY_STATE.json
  await fs.promises.writeFile(path.join(dataDir, 'COLONY_STATE.json'), JSON.stringify(colonyState));

  // Create pheromones.json with optional pre-seeded signals
  await fs.promises.writeFile(path.join(dataDir, 'pheromones.json'), JSON.stringify({
    signals: opts.pheromoneSignals || [],
    version: '1.0.0'
  }));
}

function runAetherUtil(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = { ...process.env, AETHER_ROOT: tmpDir, DATA_DIR: path.join(tmpDir, '.aether', 'data') };
  return execSync(`bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`, {
    encoding: 'utf8', env, cwd: tmpDir
  });
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| constraints.json as signal store | pheromones.json as source of truth | v1.0 era | pheromone-write still writes to both for backward compat |
| Signals have tags[], scope{} | Signals have strength, reason, expires_at | v1.1 era | 2 of 3 real signals lack the new fields |
| resume reads constraints.json | resume SHOULD read pheromones.json | NOT YET CHANGED | This is a bug that Phase 3 must fix |
| No decay math | Linear decay with type-specific periods | v1.1 era | Implemented but never tested with known timestamps |

**Deprecated/outdated:**
- `constraints.json` as pheromone store: Still written by pheromone-write for backward compat but should NOT be read as the source of truth. `/ant:resume` still reads it (bug).
- Signal format with `tags` and `scope` fields: Older format, not generated by current pheromone-write. Existing signals should be migrated or the fallback documented.

## Specific Findings Per Requirement

### PHER-01: End-to-End Injection Chain

**Status: WORKING but UNVERIFIED**

The complete chain has been traced through code:
1. `/ant:focus "X"` -> focus.md calls `pheromone-write FOCUS "X"` -- VERIFIED in focus.md
2. `pheromone-write` appends to pheromones.json with locking -- VERIFIED in aether-utils.sh:6786
3. `colony-prime --compact` calls `pheromone-prime --compact` which reads pheromones.json -- VERIFIED in aether-utils.sh:7560+7381
4. `build-context.md` Step 4 calls `colony-prime --compact` and stores `prompt_section` -- VERIFIED in build-context.md
5. `build-wave.md` Step 5.1 injects `{ prompt_section }` into builder worker prompts -- VERIFIED in build-wave.md

**What is needed:** An integration test that exercises the full path: write signal -> call colony-prime -> verify signal text appears in prompt_section output. The existing pheromone-auto-emission.test.js already tests steps 1-3 (see "auto:decision pheromones appear in colony-prime output" test). What is missing is a test specifically for the user-emitted signal path (source: "user") and verification that the signal appears with correct effective_strength in the prompt_section.

### PHER-02: Signal Lifecycle

**Status: PARTIALLY WORKING -- bugs found**

- **phase_end expiration:** pheromone-expire --phase-end-only works correctly. Called from continue-advance.md Step 2.1e. VERIFIED.
- **time-based decay:** Computed at read-time in pheromone-read, pheromone-display, and pheromone-prime. VERIFIED but UNTESTED.
- **Garbage collection:** Only runs during /ant:continue (pheromone-expire). During long builds without continue cycles, stale signals accumulate. No GC during build-context or build-wave.
- **BUG: Dual epoch functions:** pheromone-expire uses `approx_epoch` (365.25 days/year, 30.4375 days/month) while pheromone-read/display/prime use `to_epoch` (365 days/year, 30 days/month). For a signal created on 2026-01-01, the epoch difference between the two functions is approximately 5 hours by March 2026. This could cause a signal to be expired by pheromone-expire while still showing as active in pheromone-read.
- **BUG: Expired signal still marked active=true:** sig_redirect_001 has `expires_at: "2026-03-16T08:00:00Z"` which is 3 days past. pheromone-read would filter it out (via epoch comparison), but pheromone-count reports it as active because it only checks `.active == true` without decay.

### PHER-06: Decay Math Testing

**Status: NO TESTS EXIST**

No test file matches "decay" or "signal". The decay math is duplicated in 4+ locations within aether-utils.sh (pheromone-display:7020, pheromone-read:7140, pheromone-prime:7412, context-capsule:8887). Each copy uses the same `to_epoch` and `decay_days` functions but they are not shared -- they are copy-pasted.

**Edge cases that must be tested:**
1. Zero time elapsed (signal created now) -> effective_strength == original_strength
2. Half-life point (FOCUS at 15 days) -> effective_strength == 0.5 * original_strength
3. Exactly at full decay (FOCUS at 30 days) -> effective_strength == 0
4. Past full decay (FOCUS at 31 days) -> effective_strength == 0 (clamped)
5. Missing strength field -> fallback to 0.8
6. expires_at == "phase_end" -> never time-expired by decay, only by explicit pheromone-expire
7. expires_at is ISO timestamp in the past -> signal marked inactive
8. expires_at is ISO timestamp in the future -> signal remains active
9. Signal with `active: false` already set -> stays inactive regardless of decay

### PHER-07: Session Persistence

**Status: BUG -- resume reads wrong file**

pheromones.json IS a file on disk that survives `/clear` (context window reset). The file is in `.aether/data/` which is never cleared by any command. So pheromones DO persist across sessions.

However, `/ant:resume` (resume.md Step 3) reads `constraints.json` instead of `pheromones.json`:
```
### Step 3: Read Pheromone Signals
Use the Read tool to read `.aether/data/constraints.json`.
```

This means the user's actual pheromone signals (from pheromones.json) are NOT displayed on resume. Instead, they see stale constraints from the XML era.

**Fix:** Update resume.md Step 3 to:
1. Call `pheromone-read` or `pheromone-display` to get actual signals from pheromones.json
2. Remove the incorrect claim "Pheromones persist until explicitly cleared -- no decay"
3. Show decay-aware signal state (effective_strength, remaining days)

**Additional persistence paths to verify:**
- `/ant:status` -> Does it show signals from pheromones.json? (via pheromone-display -- needs verification)
- `/ant:resume-colony` -> Does the full colony resume show signals? (needs verification)
- `build-context.md` -> Yes, calls colony-prime which reads pheromones.json (verified)

## Open Questions

1. **Should pheromone-count apply decay?**
   - What we know: pheromone-count is used in focus.md/redirect.md/feedback.md Step 3 to display "Active signals: N FOCUS, N REDIRECT, N FEEDBACK" after emitting a signal. It reports raw .active==true counts without decay.
   - What is unclear: Is the inflated count a user-visible problem? In practice, most signals are fresh when emitted.
   - Recommendation: Document the inconsistency but do not fix in Phase 3 unless it causes test failures. Track as a known issue.

2. **Should old-format signals be migrated?**
   - What we know: 2 of 3 real signals use an older format (tags, scope, no strength). They work with fallback defaults.
   - What is unclear: Will these signals be confusing if they persist long-term?
   - Recommendation: Phase 1 cleaned test data. These 3 signals are "real" data. The sig_redirect_001 is expired (past expires_at). The sig_redirect third signal is inactive. Only sig_feedback_001 is genuinely active. Consider cleaning the expired ones as part of lifecycle verification.

3. **Should pheromone-expire also run during build-context.md?**
   - What we know: Currently GC only runs during /ant:continue. Long build sessions accumulate stale signals.
   - What is unclear: Would adding expire to build-context introduce timing issues or slow down builds?
   - Recommendation: Out of scope for Phase 3. If needed, track as Phase 4 enhancement.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 6786-8120 -- All pheromone subcommands traced: pheromone-write, pheromone-count, pheromone-display, pheromone-read, pheromone-prime, colony-prime, pheromone-expire
- `.aether/docs/command-playbooks/build-context.md` -- colony-prime integration verified
- `.aether/docs/command-playbooks/build-wave.md` -- prompt_section injection verified
- `.aether/docs/command-playbooks/continue-advance.md` -- pheromone-expire lifecycle verified
- `.claude/commands/ant/focus.md`, `redirect.md`, `feedback.md` -- signal emission commands verified
- `.claude/commands/ant/resume.md` -- BUG FOUND: reads constraints.json instead of pheromones.json
- `.aether/data/pheromones.json` -- 3 real signals, 2 in older format, 1 expired
- `.aether/data/constraints.json` -- stale XML-era constraints, still read by resume
- `tests/integration/pheromone-auto-emission.test.js` -- 9 existing tests covering auto-emission chain

### Secondary (MEDIUM confidence)
- `.planning/research/ARCHITECTURE.md` -- signal propagation analysis confirmed by code reading
- `.planning/research/PITFALLS.md` -- decay math and dual-store pitfalls confirmed by code reading

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies needed; existing tools are correct
- Architecture: HIGH -- code traced line-by-line through all pheromone subcommands
- Pitfalls: HIGH -- bugs found by direct code comparison (dual epoch functions, resume reading wrong file)
- Test patterns: HIGH -- existing integration test patterns well-established

**Research date:** 2026-03-19
**Valid until:** 2026-04-19 (stable domain -- bash subcommands unlikely to change without this phase)
