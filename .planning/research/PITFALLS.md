# Domain Pitfalls

**Domain:** Multi-agent colony orchestration system maintenance (pheromone integration, cleanup, fresh-install polish)
**Researched:** 2026-03-19
**Overall Confidence:** HIGH (grounded in codebase audit + multi-agent failure research)

---

## Critical Pitfalls

Mistakes that cause rewrites, broken colonies, or cascading test failures.

### Pitfall 1: Test Data Cleanup Corrupts the Canonical State Templates

**What goes wrong:** Cleaning test artifacts from QUEEN.md, pheromones.json, and constraints.json seems like a simple find-and-delete, but the line between "test data" and "template data" is blurry. The QUEEN.md currently has legitimate entries (`colony-a` through `colony-e`) mixed with 25+ junk `test-colony` entries. Deleting too aggressively removes the seed data that new colonies need. Deleting too conservatively leaves test artifacts that pollute every future colony.

**Why it happens:** Tests wrote directly to the canonical `.aether/` files instead of isolated temp directories. The integration tests (pheromone-auto-emission.test.js, suggest-pheromones.test.js) correctly use temp dirs, but earlier development testing wrote to the real files. There is no flag distinguishing "seed data" from "test residue."

**Consequences:**
- If seed entries are removed: `colony-prime` produces empty wisdom sections; new colonies lack starter philosophies
- If test entries remain: every colony inherits "Test pattern" and "Immediate decree test" entries, making the system look broken to new users
- COLONY_STATE.json contains a goal from a completely different project (Electron-to-Xcode migration) which will confuse fresh sessions

**Prevention:**
1. Before cleanup, document which QUEEN.md entries are canonical seeds (colony-a through colony-e) vs test residue (everything with "test-colony" source)
2. Create a "golden state" snapshot: pristine QUEEN.md, empty pheromones.json, clean constraints.json -- commit as the known-good baseline
3. Add a CI check: `npm run lint:state` that validates state files against expected schemas and rejects entries with "test" in source/colony fields
4. Ensure all future tests use `AETHER_ROOT=$tmpDir` isolation (the existing integration tests already do this correctly)

**Detection:**
- Run `grep -c "test-colony\|test.*pattern\|Immediate decree" .aether/QUEEN.md` -- count > 0 means test pollution remains
- Check if COLONY_STATE.json goal mentions "Electron" or "Xcode" -- stale cross-project state
- constraints.json focus array containing "test area" or "sanity signal" entries

**Phase mapping:** This should be the FIRST thing addressed in Phase 1 (cleanup). Everything downstream depends on clean state.

---

### Pitfall 2: Pheromone Integration Creates a "Write-Only" Signal System

**What goes wrong:** The pheromone system has complete write infrastructure (pheromone-write, pheromone-emit, suggest-analyze, suggest-approve) and complete read infrastructure (pheromone-read, pheromone-display, colony-prime). The gap is behavioral: workers receive pheromone signals via `colony-prime --compact` in the build-context playbook, but no worker agent definition actually references or acts on these signals. The builder agent definition does not mention pheromones at all. Signals are injected into prompts but there is no enforcement, feedback loop, or verification that workers changed behavior based on signals.

**Why it happens:** This is the classic "event-driven architecture without consumers" anti-pattern. The plumbing was built bottom-up (storage, emission, display) without top-down validation (do workers actually read and respond to these signals?). Research on multi-agent system failures shows that 79% of coordination failures originate from specification issues, not technical implementation -- the specs (agent definitions) never required pheromone consumption.

**Consequences:**
- Users emit FOCUS/REDIRECT/FEEDBACK signals expecting behavior change, but workers ignore them
- The suggest-analyze system auto-creates signals that nobody reads, creating signal noise
- Pheromones accumulate without effect, training users that the feature is decorative
- No feedback loop means signal quality cannot improve over time

**Prevention:**
1. Integration must be top-down: modify worker agent definitions (especially aether-builder.md, aether-watcher.md) to explicitly reference and act on pheromone signals
2. Add a "signal acknowledgment" mechanism: workers report which signals influenced their decisions
3. Start with REDIRECT (highest priority, simplest behavior: "avoid X") before FOCUS or FEEDBACK
4. Add integration test: emit REDIRECT signal, run build, verify worker output references the constraint
5. Validate pheromone consumption in the build-verify playbook: check worker outputs for signal acknowledgment

