# Phase 40: Pheromone Propagation - Research

**Researched:** 2026-03-30
**Domain:** Bash shell scripting, git worktree integration, pheromone signal propagation across branches
**Confidence:** HIGH

## Summary

Phase 40 wires four already-implemented pheromone subcommands (`pheromone-snapshot-inject`, `pheromone-export-branch`, `pheromone-merge-back`, `pheromone-merge-log`) into the colony workflow at three integration points: worktree creation, seal, and post-merge. The subcommands are fully implemented with 43 passing tests. The design doc at `.aether/docs/pheromone-propagation-design.md` is thorough and matches the implementation closely. The primary work is integration wiring, not new code.

The three wiring points are: (1) calling `pheromone-snapshot-inject` inside `_worktree_create` in worktree.sh after the worktree directory is created and `.aether/data/` is copied, (2) calling `pheromone-export-branch` during the seal ceremony (near the existing XML export step), and (3) calling `pheromone-merge-back` as a post-merge step. Each wiring point has a clear insertion location in existing code.

The one significant technical challenge is that `.aether/data/` is gitignored (line 80 of `.gitignore`), so `pheromone-branch-export.json` must either be written outside `.aether/data/` or get a gitignore exception. The CONTEXT.md flags this as Claude's discretion.

**Primary recommendation:** This is an integration phase, not an implementation phase. Wire the three existing subcommands into their designated call sites, add a handful of integration tests for each wiring point, and solve the gitignore exception for the export file. Expect 2-3 small plans, not 5+.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Wire `pheromone-snapshot-inject` into `_worktree_create` in spawn.sh -- after worktree is created, run snapshot-inject with main HEAD SHA so agents start with correct signals
- **D-02:** Wire `pheromone-export-branch` into `/ant:seal` Step 3.7 -- before sealing, export branch signals as a tracked file for merge-back
- **D-03:** Wire `pheromone-merge-back` into `/ant:continue` or the post-merge flow -- after a PR merges to main, run merge-back to collect branch-discovered signals
- **D-04:** Do NOT wire auto-injection into `/ant:build` -- build should assume signals are already correct from worktree creation. Auto-injection at build time would mask setup bugs.
- **D-05:** `pheromone-branch-export.json` should be written to `.aether/data/` (not gitignored for this file specifically). The design doc says it should be git-tracked so it survives merge. This requires a targeted gitignore exclusion or writing it to a tracked location.
- **D-06:** Follow the design doc's conflict resolution priority: REDIRECT > FOCUS > FEEDBACK, with strength-based dedup. This is already implemented in the merge-back subcommand -- no changes needed to the algorithm, just verify it's tested.

### Claude's Discretion
- Exact placement of merge-back trigger (post-merge hook vs command vs manual)
- Whether to add `--auto-inject` flag to build for cases where worktree was created outside Aether
- How to handle the `.aether/data/` gitignore exception for the export file

### Deferred Ideas (OUT OF SCOPE)
- Contradiction detection (advisory) -- the design doc mentions heuristic keyword overlap detection for contradictory FOCUS signals. Nice-to-have, not blocking.
- Auto-injection at build time for non-Aether-created worktrees -- add as optional flag later if needed.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PHERO-01 | `pheromone-snapshot-inject` copies active signals from main into a new worktree branch during creation, ensuring REDIRECT and FOCUS signals reach all workers | Subcommand fully implemented (pheromone.sh:2030-2254), 43 tests passing. Wiring into `_worktree_create` in worktree.sh is the implementation task. |
| PHERO-02 | `pheromone-export-branch` exports branch-specific pheromone signals as a snapshot before PR submission | Subcommand fully implemented (pheromone.sh:2260-2388). Wiring into seal ceremony at Step 6.5 (near XML export). |
| PHERO-03 | `pheromone-merge-back` merges eligible branch signals (user-created, not auto-generated) into main after PR merge, with dedup and conflict resolution | Subcommand fully implemented (pheromone.sh:2394-2641). Conflict resolution algorithm verified in tests 6-8. Wiring into post-merge flow is the implementation task. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| bash | 3.2.57 | Shell scripting | macOS default, project standard |
| jq | 1.8.1 | JSON manipulation | All pheromone operations use jq; already a hard dependency |
| git | 2.52.0 | Worktree management, branch operations | Core infrastructure for propagation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| atomic_write | (internal) | Safe file writes | All pheromone write operations |
| acquire_lock/release_lock | (internal) | File locking for concurrent access | merge-back uses this for main pheromones.json |
| _pheromone_write | (internal) | Signal write with content_hash dedup | snapshot-inject reuses this for each injected signal |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| gitignore exception for export file | Write export to tracked location (e.g., `.aether/exchange/`) | exchange/ is already tracked and designed for cross-colony data; cleaner than gitignore exception |

