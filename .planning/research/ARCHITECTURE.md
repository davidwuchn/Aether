# Architecture Research: Autonomous Iterative Deep Research Engine

**Domain:** Autonomous AI deep research loops (file-based state, stateless iteration agents)
**Researched:** 2026-03-13
**Confidence:** HIGH (architecture patterns well-established across RALPH, FS-Researcher, EDR, and production systems)

## Standard Architecture

### System Overview

```
+-------------------------------------------------------------------+
|                     COMMAND LAYER (oracle.md)                      |
|  Interactive wizard -> research config -> launch control           |
+---------------------------------+---------------------------------+
                                  |
                                  v
+---------------------------------+---------------------------------+
|                      ORCHESTRATOR (oracle.sh)                      |
|  Bash loop: spawn -> wait -> check signals -> repeat               |
+---------------------------------+---------------------------------+
                                  |
          reads state             |            writes state
          before work             |            after work
              +-------------------+-------------------+
              |                                       |
              v                                       v
+-----------------------------+   +-----------------------------+
|      STATE LAYER (JSON)     |   |    KNOWLEDGE LAYER (MD)     |
|                             |   |                             |
|  research.json (config)     |   |  findings/                  |
|  state.json (iteration      |   |    001-<topic>.md           |
|    metadata, frontier,      |   |    002-<topic>.md           |
|    coverage tracking)       |   |    ...                      |
|  plan.json (research plan   |   |  synthesis.md (running      |
|    with question tree)      |   |    summary, compressed)     |
|                             |   |  gaps.md (open questions,   |
+-----------------------------+   |    contradictions)           |
                                  +-----------------------------+
              |                                       |
              v                                       v
+-----------------------------+   +-----------------------------+
|     CONTROL FILES           |   |    ARCHIVE LAYER            |
|                             |   |                             |
|  .stop (halt signal)        |   |  archive/<date>-<topic>/   |
|  .pause (pause signal)      |   |    research.json            |
|  .steer (mid-run steering)  |   |    state.json               |
|                             |   |    findings/                |
+-----------------------------+   |    synthesis.md             |
                                  +-----------------------------+
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Command Layer (oracle.md) | User interaction, research configuration, launch/status/stop | Slash command with interactive wizard |
| Orchestrator (oracle.sh) | Outer loop control, iteration spawning, signal checking, convergence detection | Bash script with `for` loop, spawns fresh AI CLI instances |
| Iteration Prompt (iterate.md) | Per-iteration instructions: read state, research, write structured output | Markdown prompt piped to stateless AI instance |
| State Layer | Machine-readable iteration metadata, research plan, coverage tracking | JSON files on disk |
| Knowledge Layer | Human-readable accumulated findings, synthesis, gaps | Markdown files on disk |
| Control Files | Inter-process signals (stop, pause, steer) | Sentinel files checked per iteration |
| Archive Layer | Previous research session preservation | Date-stamped directory copies |

## Recommended Project Structure

```
.aether/oracle/
+-- oracle.sh              # Orchestrator (bash loop)
+-- oracle.md              # Slash command handler (wizard, status, stop)
+-- iterate.md             # Per-iteration prompt (the core research agent)
+-- research.json          # Research configuration (topic, scope, questions)
+-- state.json             # Iteration state (NEW - replaces implicit state)
+-- plan.json              # Research plan with question tree (NEW)
+-- findings/              # Per-iteration structured findings (NEW)
|   +-- 001-initial-survey.md
|   +-- 002-api-patterns.md
|   +-- 003-error-handling.md
|   +-- ...
+-- synthesis.md           # Running compressed summary (NEW - replaces progress.md)
+-- gaps.md                # Open questions and contradictions (NEW)
+-- .stop                  # Halt signal
+-- .pause                 # Pause signal (NEW)
+-- .steer                 # Mid-run steering input (NEW)
+-- archive/               # Previous research sessions
|   +-- 2026-03-12-auth-patterns/
|   +-- ...
+-- discoveries/           # Standalone reusable discoveries (existing)
+-- prompts/               # Custom research prompts (existing)
```

### Structure Rationale

- **`findings/` directory:** Each iteration writes to its own numbered file instead of appending to a monolithic progress.md. This prevents the "append-only growth" problem where progress.md becomes too large for any single iteration to read. Numbered files create a natural audit trail.
- **`state.json` (new):** Replaces implicit state tracking. Currently, the only way to know iteration progress is to count `## Iteration` headings in progress.md. Structured JSON enables the orchestrator to make decisions (convergence, branching, re-prioritization) without parsing markdown.
- **`plan.json` (new):** Externalizes the research plan as a living document the AI can update. Currently, research questions are static in research.json. A plan with priorities, completion status, and sub-questions enables iteration N to direct iteration N+1.
- **`synthesis.md` (new):** A compressed running summary that replaces progress.md as the primary "what do we know" document. Unlike progress.md (which grows linearly), synthesis.md is rewritten each iteration to stay within a token budget, preventing context exhaustion.
- **`gaps.md` (new):** Explicit tracking of what is NOT known. This is the key missing piece -- currently the system has no structured way to represent "unanswered questions" that future iterations should pursue.
- **`.steer` (new):** Enables mid-research steering without stopping. A user can write a steering directive and the next iteration picks it up, adjusting course without losing accumulated state.