**Detection:**
- Search agent definitions for "pheromone": zero matches in aether-builder.md = workers are signal-blind
- Run a build with active REDIRECT signals and check if worker output mentions the constraint
- Count ratio of pheromone-write calls to pheromone-read calls across all playbooks

**Phase mapping:** This is the CORE work of Phase 2 or 3 (pheromone integration). Should not be attempted until cleanup is complete.

---

### Pitfall 3: Breaking 490+ Tests by Changing State Schemas or Utility Signatures

**What goes wrong:** The 10,249-line aether-utils.sh has 150 subcommands. Changing the JSON structure of pheromones.json, modifying pheromone-write arguments, or altering colony-prime output format breaks tests across multiple test suites (unit, integration, bash) simultaneously. The tests are spread across 47+ files, and a single schema change can create a "red wall" of failures that is demoralizing and hard to diagnose.

**Why it happens:** The monolithic utility file means changes have a blast radius equal to the entire system. There is no module boundary between pheromone logic, state management, wisdom promotion, and spawn management. A change to how signals are structured in pheromones.json can break colony-prime, which breaks build-context, which breaks integration tests.

**Consequences:**
- Developer changes one thing, sees 50 test failures, panics, reverts everything
- "Fix forward" attempts create more breakage as fixes cascade
- Maintenance paralysis: nobody wants to touch the utility file because everything depends on it

**Prevention:**
1. Before ANY schema change, run full test suite and record baseline: `npm test 2>&1 | tail -5` to capture pass/fail count
2. Make schema changes additive-only: new fields yes, renamed/removed fields no. If pheromones.json needs new structure, add fields alongside old ones
3. Use feature flags for new behavior: `if [[ -n "${AETHER_V2_SIGNALS:-}" ]]; then` -- allows gradual rollout
4. Write the new tests FIRST (TDD), then modify the implementation to pass them
5. Run `npm test` after EVERY atomic change, not after a batch of changes

**Detection:**
- Before starting work: `npm test` baseline must be green (490+ passing)
- After each file save: run affected test subset
- If more than 5 tests break from one change, the change is too broad -- split it

**Phase mapping:** Every phase must run full test suite as exit gate. The cleanup phase is especially risky because it touches state files that tests may implicitly depend on.

---

### Pitfall 4: XML Exchange System Removal Creates Dead Code References

**What goes wrong:** The XML exchange system (.aether/exchange/, .aether/schemas/) has 6 files and ~500 lines of shell code that are completely unwired from any command or playbook. No command references pheromone-export-xml, pheromone-import-xml, wisdom-xml, or registry-xml. However, the aether-utils.sh help listing advertises these commands, and existing tests (tests/bash/test-pheromone-xml.sh, tests/bash/test-xinclude-composition.sh) exercise them. Archiving or removing the XML system without updating ALL references creates a broken help system and failing tests.

