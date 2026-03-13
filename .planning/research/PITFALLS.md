# Pitfalls Research

**Domain:** Autonomous AI Deep Research Loops (iterative deepening with stateless LLM instances)
**Researched:** 2026-03-13
**Confidence:** HIGH (verified against Anthropic engineering blog, Karpathy's autoresearch, OpenAI deep research system card, LangChain context management research, and Aether's own oracle archive)

## Critical Pitfalls

### Pitfall 1: Append-Only Progress File Becomes Unreadable

**What goes wrong:**
The progress file grows unboundedly across iterations. Each stateless instance reads the full file, but as it exceeds thousands of lines, later instances cannot effectively process it. The LLM's attention degrades over long contexts ("context rot"), and critical early findings get buried under layers of later, potentially redundant content. In Aether's archived oracle run, 50 iterations produced 258 lines with repetitive section headers ("Iteration 1-5", "Iteration 6-10") where later iterations restated earlier findings rather than building on them.

**Why it happens:**
The oracle.md instruction says "APPEND to progress.md (never replace, always append)." This is a reasonable safety measure (prevents data loss), but without any compaction or restructuring mechanism, the file becomes an append-only log rather than an evolving knowledge base. Anthropic's own engineering blog identifies this pattern: "overly aggressive compression can result in the loss of subtle but critical context whose importance only becomes apparent later," but the opposite -- no compression at all -- causes context rot where everything gets diluted.

**How to avoid:**
Separate the accumulation file from the handoff file. Maintain a raw `findings.md` append-only log for safety, but produce a structured `state.json` (or equivalent) that each iteration reads as its primary input. The state file should contain: (1) answered questions with confidence levels, (2) unanswered questions ranked by priority, (3) a compact summary of key findings, and (4) explicit "do NOT re-investigate" markers. Each iteration rewrites the state file entirely -- it is the iteration's primary output, not an append.

Karpathy's autoresearch solves this elegantly: the agent reads `train.py` (current state) and `results.tsv` (compact log of all attempts), not a growing narrative. The state IS the code; the log is structured data.

**Warning signs:**
- Progress file exceeds 200 lines (roughly 4K tokens) by iteration 10
- Later iterations repeat findings from earlier iterations verbatim
- Confidence score plateaus but iterations keep running
- Each iteration's "new findings" section shrinks relative to its "reading previous findings" overhead

**Phase to address:**
Phase 1 (State Architecture) -- this is the foundational design decision. Getting the state format wrong poisons every subsequent feature.

---

### Pitfall 2: Circular Research (Covering the Same Ground Repeatedly)

**What goes wrong:**
Stateless instances have no memory of what they already investigated. Each reads the progress file, picks what seems most important, and often gravitates toward the same subtopic. The oracle runs 50 iterations but only covers 3-4 distinct research threads because each fresh instance independently concludes the same area is most promising. Anthropic's multi-agent research system documented this exact failure: "three agents investigating the same 2025 semiconductor supply chains separately."

**Why it happens:**
The current oracle.md gives no guidance on which questions to prioritize or how to avoid re-covering ground. It says "Work on ONE focused area per iteration" but does not track which areas have been covered. Without an explicit "what has been explored" manifest, each stateless instance makes the same priority judgment from scratch. LLMs exhibit strong recency and salience biases -- they will repeatedly investigate whatever is most prominently described in the progress file.

**How to avoid:**
Implement an explicit exploration tracker in the state file. Structure it as a research tree where each node tracks: question, status (unexplored/in-progress/answered/blocked), confidence, and iteration last touched. Each iteration's instructions should say: "Pick the highest-priority UNEXPLORED or lowest-confidence IN-PROGRESS question. Do NOT re-investigate questions marked ANSWERED unless you have specific contradicting evidence."

Autoresearch prevents this via the ratcheting mechanism (monotonic improvement on a metric). For research (which lacks a numeric metric), use question-answering coverage as the ratchet: once a question is marked answered at 90%+ confidence, it stays answered unless explicitly challenged.

**Warning signs:**
- Multiple iterations produce findings about the same subtopic
- The progress file has near-duplicate paragraphs across iteration blocks
- Confidence score oscillates rather than monotonically increasing
- New iterations do not reference or build upon specific prior findings

