# Architecture Research: Signal/Event Propagation in Aether's Multi-Agent Colony

**Domain:** Multi-agent CLI orchestration system with stigmergic coordination
**Researched:** 2026-03-19
**Confidence:** HIGH (primary source is the codebase itself; patterns verified against multi-agent literature)

## Current System Overview

```
 USER COMMANDS                          STATE STORES
 (/ant:focus, /ant:redirect,           (pheromones.json, COLONY_STATE.json,
  /ant:feedback, /ant:build N)           learning-observations.json, midden.json)
        |                                       ^       |
        v                                       |       |
 +--------------+                               |       |
 | COMMAND      |--- pheromone-write ---------->+       |
 | LAYER        |                                       |
 | (slash cmds) |                                       |
 +------+-------+                                       |
        |                                               |
        v                                               |
 +--------------+                                       |
 | PLAYBOOK     |--- colony-prime ---> prompt_section   |
 | ORCHESTRATOR |                          |            |
 | (build-*.md, |<--- pheromone-display ---+            |
 |  continue-*) |                                       |
 +------+-------+                                       |
        |                                               |
        | prompt_section injected                       |
        | into worker prompts                           |
        v                                               |
 +--------------------+                                 |
 | WORKER AGENTS      |--- memory-capture ------------->+
 | (Builder, Watcher,  |--- midden-write -------------->+
 |  Chaos, Ambassador, |--- pheromone-write (auto) ---->+
 |  Measurer, etc.)    |                                |
 +--------------------+                                 |
                                                        |
 +--------------------+                                 |
 | UTILITY LAYER      |<-------------------------------+
 | (aether-utils.sh)  |
 | 150 subcommands    |
 | File locking       |
 | Atomic writes      |
 +--------------------+
```

### How Signals Flow Today (The Working Path)

1. **User emits signal** via `/ant:focus`, `/ant:redirect`, `/ant:feedback`
2. **pheromone-write** appends signal to `pheromones.json` (with locking, atomic write, decay metadata)
3. **colony-prime --compact** reads `pheromones.json`, extracts active signals, formats them into `prompt_section` markdown
4. **build-context.md** (Step 4) calls `colony-prime --compact` and stores the result as `prompt_section`
5. **build-wave.md** (Step 5.1) injects `prompt_section` into each Builder worker's prompt text
6. **Workers execute** with signals as inline context -- they see the signals as text in their prompt
7. **Workers also told** to periodically check `pheromone-read` at "natural breakpoints" (instruction in prompt, not enforced)

### The Gap: Where Integration Breaks Down

The signal propagation chain has clear weak points:

| Integration Point | Status | Issue |
|-------------------|--------|-------|
| User -> pheromones.json | WORKING | pheromone-write handles locking, validation, decay |
| pheromones.json -> colony-prime | WORKING | colony-prime reads, filters expired, formats |
| colony-prime -> prompt_section | WORKING | build-context.md stores result |
| prompt_section -> worker prompts | PARTIAL | Injected into Builder prompts but workers don't enforce reading |
| Auto-emitted signals (memory-capture) | WORKING | Writes to pheromones.json, but only consumed next build cycle |
| Midden -> REDIRECT (threshold) | WORKING | build-wave.md Step 5.2 checks midden thresholds mid-build |
| continue -> pheromone auto-emission | WORKING | Steps 2.1a-d emit FEEDBACK/REDIRECT from learnings/errors/decisions |
| phase_end expiration | WORKING | pheromone-expire --phase-end-only runs in continue |
| Fresh-install initialization | MISSING | pheromones.json created lazily by pheromone-write, not by init |
| Signal strength display | WORKING | pheromone-display shows decay percentages |
| Mid-build signal refresh | PARTIAL | build-wave.md refreshes colony-prime per wave, but not per worker |

## Component Boundaries