## Architecture Patterns

### Recommended Project Structure
```
.aether/utils/
  pheromone.sh    -- 4 subcommands already here (lines 2030-2681)
  worktree.sh     -- _worktree_create wiring point (168 lines total)
.aether/docs/command-playbooks/
  build-complete.md  -- Seal ceremony (export-branch wiring)
  continue-advance.md -- Post-merge flow (merge-back wiring)
.claude/commands/ant/
  seal.md            -- Seal ceremony steps (export-branch insertion)
test/
  pheromone-snapshot-merge.sh  -- 43 existing tests
```

### Pattern 1: Wiring into Existing Call Sites
**What:** Call an existing subcommand at a specific point in an existing workflow
**When to use:** When the subcommand is complete and just needs to be connected
**Example:**
```bash
# In _worktree_create, after cp -r of .aether/data/:
local main_head
main_head=$(git -C "$AETHER_ROOT" rev-parse HEAD 2>/dev/null || echo "unknown")
(
  cd "$worktree_dir" && \
  AETHER_ROOT="$worktree_dir" \
  COLONY_DATA_DIR="$worktree_dir/.aether/data" \
  bash "$AETHER_ROOT/.aether/aether-utils.sh" \
    pheromone-snapshot-inject --from-branch "$base" --from-commit "$main_head" \
  2>/dev/null || true  # non-blocking
)
```

### Pattern 2: Gitignore Exception for Tracked Export File
**What:** Allow one specific file under a gitignored directory to be tracked
**When to use:** When a gitignored directory needs one exception
**Example:**
```gitignore
# .gitignore
.aether/data/
!.aether/data/pheromone-branch-export.json
```
**Alternative:** Write to `.aether/exchange/` which is already tracked and designed for cross-colony data. This avoids modifying .gitignore entirely.

### Anti-Patterns to Avoid
- **Blocking worktree creation on pheromone injection failure:** The `|| true` pattern is essential. If injection fails, the worktree still works -- just without injected signals. Log the error but do not abort.
- **Calling `_pheromone_write` while holding a lock in merge-back:** The existing implementation already handles this by writing directly to the file instead of calling `_pheromone_write` while holding the lock (see pheromone.sh:2553-2559 comment about deadlock).
- **Writing export file to gitignored location without exception:** If `.aether/data/pheromone-branch-export.json` is written but the gitignore is not updated, the file will never be committed and merge-back will fail.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Signal dedup across branches | Custom hash comparison | `_pheromone_write` with content_hash | Already handles dedup via content_hash matching |
| Concurrent write protection | Custom mutex | `acquire_lock`/`release_lock` | Already used by merge-back |
| Safe JSON file writes | echo/cat redirection | `atomic_write` | Handles temp file + rename atomically |
| Conflict resolution | Custom priority logic | Existing `_pheromone_merge_back` algorithm | Already implements REDIRECT > FOCUS > FEEDBACK with strength-based dedup |

**Key insight:** Every subcommand is already implemented. This phase is purely wiring, not building.

## Common Pitfalls

### Pitfall 1: COLONY_DATA_DIR Mismatch in Worktree Context
**What goes wrong:** `pheromone-snapshot-inject` reads `$COLONY_DATA_DIR/pheromones.json`, which points to the worktree's data directory, not main's. Since the worktree was just created with a copy of main's data, the injection would read the already-copied data and write back to the same file -- a no-op.
**Why it happens:** The `_worktree_create` function copies `.aether/data/` from main to the worktree (worktree.sh:77-79). By the time snapshot-inject runs, the worktree already has main's pheromones.json. Injection should verify the copy is correct, not read from the worktree.
**How to avoid:** The injection in this context reads from the worktree's freshly-copied pheromones.json and uses `_pheromone_write` to inject (which is a no-op if the signals already exist due to content_hash dedup). This is actually correct behavior: the copy already provides the signals, and the snapshot metadata records what was injected. The key value is the metadata trail, not the actual copy.
**Warning signs:** If snapshot-inject reports `injected_count: 0` for a worktree that should have signals, the copy may have failed.

