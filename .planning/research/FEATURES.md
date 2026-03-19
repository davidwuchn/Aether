# Feature Research: Aether v1.3 Maintenance Milestone

**Domain:** Multi-agent AI orchestration system maintenance (pheromone integration, cleanup, fresh-install polish)
**Researched:** 2026-03-19
**Confidence:** HIGH (based on deep codebase analysis + ecosystem research)

---

## Context

This is a **maintenance milestone** for an existing, mature system. The goal is not new capabilities -- it is wiring together what exists, cleaning up debris from development, and making the system solid for new users. The codebase has 150 shell subcommands, 22 agents, 36 slash commands, and 490+ tests. It works, but has accumulated integration gaps and test pollution.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels broken or untrustworthy.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Clean data files on install** | Test artifacts ("TestAnt6", "test signal", "test area") in COLONY_STATE.json, pheromones.json, learning-observations.json, constraints.json make the system look broken | LOW | 6+ files contain test pollution. `learning-observations.json` is entirely test data (11 observations, all from "test-colony"). Constraints.json has 3 test focus areas. Must ship clean templates OR add a data-reset command. |
| **Pheromones actually affect worker behavior** | The docs say "Workers read FOCUS signals and weight this area higher" but no agent definition reads pheromones | HIGH | `colony-prime` injects signals into `prompt_section`, but this is only referenced in build-wave.md builder prompts. The 22 agent definitions themselves contain zero references to `pheromone-read`, `pheromone-prime`, or `colony-prime`. Signal propagation depends entirely on the orchestrator injecting `{ prompt_section }` -- workers cannot self-check signals at "natural breakpoints" as the worker prompt instructs because no agent has this wired. |
| **Working fresh install flow** | `npm install -g aether-colony` -> `/ant:lay-eggs` -> `/ant:init` should work without errors or confusion | MEDIUM | The flow exists and is well-documented, but: (1) no smoke test validates it end-to-end, (2) templates ship with test data, (3) QUEEN.md init can fail silently, (4) version.json copy can fail silently. Users hitting any silent failure get a degraded experience with no warning. |
| **All documented commands actually work** | 36 slash commands are listed in help/docs -- all should function | MEDIUM | Some commands reference features that may have drifted. The `tunnels` command, `verify-castes` command, and `migrate-state` command need validation. Commands referencing `session.json` must verify that file exists in current state format. |
| **Stale state detection and cleanup** | Colony state from a different project (goal: "Ensure the electron version...") should not persist across projects | LOW | COLONY_STATE.json currently has a stale goal from a previous project. The init command warns but does not auto-archive. The `session-verify-fresh` mechanism exists but only checks timestamps, not goal relevance. |
| **Documentation matches reality** | Docs describe features that exist, not aspirational features | MEDIUM | CLAUDE.md references "runtime/" which was eliminated. Pheromone docs claim workers "read FOCUS signals" which is architecturally true (via injection) but not operationally true (workers cannot read them independently). Known-issues.md has FIXED items still listed as unfixed in some sections. |

### Differentiators (Competitive Advantage)