**Phase to address:**
Phase 1 (State Architecture) for the exploration tracker design. Phase 2 (Iteration Protocol) for the instructions that enforce exploration discipline.

---

### Pitfall 3: Self-Assessed Confidence Is Unreliable

**What goes wrong:**
The oracle asks each iteration to "Rate your overall confidence (0-100%) that the research is complete" and terminates at a target threshold (e.g., 95%). LLMs are systematically overconfident in self-assessment. Research shows models say "I'm 100% sure" when they are wrong, and self-reported confidence is a poor proxy for actual completeness. The oracle may terminate early (claiming 95% confidence after 5 shallow iterations) or never terminate (unable to honestly claim 99% on subjective questions).

**Why it happens:**
Confidence self-assessment requires metacognition that current LLMs lack. The model is essentially asking "how much do I know about what I don't know?" -- a fundamentally ill-posed question for a system without ground truth. Additionally, confidence calibration varies wildly by domain: factual questions can be verified, but open-ended research questions ("What gaps exist between X and best practices?") have no objective completeness measure.

**How to avoid:**
Replace subjective confidence with objective coverage metrics. Track: (1) what percentage of research questions have answers, (2) what percentage of answers cite verifiable sources, (3) whether new iterations are producing novel findings or just restating known ones (novelty rate). Terminate when the novelty rate drops below a threshold (e.g., less than 10% new information in 3 consecutive iterations) AND all research questions have at least one answer. This is a "diminishing returns" detector, not a confidence score.

OpenAI's Deep Research uses a trained model (o3) specifically optimized for multi-step research trajectories. Without that training, generic Claude instances will not produce well-calibrated confidence. Use structural completion metrics instead.

**Warning signs:**
- Confidence jumps from 40% to 90% in a single iteration
- Confidence score reaches target but obvious questions remain unanswered
- Different iterations rate the same state of knowledge at wildly different confidence levels
- Research terminates but the output is clearly shallow

**Phase to address:**
Phase 2 (Iteration Protocol) for the completion criteria redesign. Phase 4 (Quality/Verification) for testing that the new criteria actually produce good stopping behavior.

---

### Pitfall 4: Iterative Appending Without Iterative Deepening

**What goes wrong:**
The system produces a series of surface-level passes rather than progressively deeper investigation. Iteration 1 covers the topic broadly. Iteration 2 covers it broadly again from a slightly different angle. By iteration 50, you have 50 broad summaries rather than one deep analysis. This is the core problem stated in the project context: "it's iterative appending not iterative deepening."

**Why it happens:**
The oracle.md prompt treats every iteration identically. There is no concept of research phases (broad survey, then focused investigation, then synthesis, then verification). Each fresh instance reads the same "You are an Oracle Ant -- a deep research agent" prompt with no indication of where in the research lifecycle it is. Without phase-awareness, each instance defaults to the same behavior: read the topic, do a broad scan, append findings.

**How to avoid:**
Define explicit research phases that the state file tracks:

1. **Survey** (iterations 1-N): Broad exploration, identify subtopics, map the landscape. Goal: populate the research tree with questions.
2. **Investigate** (iterations N-M): Deep dive into specific questions one at a time. Goal: answer individual questions with evidence and sources.
3. **Synthesize** (iterations M-P): Cross-reference findings, identify contradictions, build coherent narrative. Goal: produce structured output.
4. **Verify** (iterations P-Q): Fact-check key claims, seek disconfirming evidence, validate sources. Goal: confidence in accuracy.

The state file should track current phase. The iteration prompt should change based on phase. This is how Anthropic's multi-agent system works: the lead agent plans distinct research phases, and subagents execute specific focused tasks rather than repeating the broad scan.

**Warning signs:**
- Each iteration's output reads like a standalone research report rather than a continuation
- Findings stay at the same depth (paragraph-level summaries) across all iterations
- No iteration ever says "Building on iteration X's finding that..."
- The word count per iteration stays roughly constant instead of decreasing as the space narrows

**Phase to address:**
Phase 2 (Iteration Protocol) for phase-aware prompting. Phase 1 (State Architecture) for tracking research lifecycle phase in state.

---

### Pitfall 5: Hallucination Accumulation Across Iterations