### Pitfall 2: Export File Lost Because gitignore
**What goes wrong:** `pheromone-export-branch` writes to `.aether/data/pheromone-branch-export.json`, but `.aether/data/` is gitignored. The file is never committed, never merged, and merge-back has nothing to read.
**Why it happens:** The `.gitignore` has a blanket `.aether/data/` rule (line 80) with no exceptions.
**How to avoid:** Either (a) add `!.aether/data/pheromone-branch-export.json` to `.gitignore`, or (b) write the export file to `.aether/exchange/` which is already git-tracked. Option (b) is cleaner because `exchange/` is designed for cross-colony data sharing and is already in the repo.
**Warning signs:** `git status` not showing the export file after `pheromone-export-branch` runs.

### Pitfall 3: Merge-Back Deadlock on File Lock
**What goes wrong:** `_pheromone_merge_back` acquires a lock on `pheromones.json`, then tries to call `_pheromone_write` which also tries to acquire the same lock.
**Why it happens:** Both functions use `acquire_lock` on the same file path.
**How to avoid:** The existing implementation already handles this correctly (pheromone.sh:2553-2559): when the lock is already held, merge-back writes directly to the file instead of calling `_pheromone_write`. Do not change this pattern.
**Warning signs:** Merge-back hanging indefinitely (deadlock).

### Pitfall 4: Merge-Back Trigger Has No Clear Home
**What goes wrong:** Merge-back needs to run after a PR merges to main, but there is no existing post-merge hook in the codebase.
**Why it happens:** The PR workflow (Phase 43) is not yet implemented. The merge-back trigger is a chicken-and-egg problem.
**How to avoid:** Wire merge-back as a manual command first (e.g., `aether pheromone-merge-back --export-file PATH`), with an optional integration into continue-advance.md for the autopilot flow. The post-merge hook can be added in Phase 43 when the full PR workflow is wired.
**Warning signs:** Merge-back tests pass but there is no way to trigger it in practice.

## Code Examples

### Wiring snapshot-inject into _worktree_create (worktree.sh)
```bash
# After line 79 (cp -r of .aether/data/), before the json_ok result:

# Inject main's pheromone signals into the worktree
# Non-blocking: if injection fails, worktree still works without injected signals
if [[ -f "$AETHER_ROOT/.aether/data/pheromones.json" ]]; then
  local main_head
  main_head=$(git -C "$AETHER_ROOT" rev-parse HEAD 2>/dev/null || echo "unknown")
  (
    cd "$worktree_dir" 2>/dev/null && \
    AETHER_ROOT="$worktree_dir" \
    DATA_DIR="$worktree_dir/.aether/data" \
    COLONY_DATA_DIR="$worktree_dir/.aether/data" \
    bash "$AETHER_ROOT/.aether/aether-utils.sh" \
      pheromone-snapshot-inject --from-branch "$base" --from-commit "$main_head" \
    2>/dev/null || true
  )
fi
```

### Wiring export-branch into seal ceremony (seal.md)
```bash
# In seal.md, near Step 6.5 (XML export), add:
# Export pheromones for merge-back if on a non-main branch
current_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")
if [[ "$current_branch" != "main" ]]; then
  export_result=$(bash .aether/aether-utils.sh pheromone-export-branch 2>/dev/null || echo '{"ok":false}')
  export_ok=$(echo "$export_result" | jq -r '.ok // false' 2>/dev/null)
  if [[ "$export_ok" == "true" ]]; then
    eligible_count=$(echo "$export_result" | jq -r '.result.eligible_count // 0' 2>/dev/null)
    echo "Pheromone export: ${eligible_count} signals eligible for merge-back"
  else
    echo "Pheromone export: failed (non-blocking)"
  fi
fi
```

### Gitignore exception approach
```gitignore
# After the .aether/data/ line:
.aether/data/
!.aether/data/pheromone-branch-export.json
```

### Alternative: Write export to exchange/ instead
```bash
# In _pheromone_export_branch, change the export file path:
# FROM: "$COLONY_DATA_DIR/pheromone-branch-export.json"
# TO:   "$AETHER_ROOT/.aether/exchange/pheromone-branch-export.json"
```
This avoids modifying `.gitignore` entirely since `.aether/exchange/` is already tracked.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Pheromones are branch-isolated (no cross-branch) | Injection + export + merge-back protocol | 2026-03-30 design doc | Signals now flow across branches selectively |
| No pheromone lifecycle tracking across merges | Merge log with conflict resolution audit trail | 2026-03-30 implementation | Full audit trail of signal propagation |