Features that set Aether apart from other multi-agent orchestration systems. Not required for basic function, but provide real value.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Signal-aware worker behavior** | Workers that actually respond to pheromone signals at runtime (not just at spawn) make the colony feel alive and responsive to user steering | HIGH | Currently workers get a static `prompt_section` at spawn time. The build-wave.md instructs builders to "check for new signals at natural breakpoints" but provides no mechanism. True signal-awareness would mean workers call `pheromone-read` mid-execution -- but Claude Code subagents cannot spawn other subagents and running bash commands mid-task for signal checks is expensive. The realistic differentiator is high-quality signal injection at spawn. |
| **Closed-loop learning pipeline** | Observations become learnings become instincts become pheromones -- a full feedback loop that improves colony behavior over time | MEDIUM | The pipeline components exist: `memory-capture` -> `learning-observe` -> `learning-promote-auto` -> `instinct-create`. But the loop is not closed: (1) instincts are created but never verified to change behavior, (2) pheromone auto-emission from midden works but is not tested in real scenarios, (3) test data pollution prevents seeing real learning in action. Cleaning the pipeline data and validating the full loop is the differentiator. |
| **Data hygiene tooling** | A `data-reset` or `data-clean` command that purges test artifacts while preserving real colony state | LOW | No multi-agent framework offers this because most do not persist state across sessions. Aether's persistence model makes this uniquely necessary and valuable. |
| **XML exchange system for cross-colony communication** | Pheromone/wisdom XML export/import for sharing signals between colonies | LOW (exists) | The exchange system (`pheromone-xml.sh`, `wisdom-xml.sh`, `registry-xml.sh`) is fully built with 5 XML utility scripts, but zero commands or playbooks reference it. It is dead code. The value is there -- cross-colony signal sharing is a differentiator no competitor has -- but it needs a surface (command or automatic trigger) to be useful. |
| **Graveyard caution system** | Files with troubled history get flagged for workers with "proceed carefully" warnings | LOW (exists) | Already implemented via `grave-check` in build-wave.md. This is a genuine differentiator -- no other multi-agent system uses git history to inform worker behavior. Needs validation that it works in practice. |
| **Midden threshold auto-REDIRECT** | Recurring failures automatically emit REDIRECT pheromones to steer the colony away from problematic patterns | LOW (exists) | Implemented in build-wave.md Step 5.2. Needs real-world validation -- has only been tested with synthetic data. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems in this maintenance context.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **New agent types** | "Add a Debugger agent, a Deployer agent" | 22 agents already strain context windows. Adding more increases the surface area to maintain without solving integration gaps. | Fix the integration between existing 22 agents first. The current agents cover all needed castes. |
| **Real-time inter-worker messaging** | "Workers should talk to each other during execution" | Claude Code Task tool subagents cannot communicate with each other mid-execution. The Agent Teams feature (Feb 2026) enables this, but migrating to it is a v2 concern, not a maintenance fix. | Keep using the Queen-as-coordinator pattern with `prompt_section` injection at spawn time. |
| **Complex pheromone decay algorithms** | "Signals should decay based on relevance, not just time" | Over-engineering. The current linear decay with configurable half-lives (15/30/45 days) is sufficient. Relevance-based decay requires understanding intent, which is an unsolved problem. | Keep current decay model. Add a manual `pheromone-clear-expired` command for housekeeping. |
| **Automatic XML migration** | "Convert all JSON state to XML" | The XML system was built for cross-colony exchange, not as a replacement for JSON state management. Migrating internal state to XML adds complexity without user-facing value. | Keep JSON for internal state, XML for exchange/export. Document this boundary clearly. |
| **Dashboard web UI** | "Visualize colony state in a browser" | Massive scope increase. Aether runs in terminal/CLI context. A web UI requires a server, frontend framework, state synchronization -- all orthogonal to the core value. | Keep the ASCII-art dashboards (`pheromone-display`, `swarm-display`, `/ant:status`). They work well in the terminal context. |
| **Per-worker model routing** | "Different models for different castes" | Was built, tested, and archived because Claude Code Task tool does not support per-subagent environment variables. The architecture cannot support this until the platform changes. | Use session-level model selection as documented. Preserve the archive for future platform evolution. |

---

## Feature Dependencies

```
[Test data cleanup]
    └──enables──> [Learning pipeline validation]
                       └──enables──> [Closed-loop verification]

[Pheromone signal injection audit]
    └──enables──> [Worker signal-awareness validation]
    └──enables──> [Auto-emission verification]

[Fresh install flow validation]
    └──requires──> [Clean templates]
    └──requires──> [Working command chain]

[XML exchange activation]
    └──requires──> [Pheromone system working correctly]
    └──requires──> [Command surface for XML operations]

[Documentation update]
    └──requires──> [All other fixes complete]
    └──requires──> [Feature audit complete]

[Stale state cleanup]
    └──enhances──> [Fresh install flow]
    └──conflicts──> [Running during active colony] (must not clobber active work)
```

### Dependency Notes

- **Test data cleanup enables learning pipeline validation:** Cannot verify the learning pipeline works correctly when all observation data is synthetic test entries. Clean first, then validate.
- **Pheromone injection audit enables worker awareness:** Must understand exactly how `prompt_section` flows through the system before claiming signals affect behavior. The audit may reveal that injection is working fine and the gap is only in documentation -- or it may reveal real wiring issues.
- **Documentation update requires all other fixes:** Docs should be updated last because they describe the system as it is, not as it was. Updating docs before fixing things creates a second round of doc updates.
- **XML exchange activation requires working pheromones:** Cannot export/import signals that are not correctly read. Fix the core system before extending it.
- **Stale state cleanup conflicts with active colony:** A `data-reset` command must check for active colony state and warn/abort if a colony is mid-execution. Cleaning data during a build would corrupt state.

---

## MVP Definition

### Launch With (Maintenance Milestone v1)

Minimum viable deliverables for this milestone -- what is needed to call the maintenance work complete.