| Component | Responsibility | Communicates With | Data Format |
|-----------|---------------|-------------------|-------------|
| **pheromone-write** | Create/append signals | pheromones.json, constraints.json (backward compat) | JSON signal objects |
| **pheromone-read** | Read active signals with decay calculation | pheromones.json | Filtered JSON array |
| **pheromone-display** | Human-readable signal table | pheromones.json | Formatted text |
| **pheromone-prime** | Format signals + instincts for prompt injection | pheromones.json, COLONY_STATE.json (instincts) | Markdown prompt section |
| **pheromone-expire** | Archive expired signals to midden | pheromones.json, midden.json | JSON mutation |
| **colony-prime** | Unified priming: wisdom + signals + learnings + capsule | QUEEN.md, pheromones.json, COLONY_STATE.json, CONTEXT.md | Combined prompt section |
| **memory-capture** | Unified event handler: observe + emit pheromone + check promotion | learning-observations.json, pheromones.json, QUEEN.md | Multi-step pipeline |
| **suggest-analyze** | Codebase analysis for suggested signals | File system, pheromones.json | JSON suggestions |
| **suggest-approve** | Interactive approval for suggested signals | pheromones.json | JSON + interactive |
| **build-context.md** | Loads colony-prime into prompt_section at build start | colony-prime, pheromone-display | In-memory variable |
| **build-wave.md** | Refreshes colony-prime per wave, injects into worker prompts | colony-prime, worker Task calls | Text in worker prompts |
| **continue-advance.md** | Auto-emits signals from learnings/errors/decisions, expires phase_end signals | pheromone-write, pheromone-expire, memory-capture | JSON mutations |

## Data Flow: Signal Lifecycle

### 1. Signal Creation Flow

```
User/System Decision
    |
    v
pheromone-write(type, content, --strength, --ttl, --source, --reason)
    |
    +---> Validate type (FOCUS/REDIRECT/FEEDBACK)
    +---> Sanitize content (XSS-like prevention, 500 char limit)
    +---> Generate ID (sig_{type}_{epoch}_{random})
    +---> Compute expires_at from TTL
    +---> Acquire file lock
    +---> Append signal to pheromones.json .signals[]
    +---> Write constraints.json (backward compatibility)
    +---> Release lock
    |
    v
Signal stored with: {id, type, priority, source, created_at, expires_at, active, strength, reason, content}
```

### 2. Signal Consumption Flow (During Build)

```
/ant:build N
    |
    v
build-context.md Step 4:
    colony-prime --compact
        |
        +---> Read QUEEN.md (global + local wisdom)
        +---> Call pheromone-prime --compact --max-signals 8 --max-instincts 3
        |         |
        |         +---> Read pheromones.json
        |         +---> Filter: active==true, not expired (decay calc)
        |         +---> Sort by priority (high->normal->low), then strength
        |         +---> Take top N signals
        |         +---> Read instincts from COLONY_STATE.json
        |         +---> Format as markdown section
        |         v
        +---> Append context capsule (context-capsule --compact)
        +---> Append phase learnings from COLONY_STATE.json
        +---> Combine into prompt_section
        v
    prompt_section variable stored in-memory

build-wave.md Step 5.1:
    Before each wave:
        colony-prime --compact (REFRESH)
        |
        v
    Updated prompt_section
        |
        v
    Injected into each worker's Task tool prompt:
        "{ prompt_section }"
        |
        v
    Worker reads signals AS TEXT (no structured API)
    Worker instructed to "check for new signals" at breakpoints
    (but this is advisory -- no enforcement mechanism)
```

### 3. Signal Auto-Emission Flow (After Build)

```
continue-advance.md Step 2.1:
    |
    +---> 2.1a: FEEDBACK from phase outcome (learnings summary, strength 0.6, TTL 30d)
    +---> 2.1b: FEEDBACK from phase decisions (from CONTEXT.md, strength 0.6, TTL 30d, cap 3)
    +---> 2.1c: REDIRECT from midden error patterns (3+ occurrences, strength 0.7, TTL 30d, cap 3)
    +---> 2.1d: FEEDBACK from recurring success criteria (2+ phases, strength 0.6, TTL 30d, cap 2)
    +---> 2.1e: Expire phase_end signals, archive to midden
    |
    v
All auto-emitted signals have source="auto:*" and are deduplicated against existing active signals

memory-capture (called throughout build):
    |
    +---> learning-observe (record in learning-observations.json)
    +---> pheromone-write (auto-emit REDIRECT for failures, FEEDBACK for learnings)
    +---> learning-promote-auto (check if observation meets QUEEN.md promotion threshold)
    v
Creates signals during build that are consumed in NEXT build cycle
```