**What goes wrong:**
An early iteration introduces an inaccurate claim (hallucination). Subsequent iterations read it from the progress file, treat it as established fact, and build upon it. Over many iterations, the hallucination becomes deeply embedded in the research -- cited, cross-referenced, and relied upon for conclusions. Research on multi-agent hallucination identifies this as "full-chain error propagation across multiple interdependent components" -- minor initial errors compound into systemic inaccuracies.

**Why it happens:**
Stateless instances have no way to distinguish between verified findings and unverified assertions in the progress file. Everything written by a previous iteration looks equally authoritative. The current oracle.md has no source citation requirement, no verification step, and no mechanism to flag or challenge prior findings. Each iteration implicitly trusts all prior iterations' output.

**How to avoid:**
Require structured source tracking for every factual claim. The state file should track, for each finding: the claim, the source (URL, file path, or "inference"), a verification status (unverified/single-source/multi-source/contradicted), and which iteration produced it. Include a "Verify" research phase where iterations specifically attempt to disprove or find counter-evidence for key claims. Flag any finding that relies solely on LLM inference (no external source) as LOW confidence.

Anthropic's research system treats citations as "first-class data" where each chunk carries provenance metadata. Implement a simpler version: each finding gets a source field, and the synthesis phase specifically targets unverified claims for validation.

**Warning signs:**
- Findings appear with no source attribution
- Later iterations cite earlier iteration findings as authoritative (circular citation)
- Contradictory claims exist in different sections without acknowledgment
- "Phantom sources" -- URLs or references that were hallucinated and never verified

**Phase to address:**
Phase 2 (Iteration Protocol) for source citation requirements. Phase 4 (Quality/Verification) for the verification phase and hallucination detection.

---

### Pitfall 6: Stateless Instance Cannot Understand Context of Prior Failures

**What goes wrong:**
An iteration tries a research approach (e.g., searching for a specific resource, attempting to access a URL that 404s, looking for a codebase pattern that does not exist) and fails. It logs the failure in the progress file as a finding. The next iteration reads this but may attempt the same approach because the progress file captures WHAT was found, not WHY an approach failed or WHAT was tried and abandoned. The state file lacks negative knowledge -- "we tried X and it didn't work, don't try X again."

**Why it happens:**
Append-only logs naturally capture results, not process. "URL X returned 404" is recorded, but "I searched for documentation on Y and found none exists" is often omitted because the agent focuses on what it found, not what it tried and failed to find. Karpathy's autoresearch explicitly tracks failed experiments in results.tsv (recording "crash" and "discard" outcomes alongside successes) -- most research loop implementations only record successes.

**How to avoid:**
Add a "dead ends" or "attempted approaches" section to the state file. Each iteration should log: what it tried, what worked, what failed, and what it explicitly chose NOT to do (with reasoning). This is negative knowledge -- equally valuable as positive findings. The state file schema should include a `dead_ends` array alongside `findings`.

**Warning signs:**
- Error messages or "not found" notes repeat across iterations
- Iterations attempt to access the same broken URLs
- The same unsuccessful search queries appear in multiple iterations' tool use
- Research stalls because every iteration re-discovers the same dead ends

**Phase to address:**
Phase 1 (State Architecture) for the dead-ends tracking structure. Phase 2 (Iteration Protocol) for instructing iterations to record negative findings.

---

### Pitfall 7: Prompt Overloading Degrades Per-Iteration Quality

**What goes wrong:**
As the system matures, the iteration prompt grows to include: research questions, current phase, exploration tracker, source citation requirements, dead-end avoidance, output format specifications, and behavioral constraints. The prompt becomes so long and instruction-dense that the LLM's adherence to any individual instruction degrades. Each new requirement added to the prompt slightly reduces compliance with all existing requirements.

**Why it happens:**
This is "attention budget depletion" -- Anthropic's context engineering guide identifies that each token "depletes the LLM's limited attention budget by some amount." Complex prompts with many concurrent requirements produce worse results than focused prompts with fewer requirements. The oracle currently has a single oracle.md prompt that handles all iteration types identically, and adding more instructions to it will hit diminishing returns.