- [x] **Test data purge** -- Remove all synthetic test data from `.aether/data/` files (pheromones.json, constraints.json, learning-observations.json, COLONY_STATE.json, spawn-tree.txt, midden). Either clean the committed files or ensure templates are clean and add a reset command.
- [ ] **Pheromone injection audit** -- Trace the full path from user running `/ant:focus "X"` through to a builder worker receiving that signal in its prompt. Document exactly where signals flow, identify gaps, fix any broken links.
- [ ] **Fresh install smoke test** -- Create an automated test that validates `lay-eggs` -> `init` -> `plan` -> `build` on a clean repo. This prevents regressions in the onboarding experience.
- [ ] **Broken command audit** -- Run every slash command with `--help` or minimal args. Document which ones error. Fix critical ones, log non-critical ones as known issues.
- [ ] **Documentation accuracy pass** -- Update CLAUDE.md, pheromones.md, known-issues.md to reflect current reality. Remove references to eliminated features (runtime/), update FIXED statuses, correct pheromone behavior descriptions.

### Add After Validation (v1.x)

Features to add once the core cleanup is validated working.

- [ ] **`data-clean` command** -- Dedicated utility to purge test artifacts, expired pheromones, and stale signals. Safe to run anytime with confirmation prompt. Trigger: users report confusion from old data.
- [ ] **XML exchange command surface** -- Create `/ant:export-signals` and `/ant:import-signals` commands that use the existing `pheromone-xml.sh` exchange module. Trigger: users with multiple colonies want to share signals.
- [ ] **Learning pipeline end-to-end test** -- Integration test that follows an observation through the full pipeline: `memory-capture` -> `learning-observe` -> threshold met -> `learning-promote-auto` -> `instinct-create` -> instinct appears in `colony-prime` output. Trigger: pipeline components are clean and individually tested.
- [ ] **Error code standardization** -- Address BUG-004, BUG-007, BUG-008, BUG-009, BUG-010, BUG-012: replace all hardcoded error strings with `E_*` constants. Trigger: when touching aether-utils.sh for other fixes.

### Future Consideration (v2+)

Features to defer until the maintenance milestone is complete and system is stable.

- [ ] **Agent Teams migration** -- Claude Code Agent Teams (released Feb 2026) enables inter-worker communication. Migrating from Task tool subagents to teammates would unlock real-time signal propagation. Defer because: requires significant architecture changes and the current system works.
- [ ] **Cross-colony wisdom sync** -- Use the XML exchange to automatically sync wisdom across colonies in the same registry. Defer because: requires the exchange system to be activated and tested first.
- [ ] **Pheromone visualization improvements** -- Enhanced ASCII/terminal UI for signal display with time-series decay visualization. Defer because: current `pheromone-display` works, this is polish.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Test data purge | HIGH | LOW | P1 |
| Pheromone injection audit | HIGH | MEDIUM | P1 |
| Fresh install smoke test | HIGH | MEDIUM | P1 |
| Documentation accuracy pass | HIGH | LOW | P1 |
| Broken command audit | MEDIUM | MEDIUM | P1 |
| `data-clean` command | MEDIUM | LOW | P2 |
| Learning pipeline e2e test | MEDIUM | MEDIUM | P2 |
| XML exchange commands | LOW | LOW | P2 |
| Error code standardization | LOW | MEDIUM | P2 |
| Agent Teams migration | HIGH | HIGH | P3 |
| Cross-colony wisdom sync | MEDIUM | HIGH | P3 |
| Pheromone visualization | LOW | MEDIUM | P3 |

**Priority key:**
- P1: Must have for this milestone -- fixes broken or misleading behavior
- P2: Should have -- improves reliability and developer experience
- P3: Future work -- requires architecture changes or depends on P2 completion

---

## Competitor Feature Analysis

| Feature | CrewAI | AutoGen | LangGraph | OpenAI Swarm | Aether |
|---------|--------|---------|-----------|--------------|--------|
| Signal/pheromone system | None | None | State channels (basic) | Handoff only | Full pheromone system with FOCUS/REDIRECT/FEEDBACK, decay, TTL |
| Cross-session memory | None (stateless) | None | Checkpointing | None (stateless) | COLONY_STATE.json + session.json + QUEEN.md wisdom |
| Learning pipeline | None | None | None | None | Observation -> learning -> instinct pipeline (unique) |
| Worker specialization | Role-based agents | Conversational agents | Graph nodes | Function-based | 22 typed agents with caste system |
| Failure tracking | Basic logging | Conversation history | State persistence | None | Midden system with threshold-based auto-REDIRECT |
| Fresh install experience | pip install + API key | pip install + API key | pip install + API key | pip install | npm install + lay-eggs + init (multi-step) |
| XML exchange | None | None | None | None | Built but unactivated |
| State cleanup | Not needed (stateless) | Not needed | Manual | Not needed | Needed but missing |

**Key observation:** Aether's persistent state model is its biggest differentiator AND its biggest maintenance burden. Competitors avoid this problem by being stateless -- but that means they cannot learn across sessions. The maintenance milestone should lean into persistence as a strength while fixing the hygiene issues it creates.

---

## Current State Evidence

### Test Data Pollution (confirmed via codebase analysis)

