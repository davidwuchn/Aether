# Feature Research: Deep Research Engine (Oracle v2)

**Domain:** Autonomous AI deep research systems
**Researched:** 2026-03-13
**Confidence:** MEDIUM-HIGH

## Current Oracle State (Baseline)

Before mapping the landscape, here is what the current Oracle already has:

| Capability | Current State | Gap |
|-----------|---------------|-----|
| Iterative loop | Bash for-loop, fresh AI instance per iteration | No memory between iterations except flat file append |
| Research config | Interactive wizard (topic, depth, confidence, scope) | No strategy config (breadth vs depth) |
| Progress tracking | Flat progress.md append log | No structure, no knowledge graph, no gap tracking |
| Completion detection | Grep for `<oracle>COMPLETE</oracle>` sentinel | No nuanced confidence scoring per sub-question |
| Stop mechanism | .stop file for user interruption | No mid-session steering, no pause/refine/resume |
| Source handling | None | No source tracking, no citation, no verification |
| Output format | Flat markdown append | No structured report, no sections, no synthesis |
| Knowledge integration | None | Research does not feed back into colony state |
| Search strategy | Single linear pass through questions | No adaptive strategy, no branching, no backtracking |

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features that every competent deep research tool provides. Without these, the Oracle feels like a toy.

| # | Feature | Why Expected | Complexity | Notes |
|---|---------|--------------|------------|-------|
| T1 | **Sub-question decomposition** | Every major deep research tool (OpenAI, Perplexity, Gemini, Open Deep Research) breaks the top-level question into 3-8 targeted sub-questions. Without this, research stays shallow because each iteration tackles the same vague topic instead of drilling into specifics. | MEDIUM | The current Oracle already generates questions in research.json but does not track which are answered, partially answered, or still open. Need per-question status tracking. |
| T2 | **Iterative gap identification** | After each iteration, the system must identify what it still does not know and what contradictions exist. OpenAI Deep Research calls this "backtracking from dead-ends." Perplexity calls it the "verification" stage. Without it, iterations repeat what is already known instead of deepening. | HIGH | This is the single biggest gap in the current Oracle. The prompt says "focus on filling knowledge gaps" but provides no mechanism to track or identify them. Each iteration reads the entire progress.md and hopes the LLM notices what is missing. |
| T3 | **Source tracking and citation** | AI citation hallucination rates are 18-55% (GPT-4: 18-28%). Users cannot trust research findings without knowing where claims come from. Every commercial deep research tool (Perplexity, Elicit, OpenAI) provides inline citations. | MEDIUM | Each claim should link to a source URL. Sources should be collected in a dedicated section. Does not need full academic citation formatting -- URL + title + date is sufficient for a dev tool. |
| T4 | **Structured output (not flat append)** | Current Oracle appends to a single flat progress.md. Every commercial tool produces a structured report with sections, headings, executive summary, and organized findings. Flat append creates an unreadable wall of text that makes the research output nearly unusable. | MEDIUM | The output should be a synthesized report, not a log of iterations. Iterations are intermediate work; the final output should be a clean document. |
| T5 | **Per-question confidence scoring** | Current Oracle has a single global confidence threshold (e.g., 95%). Real deep research systems track confidence per sub-question so that unanswered areas get more attention and well-covered areas stop consuming iterations. OpenAI Deep Research targets "2+ independent sources per sub-question." | MEDIUM | Track confidence per sub-question (0-100%). Use per-question confidence to drive which areas get researched next. Global confidence = weighted average of sub-question confidences. |
| T6 | **Research plan visibility** | Users need to see what the system plans to research, what it has covered, and what remains. Enterprise Deep Research (EDR) uses a visible todo.md as both execution plan and progress tracker. Without this, the user has no idea what the Oracle is doing or whether it is stuck in a loop. | LOW | Expose the research plan as a readable artifact (e.g., research-plan.md with status per question). Update after each iteration. |
| T7 | **Source verification (cross-referencing)** | Claims should be verified across multiple independent sources before being stated as findings. Single-source claims should be flagged as low confidence. This is table stakes because AI hallucination rates are so high that unverified claims actively harm the user. | HIGH | At minimum: flag claims supported by only one source. Ideal: require 2+ sources for key claims. Track source count per finding. |
| T8 | **Graceful interruption with partial results** | Current .stop file halts the loop but does not synthesize what was found. Users should get a useful partial report even when stopping early or hitting max iterations. | LOW | On stop/max-iterations: run a synthesis pass that organizes current findings into the structured output format. |