**How to avoid:**
Use phase-specific prompts rather than one monolithic prompt. The survey phase gets a survey-focused prompt (short, emphasizing breadth). The investigate phase gets an investigate-focused prompt (emphasizing depth and source citation). The verify phase gets a verification-focused prompt (emphasizing skepticism and counter-evidence). The bash script selects the appropriate prompt based on the current phase from the state file. This keeps each prompt focused and within the model's effective attention budget.

This mirrors Aether's existing "split playbooks" pattern (build-prep.md, build-context.md, etc.) -- apply the same principle to oracle prompts.

**Warning signs:**
- The oracle.md prompt exceeds 2,000 tokens
- Iterations ignore specific instructions (e.g., source citation) while following others (e.g., append format)
- Adding a new instruction causes previously-followed instructions to be dropped
- Per-iteration output quality decreases as the prompt grows

**Phase to address:**
Phase 2 (Iteration Protocol) for designing phase-specific prompts. Phase 3 (Prompt Engineering) for testing and tuning individual phase prompts.

---

### Pitfall 8: No Mechanism to Correct Course (Autonomy Without Steering)

**What goes wrong:**
The oracle runs autonomously for hours. The user cannot steer it mid-run except by creating a `.stop` file. If the research goes in an unproductive direction (e.g., investigating tangential subtopics, fixating on one question while ignoring others), there is no mechanism to redirect without stopping and restarting the entire run. The existing pheromone system (FOCUS/REDIRECT/FEEDBACK) is not integrated with the oracle loop.

**Why it happens:**
The oracle was designed as a "fire and forget" system -- launch it, let it run, read results. This is appropriate for short runs (5 iterations) but dangerous for long runs (50 iterations). Without mid-flight steering, small trajectory errors compound over many iterations. Anthropic's system explicitly includes "scaling rules" and task decomposition to prevent individual agents from going off-track, but Aether's oracle has no equivalent.

**How to avoid:**
Integrate Aether's existing pheromone system with the oracle loop. Between iterations, the bash script should check for pheromone signals: FOCUS signals reprioritize remaining questions, REDIRECT signals mark approaches as dead ends, FEEDBACK signals adjust research depth or direction. This gives the user steering capability without requiring a full stop-and-restart. Additionally, generate a brief "iteration summary" output after each iteration (visible in tmux) so the user can monitor trajectory and intervene if needed.

**Warning signs:**
- Research output diverges from the original topic by iteration 15+
- User discovers poor trajectory only after the full run completes
- No way to tell mid-run whether the research is productive
- Oracle spends many iterations on low-priority subtopics while skipping high-priority ones

**Phase to address:**
Phase 3 (Integration) for wiring pheromones into the oracle loop. Phase 2 (Iteration Protocol) for mid-run visibility and status reporting.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Unstructured markdown as sole state format | Simple to read/write, human-friendly | Cannot be programmatically queried, grows unbounded, no schema validation | Never for inter-iteration state. OK for final human-readable output |
| Single prompt for all iteration types | One file to maintain, simple architecture | Per-iteration quality degrades as prompt grows, cannot optimize per phase | MVP/prototype only. Must split before production use |
| Self-assessed confidence as termination criterion | Easy to implement (one number comparison) | Unreliable, causes premature termination or infinite loops | Never as sole criterion. OK as supplementary signal alongside structural metrics |
| Trusting all prior iteration output equally | Simpler state model, no provenance tracking | Hallucinations accumulate, no way to distinguish verified from inferred | Never for factual research. Acceptable for brainstorming/ideation modes |
| Append-only with no compaction | Zero risk of data loss | Context rot, repeated findings, growing token costs | First 5-10 iterations only. Must introduce structured state before iteration 15 |

## Integration Gotchas

Common mistakes when connecting the oracle loop to external systems.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Claude CLI (`--print` mode) | Assuming output is clean text; it may include tool call formatting, error messages, or partial responses | Parse output for completion signal only; let the iteration write to files directly rather than capturing stdout |
| WebSearch/WebFetch tools | Feeding full web page content into progress file, blowing up token count | Instruct iterations to extract and summarize relevant findings only; never paste raw HTML or full page content |
| Pheromone system | Checking pheromones inside the Claude instance (which is stateless and may not have access to latest state) | Check pheromones in the bash loop BETWEEN iterations, before launching the next instance; inject relevant signals as context |
| tmux session | Assuming tmux is always available; no fallback | The current oracle.sh already handles this correctly with a TMUX_FAIL fallback. Maintain this pattern |
| Git state (for code research) | Research iterations making git commits or modifying working tree | The current "Non-Invasive Guarantee" is correct. Enforce read-only access to everything outside `.aether/oracle/` |