**Deprecated/outdated:**
- None in this phase -- all code is current

## Open Questions

1. **Where exactly does merge-back get triggered?**
   - What we know: D-03 says `/ant:continue` or post-merge flow. The post-merge hook does not exist yet (Phase 43 builds it).
   - What's unclear: Whether to wire into continue-advance.md now (for autopilot flow) or leave as a manual command until Phase 43.
   - Recommendation: Wire as a subcommand call in continue-advance.md after state advancement. Add a check for `pheromone-branch-export.json` existence. If the file exists, run merge-back. This works for both autopilot and manual flows. The Phase 43 PR workflow will add the post-merge hook later.

2. **Export file location: `.aether/data/` with gitignore exception vs `.aether/exchange/`?**
   - What we know: `.aether/exchange/` is already git-tracked and designed for cross-colony data. `.aether/data/` is gitignored.
   - What's unclear: Whether changing the export path breaks the existing 43 tests (which use `COLONY_DATA_DIR` for the export path).
   - Recommendation: Use `.aether/exchange/` for the export file. Update the export subcommand to write there. Update tests accordingly. This is cleaner than a gitignore exception and aligns with the exchange/ directory's purpose.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| bash | All subcommands | Yes | 3.2.57 | -- |
| jq | JSON manipulation | Yes | 1.8.1 | -- |
| git | Worktree, branch ops | Yes | 2.52.0 | -- |
| GNU date / BSD date | Timestamp handling | Yes (BSD) | macOS | Subcommands use `date -u` which works on both |

**Missing dependencies with no fallback:**
- None

**Missing dependencies with fallback:**
- None

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Bash (custom harness with assert helpers) |
| Config file | None -- tests are standalone shell scripts |
| Quick run command | `bash test/pheromone-snapshot-merge.sh` |
| Full suite command | `npm test` (runs all 509 tests) |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PHERO-01 | snapshot-inject copies signals during worktree creation | integration | `bash test/pheromone-snapshot-merge.sh` (tests 1-2) + new worktree integration test | Partial -- injection tested, worktree wiring not tested |
| PHERO-02 | export-branch produces valid snapshot | unit | `bash test/pheromone-snapshot-merge.sh` (tests 3, 8, 13) | Yes |
| PHERO-03 | merge-back merges signals with dedup and conflict resolution | unit | `bash test/pheromone-snapshot-merge.sh` (tests 5-7, 9-12) | Yes |

### Sampling Rate
- **Per task commit:** `bash test/pheromone-snapshot-merge.sh`
- **Per wave merge:** `npm test`
- **Phase gate:** Full suite green (509+ passing) before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `test/worktree-pheromone-integration.sh` -- covers worktree creation triggering snapshot-inject
- [ ] Update existing export-branch tests if export path changes to `.aether/exchange/`
- [ ] Integration test for seal ceremony triggering export-branch
- [ ] Integration test for continue flow triggering merge-back

## Sources

### Primary (HIGH confidence)
- `.aether/docs/pheromone-propagation-design.md` -- Complete protocol specification, verified against implementation
- `.aether/docs/state-contract-design.md` -- Branch-local vs hub-global state rules
- `.aether/utils/pheromone.sh` lines 2030-2681 -- All four subcommands, verified line-by-line
- `.aether/utils/worktree.sh` lines 18-98 -- `_worktree_create` with existing data copy logic
- `test/pheromone-snapshot-merge.sh` -- 43 tests covering all subcommands, all passing

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/seal.md` -- Seal ceremony steps, verified XML export section for insertion point
- `.aether/docs/command-playbooks/continue-advance.md` -- Continue flow for merge-back wiring

### Tertiary (LOW confidence)
- None -- all findings verified against codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- bash, jq, git are the project's existing stack; no new dependencies
- Architecture: HIGH -- design doc is thorough and verified against implementation; integration points are identified in existing code
- Pitfalls: HIGH -- identified from reading the actual code (lock handling, gitignore, COLONY_DATA_DIR scoping)

**Research date:** 2026-03-30
**Valid until:** 2026-04-30 (stable -- no external dependencies)