## Architectural Patterns

### Pattern 1: Structured State Bridge

**What:** Use structured JSON files as the "handoff protocol" between stateless AI iterations. Each iteration reads state, does work, writes updated state. The state file is the only continuity mechanism.

**When to use:** Always. This is the fundamental pattern that makes stateless iterations achieve stateful depth.

**Trade-offs:** JSON is machine-parseable but requires schema discipline. Schema drift across iterations can corrupt state. Must validate JSON on write.

**Current state vs. recommended:**

Current `research.json` (static config only):
```json
{
  "topic": "How auth works",
  "scope": "both",
  "questions": ["Q1", "Q2", "Q3"],
  "max_iterations": 50,
  "target_confidence": 95
}
```

Recommended `state.json` (living iteration state):
```json
{
  "iteration": 7,
  "phase": "deepening",
  "confidence": 62,
  "coverage": {
    "Q1": { "status": "complete", "confidence": 90, "findings": ["001", "003"] },
    "Q2": { "status": "partial", "confidence": 45, "findings": ["002"] },
    "Q3": { "status": "not_started", "confidence": 0, "findings": [] }
  },
  "frontier": ["How does token refresh interact with SSO?", "What happens on network partition?"],
  "completed_areas": ["basic auth flow", "password hashing", "session management"],
  "next_priority": "Q2",
  "stalled_count": 0,
  "last_finding_file": "003-error-handling.md"
}
```

### Pattern 2: Research Plan as Living Document

**What:** The research plan (questions to investigate) is a mutable tree, not a static list. Iterations can add sub-questions, mark questions answered, re-prioritize, and branch into unexpected areas.

**When to use:** Any research deeper than a surface scan (more than 5 iterations).

**Trade-offs:** Plan mutation means the research can drift from original intent. Mitigate by anchoring to the original questions in research.json (immutable) while allowing plan.json to evolve.

Recommended `plan.json`:
```json
{
  "original_questions": ["Q1", "Q2", "Q3"],
  "question_tree": [
    {
      "id": "Q1",
      "text": "How does authentication work?",
      "status": "complete",
      "confidence": 90,
      "children": [
        {
          "id": "Q1.1",
          "text": "How are tokens refreshed?",
          "status": "in_progress",
          "confidence": 30,
          "added_by_iteration": 4,
          "children": []
        }
      ]
    },
    {
      "id": "Q2",
      "text": "What are the error handling patterns?",
      "status": "in_progress",
      "confidence": 45,
      "children": []
    }
  ],
  "abandoned": [
    { "id": "Q1.2", "text": "LDAP integration?", "reason": "Not relevant to this codebase", "abandoned_at_iteration": 5 }
  ]
}
```

### Pattern 3: Synthesis with Compression

**What:** Instead of append-only accumulation (progress.md growing forever), maintain a compressed synthesis that is rewritten each iteration. The synthesis stays within a token budget (e.g., 4000 tokens) so every iteration can read the full state of knowledge without context exhaustion.

**When to use:** Any research session exceeding 10 iterations. Below 10, append-only is fine.

**Trade-offs:** Rewriting synthesis risks losing detail. Mitigate by keeping per-iteration findings/ files as the source of truth -- synthesis is a compressed index, not the primary record.