## Performance Traps

Patterns that work at small scale but fail as iteration count grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Reading full progress.md every iteration | Iterations slow down; later iterations spend more time reading than researching | Switch to structured state file; keep raw log separate | 15+ iterations (~5K+ tokens in progress file) |
| No deduplication of findings | Same facts restated in slightly different words across iterations | Hash-based or keyword-based dedup in state file; mark findings with unique IDs | 10+ iterations |
| Spawning Claude CLI per iteration without cleanup | Zombie processes, orphaned tool connections, temp file accumulation | Add cleanup step in bash loop between iterations (existing `sleep 2` could become a cleanup phase) | 30+ iterations in long runs |
| Web search for same queries across iterations | Rate limiting, wasted API calls, identical results | Track searched queries in state file; skip if already searched within N iterations | 20+ iterations with web scope |

## Security Mistakes

Domain-specific security issues for autonomous research loops.

| Mistake | Risk | Prevention |
|---------|------|------------|
| `--dangerously-skip-permissions` with web research scope | AI instance can fetch arbitrary URLs, potentially triggering SSRF or leaking local network info | Acceptable tradeoff for oracle (needed for autonomy), but document the risk; never run oracle on untrusted networks |
| Progress file contains hallucinated "findings" presented as authoritative | User trusts and acts on inaccurate information (e.g., "Library X has vulnerability Y") | Source attribution and verification status on every factual claim; clearly label unverified findings |
| Oracle writing to files outside `.aether/oracle/` | Could corrupt colony state, modify code, or overwrite user files | The existing "Non-Invasive Guarantee" is correct. Test and enforce this boundary |
| Sensitive information in research.json (API keys, credentials) | Exposed in archive directory, potentially committed to git | Validate research.json contents; warn if it contains patterns matching secrets |

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No visibility into what oracle is doing mid-run | User waits hours, checks progress.md, finds it went sideways | Emit a one-line status per iteration: "Iteration 5/30: Investigating [question]. 3/5 questions answered." Visible in tmux |
| Progress.md is the only output (no structured summary) | User must read through 50 iterations of raw notes to extract conclusions | Produce a structured `report.md` during synthesis phase with executive summary, key findings, and confidence levels |
| "Marathon (50 iterations)" sounds productive but wastes resources | User selects max depth thinking more = better; gets diminishing returns after 15 | Recommend depth based on topic complexity; show diminishing returns warning in wizard |
| Confidence percentage without context | "85% confidence" means nothing without knowing what is missing | Show: "85% complete -- 4/5 questions answered, 1 question unresolved: [specific question]" |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Research output:** Has findings but no source attribution -- verify every factual claim has a source
- [ ] **Structured state:** State file exists but does not track exploration coverage -- verify all research questions have entries
- [ ] **Completion criteria:** Confidence target reached but novelty rate was not checked -- verify last 3 iterations produced new findings
- [ ] **Phase transitions:** Survey phase "complete" but investigation phase starts without an explicit question queue -- verify the research tree is populated before transitioning
- [ ] **Verification phase:** Claims are "verified" but only by re-reading the same source -- verify counter-evidence was sought
- [ ] **Dead ends:** No dead ends recorded (suspicious) -- verify iterations actually attempted diverse approaches
- [ ] **Final report:** Report exists but was generated from raw progress file, not synthesized -- verify it has a coherent narrative, not a list of iteration summaries

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Progress file bloated beyond usefulness | LOW | Pause oracle. Manually summarize progress.md into a structured state file. Resume from structured state |
| Circular research (same ground covered 5+ times) | LOW | Pause oracle. Mark covered questions as ANSWERED in state file. Add explicit "investigate NEXT: [specific question]" directive. Resume |
| Hallucination embedded in findings | MEDIUM | Pause oracle. Manually review findings for source attribution. Remove or flag unsourced claims. Add verification pass to remaining iterations |
| Confidence-based termination fired too early | LOW | Re-run with structural completion criteria instead. Previous findings are not lost -- they seed the new run |
| Oracle went off-topic for 30+ iterations | MEDIUM | Archive the off-topic run. Extract any relevant findings manually. Create a new research.json with more specific questions. Start fresh |
| Prompt overloading causing instruction non-compliance | MEDIUM | Split into phase-specific prompts. Test each prompt independently before integration. Reduce instruction count per prompt to under 10 directives |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Append-only progress file bloat | Phase 1 (State Architecture) | State file under 2K tokens at iteration 20; raw log separated from handoff state |
| Circular research | Phase 1 (State Architecture) + Phase 2 (Iteration Protocol) | Exploration tracker shows unique questions covered per iteration; no duplicates across 10+ iterations |
| Unreliable self-assessed confidence | Phase 2 (Iteration Protocol) | Structural completion criteria (coverage %, novelty rate) used instead of or alongside self-assessed confidence |
| Iterative appending not deepening | Phase 2 (Iteration Protocol) | Phase transitions visible in state; later iterations demonstrably deeper than earlier ones |
| Hallucination accumulation | Phase 2 (Iteration Protocol) + Phase 4 (Quality) | Every factual claim in final report has source attribution; verification phase explicitly seeks counter-evidence |
| Lost negative knowledge | Phase 1 (State Architecture) | Dead-ends tracked in state file; no iteration repeats a known-failed approach |
| Prompt overloading | Phase 2 (Iteration Protocol) + Phase 3 (Prompt Engineering) | Phase-specific prompts each under 1,500 tokens; measurable instruction compliance rate |
| No mid-run steering | Phase 3 (Integration) | Pheromone signals checked between iterations; user can FOCUS/REDIRECT without stopping |

