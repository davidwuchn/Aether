# Technology Stack

**Project:** Aether Oracle v2 -- Deep Research Engine Upgrade
**Researched:** 2026-03-13
**Mode:** Ecosystem (Stack dimension)

---

## Context: What We Already Have

The oracle is a 134-line bash script (`oracle.sh`) that spawns fresh Claude CLI instances in a for-loop. Each iteration reads a static prompt (`oracle.md`), reads a flat `progress.md` file, appends findings, and checks for a completion sentinel. The existing infrastructure is bash-based (3.2 compatible), uses `jq` for JSON manipulation, and runs via `claude --print` or tmux.

**This is NOT a greenfield build.** We are upgrading an existing working system. The stack recommendations below are constrained to what integrates with the existing bash/CLI architecture.

---

## Recommended Stack

### Core Runtime: Bash + Claude CLI (Keep What Works)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Bash | 3.2+ | Loop orchestrator, state machine | Already proven in oracle.sh; entire Aether ecosystem is bash-based. No reason to rewrite in Python/Node. |
| Claude CLI (`claude -p`) | Latest | AI reasoning engine per iteration | Already used. The `--print` flag enables non-interactive automation. New capabilities (--output-format json, --json-schema, --continue, --resume, --append-system-prompt) unlock structured output and session continuity. |
| jq | 1.6+ | JSON state manipulation | Already a dependency via aether-utils.sh (150+ subcommands use it). Handles all structured state read/write needs. |

**Confidence: HIGH** -- These are already in use and well-documented in official Claude Code docs.

### Structured Output: Claude CLI JSON Schema Mode

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `--output-format json` | Claude CLI current | Machine-parseable iteration results | Enables reliable extraction of structured findings from each iteration instead of grep-based sentinel detection. |
| `--json-schema` | Claude CLI current | Enforce iteration output structure | Forces each iteration to produce a validated JSON object with claims, sources, confidence, and gaps -- eliminating freeform append chaos. |
| `--append-system-prompt` | Claude CLI current | Dynamic per-iteration context injection | Allows injecting current research state, gaps, and priorities without modifying the base oracle.md prompt. |

**Confidence: HIGH** -- Verified in official Claude Code headless docs (code.claude.com/docs/en/headless). The `--json-schema` flag uses constrained decoding to guarantee schema compliance.

### State Management: Structured JSON Research State

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `research-state.json` (new) | N/A | Structured knowledge accumulator | Replaces flat progress.md as the machine-readable state. Contains claims, sources, gaps, confidence scores. Each iteration reads this, extends it, writes it back. |
| `progress.md` (retained) | N/A | Human-readable research narrative | Keep as append-only prose log for human review. Generated FROM research-state.json, not the source of truth. |
| `jq` merges | N/A | State transitions between iterations | Use `jq -s '.[0] * .[1]'` patterns to merge new iteration findings into accumulated state. Atomic writes via existing `atomic-write.sh` utility. |