**How it works:**
```
Iteration N reads:
  1. state.json (what iteration is this, what's the priority)
  2. plan.json (what questions remain)
  3. synthesis.md (compressed summary of ALL prior findings)
  4. gaps.md (what we don't know yet)

Iteration N does:
  5. Research (tools: Glob, Grep, Read, WebSearch, WebFetch)
  6. Write findings/00N-<topic>.md (detailed per-iteration output)

Iteration N writes:
  7. Update state.json (increment iteration, update coverage, set next_priority)
  8. Update plan.json (mark questions, add sub-questions)
  9. Rewrite synthesis.md (merge new findings into compressed summary)
  10. Update gaps.md (add new gaps, remove resolved ones)
```

### Pattern 4: Frontier-Based Iteration Direction

**What:** Each iteration explicitly identifies the "frontier" -- the most valuable next area to research. This replaces the current implicit approach where each iteration independently decides what to work on, often duplicating effort.

**When to use:** Always. This is what makes iteration N+1 meaningfully different from iteration N.

**Trade-offs:** Requires discipline from the AI to actually set a frontier. The iterate.md prompt must enforce this.

**Example frontier in state.json:**
```json
{
  "frontier": [
    "Token refresh flow under SSO -- found reference in auth.js:142 but not traced yet",
    "Error recovery in database connection pooling -- mentioned in 3 places but contradictory"
  ],
  "next_priority": "Token refresh flow under SSO",
  "priority_reason": "Blocks understanding of Q1.1 which is the last gap in authentication coverage"
}
```

### Pattern 5: Convergence Detection

**What:** The orchestrator (bash loop) detects when research has converged -- when iterations stop producing new knowledge. This replaces the current "self-reported confidence" which is unreliable (the AI tends to be overconfident or overly conservative).

**When to use:** Always. Self-reported confidence should be one signal among several, not the sole termination criterion.

**Detection signals (checked by oracle.sh):**

| Signal | How to detect | Weight |
|--------|--------------|--------|
| Self-reported confidence | AI writes confidence to state.json | Low (unreliable alone) |
| Findings size trend | Track byte count of findings/00N.md -- shrinking findings = diminishing returns | Medium |
| Gap resolution rate | Count open gaps in gaps.md over time -- plateau = convergence | High |
| Coverage completeness | All questions in plan.json at "complete" or "abandoned" | High |
| Stall detection | Same frontier repeated 3+ iterations without progress | High (signals stuck) |

**Implementation in oracle.sh:**
```bash
# After each iteration, check convergence
CONFIDENCE=$(jq -r '.confidence' state.json)
STALL_COUNT=$(jq -r '.stalled_count' state.json)
OPEN_GAPS=$(wc -l < gaps.md)  # simplified
FINDING_SIZE=$(wc -c < "findings/$(printf '%03d' $i)-*.md" 2>/dev/null || echo 0)

# Convergence = high confidence + low gaps + no stall
if [[ "$CONFIDENCE" -ge "$TARGET" && "$STALL_COUNT" -eq 0 ]]; then
  echo "CONVERGED: confidence target met"
  exit 0
fi

# Stall = 3+ iterations with same frontier
if [[ "$STALL_COUNT" -ge 3 ]]; then
  echo "STALLED: research not progressing, consider stopping"
  # Don't auto-stop, but warn
fi
```

## Data Flow

### Per-Iteration Data Flow

```
oracle.sh (orchestrator)
    |
    | spawns fresh AI instance with iterate.md prompt
    v
+-- AI Instance (stateless, fresh 200K context) --+
|                                                   |
|  READS (input to this iteration):                 |
|    research.json  -- what are we researching       |
|    state.json     -- where are we in the process   |
|    plan.json      -- what questions remain          |
|    synthesis.md   -- compressed prior knowledge     |
|    gaps.md        -- what we don't know             |
|    .steer         -- any user steering directives   |
|                                                   |
|  DOES (the actual research):                       |
|    Glob/Grep/Read -- explore codebase              |
|    WebSearch/WebFetch -- external research          |
|    Analyze, synthesize, connect                    |
|                                                   |
|  WRITES (output of this iteration):                |
|    findings/00N-<topic>.md -- detailed findings     |
|    state.json     -- updated iteration state        |
|    plan.json      -- updated question tree          |
|    synthesis.md   -- rewritten compressed summary   |
|    gaps.md        -- updated open questions          |
|                                                   |
+---------------------------------------------------+
    |
    | AI instance exits, output captured
    v
oracle.sh (orchestrator)
    |
    | checks: completion signal? convergence? stop file?
    | decides: continue loop or terminate
    v
[Next iteration or COMPLETE]
```

### Cross-Iteration Knowledge Flow