### 4. Signal Expiration Flow

```
Two expiration mechanisms:

1. Phase-scoped: expires_at == "phase_end"
   - Expired by: pheromone-expire --phase-end-only (in continue-advance Step 2.1e)
   - Archived to: midden/midden.json

2. Time-scoped: expires_at == ISO-8601 timestamp
   - Expired by: pheromone-read decay calculation (real-time filtering on read)
   - Natural decay reduces effective strength over time:
     FOCUS:    15-day half-life, 30-day full decay
     REDIRECT: 30-day half-life, 60-day full decay
     FEEDBACK: 45-day half-life, 90-day full decay
   - Signals below 10% strength treated as inactive

3. Pause-aware: Wall-clock TTLs extended by colony pause duration
```

## Architectural Patterns Applied

### Pattern: Stigmergic Blackboard (Already in Use)

**What:** Agents coordinate through a shared environment (pheromones.json) rather than direct messaging. Signals decay naturally, preventing stale data accumulation. The environment (JSON files) IS the communication medium.

**Aether's implementation:** This is the core pattern. Workers never message each other directly. All coordination flows through pheromones.json and COLONY_STATE.json. The system is textbook stigmergy -- agents modify the environment, other agents sense those modifications and adjust behavior.

**Strength:** Scales naturally. Adding a 23rd agent type requires zero changes to the signal system. Agents are fully decoupled from each other.

### Pattern: Hierarchical Decomposition (Already in Use)

**What:** Queen orchestrator breaks work into waves, delegates to specialized workers, gathers results.

**Aether's implementation:** Build playbook decomposes phases into dependency-ordered waves. Queen spawns Builder/Watcher/Chaos agents per wave. Results bubble up through JSON return values.

### Pattern: Generator-Critic (Already in Use)

**What:** Builder generates code, Watcher validates independently.

**Aether's implementation:** Builder agents implement tasks, Watcher agent independently verifies. This is enforced by the Spawn Gate and Watcher Gate in continue-gates.md.

## Patterns to Follow for Integration Work

### Pattern 1: Signal Injection at Construction Time

**What:** Signals must be part of the worker's initial context, not something workers discover mid-execution. The prompt_section from colony-prime is the canonical injection point.

**When:** Every time a worker is spawned.

**Implementation note:** This already works. The key insight is that `prompt_section` is the single bottleneck for signal delivery. Any new signal type or integration point must flow through colony-prime to reach workers.

### Pattern 2: Event-Carried State Transfer

**What:** When signals change, the change event carries enough state for the consumer to act without querying the source. This is what memory-capture does -- it observes an event, writes the observation, emits a pheromone, and checks promotion thresholds in a single pipeline.

**When:** Any time a build outcome, failure, or learning is recorded.

**Implementation note:** memory-capture is the unified pipeline. All new event types should flow through it rather than creating parallel paths.

### Pattern 3: Deduplication by Content Hash

**What:** Auto-emitted signals check for existing active signals with matching content before creating duplicates. This prevents signal pile-up across multiple build/continue cycles.

**When:** Every auto-emission (Steps 2.1a-d in continue-advance).

**Implementation note:** Each auto-emission step includes a jq-based deduplication check. New auto-emission paths must follow the same pattern: check `.signals[] | select(.active == true and .source == "{source}" and (.content.text | contains($text)))`.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Direct Agent-to-Agent Communication

**What people do:** Try to have one worker message another worker directly (e.g., Builder telling Watcher "focus on file X").
**Why it's wrong:** Violates the stigmergic model. Creates coupling between agent types. Breaks when agent types are added/removed.
**Do this instead:** Have the Builder emit a FOCUS pheromone via pheromone-write. The Watcher picks it up through its own colony-prime injection on the next cycle.

### Anti-Pattern 2: Signal Checking as Runtime Obligation