## Sources

- [Anthropic: How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) -- pitfalls in multi-agent coordination, duplication, source bias, cascading failures (HIGH confidence)
- [Anthropic: Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) -- context rot, attention budget depletion, prompt overloading, dead-end chasing (HIGH confidence)
- [LangChain: Context Management for Deep Agents](https://blog.langchain.com/context-management-for-deepagents/) -- context rot, goal drift after summarization, trajectory loss (MEDIUM confidence)
- [Karpathy autoresearch: DeepWiki analysis](https://deepwiki.com/karpathy/autoresearch/4-agent-operation) -- ratcheting mechanism, results.tsv tracking, monotonic improvement (HIGH confidence)
- [OpenAI Deep Research System Card](https://cdn.openai.com/deep-research-system-card.pdf) -- multi-step research trajectory optimization, backtracking (MEDIUM confidence)
- [Galileo: Why Multi-Agent LLM Systems Fail](https://galileo.ai/blog/multi-agent-llm-systems-fail) -- circular exchanges, context loss, coordination breakdowns (MEDIUM confidence)
- [Solving LLM Repetition Problem in Production](https://arxiv.org/html/2512.04419v1) -- self-reinforcement effect, repetition probability enhancement (MEDIUM confidence)
- [Ralph Wiggum / Fresh Context Pattern](https://deepwiki.com/FlorianBruniaux/claude-code-ultimate-guide/7.3-fresh-context-pattern-(ralph-loop)) -- progress.txt growth, context window exhaustion (HIGH confidence)
- [11 Tips For AI Coding With Ralph Wiggum](https://www.aihero.dev/tips-for-ai-coding-with-ralph-wiggum) -- progress file management, truncation, local minimum traps (MEDIUM confidence)
- [LLM-based Agents Suffer from Hallucinations survey](https://arxiv.org/html/2509.18970v1) -- multi-step hallucination accumulation, chain error propagation (MEDIUM confidence)
- [CEA: Context Engineering Agent for Enhanced Reliability](https://openreview.net/forum?id=6QUNblHtto) -- structured context management, token efficiency vs memory integrity tradeoff (MEDIUM confidence)
- Aether oracle archive (`.aether/oracle/archive/2026-02-16-191250-progress.md`) -- first-hand evidence of iterative appending without deepening, repetitive iteration blocks (HIGH confidence, direct observation)

---
*Pitfalls research for: Autonomous AI Deep Research Loops*
*Researched: 2026-03-13*