```
Iteration 1         Iteration 2         Iteration 3         Iteration N
+-----------+       +-----------+       +-----------+       +-----------+
| Survey    |       | Deepen    |       | Synthesize|       | Verify    |
| landscape |  -->  | Q1 focus  |  -->  | Connect   |  -->  | Fill gaps |
| Set plan  |       | Find gaps |       | patterns  |       | Converge  |
+-----------+       +-----------+       +-----------+       +-----------+
     |                   |                   |                   |
     v                   v                   v                   v
state.json:          state.json:          state.json:          state.json:
 iter=1               iter=2               iter=3               iter=N
 conf=15              conf=35              conf=60              conf=92
 phase=survey         phase=deepening      phase=synthesis      phase=verification
     |                   |                   |                   |
     v                   v                   v                   v
synthesis.md:        synthesis.md:        synthesis.md:        synthesis.md:
 "Found 3 main       "Auth uses JWT       "3 patterns          "All questions
  components..."       with refresh..."     identified..."       answered..."
     |                   |                   |                   |
     v                   v                   v                   v
gaps.md:             gaps.md:             gaps.md:             gaps.md:
 "Q1: ?"              "Q1.1: token         "Q1.1: edge          (empty or
  "Q2: ?"              refresh?"            case in SSO?"        minimal)
  "Q3: ?"             "Q2: ?"
```

### Key Data Flows

1. **Config flow (immutable):** `wizard -> research.json -> iterate.md reads it` -- The original research parameters never change. They anchor the research to the user's intent.

2. **State flow (mutable per iteration):** `iterate.md writes state.json -> oracle.sh reads for convergence -> iterate.md reads next iteration` -- The state file is the primary handoff mechanism between iterations.

3. **Knowledge flow (accumulative):** `iterate.md writes findings/00N.md -> iterate.md reads synthesis.md (compressed prior) -> iterate.md rewrites synthesis.md with new knowledge` -- Knowledge accumulates in findings/ and is compressed into synthesis.md.

4. **Direction flow (adaptive):** `iterate.md writes plan.json + frontier -> iterate.md reads plan.json next iteration -> focuses on frontier` -- Each iteration steers the next toward the most valuable unexplored area.

5. **Control flow (external signals):** `user touches .stop/.steer -> oracle.sh or iterate.md reads signal -> adjusts behavior` -- External control without interrupting the research process.

## Research Phases (Emergent Structure)

The iterate.md prompt should recognize and adapt to distinct research phases. These are not rigidly sequential -- the AI should identify which phase is appropriate based on state.json.

| Phase | Iterations | Focus | Transition Criterion |
|-------|-----------|-------|---------------------|
| **Survey** | 1-3 | Broad landscape scan, identify major areas | All original questions have initial coverage |
| **Deepening** | 4-N | Focused investigation of highest-priority gaps | Coverage > 60% on all questions |
| **Synthesis** | N-M | Connect findings, identify patterns, resolve contradictions | All questions at "complete" or "abandoned" |
| **Verification** | M-end | Verify claims, fill remaining gaps, compress final output | Confidence > target, gaps resolved |

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Quick scan (5 iterations) | Skip plan.json, synthesis.md = simple append, no convergence detection needed |
| Standard research (15 iterations) | Full architecture as described, synthesis rewriting kicks in at iteration 10 |
| Deep dive (30+ iterations) | Add findings/ file size monitoring, consider splitting synthesis.md by question, add stall-breaking heuristics |
| Marathon (50+ iterations) | Add checkpoint system (snapshot state every 10 iterations), implement "rotation" where the AI re-reads original research.json to prevent drift |

### Scaling Priorities

1. **First bottleneck: synthesis.md growth.** Without compression, the synthesis grows linearly and eventually exceeds what an iteration can read. Solution: enforce a token budget on synthesis.md and rewrite (not append) each iteration.

2. **Second bottleneck: findings/ directory size.** At 50+ iterations, the AI cannot read all findings. Solution: synthesis.md is the compressed index -- iterations should read synthesis.md, not all findings. Findings/ is the audit trail, not the primary input.

3. **Third bottleneck: research drift.** After 30+ iterations, the research can wander far from original intent. Solution: "rotation" pattern -- every 10 iterations, re-read research.json and explicitly compare current direction against original questions.

## Anti-Patterns

### Anti-Pattern 1: Append-Only Knowledge Accumulation