**What people do:** Instruct workers to "periodically check pheromone-read during execution."
**Why it's wrong:** Workers are sub-agents with limited tool call budgets. Checking signals mid-task burns tool calls on infrastructure instead of task work. Workers cannot reliably be trusted to follow advisory instructions.
**Do this instead:** Front-load all signal context at spawn time via prompt_section. Workers should receive all relevant signals before they start, not discover them mid-flight. The per-wave colony-prime refresh (build-wave.md Step 5.1) handles inter-wave signal updates.

### Anti-Pattern 3: Parallel Signal Stores

**What people do:** Store signals in both pheromones.json AND constraints.json AND COLONY_STATE.json.
**Why it's wrong:** Creates consistency bugs. The backward-compatibility write to constraints.json in pheromone-write already demonstrates this risk -- it's a maintenance burden.
**Do this instead:** pheromones.json is the single source of truth for active signals. constraints.json should be deprecated in favor of reading REDIRECT signals from pheromones.json directly. COLONY_STATE.json stores instincts (learned patterns), which are a different concern.

### Anti-Pattern 4: Unbounded Signal Accumulation

**What people do:** Auto-emit signals without caps, creating dozens of signals that dilute attention.
**Why it's wrong:** colony-prime with --compact takes top 8 signals. If there are 50 active signals, the most relevant may be pushed out. Workers can only process a limited context window.
**Do this instead:** Cap auto-emissions per cycle (already done: 3 decision pheromones, 3 error patterns, 2 success criteria per continue). Decay and expire aggressively. The 8-signal compact limit in pheromone-prime is correct.

## Recommended Project Structure for Integration Work

```
.aether/
 aether-utils.sh            # Signal subcommands live here (pheromone-*, colony-prime, memory-capture)
 data/
   pheromones.json           # Active signals (source of truth)
   COLONY_STATE.json         # Colony state + instincts + phase learnings
   learning-observations.json # Observation counts for promotion tracking
   midden/
     midden.json             # Archived expired signals + failure records
   constraints.json          # DEPRECATED: backward compat only
 templates/
   pheromones.template.json  # Initial pheromones.json structure
 docs/
   pheromones.md             # User guide for signal system
   command-playbooks/
     build-context.md        # Where colony-prime is called (Step 4)
     build-wave.md           # Where prompt_section is injected (Step 5.1)
     continue-advance.md     # Where auto-emission and expiration happen (Step 2.1)
```

### Structure Rationale

- **pheromones.json is the hub:** All signal reads and writes go through it. There is no secondary signal store.
- **colony-prime is the aggregator:** It combines wisdom + signals + learnings + context capsule into a single injectable section. Downstream consumers never read pheromones.json directly -- they consume colony-prime output.
- **memory-capture is the pipeline:** All event recording (learning, failure, success, redirect) flows through this single entry point, which handles observe + emit + promote in sequence.
- **Playbooks are the integration points:** build-context.md and build-wave.md are where signals enter the worker execution path. continue-advance.md is where signals are emitted and expired.

## Integration Points for the Milestone

### Integration Point 1: Fresh-Install Initialization

| Boundary | Communication | Status |
|----------|---------------|--------|
| /ant:init -> pheromones.json | Should create from template | MISSING |
| /ant:lay-eggs -> pheromones.json | Should create from template | NEEDS VERIFICATION |

**Current behavior:** pheromones.json is created lazily by pheromone-write when the first signal is emitted. This means colony-prime (called in build-context.md) may encounter a missing file on a fresh install before any signals are emitted. It handles this gracefully (warns and continues), but the system would be cleaner if initialized upfront.

### Integration Point 2: Pheromone Display in Status

| Boundary | Communication | Status |
|----------|---------------|--------|
| /ant:status -> pheromone-display | Should show active signals in status dashboard | NEEDS VERIFICATION |

### Integration Point 3: Signal Cleanup on Colony Seal/Entomb

| Boundary | Communication | Status |
|----------|---------------|--------|
| /ant:seal -> pheromone-expire | Should expire all signals when colony is sealed | NEEDS VERIFICATION |
| /ant:entomb -> pheromones.json | Should archive all signals to completed colony archive | NEEDS VERIFICATION |