### Differentiators (Competitive Advantage)

Features that elevate Oracle beyond what basic AI research tools offer. These are not expected, but make the system genuinely powerful.

| # | Feature | Value Proposition | Complexity | Notes |
|---|---------|-------------------|------------|-------|
| D1 | **Configurable search strategy (breadth vs depth)** | Users should choose whether to explore broadly (many sub-questions, moderate depth each) or deeply (fewer sub-questions, exhaustive coverage). Research shows hybrid strategies perform best: "explore widely first, then think deeply only when it matters." No open-source tool currently exposes this as a user-facing config. | MEDIUM | Three modes: breadth-first (cover all sub-questions at surface level, then deepen), depth-first (fully answer one sub-question before moving to next), adaptive (system decides based on what it finds). Default to adaptive. |
| D2 | **Mid-session steering** | Enterprise Deep Research's core innovation. Users can redirect research mid-session by sending natural language directives (e.g., "focus on peer-reviewed sources," "stop looking at X, investigate Y instead"). Current Oracle has no mechanism for this -- .stop is binary. Aether already has the pheromone system (FOCUS/REDIRECT/FEEDBACK) which maps perfectly to this need. | HIGH | Read pheromone signals between iterations. FOCUS = prioritize this sub-area. REDIRECT = stop researching X, investigate Y. FEEDBACK = adjust approach. This makes Aether's existing pheromone system the steering mechanism for research. |
| D3 | **Knowledge graph construction** | As research progresses, build a graph of entities, relationships, and claims. Recent research (Agentic Deep Graph Reasoning, 2025) shows knowledge graphs enable discovery of "bridge nodes" -- concepts that connect disparate knowledge areas. This turns research from a flat document into a navigable knowledge structure. | HIGH | Build incrementally across iterations. Nodes = concepts/entities. Edges = relationships. Store as JSON. Enables: finding connections the user did not ask about, identifying under-explored areas, visualizing the research landscape. |
| D4 | **Colony knowledge integration** | Research findings should feed back into Aether's colony state. Discovered patterns become instincts. Key findings become learnings. Confidence-scored claims become reference material for builders. No other research tool does this because no other tool has a colony system. This is Aether's unique advantage. | MEDIUM | After research completes: extract high-confidence findings as colony learnings, extract patterns as colony instincts, store the full report in an accessible location. Wire into the existing event/learning/instinct system. |
| D5 | **Research strategy templates** | Pre-built research strategies for common patterns: "Technology evaluation" (compare options), "Architecture review" (analyze codebase patterns), "Bug investigation" (root cause analysis), "Best practices survey" (community patterns). Each template pre-configures questions, search strategy, and output format. | LOW | Templates are just pre-filled research.json configs with tailored questions and output format. Easy to implement once the core engine supports structured config. |
| D6 | **Parallel sub-question research** | Instead of sequential iterations that address one question at a time, spawn parallel research threads for independent sub-questions. Open Deep Research uses this pattern with its Supervisor-Researcher architecture. Dramatically reduces wall-clock time for broad research. | HIGH | Requires spawning multiple AI instances (which Aether's existing tmux-based build system already supports). Independent sub-questions can be researched in parallel. Dependent ones remain sequential. Use the DAG approach from the Egnyte architecture. |
| D7 | **Source credibility scoring** | Assign trust scores to sources based on domain authority (academic journals > blogs), recency (newer > older for tech topics), and cross-reference frequency. Weight findings by source credibility in the final synthesis. | MEDIUM | Domain-based scoring: .edu/.gov/official docs = high, established tech blogs = medium, random forums = low. Recency scoring: configurable decay. Aggregate score per claim = sum of source credibility scores. |
| D8 | **Reflection and self-correction loop** | After each iteration, a dedicated reflection step evaluates: Did this iteration actually advance understanding? Are we going in circles? Should we change strategy? Enterprise Deep Research calls this "structured evaluation after each loop." Prevents the common failure mode where iterations repeat similar content. | MEDIUM | Add a meta-evaluation step between iterations that compares new findings against existing knowledge. If overlap > threshold, flag "diminishing returns" and suggest strategy change. Track iteration productivity metrics. |
| D9 | **Multi-scope research (codebase + web + docs)** | The current Oracle already supports scope selection (codebase/web/both). Enhance this with intelligent scope-switching: start with codebase analysis, identify unknowns, automatically pivot to web research for those unknowns, then return to codebase to verify applicability. | MEDIUM | Scope becomes dynamic per sub-question rather than global per session. Some questions are best answered by code, others by docs, others by web. Let the system decide per-question. |
| D10 | **Research artifacts (discoverable outputs)** | Beyond the main report, produce discrete artifacts: comparison tables, decision matrices, architecture diagrams (as text), glossaries, reference lists. These are individually useful outputs that can be referenced later without re-reading the full report. | LOW | Store in .aether/oracle/discoveries/ (directory already exists but unused). Each artifact is a small standalone file. Makes research outputs more reusable and discoverable. |

### Anti-Features (Deliberately NOT Building)

| # | Feature | Why Requested | Why Problematic | Alternative |
|---|---------|---------------|-----------------|-------------|
| A1 | **Real-time web scraping / browser automation** | "The agent should be able to browse real websites like a human." | Massive complexity. Browser automation is fragile, slow, and requires headless browser infrastructure. OpenAI can do this because they have dedicated infrastructure. A CLI tool should not attempt it. | Use WebSearch and WebFetch tools which are already available in the AI CLI. These handle 95% of web research needs without browser automation overhead. |
| A2 | **Academic database integration (PubMed, arXiv APIs)** | "Should search academic papers directly." | Adds significant complexity for a development research tool. Most Oracle use cases are technology evaluation, architecture review, and codebase analysis -- not academic literature review. | Use web search which surfaces academic results when relevant. If a user needs academic-specific research, they should use Elicit or Semantic Scholar directly. |
| A3 | **Real-time collaboration / multi-user steering** | "Multiple team members should steer research simultaneously." | Aether is a single-developer CLI tool. Multi-user coordination adds massive complexity (conflict resolution, permissions, real-time sync) for marginal value. | Single-user pheromone-based steering covers the use case. Team members can review results after completion. |
| A4 | **Autonomous scope expansion** | "The agent should discover related topics and research those too." | Runaway scope expansion is the number one failure mode in research agents. An agent that discovers "related" topics will explore indefinitely, burning tokens and producing unfocused output. | Strictly scope research to user-defined questions. If the agent discovers a relevant adjacent topic, add it to a "suggested follow-up" list rather than auto-researching it. |
| A5 | **Custom LLM model selection per research phase** | "Use GPT-4 for analysis and Claude for synthesis." | Adds configuration complexity. The Oracle already uses whatever CLI is available (claude or opencode). Multi-model orchestration requires API key management, model-specific prompt engineering, and fallback logic. | Use the single available AI CLI. Model quality improvements come from better prompts, not model switching. |
| A6 | **PDF/image analysis** | "Should read PDFs and analyze charts." | Requires multimodal capabilities that may not be available in all AI CLIs. Adds complexity for edge cases. | If the AI CLI supports multimodal input, it works automatically through WebFetch. Do not build custom PDF parsing infrastructure. |
| A7 | **Persistent cross-session research memory** | "The Oracle should remember all previous research sessions." | Creates unbounded context growth. Old research becomes stale. Mixing current and historical research confuses findings. | Archive completed research (already implemented). Colony integration (D4) captures the durable knowledge. Full session history stays in archive/ for manual reference. |

---

## Feature Dependencies

```
[T1] Sub-question decomposition
    +--requires--> [T5] Per-question confidence scoring
    +--requires--> [T6] Research plan visibility
    +--enables---> [D1] Configurable search strategy
    +--enables---> [D6] Parallel sub-question research

[T2] Iterative gap identification
    +--requires--> [T1] Sub-question decomposition
    +--requires--> [T5] Per-question confidence scoring
    +--enables---> [D8] Reflection and self-correction

[T3] Source tracking
    +--enables---> [T7] Source verification
    +--enables---> [D7] Source credibility scoring

[T4] Structured output
    +--enables---> [D10] Research artifacts
    +--enhanced-by-> [T3] Source tracking (citations in output)

[T5] Per-question confidence
    +--enables---> [T2] Iterative gap identification (know WHERE gaps are)
    +--enables---> [D1] Search strategy (know where to go deeper)

[D2] Mid-session steering
    +--requires--> [T6] Research plan visibility (must see plan to steer it)
    +--integrates-> Aether pheromone system (FOCUS/REDIRECT/FEEDBACK)

[D3] Knowledge graph
    +--requires--> [T1] Sub-question decomposition
    +--requires--> [T3] Source tracking
    +--enables---> [D4] Colony knowledge integration

[D4] Colony knowledge integration
    +--requires--> [T5] Per-question confidence (only integrate high-confidence findings)
    +--requires--> [T4] Structured output (need organized findings to extract from)
    +--integrates-> Aether colony state (instincts, learnings, events)

[D6] Parallel sub-question research
    +--requires--> [T1] Sub-question decomposition
    +--requires--> [T6] Research plan visibility (coordinate parallel threads)

[D8] Reflection loop
    +--requires--> [T2] Gap identification
    +--requires--> [T5] Per-question confidence
    +--enables---> [D1] Adaptive strategy switching
```

### Dependency Notes

- **T1 is the foundation.** Almost every other feature depends on proper sub-question decomposition. Build this first.
- **T5 (confidence scoring) is the second foundation.** Gap identification, strategy selection, and colony integration all depend on knowing confidence levels per question.
- **D2 (mid-session steering) is uniquely Aether.** It leverages the existing pheromone system, making it a natural integration rather than building from scratch.
- **D3 (knowledge graph) and D6 (parallel research) are high-complexity features that depend on multiple table-stakes features being solid first.** Defer these to later phases.
- **D4 (colony integration) is the ultimate payoff.** It is what makes Oracle research valuable beyond the session. But it requires structured output (T4) and confidence scoring (T5) to extract meaningful knowledge.

---

## MVP Definition

### Launch With (v2.0 -- Core Engine)

The minimum set of features that makes Oracle v2 meaningfully better than v1.

- [x] **T1: Sub-question decomposition** -- Break topic into tracked sub-questions with status (open/partial/answered)
- [x] **T2: Iterative gap identification** -- After each iteration, identify what is still unknown and target next iteration at the biggest gap
- [x] **T5: Per-question confidence scoring** -- Track confidence (0-100%) per sub-question; use to drive iteration focus
- [x] **T6: Research plan visibility** -- Expose research-plan.md showing questions, status, confidence, and next steps
- [x] **T4: Structured output** -- Final synthesis into organized report with sections, executive summary, and findings organized by sub-question
- [x] **T8: Graceful interruption** -- On stop/max-iterations, synthesize partial results into structured output

**Why this set:** These six features transform Oracle from "append random findings to a flat file" into "systematically deepen understanding with visible progress." Every feature in this set addresses the core complaint: "iterations don't build on each other meaningfully."

### Add After Validation (v2.1 -- Trust and Steering)

Features to add once the core iterative engine is working and proven.

- [ ] **T3: Source tracking** -- When the core iteration loop works, add source URL collection and inline citation
- [ ] **T7: Source verification** -- Cross-reference claims across sources; flag single-source claims as low confidence
- [ ] **D2: Mid-session steering** -- Integrate pheromone signals (FOCUS/REDIRECT) as research directives between iterations
- [ ] **D1: Configurable search strategy** -- Expose breadth/depth/adaptive strategy selection in the wizard
- [ ] **D8: Reflection loop** -- Meta-evaluation step to detect diminishing returns and suggest strategy changes

**Trigger for adding:** Core engine produces structured, gap-aware research but users want more control and more trustworthy outputs.

### Future Consideration (v2.2+ -- Advanced)

Features to defer until the core engine and trust layer are battle-tested.

- [ ] **D4: Colony knowledge integration** -- Requires stable structured output and reliable confidence scoring; defer because the colony integration API needs careful design
- [ ] **D5: Research strategy templates** -- Easy to build but low urgency; the wizard already handles basic config
- [ ] **D7: Source credibility scoring** -- Useful but requires domain classification heuristics; diminishing returns over basic source verification
- [ ] **D10: Research artifacts** -- Nice-to-have discrete outputs; depends on structured output being mature
- [ ] **D3: Knowledge graph** -- High complexity, high potential, but needs mature sub-question decomposition and source tracking before it adds real value
- [ ] **D6: Parallel sub-question research** -- Requires robust orchestration; current serial approach works and is simpler to debug
- [ ] **D9: Multi-scope research** -- Enhancement to existing scope system; not critical for v2

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| T1: Sub-question decomposition | HIGH | MEDIUM | **P1** |
| T2: Iterative gap identification | HIGH | HIGH | **P1** |
| T5: Per-question confidence | HIGH | MEDIUM | **P1** |
| T6: Research plan visibility | HIGH | LOW | **P1** |
| T4: Structured output | HIGH | MEDIUM | **P1** |
| T8: Graceful interruption | MEDIUM | LOW | **P1** |
| T3: Source tracking | HIGH | MEDIUM | **P2** |
| T7: Source verification | HIGH | HIGH | **P2** |
| D2: Mid-session steering | HIGH | HIGH | **P2** |
| D1: Search strategy config | MEDIUM | MEDIUM | **P2** |
| D8: Reflection loop | MEDIUM | MEDIUM | **P2** |
| D4: Colony integration | HIGH | MEDIUM | **P3** |
| D5: Strategy templates | LOW | LOW | **P3** |
| D7: Source credibility | MEDIUM | MEDIUM | **P3** |
| D10: Research artifacts | LOW | LOW | **P3** |
| D3: Knowledge graph | MEDIUM | HIGH | **P3** |
| D6: Parallel research | MEDIUM | HIGH | **P3** |
| D9: Multi-scope research | LOW | MEDIUM | **P3** |

**Priority key:**
- P1: Must have for v2 launch (core engine)
- P2: Should have, add in v2.1 (trust and steering)
- P3: Nice to have, future consideration (advanced)

---

## Competitor Feature Analysis

| Feature | OpenAI Deep Research | Perplexity Deep Research | Google Gemini Deep Research | LangChain Open Deep Research | Stanford STORM | Current Aether Oracle |
|---------|---------------------|--------------------------|----------------------------|------------------------------|----------------|----------------------|
| Sub-question decomposition | Yes (implicit via o3 reasoning) | Yes (3-5 sequential searches) | Yes (multi-step) | Yes (section-based plan) | Yes (perspective-based) | Partial (questions in config, no tracking) |
| Iterative gap identification | Yes (backtracking from dead-ends) | Yes (verification stage) | Yes | Yes (feedback loops) | Yes (via multi-perspective gaps) | No |
| Source tracking & citation | Yes (inline) | Yes (inline, most transparent) | Yes | Yes | Yes (Wikipedia-style) | No |
| Structured output | Yes (full report) | Yes (narrative with citations) | Yes (full report) | Yes (section-based report) | Yes (Wikipedia-style article) | No (flat append log) |
| Confidence scoring | Implicit (chain of thought) | Implicit | Implicit | No | No | Single global threshold only |
| Mid-session steering | Yes (interrupt and refine) | Limited (follow-up questions) | Limited | No | No | No (.stop is binary) |
| Source verification | Yes (2+ sources per claim) | Yes (conflicting claims flagged) | Yes | Partial | Yes (grounded in sources) | No |
| Search strategy config | No (automatic) | No (automatic) | No (automatic) | Yes (via config) | No | No |
| Knowledge graph | No | No | No | No | No | No |
| Colony/workflow integration | No | No | No | Partial (via LangGraph state) | No | No |
| Parallel research | Implicit | Yes (parallel subtopic search) | Yes | Yes (parallel researchers) | Yes (parallel conversations) | No |
| Research plan visibility | Yes (real-time progress tracking) | Limited | Limited | Yes (via LangGraph nodes) | Partial (outline visible) | No |
| Cost per session | ~$0.50-2.00 (API) | $20/mo subscription | $19.99/mo subscription | Self-hosted (LLM costs) | Self-hosted (LLM costs) | Self-hosted (CLI costs) |

### Key Competitive Insights

1. **Sub-question decomposition + gap identification is universal.** Every serious tool does this. The current Oracle does not. This is the clearest gap to close.
2. **Source tracking is universal.** Every serious tool provides citations. Oracle has none. This is table stakes.
3. **Mid-session steering is rare.** Only OpenAI offers real-time steering. Enterprise Deep Research (academic) provides it. This is a genuine differentiator that Aether's pheromone system enables naturally.
4. **Knowledge graph is absent everywhere.** No commercial or open-source deep research tool builds a knowledge graph during research. This is a potential differentiator for v2.2+.
5. **Colony/workflow integration is unique to Aether.** No other tool feeds research findings back into a development workflow. This is Aether's strongest competitive advantage.
6. **Search strategy configuration is rare.** Only Open Deep Research exposes this. Most tools are black-box. Making strategy visible and configurable is a differentiator.

---

## Sources

### Primary Sources (HIGH confidence)
- [OpenAI: Introducing Deep Research](https://openai.com/index/introducing-deep-research/) -- Official feature announcement
- [How OpenAI's Deep Research Works](https://blog.promptlayer.com/how-deep-research-works/) -- Technical architecture breakdown
- [Perplexity: Introducing Deep Research](https://www.perplexity.ai/hub/blog/introducing-perplexity-deep-research) -- Official feature announcement
- [LangChain: Open Deep Research](https://blog.langchain.com/open-deep-research/) -- Open-source implementation
- [Stanford STORM](https://github.com/stanford-oval/storm) -- Academic research system
- [Enterprise Deep Research (EDR)](https://arxiv.org/html/2510.17797v1) -- Steerable multi-agent research paper

### Secondary Sources (MEDIUM confidence)
- [Deep Research AI Agents: Complete Guide](https://calmops.com/ai/deep-research-ai-agents-complete-guide/) -- Feature landscape overview
- [Inside the Architecture of a Deep Research Agent](https://www.egnyte.com/blog/post/inside-the-architecture-of-a-deep-research-agent/) -- Architecture patterns
- [Deep Research Agents: A Systematic Examination](https://arxiv.org/html/2506.18096v1) -- Comprehensive survey
- [Karpathy autoresearch](https://github.com/karpathy/autoresearch) -- Autonomous experiment loop patterns
- [Agentic Deep Graph Reasoning](https://arxiv.org/html/2502.13025v1) -- Knowledge graph construction via iterative expansion

### Tertiary Sources (LOW confidence -- patterns only)
- [7 Best AI Tools for Deep Research](https://www.index.dev/blog/ai-tools-for-deep-research) -- Tool comparison
- [Google Deep Research vs Perplexity vs ChatGPT](https://freeacademy.ai/blog/google-deep-research-vs-perplexity-vs-chatgpt-comparison-2026) -- Feature comparison
- [AI Hallucinations in Research](https://www.inra.ai/blog/ai-hallucinations) -- Hallucination rate statistics

---
*Feature research for: Autonomous AI Deep Research Engine (Oracle v2)*
*Researched: 2026-03-13*
