# Project Research Summary

**Project:** Aether Oracle v2 — Deep Research Engine Upgrade
**Domain:** Autonomous iterative AI deep research loops
**Researched:** 2026-03-13
**Confidence:** HIGH

## Executive Summary

Aether Oracle v2 is an upgrade to an existing working bash-based autonomous research loop. The current Oracle (`oracle.sh` + `oracle.md`) uses a proven iterative pattern — spawning fresh Claude CLI instances in a for-loop — but fails at the state management layer: it appends all findings to a flat `progress.md` file with no structure, no gap tracking, and no knowledge synthesis. The result is "iterative appending, not iterative deepening." Every serious deep research system (OpenAI, Perplexity, GPT Researcher, Stanford STORM, Anthropic's own multi-agent research system) solves this through structured, machine-readable state between iterations. This is the core upgrade: replace flat markdown accumulation with a JSON-based knowledge state that each iteration reads, extends, and writes back.

The recommended approach is a zero-new-dependency upgrade. No Python, no Node.js, no external APIs. The existing stack (bash, jq, Claude CLI) is sufficient — what changes is the data architecture. Five files replace the current single `progress.md`: `state.json` (iteration metadata and coverage tracking), `plan.json` (living question tree with per-question confidence), `synthesis.md` (token-budget-constrained compressed summary rewritten each iteration), `gaps.md` (explicit open questions driving next iteration), and `findings/00N-<topic>.md` (per-iteration audit trail). The Claude CLI's `--json-schema` flag enforces structured output from each iteration, making state transitions reliable.

The critical risk is prompt design, not engineering. Getting `iterate.md` right — so every iteration reliably reads all state, focuses on the highest-priority gap, and writes all state updates — determines whether the entire system works. The bash orchestrator and JSON schemas are straightforward; the iteration prompt is where the intelligence lives. A secondary risk is the `--json-schema` flag availability in the user's Claude CLI version. The fallback (prompt-based JSON enforcement) is documented and functional but less reliable. Both risks are manageable and should be caught in Phase 2 testing before the system is used for real research.

## Key Findings

### Recommended Stack

The stack is constrained by a clear principle: Aether Oracle runs in any repo where Aether is installed, so zero new runtime dependencies. This rules out Python frameworks (LangChain, LangGraph, CrewAI), external search APIs (Tavily, Firecrawl), and database storage (SQLite, Neo4j). The upgrade uses three existing tools in new patterns.

**Core technologies:**
- **Bash 3.2+**: Orchestrator and state machine — already proven in oracle.sh across the entire Aether ecosystem
- **Claude CLI (current)**: AI reasoning engine per iteration — `--output-format json`, `--json-schema`, and `--append-system-prompt` flags unlock structured output and dynamic context injection
- **jq 1.6+**: JSON state manipulation — already a dependency of aether-utils.sh; handles all structured state read/write via existing `atomic-write.sh` utility

**Critical note on `--continue` / `--resume`:** Do NOT use these flags. Counterintuitively, accumulating context across iterations causes "context rot" — the exact problem this upgrade solves. Each iteration must start fresh with bounded context read from disk files.

### Expected Features

The feature research mapped the current Oracle against every major deep research tool and identified a clear MVP set. T1 (sub-question decomposition) is the foundation — almost every other feature depends on it.

**Must have for v2.0 (core engine):**
- **T1: Sub-question decomposition** — break topic into tracked sub-questions with status (open/partial/answered); every serious tool does this, Oracle does not
- **T2: Iterative gap identification** — after each iteration, identify what is still unknown and target the next iteration at the biggest gap; this is the single biggest current gap
- **T5: Per-question confidence scoring** — track confidence per sub-question (0-100%) to drive iteration focus; replaces unreliable single global threshold
- **T6: Research plan visibility** — expose research-plan.md with question status and next steps so users can see what Oracle is doing
- **T4: Structured output** — final synthesis into organized report with sections and executive summary; replace the current flat append log
- **T8: Graceful interruption** — on stop/max-iterations, synthesize partial results into structured output

**Should have for v2.1 (trust and steering):**
- **T3: Source tracking and citation** — URL + title per claim; AI citation hallucination rates are 18-55%, unverified claims actively harm users
- **T7: Source verification** — cross-reference claims across sources; flag single-source claims as low confidence
- **D2: Mid-session steering** — integrate Aether pheromone signals (FOCUS/REDIRECT/FEEDBACK) as research directives between iterations; this is uniquely Aether and uses existing infrastructure
- **D1: Configurable search strategy** — expose breadth/depth/adaptive strategy selection
- **D8: Reflection loop** — meta-evaluation to detect diminishing returns and suggest strategy changes

**Defer to v2.2+ (advanced):**
- D4: Colony knowledge integration — requires stable structured output and reliable confidence scoring first
- D3: Knowledge graph — high complexity, high potential, but needs mature foundations
- D6: Parallel sub-question research — requires robust orchestration; serial approach works and is simpler to debug
- D5, D7, D9, D10: Strategy templates, source credibility scoring, multi-scope research, research artifacts

**Explicit anti-features (do not build):**
- Real-time browser automation (fragile infrastructure, WebSearch/WebFetch already covers 95% of needs)
- Persistent cross-session memory (creates stale knowledge, colony integration covers durable findings)
- Autonomous scope expansion (the number-one research agent failure mode)

### Architecture Approach

The architecture is a five-layer system: Command Layer (oracle.md wizard), Orchestrator (oracle.sh bash loop), State Layer (JSON files), Knowledge Layer (markdown files), and Control Files (sentinel signals). The key pattern is the Structured State Bridge: JSON files are the handoff protocol between stateless AI iterations. Each iteration reads `state.json + plan.json + synthesis.md + gaps.md`, does focused research, writes `findings/00N.md` plus updated state, and exits. The bash loop checks convergence signals and spawns the next iteration. State flows are strictly separated: config is immutable (research.json), iteration state is mutable (state.json), knowledge accumulates in findings/ and is compressed into synthesis.md (rewritten each iteration to stay within a token budget).

**Major components:**
1. **Command Layer (oracle.md)** — interactive wizard, status display, stop/pause control; updated to show new state format
2. **State Layer (state.json + plan.json)** — living iteration metadata with coverage tracking and question tree; the primary handoff mechanism between iterations
3. **Iteration Prompt (iterate.md)** — phase-specific prompts that instruct the AI to read state, research the highest-priority gap, and write structured updates; this is the hardest part and the critical path
4. **Orchestrator (oracle.sh)** — multi-signal convergence detection (coverage completeness, gap resolution rate, findings size trend, stall count) replacing single self-reported confidence threshold
5. **Knowledge Layer (synthesis.md + findings/ + gaps.md)** — compressed synthesis rewritten each iteration plus per-iteration audit trail plus explicit gap tracking

### Critical Pitfalls

1. **Append-only progress file causes context rot** — By iteration 15-20, a growing progress.md exceeds what a fresh AI instance can meaningfully process; later iterations restate earlier findings rather than deepening. Fix: separate raw findings (append to `findings/` directory) from compressed synthesis (rewrite `synthesis.md` each iteration with a token budget cap).

2. **Circular research covers the same ground repeatedly** — Without an explicit exploration tracker, each stateless instance independently gravitates to the most salient subtopic, producing redundant iterations. Fix: track question status in state.json (unexplored/in-progress/answered); require each iteration to address the highest-priority unexplored or lowest-confidence question.

3. **Self-assessed confidence terminates research prematurely or never** — LLMs are poorly calibrated on metacognitive confidence; they may claim 95% confidence after 5 shallow iterations, or refuse to exceed 90% on open-ended questions. Fix: use structural completion metrics (coverage percentage, gap resolution rate, novelty rate across last 3 iterations) as primary termination signals; treat self-assessed confidence as one supplementary signal.

4. **Hallucination accumulation compounds across iterations** — An early inaccurate claim gets read by subsequent iterations and treated as established fact, becoming deeply embedded over many iterations. Fix: require source attribution for every factual claim in state.json; include a dedicated Verify phase that specifically seeks counter-evidence; flag any finding with no external source as LOW confidence.

5. **Prompt overloading degrades per-iteration quality** — As requirements grow, a monolithic iterate.md prompt with 10+ concurrent instructions produces worse results than focused prompts with fewer instructions. Fix: use phase-specific prompts (survey prompt, investigate prompt, synthesize prompt, verify prompt) that the orchestrator selects based on current phase from state.json; mirrors Aether's existing split-playbooks pattern.

## Implications for Roadmap

Based on research, the architecture has a clear dependency chain. The state schema must exist before the iteration prompt can reference it. The iteration prompt must work before the orchestrator can detect convergence from state. Only once the core loop is working do trust features (source tracking) and steering features (pheromone integration) become meaningful additions.

### Phase 1: State Architecture Foundation

**Rationale:** Everything downstream depends on the state schema. Getting this wrong poisons every subsequent feature — this is the highest-risk foundational decision. The PITFALLS research explicitly flags this as "Phase 1, must address first." The schema must cover: question coverage, iteration metadata, frontier tracking, dead-ends (negative knowledge), and convergence window.

**Delivers:** `state.json` schema + validation, `plan.json` question tree schema, `gaps.md` format specification, `findings/` directory structure, `synthesis.md` token-budget contract. All tested with `jq` validation.

**Addresses:** T1 (sub-question structure in plan.json), T5 (per-question confidence in state.json), T6 (research-plan.md generated from plan.json)

**Avoids:** Pitfall 1 (append-only bloat), Pitfall 2 (circular research via exploration tracker), Pitfall 6 (lost negative knowledge via dead-ends tracking)

**Research flag:** Standard patterns — JSON schemas and jq manipulation are well-documented. No additional research needed.

### Phase 2: Iteration Prompt Engineering

**Rationale:** The iterate.md prompt is the hardest part and the critical path per the architecture research. It must reliably instruct a stateless AI to read all state files, focus on the highest-priority gap, do actual research, and write all state updates. Phase-specific prompts (survey / investigate / synthesize / verify) keep each prompt focused and within the model's attention budget.

**Delivers:** `iterate.md` (or phase-specific variants), tested across multiple iteration cycles with verifiable state transitions. State files consistently valid after each iteration.

**Addresses:** T2 (iterative gap identification — prompt enforces gap-targeting), T4 (structured output — synthesis phase prompt produces organized report), T8 (graceful interruption — synthesis phase callable on demand)

**Avoids:** Pitfall 3 (unreliable self-assessed confidence — structural completion criteria in prompt), Pitfall 4 (iterative appending not deepening — phase-aware prompting), Pitfall 7 (prompt overloading — phase-specific prompts under 1,500 tokens each)

**Research flag:** Needs validation — prompt engineering for structured state output requires iteration. Plan for multiple test cycles with real research topics before declaring stable. The `--json-schema` flag availability needs early verification.

### Phase 3: Orchestrator Upgrade

**Rationale:** Once iterate.md reliably produces valid state transitions, the orchestrator can make intelligent decisions. Multi-signal convergence detection replaces the current single confidence threshold. State validation after each iteration (jq check) prevents cascade failures from malformed JSON.

**Delivers:** Updated `oracle.sh` with: multi-signal convergence detection (coverage completeness + gap resolution rate + findings size trend + stall count + self-assessed confidence), state validation after each iteration, findings directory management, phase-transition logic.

**Addresses:** T5 (per-question confidence aggregated for convergence), T8 (graceful interruption — synthesis triggered on stop/max-iterations)

**Avoids:** Pitfall 3 (unreliable confidence — multi-signal detection), Pitfall 5 (hallucination accumulation — stall detection flags suspicious non-progress)

**Research flag:** Standard patterns — bash loop mechanics and jq-based state validation are well-documented. Convergence threshold numbers will need empirical tuning.

### Phase 4: Source Tracking and Trust Layer

**Rationale:** Source tracking is table stakes per the feature research (every serious tool provides it), but it depends on the core iteration loop being stable first. Adding source requirements to an unstable iteration prompt would compound complexity. Once Phase 2 is validated, source fields can be added to the state schema and iteration prompts.

**Delivers:** Source attribution for every factual claim in state.json (URL + title + type + credibility), inline citations in final report, single-source claim flagging, Verify phase prompt seeking counter-evidence.

**Addresses:** T3 (source tracking), T7 (source verification), Pitfall 5 (hallucination accumulation)

**Avoids:** The anti-pattern of trusting all prior iteration output equally

**Research flag:** Standard patterns for source tracking in the state schema. The credibility scoring heuristics (domain-based: .edu/.gov > established blogs > forums) are straightforward to implement.

### Phase 5: Steering Integration

**Rationale:** D2 (mid-session steering) is Aether's unique differentiator — no commercial deep research tool offers mid-run pheromone-based steering. It maps directly to the existing pheromone system (FOCUS/REDIRECT/FEEDBACK). This phase wires the oracle loop to read pheromone signals between iterations. Deferred to Phase 5 because it requires the core loop (Phases 1-3) to be stable before adding control-plane complexity.

**Delivers:** Updated `oracle.sh` that reads FOCUS/REDIRECT/FEEDBACK pheromones between iterations, `.steer` file support for mid-run directives, per-iteration status output visible in tmux ("Iteration 5/30: Investigating [question]. 3/5 answered."), updated `/ant:oracle` status display.

**Addresses:** D2 (mid-session steering), D1 (configurable search strategy via FOCUS signals)

**Avoids:** Pitfall 8 (no mid-run steering — oracle can wander for 30+ iterations without correction)

**Research flag:** Integration-specific — pheromone file format is already documented in Aether. The key integration pattern (check pheromones in bash loop between iterations, NOT inside Claude instance) is documented in the pitfalls. No external research needed.

### Phase 6: Colony Knowledge Integration

**Rationale:** D4 (colony knowledge integration) is the ultimate payoff — research findings promoting to colony instincts, learnings, and pheromones. Deferred last because it requires reliable structured output (Phase 1-2), trustworthy confidence scoring (Phase 4), and stable steering (Phase 5). The colony integration API needs careful design to avoid corrupting COLONY_STATE.json.

**Delivers:** Post-research pipeline that extracts high-confidence findings as colony learnings, patterns as colony instincts, and stores the full report accessibly. Wired into existing `instinct-add`, `learning-add` subcommands.

**Addresses:** D4 (colony knowledge integration)

**Avoids:** Accidental state corruption (oracle remains sandboxed to .aether/oracle/ during research; only writes to colony state explicitly post-completion)

**Research flag:** Needs design work — the integration API between oracle findings and colony state needs careful boundary definition. Consider a `/ant:oracle integrate` command as a deliberate human-triggered step rather than automatic post-research execution.

### Phase Ordering Rationale

- Phases 1-2 are strictly sequential and the critical path: state schema must exist before the iteration prompt can reference it.
- Phase 3 depends on Phase 2 being validated: convergence detection is meaningless if iterate.md doesn't produce reliable state.
- Phase 4 can begin in parallel with late Phase 3 testing: source schema fields can be designed while orchestrator is being tuned.
- Phase 5 is independent of Phase 4 and can be developed in parallel once Phase 3 is stable.
- Phase 6 requires all prior phases because it consumes the final structured output from the complete system.
- The architecture research provides an explicit build order that matches this phase sequence exactly.

### Research Flags

Phases needing deeper research during planning:
- **Phase 2 (Iteration Prompt):** Prompt engineering for reliable structured state output requires empirical testing. Plan multiple test cycles. Verify `--json-schema` Claude CLI flag availability on day one — this is the highest-risk dependency.
- **Phase 6 (Colony Integration):** The boundary between oracle state and colony state needs careful design. A wrong integration could corrupt COLONY_STATE.json. Requires a deliberate API design session before implementation.

Phases with standard patterns (skip additional research):
- **Phase 1 (State Architecture):** JSON schemas and jq manipulation are well-documented. The schemas proposed in STACK.md and ARCHITECTURE.md are complete and can be implemented directly.
- **Phase 3 (Orchestrator):** Bash loop mechanics with multi-signal convergence are established patterns. Threshold tuning is empirical, not research-dependent.
- **Phase 4 (Source Tracking):** Source attribution in JSON state is a straightforward schema extension. Domain-based credibility heuristics don't require additional research.
- **Phase 5 (Steering):** Pheromone integration pattern is fully documented in Aether's existing system and the pitfalls research.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Zero new dependencies. All three core technologies are already in use. Claude CLI flags verified in official docs. Only risk is `--json-schema` availability in specific CLI versions. |
| Features | HIGH | MVP feature set (T1, T2, T4, T5, T6, T8) is directly derived from universal patterns across all major deep research tools. v2.1 and v2.2+ features are clearly bounded. |
| Architecture | HIGH | Five core architectural patterns (Structured State Bridge, Living Research Plan, Synthesis with Compression, Frontier-Based Direction, Multi-Signal Convergence) are validated across RALPH, FS-Researcher, EDR, and production systems with academic backing. |
| Pitfalls | HIGH | All 8 critical pitfalls are verified against multiple independent sources including Anthropic's own engineering blog and direct observation in Aether's oracle archive. Prevention strategies are specific and testable. |

**Overall confidence:** HIGH

### Gaps to Address

- **`--json-schema` flag availability:** Verify this Claude CLI flag is available in the first iteration of Phase 2. If unavailable, fall back to prompt-based JSON enforcement with the `--output-format json` flag and explicit schema instructions in the prompt. This fallback is documented and functional.

- **Convergence threshold numbers:** The specific thresholds (e.g., "fewer than 2 new claims in 3 consecutive iterations") are from a single LOW-confidence source and need empirical tuning. Treat initial values as starting points, not targets. Build observability into Phase 3 to measure actual convergence signal behavior across test research sessions.

- **Phase-specific prompt token budgets:** The "under 1,500 tokens per phase prompt" recommendation comes from general attention budget research, not oracle-specific testing. Validate against actual iteration compliance rates during Phase 2.

- **Colony integration API design:** Phase 6 has no established pattern to follow. The boundary between oracle output and colony state needs deliberate design before implementation begins. Flag for a planning session before Phase 6 starts.

## Sources

### Primary (HIGH confidence)
- [Claude Code Headless Mode](https://code.claude.com/docs/en/headless) — `--print`, `--output-format json`, `--json-schema`, `--append-system-prompt` flags
- [Claude Structured Outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs) — JSON schema enforcement
- [How Anthropic built its multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) — orchestrator-worker pattern, source quality, task decomposition, pitfalls in multi-agent coordination
- [Anthropic: Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — context rot, attention budget depletion, prompt overloading
- [FS-Researcher: File-System-Based Agents](https://arxiv.org/abs/2602.01566) — findings/ + synthesis.md separation pattern
- [Enterprise Deep Research (EDR)](https://arxiv.org/html/2510.17797v1) — steerable multi-agent research, convergence detection, gap tracking
- [Deep Research Agents: Systematic Examination](https://arxiv.org/html/2506.18096v1) — comprehensive survey of architectures and failure taxonomies
- Aether oracle archive (`.aether/oracle/archive/2026-02-16-191250-progress.md`) — direct observation of iterative appending without deepening

### Secondary (MEDIUM confidence)
- [RALPH (snarktank/ralph)](https://github.com/snarktank/ralph) — original bash loop pattern, fresh context per iteration, file-based state
- [From ReAct to Ralph Loop](https://www.alibabacloud.com/blog/from-react-to-ralph-loop-a-continuous-iteration-paradigm-for-ai-agents_602799) — context rot problem, file-based vs token-based state
- [GPT Researcher](https://github.com/assafelovic/gpt-researcher) — planner-executor-publisher pipeline, 20+ source aggregation, multi-source consensus
- [dzhng/deep-research](https://github.com/dzhng/deep-research) — recursive depth/breadth parameters, gap-driven follow-up
- [FlowSearch: Dynamic Structured Knowledge Flow](https://arxiv.org/html/2510.08521v1) — DAG-based research state, gap-driven iteration
- [Karpathy autoresearch](https://github.com/karpathy/autoresearch) — ratcheting mechanism, results.tsv tracking, monotonic improvement
- [OpenAI: Introducing Deep Research](https://openai.com/index/introducing-deep-research/) — feature landscape, backtracking from dead-ends
- [Enterprise Deep Research (EDR)](https://arxiv.org/html/2510.17797v1) — steerable research, visible todo.md, mid-session steering

### Tertiary (LOW confidence — patterns only)
- [Multi-Agent Deep Research Architecture (Trilogy AI)](https://trilogyai.substack.com/p/multi-agent-deep-research-architecture) — convergence thresholds, coverage-based termination
- [Solving LLM Repetition Problem in Production](https://arxiv.org/html/2512.04419v1) — self-reinforcement effect
- [AI Hallucinations in Research](https://www.inra.ai/blog/ai-hallucinations) — hallucination rate statistics (18-55%)

---
*Research completed: 2026-03-13*
*Ready for roadmap: yes*