### Integration Point 4: constraints.json Deprecation

| Boundary | Communication | Status |
|----------|---------------|--------|
| pheromone-write -> constraints.json | Backward compat write | SHOULD DEPRECATE |
| Any consumer of constraints.json | Should read from pheromones.json instead | NEEDS AUDIT |

## Scaling Considerations

| Concern | At 5-10 signals | At 20-50 signals | At 100+ signals |
|---------|-----------------|-------------------|-----------------|
| File I/O | Negligible | Acceptable | Could slow jq parsing; consider cleanup |
| Prompt token cost | ~200 tokens (compact) | ~500 tokens | colony-prime --compact caps at 8 signals, so stays bounded |
| Signal relevance | All visible | Ranking matters (priority + strength) | Many signals lost to compact limit; need stronger decay |
| Deduplication cost | Trivial | O(n) per emission | jq `.signals[]` scan gets expensive; consider indexing by content hash |

### Scaling Priorities

1. **First bottleneck:** Signal accumulation without cleanup. The pheromone-expire mechanism only runs during `/ant:continue`. Long build sessions without continue cycles can accumulate stale signals. Mitigation: also expire during build-context.md Step 4.
2. **Second bottleneck:** jq parsing of large pheromones.json. Each read/write operation parses the entire file. At 100+ signals (including inactive), jq performance degrades. Mitigation: periodic compaction that removes inactive signals entirely.

## Build Order Implications

Based on the component boundaries and data flow analysis:

1. **Initialization first** -- Ensure pheromones.json is created during /ant:init and /ant:lay-eggs from the template. This unblocks all downstream consumers.
2. **Audit constraints.json consumers** -- Before deprecating, find all code paths that read constraints.json and migrate to pheromones.json reads.
3. **Verify status integration** -- Ensure /ant:status shows active signals via pheromone-display.
4. **Verify lifecycle cleanup** -- Ensure /ant:seal and /ant:entomb properly handle signal archival.
5. **Polish fresh-install flow** -- Test the entire path: lay-eggs -> init -> focus -> build -> continue -> signals appear in worker prompts.

## Sources

- **Codebase analysis:** Direct reading of aether-utils.sh (pheromone-write, pheromone-read, pheromone-prime, colony-prime, memory-capture), build playbooks (build-context.md, build-wave.md, build-verify.md, build-complete.md), continue playbooks (continue-verify.md, continue-gates.md, continue-advance.md, continue-finalize.md), pheromones.md user guide. **HIGH confidence.**
- [Google's Eight Essential Multi-Agent Design Patterns (InfoQ)](https://www.infoq.com/news/2026/01/multi-agent-design-patterns/) -- Validates Aether's hierarchical decomposition + generator-critic patterns. **MEDIUM confidence.**
- [Four Design Patterns for Event-Driven Multi-Agent Systems (Confluent)](https://www.confluent.io/blog/event-driven-multi-agent-systems/) -- Event-carried state transfer pattern matches memory-capture pipeline. **MEDIUM confidence.**
- [Why Multi-Agent Systems Don't Need Managers: Lessons from Ant Colonies](https://www.rodriguez.today/articles/emergent-coordination-without-managers) -- Pressure field + decay model validates Aether's signal decay design. **MEDIUM confidence.**
- [Introducing SBP: Multi-Agent Coordination via Digital Pheromones](https://dev.to/naveentvelu/introducing-sbp-multi-agent-coordination-via-digital-pheromones-2j4e) -- Stigmergic Blackboard Protocol mirrors Aether's pheromones.json approach. **MEDIUM confidence.**
- [Stigmergy (Wikipedia)](https://en.wikipedia.org/wiki/Stigmergy) -- Foundational concept validation. **HIGH confidence.**
- [Self-Organising in Multi-agent Coordination Using Stigmergy (Springer)](https://link.springer.com/chapter/10.1007/978-3-540-24701-2_8) -- Academic validation of pheromone-based coordination patterns. **MEDIUM confidence.**

---
*Architecture research for: Aether pheromone integration and signal propagation*
*Researched: 2026-03-19*