**Confidence: HIGH** -- This is the core insight from the research. Every successful deep research system (GPT Researcher, FlowSearch, Anthropic's own multi-agent system) uses structured state between iterations rather than flat text append. The Ralph Loop pattern specifically advocates "shifting state management from the LLM's memory (token sequence) to the disk (file system)."

### Source Verification: Multi-Source Consensus Pattern

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| WebSearch tool (via Claude) | N/A | Primary source discovery | Claude CLI has native WebSearch and WebFetch tools. Each iteration can search, fetch, and verify. |
| Claim-source tracking | N/A | Audit trail for every assertion | Each claim in research-state.json links to 1+ source URLs with fetch timestamps and credibility indicators. |
| Cross-reference scoring | N/A | Confidence via source agreement | Claims verified by 3+ independent sources get HIGH confidence. Single-source claims get LOW. This is the pattern used by GPT Researcher (20+ sources per topic) and recommended by Anthropic's own research system. |

**Confidence: MEDIUM** -- The pattern is well-established across the ecosystem. Implementation specifics (how to score credibility within a bash/CLI context) will need iteration. Claude CLI's WebSearch/WebFetch tools handle the mechanics; the challenge is structuring the prompt to produce consistent source attribution.

### Iteration Intelligence: Gap-Driven Research Direction

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Knowledge gap analysis | N/A | Determines what each iteration researches | Instead of "research more about topic X," each iteration receives the current research-state.json and must identify and fill specific gaps. This is the FlowSearch "Knowledge Flow Refiner" pattern. |
| Depth/breadth parameters | N/A | Controls research exploration shape | Configurable via research.json (already exists). Breadth = how many subtopics per iteration. Depth = how many follow-up levels. |
| Convergence detection | N/A | Replaces arbitrary confidence self-rating | Track new-claims-per-iteration. When 3 consecutive iterations produce <2 new claims, research is converging. More reliable than asking the LLM "how confident are you?" |

**Confidence: MEDIUM** -- Gap-driven iteration is the pattern used by FlowSearch, OpenAI's Deep Research API, and Anthropic's research system. Convergence detection based on diminishing returns is a pattern from the academic deep research literature (DeepSearchQA, DeepResearchBench). The specific threshold numbers will need tuning.

### Report Generation: Structured-to-Prose Pipeline

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Research-state-to-report | N/A | Final synthesis step | A dedicated Claude CLI call that reads the complete research-state.json and produces a structured report with citations, organized by topic, with confidence annotations. |
| Markdown output | N/A | Universal format | Reports in markdown for compatibility with any downstream consumer (colony state, CLAUDE.md, developer docs). |
| Discoveries extraction | N/A | Actionable findings | Separate JSON file with machine-actionable discoveries (already has `discoveries/` dir). Used by colony to create instincts, pheromones, or wisdom. |

**Confidence: HIGH** -- This is a standard pattern. The key insight is separating accumulation (JSON state) from presentation (markdown report). GPT Researcher, dzhng/deep-research, and FlowSearch all use this two-phase approach.

---

## Critical Architecture Pattern: The Research State Schema

This is the most important technical decision. The schema for `research-state.json` determines everything downstream.

```json
{
  "meta": {
    "topic": "string",
    "scope": "codebase|web|both",
    "started_at": "ISO-8601",
    "iteration_count": 0,
    "status": "active|converging|complete",
    "convergence_window": []
  },
  "questions": [
    {
      "id": "q1",
      "text": "Original research question",
      "status": "open|partially_answered|answered",
      "sub_questions": ["q1.1", "q1.2"]
    }
  ],
  "claims": [
    {
      "id": "c1",
      "text": "Specific factual claim",
      "question_id": "q1",
      "confidence": "HIGH|MEDIUM|LOW",
      "source_ids": ["s1", "s2"],
      "first_seen_iteration": 1,
      "last_verified_iteration": 3,
      "contradicted_by": [],
      "superseded_by": null
    }
  ],
  "sources": [
    {
      "id": "s1",
      "url": "https://...",
      "title": "Source title",
      "type": "official_docs|academic|blog|forum|code",
      "fetched_at": "ISO-8601",
      "credibility": "HIGH|MEDIUM|LOW"
    }
  ],
  "gaps": [
    {
      "id": "g1",
      "description": "What we don't know yet",
      "question_id": "q1",
      "priority": "HIGH|MEDIUM|LOW",
      "status": "open|investigating|resolved",
      "resolved_by_claims": []
    }
  ],
  "contradictions": [
    {
      "claim_a_id": "c1",
      "claim_b_id": "c2",
      "description": "How they conflict",
      "resolution": null,
      "resolution_source_id": null
    }
  ]
}
```

**Why this schema:**
- **Claims are atomic.** Each claim is a single verifiable assertion with source attribution.
- **Sources are deduplicated.** Multiple claims can reference the same source without duplication.
- **Gaps drive iteration.** The LLM reads open gaps and prioritizes filling them.
- **Contradictions are tracked explicitly.** Rather than silently overwriting, conflicts are surfaced.
- **Convergence is measurable.** Track claims-per-iteration in `convergence_window` array.

**Confidence: MEDIUM** -- This schema synthesizes patterns from FlowSearch (DAG nodes), GPT Researcher (multi-source aggregation), and Anthropic's research system (task decomposition). The specific field names and nesting will likely evolve during implementation, but the core concepts (atomic claims, source dedup, explicit gaps, contradiction tracking) are consistent across all successful implementations.

---

## Iteration Output Schema

Each Claude CLI iteration must output JSON conforming to this schema (enforced via `--json-schema`):

```json
{
  "new_claims": [
    {
      "text": "string",
      "question_id": "string",
      "confidence": "HIGH|MEDIUM|LOW",
      "sources": [
        {
          "url": "string",
          "title": "string",
          "type": "string"
        }
      ]
    }
  ],
  "updated_claims": [
    {
      "claim_id": "string",
      "new_confidence": "HIGH|MEDIUM|LOW",
      "new_sources": [],
      "reason": "string"
    }
  ],
  "new_gaps": [
    {
      "description": "string",
      "question_id": "string",
      "priority": "HIGH|MEDIUM|LOW"
    }
  ],
  "resolved_gaps": [
    {
      "gap_id": "string",
      "resolved_by_claim_ids": []
    }
  ],
  "contradictions_found": [
    {
      "existing_claim_id": "string",
      "new_claim_text": "string",
      "description": "string"
    }
  ],
  "narrative_summary": "string",
  "self_assessed_progress": 0
}
```

**Why:** This separates what the LLM produces (deltas) from what the system accumulates (full state). The bash orchestrator merges deltas into state using jq. This is the same pattern as GPT Researcher's planner-executor-publisher pipeline -- the LLM is the executor, bash is the publisher.

---

## What NOT to Use and Why

### Do NOT Use: Python/Node.js Frameworks

| Framework | Why Not |
|-----------|---------|
| LangChain | Adds Python dependency to a bash-native system. The entire Aether ecosystem runs without Python. Oracle improvements must compose with existing aether-utils.sh. |
| LangGraph | Same -- Python dependency. The graph-based research pattern (FlowSearch) can be implemented with JSON + jq without a framework. |
| CrewAI | Multi-agent Python framework. Aether already has its own agent spawning system. |
| AutoGen | Microsoft's multi-agent framework. Wrong language, wrong paradigm for this codebase. |
| OpenAI Agents SDK | Wrong vendor (Aether uses Claude), Python-only. |
| Vercel AI SDK | TypeScript. Would require adding a build step to a zero-dependency bash tool. |

**Rationale:** The oracle runs in ANY repo where Aether is installed. Adding a runtime dependency (Python, Node) would break portability. The Claude CLI already provides the AI reasoning engine. Bash + jq + Claude CLI is sufficient for the patterns we need.

### Do NOT Use: External Search APIs

| API | Why Not |
|-----|---------|
| Tavily | Paid API, requires API key management. Claude CLI's built-in WebSearch tool already handles web search within each iteration. |
| Firecrawl | Paid web scraping API. Claude CLI's WebFetch tool handles page content extraction. |
| Serper | Same -- paid, key management overhead. |
| Brave Search API | Requires key management and HTTP client in bash. Claude's native search is simpler. |

**Rationale:** Claude CLI has WebSearch and WebFetch as built-in tools. When running with `--dangerously-skip-permissions` (which oracle.sh already does), these tools work autonomously. Adding external search APIs adds cost, API key management, and HTTP client dependencies for marginal benefit.

### Do NOT Use: Database/Graph Storage

| Technology | Why Not |
|------------|---------|
| SQLite | Adds binary dependency. JSON files + jq provide sufficient structured storage for research state. |
| Neo4j | Massive overhead for a knowledge graph that lives for the duration of one research session. |
| Redis | In-memory store adds operational complexity. File-based state with atomic writes (already in aether-utils.sh) is simpler and more portable. |
| Graphiti | Knowledge graph library requiring Python + Neo4j. Way too heavy. |

**Rationale:** Research state is session-scoped (minutes to hours), not persistent across weeks. JSON files are the right storage for this scope. The existing `atomic-write.sh` utility provides safe writes without race conditions.

### Do NOT Use: Session Continuity (`--continue` / `--resume`)

| Feature | Why Not |
|---------|---------|
| `claude -p --continue` | Accumulates context across iterations, leading to "context rot" -- the exact problem the Ralph Loop pattern was designed to solve. Each iteration MUST start fresh with bounded context. |
| `claude -p --resume $session_id` | Same issue. We want fresh context per iteration with state on disk, not in the LLM's context window. |

**Rationale:** This is counterintuitive but critical. Every successful autonomous loop pattern (Ralph, GPT Researcher, Anthropic's own system) uses fresh context per iteration. The reason: LLM performance degrades as context fills, and errors compound. By reading state from disk and writing deltas back, each iteration gets optimal attention across the entire research state.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Orchestrator | Bash script (enhanced oracle.sh) | Python agent framework | Breaks Aether's zero-dependency bash portability |
| State format | JSON (research-state.json) | Markdown (current progress.md) | Can't be queried, merged, or validated programmatically. LLM appends are unreliable (formatting drift, duplication). |
| Iteration output | `--json-schema` enforced JSON | Free-text + grep for sentinels | Current approach. Produces unreliable, unparseable output. No structured claims or sources. |
| Source verification | Multi-source consensus via prompt engineering | External fact-checking API | No reliable free API exists. Prompt engineering the LLM to cross-reference and score is the pattern used by all major deep research systems. |
| Completion detection | Convergence tracking (claims-per-iteration) | LLM self-assessed confidence % | LLMs are bad at self-calibrating confidence. Convergence tracking (diminishing new claims) is an objective measure. |
| Knowledge accumulation | Claim-based JSON with source links | Append-only markdown log | Current approach. Produces unstructured, duplicative output that can't be synthesized into a coherent report. |
| Report generation | Final synthesis pass from structured state | Progressive append to report file | Append produces disjointed prose. A single synthesis pass from complete state produces coherent, well-structured reports. |

---

## Integration Points with Existing Aether

| Aether Component | Integration | How |
|------------------|-------------|-----|
| `aether-utils.sh` | New subcommands | Add `oracle-state-merge`, `oracle-convergence-check`, `oracle-report-generate` subcommands |
| `atomic-write.sh` | State file safety | Use existing atomic write utility for research-state.json updates |
| `file-lock.sh` | Concurrency safety | Use existing file lock for state mutations (prevents corruption if user runs multiple oracles) |
| `/ant:oracle` command | Enhanced wizard | Update research.json config to include new parameters (breadth, depth, convergence threshold) |
| `oracle.md` prompt | Major rewrite | New prompt must instruct Claude to read research-state.json, identify gaps, produce structured delta output |
| `discoveries/` directory | Actionable findings | Final report extraction writes machine-actionable discoveries here |
| Colony instincts | Knowledge promotion | High-confidence findings can be promoted to colony instincts via existing `instinct-add` subcommand |
| Pheromone system | Research signals | Oracle findings can generate FOCUS pheromones for related build work |

---

## Installation / Dependencies

```bash
# No new dependencies required. Verify existing ones:
which jq          # JSON processor (already required by aether-utils.sh)
which claude      # Claude CLI (already required by oracle.sh)

# Optional but recommended:
which tmux        # Background execution (already used by oracle.sh)
```

**Zero new dependencies.** This is a key constraint. The upgrade uses existing tools (bash, jq, claude CLI) with new patterns (structured output, schema enforcement, gap-driven iteration).

---

## Version Compatibility Notes

| Component | Minimum Version | Required Feature |
|-----------|----------------|-----------------|
| Bash | 3.2 | Arrays, string manipulation. Already tested across macOS/Linux. |
| jq | 1.5 | `--slurpfile`, `--argjson`. Version 1.6+ preferred for better error messages. |
| Claude CLI | Current (2026) | `--output-format json`, `--json-schema`, `--append-system-prompt`. These are recent additions -- verify availability. |

**Risk: Claude CLI `--json-schema` flag availability.** This is the most critical dependency. If `--json-schema` is not available in the user's Claude CLI version, the fallback is to use `--output-format json` with schema instructions in the prompt (less reliable but functional). Need to verify during implementation.

**Confidence: MEDIUM** -- The `--json-schema` flag was verified in official docs (code.claude.com/docs/en/headless) but availability may vary by Claude CLI version. The fallback approach (prompt-based JSON enforcement) is well-documented in Anthropic's structured output docs.

---

## Sources

### HIGH Confidence (Official Documentation)
- [Claude Code Headless Mode / Programmatic Usage](https://code.claude.com/docs/en/headless) -- Official docs for `--print`, `--output-format json`, `--json-schema`, `--continue`, `--resume`
- [Claude Structured Outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs) -- Official docs for JSON schema enforcement

### HIGH Confidence (Anthropic Engineering)
- [How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) -- Anthropic's own architecture for orchestrator-worker research pattern, task decomposition, source quality heuristics

### MEDIUM Confidence (Verified Multi-Source)
- [Ralph / snarktank](https://github.com/snarktank/ralph) -- Original Ralph Loop pattern: fresh context per iteration, file-based state, completion sentinels
- [From ReAct to Ralph Loop](https://www.alibabacloud.com/blog/from-react-to-ralph-loop-a-continuous-iteration-paradigm-for-ai-agents_602799) -- Context rot problem, file-based state vs token-based state
- [Self-Improving Coding Agents (Addy Osmani)](https://addyosmani.com/blog/self-improving-agents/) -- AGENTS.md pattern, four-channel persistent memory, knowledge accumulation best practices
- [GPT Researcher](https://github.com/assafelovic/gpt-researcher) -- Planner-executor-publisher pipeline, 20+ source aggregation, multi-source consensus
- [dzhng/deep-research](https://github.com/dzhng/deep-research) -- Recursive depth/breadth parameters, gap-driven follow-up, markdown report generation

### MEDIUM Confidence (Academic / ArXiv)
- [FlowSearch: Dynamic Structured Knowledge Flow](https://arxiv.org/html/2510.08521v1) -- DAG-based research state, six graph transformation operations, JSON node/edge representation
- [Deep Research Agents: A Systematic Examination](https://arxiv.org/html/2506.18096v1) -- Failure taxonomies, evaluation dimensions (Information Recall, Analysis, Presentation)
- [From Fluent to Verifiable: Claim-Level Auditability](https://arxiv.org/html/2602.13855) -- Provenance graphs, entailment strength scoring, source reliability tracking

### LOW Confidence (Single Source, Needs Validation)
- [Multi-Agent Deep Research Architecture (Trilogy AI)](https://trilogyai.substack.com/p/multi-agent-deep-research-architecture) -- Convergence thresholds, coverage-based termination, knowledge graph versioning
- [DeepFact: Benchmarks for Research Factuality](https://arxiv.org/html/2603.05912v1) -- Factuality scoring methods for deep research agents