| File | Test Artifacts Found |
|------|---------------------|
| `COLONY_STATE.json` | "TestAnt6" in memory.phase_learnings, "test error" in errors.records |
| `pheromones.json` | 3 signals with "test" content: "test area for pheromone unification", "test area", "test signal" |
| `constraints.json` | 3 focus entries: "test area for pheromone unification", "test area", "test signal" |
| `learning-observations.json` | **Entire file is test data** (11 observations, all from "test-colony", "alpha-colony", "beta-colony", etc.) |
| `spawn-tree.txt` | "TestAnt6" entry |

### Pheromone Flow Gap (confirmed via grep analysis)

| Component | Pheromone Awareness | Status |
|-----------|-------------------|--------|
| `pheromone-write` (aether-utils.sh) | Writes signals | WORKING |
| `pheromone-read` (aether-utils.sh) | Reads signals with decay | WORKING |
| `pheromone-prime` (aether-utils.sh) | Compiles signals for injection | WORKING |
| `colony-prime` (aether-utils.sh) | Combines wisdom + signals + instincts | WORKING |
| `build-context.md` (playbook) | Calls colony-prime, stores prompt_section | WORKING |
| `build-wave.md` (playbook) | Injects `{ prompt_section }` into builder prompts | WORKING |
| Agent definitions (22 files) | **None read pheromones independently** | GAP -- by design (injection model) |
| Worker self-check ("At natural breakpoints: Check for new signals") | Instruction in worker prompt but no mechanism provided | GAP -- aspiration vs reality |

**Assessment:** The injection model (Queen reads signals, injects into worker prompts) is architecturally sound. The gap is documentation claiming workers "read signals" when they receive injected context. Fix the docs, not the architecture.

### XML Exchange System (confirmed via file analysis)

| Component | Status |
|-----------|--------|
| `pheromone-xml.sh` | Built, tested (test-pheromone-xml.sh exists) |
| `wisdom-xml.sh` | Built |
| `registry-xml.sh` | Built |
| `xml-core.sh` | Built |
| `xml-convert.sh`, `xml-query.sh`, `xml-compose.sh`, `xml-utils.sh` | Built |
| Commands referencing XML | **None** -- zero slash commands use the exchange system |
| Playbooks referencing XML | **None** -- zero playbooks call exchange functions |

**Assessment:** The XML exchange is complete, tested infrastructure with zero activation surface. Either create commands to use it or explicitly document it as "infrastructure for future use."

---

## Sources

- Codebase analysis (HIGH confidence):
  - `.aether/data/pheromones.json` -- confirmed test pollution
  - `.aether/data/COLONY_STATE.json` -- confirmed stale state and test data
  - `.aether/data/learning-observations.json` -- confirmed entirely test data
  - `.aether/data/constraints.json` -- confirmed test focus entries
  - `.claude/agents/ant/*.md` -- confirmed no agent reads pheromones independently
  - `.aether/docs/command-playbooks/build-wave.md` -- confirmed prompt_section injection model
  - `.aether/docs/command-playbooks/build-context.md` -- confirmed colony-prime integration
  - `.aether/exchange/*.sh` -- confirmed XML system built but unused
  - `.claude/commands/ant/lay-eggs.md` -- confirmed fresh install flow
  - `.aether/docs/known-issues.md` -- confirmed documentation staleness

- External research (MEDIUM confidence):
  - [Multi-Agent Systems & AI Orchestration Guide 2026](https://www.codebridge.tech/articles/mastering-multi-agent-orchestration-coordination-is-the-new-scale-frontier) -- orchestration patterns
  - [Multi-Agent Orchestration Patterns](https://www.ai-agentsplus.com/blog/multi-agent-orchestration-patterns-2026) -- signal propagation patterns
  - [Memory for AI Agents: A New Paradigm of Context Engineering](https://thenewstack.io/memory-for-ai-agents-a-new-paradigm-of-context-engineering/) -- persistence patterns
  - [The 6 Best AI Agent Memory Frameworks 2026](https://machinelearningmastery.com/the-6-best-ai-agent-memory-frameworks-you-should-try-in-2026/) -- memory management
  - [Create custom subagents - Claude Code Docs](https://code.claude.com/docs/en/sub-agents) -- Claude Code subagent patterns
  - [Claude Code Agent Teams](https://code.visualstudio.com/blogs/2026/02/05/multi-agent-development) -- future multi-agent patterns
  - [Anthropic: Building Effective Agents](https://www.anthropic.com/research/building-effective-agents) -- agent design best practices
  - [OpenTelemetry AI Agent Observability](https://opentelemetry.io/blog/2025/ai-agent-observability/) -- observability standards

---

*Feature research for: Aether v1.3 Maintenance Milestone*
*Researched: 2026-03-19*