**Why it happens:** The XML system was built as a complete feature but never integrated into the actual colony lifecycle (pause/resume/seal commands don't call it). It exists as a parallel, unused layer. The CONCERNS.md audit correctly identified this as an integration gap, and the PROJECT.md marks it for archival. But "archive" is ambiguous -- does it mean delete, move to an archive directory, or comment out?

**Consequences:**
- If deleted: test-pheromone-xml.sh and test-xinclude-composition.sh fail; help listing references nonexistent commands
- If kept: security concern (XXE vulnerability in XInclude processing without circular reference detection) remains in the codebase
- If partially removed: dead references create confusing error messages

**Prevention:**
1. Define "archive" precisely: move .aether/exchange/ and .aether/schemas/ to .aether/archive/exchange/ and .aether/archive/schemas/
2. Move corresponding tests to a disabled/archive state (rename to .skip or move to tests/archived/)
3. Remove XML-related entries from the aether-utils.sh help command listing
4. Keep the archive accessible for future reference but out of the active codebase
5. Update validate-package.sh if it checks for exchange/ directory presence

**Detection:**
- After archival: `grep -r "pheromone-xml\|wisdom-xml\|registry-xml\|exchange/" .aether/aether-utils.sh` should return only archive references
- Run `npm test` and `npm run test:bash` immediately after
- Check `npm pack --dry-run` to verify archive is excluded from distribution

**Phase mapping:** Phase 1 (cleanup) -- but must be done AFTER test data cleanup and BEFORE pheromone integration, so the integration work builds on a clean foundation.

---

## Moderate Pitfalls

### Pitfall 5: Pheromone Decay Math Has No Tests and Uses Platform-Dependent Date Arithmetic

**What goes wrong:** Pheromone signals have TTL-based expiration using epoch timestamps. The expiration calculation in pheromone-write (lines 6870-6889) uses `date -r` (macOS) with fallback to `date -d` (Linux). The decay display in pheromone-display computes strength decay over time. Neither path has dedicated tests, and the date arithmetic differs between macOS and Linux.

**Why it happens:** Bash date handling is notoriously inconsistent across platforms. macOS uses BSD date, Linux uses GNU date. The code has a fallback chain but the fallback itself is untested.

**Prevention:**
1. Write explicit decay math tests with known timestamps: create signal at epoch X, check decay at epoch X+3600 (1 hour), X+86400 (1 day)
2. Test on both macOS and Linux (CI should cover this)
3. Consider using a date wrapper function that normalizes behavior across platforms
4. Test edge cases: signals created before DST transition, signals with "phase_end" TTL

**Detection:**
- Run `bash .aether/aether-utils.sh pheromone-display` and check if decay percentages make sense
- Create a signal with `--ttl 1h`, wait 30 minutes, verify strength is ~50%

**Phase mapping:** Phase 2 (pheromone integration) -- decay math must work before signals can be trusted.

---

### Pitfall 6: Fresh Install Breaks Because Global Hub Assumes Existing State

**What goes wrong:** `npm install -g aether-colony` runs postinstall which calls setupHub() to populate `~/.aether/`. If the user then runs `aether update` in a new repo, it copies files from the hub. But the hub's QUEEN.md may contain test data from the developer's environment (if the package was built from a polluted state), and the hub's version may conflict with what `aether init` expects.

**Why it happens:** The distribution pipeline copies `.aether/` contents to the npm package. If `.aether/QUEEN.md` has test pollution when `npm publish` runs, that pollution ships to every user.

**Prevention:**
1. Add a pre-publish validation step to validate-package.sh that checks QUEEN.md for test artifacts
2. Create a "clean" QUEEN.md template at `.aether/templates/QUEEN.md.template` that is used for distribution instead of the working QUEEN.md
3. Test the full install flow: `npm pack`, install the tarball in a fresh directory, run `aether init`, verify clean state

**Detection:**
- `npm pack --dry-run | grep QUEEN.md` -- if QUEEN.md is included, inspect it for test data
- After install: check `~/.aether/QUEEN.md` for "test-colony" entries

**Phase mapping:** Phase 3 or 4 (fresh-install polish) -- but cleanup in Phase 1 prevents this if done correctly.

---

### Pitfall 7: Signal Accumulation Without Garbage Collection Creates Noise

**What goes wrong:** pheromones.json currently has 9 signals, several marked `active: false` with `expires_at: "phase_end"`. But no phase has advanced (current_phase is 0), so "phase_end" never triggers. Expired signals accumulate indefinitely. The pheromone-display command shows all signals including expired ones, creating visual noise that drowns out real signals.

**Why it happens:** The pheromone-expire subcommand exists but is never called automatically. There is no cron, no lifecycle hook, and no garbage collection in the build pipeline. The build-context playbook reads signals but does not prune expired ones.

**Prevention:**
1. Add `pheromone-expire` call to the build-prep playbook (run before loading signals)
2. Add `pheromone-expire` call to the continue playbook (clean up after phase completion)
3. Implement a maximum signal count (e.g., 20 active signals) with oldest-first eviction
4. pheromone-display should filter expired signals by default (add `--all` flag for full view)

**Detection:**
- `jq '[.signals[] | select(.active == false)] | length' .aether/data/pheromones.json` -- count of inactive signals
- If inactive count exceeds active count, garbage collection is needed

**Phase mapping:** Phase 2 (pheromone integration) -- must be solved alongside signal reading.

---

### Pitfall 8: Stale COLONY_STATE.json From Wrong Project Causes Confusing UX

**What goes wrong:** The current COLONY_STATE.json has goal "Ensure the electron version of the app..." from a completely different project. A new user running `/ant:status` or `/ant:resume` sees this stale goal and is confused. The session recovery check in CLAUDE.md says to display "Previous colony session detected: {goal}" which would show the wrong project's goal.

**Why it happens:** Colony state is per-repo but not validated against the repo it belongs to. There is no repo fingerprint or project identifier in the state file. Any colony initialized in this repo leaves state that persists across unrelated sessions.

**Prevention:**
1. Add repo fingerprint to COLONY_STATE.json (e.g., first 8 chars of repo origin URL hash)
2. On `/ant:init`, validate that existing state matches current repo before offering resume
3. For the immediate cleanup: reset COLONY_STATE.json to a clean template or delete it entirely
4. Consider adding a `stale_after` timestamp that auto-invalidates state after 7 days of no activity

**Detection:**
- Check if COLONY_STATE.json goal mentions anything unrelated to current project
- Check `initialized_at` -- if months old and no phases completed, it is stale

**Phase mapping:** Phase 1 (cleanup) -- must be resolved before any colony operations make sense.

---

### Pitfall 9: Concurrent Agent Modifications During Build Create State Races

**What goes wrong:** During `/ant:build`, multiple workers are spawned in parallel waves. If two workers both try to emit pheromones (via memory-capture which calls pheromone-write), they race on the pheromones.json file lock. The lock system has a 50-second timeout with 100 retries at 500ms intervals, but no exponential backoff and no jitter. Under parallel worker load, this can cause lock contention, duplicate signals, or worker timeout.

**Why it happens:** The file-based locking system was designed for single-threaded command execution, not parallel worker builds. The CONCERNS.md identified this: "aggressive polling could waste CPU on slow filesystems" and "if colony has 10+ concurrent commands, some will timeout."

**Prevention:**
1. Make pheromone emission non-blocking: workers write to per-worker signal files, build-verify merges them
2. Add jitter to lock retry: `sleep $(( RANDOM % 500 ))ms` instead of fixed 500ms
3. Limit pheromone-write calls during build waves (batch signals, emit once after wave completes)
4. Test with 3+ parallel pheromone-write calls to verify lock correctness

**Detection:**
- During builds: watch for "Failed to acquire lock on pheromones.json" in worker output
- Check for duplicate signal IDs in pheromones.json after a multi-worker build
- Monitor lock file age: `ls -la /tmp/.aether-*.lock` during builds

**Phase mapping:** Phase 2 (pheromone integration) -- must be addressed when making pheromones auto-emit during builds.

---

## Minor Pitfalls

### Pitfall 10: Agent Definition Updates Break OpenCode Parity

**What goes wrong:** Modifying Claude agent definitions in `.claude/agents/ant/` to add pheromone awareness requires mirroring changes to `.aether/agents-claude/` (packaging mirror) and `.opencode/agents/` (structural parity). Missing any of these three locations creates drift. The `lint:sync` check verifies count parity but not content parity.

**Prevention:**
1. Always edit `.claude/agents/ant/` first, then copy to `.aether/agents-claude/`
2. Update `.opencode/agents/` with equivalent structural changes
3. Run `npm run lint:sync` after every agent definition change
4. Consider adding content hash comparison to lint:sync

**Phase mapping:** Phase 2 (pheromone integration) -- every agent definition change must update all three locations.

---

### Pitfall 11: Documentation Updates Create Inconsistency Between CLAUDE.md and Actual Behavior

**What goes wrong:** CLAUDE.md describes the pheromone system as "signals guide colony behavior" but the actual behavior is "signals are stored and displayed." Updating docs to match new pheromone integration before the integration is complete creates a different kind of inconsistency: docs promise features that do not work yet.

**Prevention:**
1. Update documentation AFTER integration is verified, not during
2. Use "[PLANNED]" markers for features in progress
3. Test documentation claims by following them literally: does doing what the docs say produce the expected result?

**Phase mapping:** Phase 3 or 4 (documentation update) -- must happen after pheromone integration is verified.

---

### Pitfall 12: Constraints.json Has Stale XML-Era Constraints

**What goes wrong:** constraints.json contains AVOID constraints from the XML migration era: "Breaking changes to existing JSON files" and "Compromised hybrid approaches that limit XML capabilities." These constraints are no longer relevant since the XML system is being archived. If pheromone integration reads these constraints, it will apply outdated guidance.

**Prevention:**
1. Review all constraints.json entries during cleanup phase
2. Remove or mark constraints that reference archived features
3. Add `archived_at` field to constraints for soft deletion

**Phase mapping:** Phase 1 (cleanup) -- alongside QUEEN.md and pheromones.json cleanup.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| State cleanup (Phase 1) | Removing seed data along with test data (Pitfall 1) | Document canonical entries before deleting; create golden snapshot |
| State cleanup (Phase 1) | Stale COLONY_STATE.json confuses subsequent phases (Pitfall 8) | Reset to clean state or delete entirely |
| State cleanup (Phase 1) | XML archival leaves dead references (Pitfall 4) | Audit all references in utils, tests, help text, package validation |
| State cleanup (Phase 1) | Constraints.json contains stale XML-era rules (Pitfall 12) | Review all entries, remove/archive irrelevant ones |
| Pheromone integration (Phase 2) | Write-only signal system (Pitfall 2) | Start from agent definitions, not from plumbing |
| Pheromone integration (Phase 2) | Decay math untested (Pitfall 5) | Write decay tests before relying on decay for behavior |
| Pheromone integration (Phase 2) | Signal accumulation noise (Pitfall 7) | Add expire calls to build lifecycle |
| Pheromone integration (Phase 2) | Lock contention during parallel builds (Pitfall 9) | Batch signal writes, add jitter to retries |
| Pheromone integration (Phase 2) | Agent definition drift across 3 mirrors (Pitfall 10) | Update all 3 locations, run lint:sync |
| Documentation update (Phase 3) | Docs promise features not yet working (Pitfall 11) | Update docs only after integration verified |
| Fresh install (Phase 4) | Polluted state ships in npm package (Pitfall 6) | Validate package contents in pre-publish step |
| All phases | Test regression from schema changes (Pitfall 3) | Run full test suite after every atomic change |

---

## Sources

### Codebase Evidence (HIGH confidence)
- `.planning/codebase/CONCERNS.md` -- 390-line audit identifying all technical debt, security issues, test gaps
- `.aether/data/pheromones.json` -- 9 signals, most test artifacts, confirms signal accumulation
- `.aether/data/COLONY_STATE.json` -- stale goal from different project, confirms cross-project pollution
- `.aether/QUEEN.md` -- 25+ test entries mixed with 5 seed entries, confirms test data pollution
- `.aether/data/constraints.json` -- XML-era constraints still active, confirms stale guidance
- `.claude/agents/ant/aether-builder.md` -- zero pheromone references, confirms workers are signal-blind
- `.aether/exchange/` -- 6 files, zero references from commands, confirms XML system is unwired
- `tests/integration/pheromone-auto-emission.test.js` -- uses temp dirs correctly, shows good test isolation pattern

### Research (MEDIUM confidence)
- [Why Your Multi-Agent System is Failing: 17x Error Trap](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) -- error amplification in multi-agent architectures without coordination topology
- [Multi-Agent System Reliability: Failure Patterns](https://www.getmaxim.ai/articles/multi-agent-system-reliability-failure-patterns-root-causes-and-production-validation-strategies/) -- state synchronization breakdowns, communication protocol failures, 79% of failures from specification issues
- [Why Do Multi-Agent LLM Systems Fail?](https://arxiv.org/html/2503.13657v1) -- research on specification ambiguities and coordination gaps as primary failure source
- [Event-Driven Architecture: The Hard Parts](https://threedots.tech/episode/event-driven-architecture/) -- complexity in debugging, eventual consistency, schema evolution
- [Digital Pheromones: Agent Coordination](https://www.distributedthoughts.org/digital-pheromones-what-ants-know-about-agent-coordination/) -- pheromone paradigm for async agent coordination, environment-as-channel pattern

### Research (LOW confidence -- verify before acting)
- [Event-Driven Architecture: 5 Pitfalls to Avoid](https://medium.com/wix-engineering/event-driven-architecture-5-pitfalls-to-avoid-b3ebf885bdb1) -- context propagation without correlation IDs (applicable to signal tracing); could not fetch full content
- [Common Pitfalls in Event-Driven Architectures](https://medium.com/insiderengineering/common-pitfalls-in-event-driven-architectures-de84ad8f7f25) -- duplicate events, unclear ownership, brittle integrations; could not fetch full content

---

*Pitfalls analysis: 2026-03-19*