**What people do:** Each iteration appends findings to a single progress.md file that grows indefinitely.
**Why it is wrong:** By iteration 20, progress.md exceeds what a fresh AI instance can meaningfully process. Later iterations either skip reading it (losing context) or read a truncated version (losing early findings). The research becomes shallow repetition rather than genuine deepening.
**Do this instead:** Separate raw findings (append to findings/ directory) from compressed synthesis (rewrite synthesis.md each iteration with a token budget). The synthesis is the "working memory," findings are the "long-term memory."

### Anti-Pattern 2: Self-Reported Confidence as Sole Termination

**What people do:** Ask the AI to rate its confidence 0-100 and stop when it exceeds the target.
**Why it is wrong:** LLMs are poorly calibrated on meta-cognitive confidence. They tend to report high confidence early (premature termination) or refuse to exceed 90% (wasted iterations). The number is arbitrary and disconnected from actual research quality.
**Do this instead:** Use multiple convergence signals: coverage completeness, gap resolution rate, findings size trend, AND self-reported confidence. The orchestrator (bash) should aggregate these signals, not rely on any single one.

### Anti-Pattern 3: Undirected Iterations

**What people do:** Give each iteration the same generic prompt ("research this topic") and hope it picks a useful area.
**Why it is wrong:** Without explicit direction, iterations 1, 5, and 15 may all investigate the same surface-level aspects. The research breadth grows but depth does not. This is "iterative appending, not iterative deepening."
**Do this instead:** Each iteration must write an explicit frontier (next_priority in state.json) that the next iteration is obligated to pursue. The iterate.md prompt must enforce: "You MUST start by addressing the frontier from state.json before exploring new areas."

### Anti-Pattern 4: Monolithic Per-Iteration Prompt

**What people do:** Put all instructions (read state, research, write findings, update plan, update synthesis, rate confidence) in one prompt.
**Why it is wrong:** A monolithic prompt with 10+ distinct responsibilities leads to partial execution. The AI completes the first few steps well but skips or shortcuts later steps, especially state file updates.
**Do this instead:** Structure the iterate.md prompt with explicit numbered steps and mandatory output sections. Consider using XML tags to delimit required outputs so completion can be verified.

### Anti-Pattern 5: No Schema Validation on State Files

**What people do:** Trust the AI to write valid JSON to state.json without verification.
**Why it is wrong:** One malformed state.json write corrupts all subsequent iterations. The AI has no error correction mechanism for its own output format errors.
**Do this instead:** The orchestrator (oracle.sh) should validate state.json after each iteration using `jq`. If validation fails, restore from the previous iteration's state and retry or halt.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Claude CLI | Spawned by oracle.sh via `claude --print` | Stateless, fresh context each call |
| OpenCode CLI | Alternative to Claude CLI | Same interface, detected at runtime |
| Web search | Used by iterate.md via WebSearch tool | Available when scope includes "web" |
| File system | Primary state persistence layer | All state lives on disk as files |
| tmux | Background session management | Optional, falls back to manual launch |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| oracle.md (wizard) <-> oracle.sh (loop) | research.json on disk | Wizard writes config, loop reads it. One-way. |
| oracle.sh (loop) <-> iterate.md (AI) | state.json, plan.json, synthesis.md, gaps.md | Bidirectional via files. AI reads and writes. |
| oracle.sh (loop) <-> user | .stop, .steer, .pause files + /ant:oracle status | User writes signal files, loop reads them. |
| oracle.sh (loop) <-> colony state | NONE (by design) | Oracle never touches COLONY_STATE.json or other colony files. Sandboxed. |
| findings/ <-> synthesis.md | iterate.md reads synthesis, writes findings | Synthesis is compressed view of all findings. |

### Integration with Aether Colony System

The oracle is intentionally isolated from the colony state system. It does NOT:
- Read or write COLONY_STATE.json
- Modify pheromones, constraints, or activity logs
- Affect colony phase progression

It DOES:
- Write to `.aether/oracle/` exclusively
- Produce a final report that can be manually consumed by colony workflows
- Respect the same CLI detection pattern (claude vs opencode)

This isolation is a feature, not a limitation. Research sessions can run alongside active colony work without interference.

## Build Order (Dependencies Between Components)

The components have clear build-order dependencies:

```
Phase 1: State Layer Foundation
  state.json schema + validation
  plan.json schema + question tree
  gaps.md format specification
  --> These must exist before iterate.md can reference them

Phase 2: Iteration Prompt (iterate.md)
  Read state -> research -> write structured output
  Depends on: state.json schema, plan.json schema, gaps.md format
  --> This is the core agent behavior, requires stable state contracts

Phase 3: Orchestrator Updates (oracle.sh)
  Multi-signal convergence detection
  State validation after each iteration
  Findings directory management
  --> Depends on state.json being written correctly by iterate.md

Phase 4: Knowledge Management
  synthesis.md compression/rewrite logic
  findings/ directory with numbered files
  --> Can be built after basic iterate.md works with append-only

Phase 5: Command Layer Updates (oracle.md)
  Status display reading new state format
  Steering file support (.steer, .pause)
  --> Polish layer, depends on everything else working

Phase 6: Advanced Features
  Convergence heuristics (stall detection, trend analysis)
  Checkpoint/snapshot system
  Research drift detection (rotation pattern)
  --> Only needed for deep/marathon sessions
```

**Critical path:** Phase 1 -> Phase 2 -> Phase 3. Everything else can be parallelized or deferred.

**The key insight:** Phase 2 (iterate.md) is the hardest part. Getting the per-iteration prompt right -- so the AI reliably reads all state, does focused research, and writes all state updates -- determines whether the entire system works. The bash loop and state schemas are straightforward engineering; the prompt is where the intelligence lives.

## How Stateless Iterations Achieve Stateful Research Depth

This is the central architectural question. The answer has five parts:

1. **Structured state files as memory.** Each iteration is stateless (fresh 200K context), but the state files on disk are the persistent memory. The state.json, plan.json, synthesis.md, and gaps.md collectively represent "everything the research knows" in a form compact enough for a fresh AI to consume.

2. **Explicit frontier as direction.** The frontier in state.json tells the next iteration exactly where to look. Without this, each iteration would independently decide what to research, leading to redundant coverage. The frontier is the mechanism by which iteration N "talks to" iteration N+1.

3. **Synthesis compression as working memory.** The synthesis.md file acts as compressed working memory. It cannot grow indefinitely (token budget), so it forces prioritization -- only the most important findings survive compression. This mirrors how human researchers maintain mental models: lossy compression of raw data into patterns and principles.

4. **Gap tracking as research agenda.** The gaps.md file explicitly tracks what is NOT known. This inverts the default behavior (reporting what was found) to also track what was not found. Gaps are the driver of depth -- each iteration should close gaps, not just add findings.

5. **Convergence detection as termination.** The orchestrator monitors multiple signals to detect when research has genuinely converged (all questions answered, gaps resolved, findings shrinking) rather than relying on the AI's self-assessment. This prevents both premature termination and infinite loops.

Together, these five mechanisms transform a sequence of independent, stateless AI invocations into a coherent, directed, progressively deepening research process.

## Sources

- [RALPH (snarktank/ralph)](https://github.com/snarktank/ralph) -- Original bash loop pattern for autonomous AI agents. Foundation for oracle.sh. HIGH confidence.
- [FS-Researcher: File-System-Based Agents](https://arxiv.org/abs/2602.01566) -- Dual-agent research framework using filesystem as external memory. Validates findings/ + synthesis.md separation pattern. HIGH confidence.
- [Enterprise Deep Research (EDR)](https://arxiv.org/html/2510.17797) -- Multi-agent deep research with steerable context engineering. Source for convergence detection, gap tracking, and steering patterns. HIGH confidence.
- [Multi-Agent Deep Research Architecture](https://trilogyai.substack.com/p/multi-agent-deep-research-architecture) -- Hierarchical iterative research pipeline. Validates question tree and synthesis patterns. MEDIUM confidence (blog post, not peer-reviewed).
- [Deep Research Agents: Systematic Examination](https://arxiv.org/html/2506.18096v1) -- Comprehensive survey of DR agent architectures. Source for taxonomy and state management patterns. HIGH confidence.
- [The Ralph Loop: Autonomous AI Agent Architecture](https://blakecrosley.com/blog/ralph-agent-architecture) -- Production experience with Ralph pattern. Source for spawn budgets, completion criteria, convergence. MEDIUM confidence (practitioner report).
- [AgentFS / Turso](https://github.com/tursodatabase/agentfs) -- Filesystem abstraction for agent state. Validates file-based state management approach. MEDIUM confidence.
- [Everything is Context: Agentic File System Abstraction](https://arxiv.org/abs/2512.05470) -- Academic grounding for filesystem as context engineering mechanism. HIGH confidence.

---
*Architecture research for: Autonomous Iterative Deep Research Engine (Oracle v2)*
*Researched: 2026-03-13*
