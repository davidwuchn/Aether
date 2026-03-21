<div align="center">

```
      █████╗ ███████╗████████╗██╗  ██╗███████╗██████╗
     ██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██╔══██╗
     ███████║█████╗     ██║   ███████║█████╗  ██████╔╝
     ██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██╔══██╗
     ██║  ██║███████╗   ██║   ██║  ██║███████╗██║  ██║
     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
```

**Multi-agent system using ant colony intelligence for Claude Code and OpenCode**

[![npm version](https://img.shields.io/npm/v/aether-colony.svg)](https://www.npmjs.com/package/aether-colony)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**v2.0.0**
</div>

<p align="center">
  <img src="./AetherArtork.png" alt="Aether artwork" width="720" />
</p>

---

## What Is Aether?

Aether brings **ant colony intelligence** to Claude Code and OpenCode. Instead of one agent doing everything sequentially, you get a colony of specialists that self-organize around your goal.

```
👑 Queen (you)
   │
   ▼ pheromone signals guide the colony
   │
🐜 Workers spawn Workers (max depth 3)
   │
   ├── 🔨🐜 Builders — implement code
   ├── 👁️🐜 Watchers — verify & test
   ├── 🔍🐜 Scouts — research docs
   ├── 🐛🐜 Trackers — investigate bugs
   ├── 🗺️🐜 Colonizers — explore codebases (4 parallel scouts)
   ├── 📋🐜 Route-setters — plan phases
   ├── 🏺🐜 Archaeologists — excavate git history
   ├── 🎲🐜 Chaos Ants — resilience testing
   └── 📚🐜 Keepers — preserve knowledge
```

When a Builder hits something complex, it spawns a Scout to research. When code is written, a Watcher spawns to verify. **The colony adapts to the problem.**

---

## Key Features

- **22 Agent Definitions** — Real subagents spawned via Task tool
- **43 Slash Commands** — Full lifecycle management
- **Hard Enforcement Guards** — Spawn budget hard-fail mode, schema-validated worker payloads, and explicit blocker gating
- **Pheromone System** — Guide the colony with FOCUS, REDIRECT, FEEDBACK signals
- **State Safety** — Lock + atomic-write protections on critical state and memory mutation paths
- **Oracle Deep Research** — 50+ iteration autonomous research loop
- **6-Phase Verification** — Build, types, lint, tests, security, diff
- **Colony Memory** — Learnings persist across sessions via QUEEN.md
- **Operational Evolution Loop** — Incident template, regression scaffolding, weekly audit script, and entropy/spawn metrics
- **Pause/Resume** — Full state serialization for context breaks
- **Autopilot** (`/ant:run`) — Automated build-verify-advance loop across phases
- **Hive Brain** — Cross-colony wisdom sharing with domain-scoped retrieval and multi-repo confidence boosting
- **User Preferences** — Colony adapts to your communication style and decision patterns
- **Pre-Seal Audit** (`/ant:patrol`) — Verify work against plan, check docs, review issues before sealing
- **Quality Gates** — Security (Gatekeeper), quality (Auditor), coverage (Probe), performance (Measurer)
- **Pheromone Hardening** — Content deduplication, prompt injection sanitization, and signal reinforcement

---

## Installation

```bash
# NPX installer (recommended)
npx aether-colony install

# Or npm global install
npm install -g aether-colony
```

This installs 22 agents to `~/.claude/agents/ant/` plus 43 slash commands to `~/.claude/commands/ant/`.

---

## Quick Start

```bash
/ant:init "Build a REST API with authentication"
/ant:plan
/ant:build 1
/ant:continue
```

---

## Command Reference

### Core Lifecycle

| Command | Description |
|---------|-------------|
| `/ant:init "goal"` | 🌱 Initialize colony with mission |
| `/ant:plan` | 📋 Generate phased roadmap |
| `/ant:build N` | 🔨 Execute phase N with worker waves |
| `/ant:continue` | ➡️ 6-phase verification, advance to next phase |
| `/ant:pause-colony` | 💾 Save state for context break |
| `/ant:resume-colony` | 🚦 Restore from pause |
| `/ant:run` | 🤖 Autopilot — build, verify, advance automatically |
| `/ant:patrol` | 🔍 Pre-seal audit — verify work against plan |
| `/ant:seal` | 🏺 Complete and archive colony |
| `/ant:entomb` | ⚰️ Create chamber from completed colony |

Implementation note:
- In Claude Code, `.claude/commands/ant/build.md` is an orchestrator and executes split playbooks under `.aether/docs/command-playbooks/` (`build-prep.md`, `build-context.md`, `build-wave.md`, `build-verify.md`, `build-complete.md`).
- OpenCode has its own command spec at `.opencode/commands/ant/build.md`.

**Core Flow:**
```
/ant:init → /ant:plan → /ant:build 1 → /ant:continue → /ant:build 2 → ... → /ant:seal
```

**Autopilot Flow:**
```
/ant:init → /ant:plan → /ant:run → /ant:seal
```

### Pheromone Signals

| Command | Emoji | Description |
|---------|-------|-------------|
| `/ant:focus "area"` | 🎯 | FOCUS signal — "Pay attention here" |
| `/ant:redirect "pattern"` | 🚫 | REDIRECT signal — "Don't do this" (hard constraint) |
| `/ant:feedback "note"` | 💬 | FEEDBACK signal — "Adjust based on this observation" |

**How pheromones work:**
- Before builds: Use FOCUS + REDIRECT to steer the colony
- After builds: Use FEEDBACK to teach preferences
- Signals persist in `.aether/data/pheromones.json`
- Auto-injected into worker prompts via `colony-prime --compact`
- Compact context capsule is injected alongside top signals (goal, phase, next action, risks, recent decisions)
- **Displayed in `/ant:build`** before workers spawn
- View active signals with `/ant:pheromones`
- Decay over time: FOCUS 30d, REDIRECT 60d, FEEDBACK 90d

### Research & Analysis

| Command | Description |
|---------|-------------|
| `/ant:colonize` | 📊🐜🗺️ 4 parallel scouts analyze your codebase |
| `/ant:oracle ["topic"]` | 🔮 Deep research with 50+ iteration loop |
| `/ant:archaeology <path>` | 🏺 Excavate git history for any file |
| `/ant:chaos <target>` | 🎲 Resilience testing, edge case probing |
| `/ant:swarm ["problem"]` | 🔥 4 parallel scouts for stubborn bugs |
| `/ant:dream` | 💭 Philosophical codebase wanderer |
| `/ant:interpret` | 🔍 Grounds dreams in reality, discusses implementation |
| `/ant:organize` | 🧹 Codebase hygiene report |

### Visibility

| Command | Description |
|---------|-------------|
| `/ant:status` | 📈 Colony overview with memory health |
| `/ant:pheromones` | 🎯 View active signals (FOCUS/REDIRECT/FEEDBACK) |
| `/ant:memory-details` | 🧠 Wisdom, pending promotions, recent failures |
| `/ant:watch` | 👁️ Real-time swarm display |
| `/ant:history` | 📜 Recent activity log |
| `/ant:flags` | 🚩 List blockers and issues |
| `/ant:help` | 🐜 Full command reference |

### Coordination & Maintenance

| Command | Description |
|---------|-------------|
| `/ant:council` | 🏛️ Clarify intent via multi-choice questions |
| `/ant:flag` | 🚩 Create project-specific flag (blocker/issue/note) |
| `/ant:data-clean` | 🧹 Remove test artifacts from colony data |
| `/ant:export-signals` | 📤 Export pheromone signals to XML |
| `/ant:import-signals` | 📥 Import pheromone signals from XML |
| `/ant:preferences` | 🎨 Add or list user preferences |

---

## The Active Castes

| Tier | Agent | Role | Spawned By |
|------|-------|------|------------|
| **Core** | Queen | Orchestrates, spawns workers | You |
| **Core** | Builder | Writes code, TDD-first | `/ant:build` |
| **Core** | Watcher | Tests, validates | `/ant:build` |
| **Core** | Scout | Researches, discovers | `/ant:build`, `/ant:oracle`, `/ant:swarm` |
| **Orchestration** | Route-Setter | Plans phases | `/ant:plan` |
| **Surveyor** | surveyor-nest | Maps directory structure | `/ant:colonize` |
| **Surveyor** | surveyor-disciplines | Documents conventions | `/ant:colonize` |
| **Surveyor** | surveyor-pathogens | Identifies tech debt | `/ant:colonize` |
| **Surveyor** | surveyor-provisions | Maps dependencies | `/ant:colonize` |
| **Specialist** | Keeper | Preserves knowledge | `/ant:continue` |
| **Specialist** | Tracker | Investigates bugs | `/ant:swarm` |
| **Specialist** | Probe | Coverage analysis | `/ant:continue` |
| **Specialist** | Weaver | Refactoring specialist | `/ant:build` |
| **Specialist** | Auditor | Quality gate | `/ant:continue` |
| **Niche** | Chaos | Resilience testing | `/ant:chaos`, `/ant:build` |
| **Niche** | Archaeologist | Excavates git history | `/ant:archaeology`, `/ant:build` |
| **Niche** | Gatekeeper | Security gate | `/ant:continue` |
| **Niche** | Includer | Accessibility audits | `/ant:build` |
| **Niche** | Measurer | Performance analysis | `/ant:continue` |
| **Niche** | Sage | Wisdom synthesis | `/ant:seal` |
| **Niche** | Ambassador | External integrations | `/ant:build` |
| **Niche** | Chronicler | Documentation | `/ant:build`, `/ant:seal` |

---

## Spawn Depth

```
👑 Queen (depth 0)
└── 🔨🐜 Builder-1 (depth 1) — can spawn 4 more
    ├── 🔍🐜 Scout-7 (depth 2) — can spawn 2 more
    │   └── 🔍🐜 Scout-12 (depth 3) — no more spawning
    └── 👁️🐜 Watcher-3 (depth 2)
```

- **Depth 1**: Up to 4 spawns
- **Depth 2**: Up to 2 spawns (only if genuinely surprised)
- **Depth 3**: Complete inline, no further spawning
- **Global cap**: 10 workers per phase

---

## 6-Phase Verification

Before any phase advances:

| Gate | Check |
|------|-------|
| Build | Project compiles/bundles |
| Types | Type checker passes |
| Lint | Linter passes |
| Tests | All tests pass |
| Security | No exposed secrets |
| Diff | Review changes |

---

## Colony Memory (QUEEN.md)

The colony learns and persists wisdom across sessions:

- **📜 Philosophies** — Core beliefs about how to build
- **🧭 Patterns** — Reusable solutions that worked
- **⚠️ Redirects** — Things to avoid (hard constraints)
- **🔧 Stack Wisdom** — Technology-specific learnings
- **🏛️ Decrees** — Immediate rules from user feedback

View memory: `/ant:memory-details`

---

## File Structure

```
<your-repo>/.aether/              # Repo-local colony files
    ├── QUEEN.md                  # Colony wisdom (persists across sessions)
    ├── workers.md                # Worker specs and spawn protocol
    ├── aether-utils.sh           # Utility layer (125 subcommands)
    ├── model-profiles.yaml       # Model routing config
    │
    ├── docs/                     # Documentation
    ├── utils/                    # Utility scripts
    ├── templates/                # File templates
    ├── schemas/                  # JSON schemas
    │
    ├── data/                     # State (NEVER synced by updates)
    │   ├── COLONY_STATE.json     # Goal, plan, memory
    │   ├── constraints.json      # Active constraints
    │   ├── pheromones.json       # Signal tracking
    │   ├── learning-observations.json  # Pattern observations
    │   └── midden/               # Failure signal tracking
    │
    ├── dreams/                   # Session notes
    └── chambers/                 # Archived colonies
```

---

## Typical Workflows

### Starting a New Project

```
1. /ant:init "Build feature X"     # Set the goal
2. /ant:colonize                    # Analyze codebase (4 parallel scouts)
3. /ant:plan                        # Generate phases
4. /ant:focus "security"            # Guide attention (optional)
5. /ant:redirect "use ORM"          # Set hard constraint (optional)
6. /ant:build 1                     # Execute phase 1
7. /ant:continue                    # Verify, advance
8. Repeat until done
9. /ant:patrol                      # Pre-seal audit
10. /ant:seal                       # Complete and archive

# Or use autopilot (replaces steps 6-8):
6. /ant:run                         # Auto build/verify/advance all phases
```

### Deep Research with Oracle

```
/ant:oracle "research topic"    # Launch Oracle (50+ iteration loop)
/ant:oracle status              # Check progress
/ant:oracle stop                # Stop if needed
# Read findings in .aether/oracle/discoveries/
```

### When Stuck on a Bug

```
/ant:swarm "bug description"    # 4 parallel scouts investigate
/ant:archaeology src/module/    # Excavate why code exists
/ant:chaos "auth flow"          # Test edge cases
```

### Providing Feedback

```
/ant:focus "performance"        # "Pay attention to performance"
/ant:redirect "jQuery"          # "Don't use jQuery"
/ant:feedback "prefer composition over inheritance"
```

---

## CLI Commands

```bash
aether version              # View version
aether update               # Update system files from hub
aether update --all         # Update all registered repos
aether telemetry            # View usage stats
aether spawn-tree           # Display worker spawn tree
aether context              # Show context including nestmates
```

---

## Safety Features

- **File Locking** — Prevents concurrent modification
- **Atomic Writes** — Temp file + rename pattern
- **State Validation** — Schema validation before modifications
- **Session Freshness Detection** — Stale sessions detected and handled
- **Git Checkpoints** — Automatic commits before phases

---

## License

MIT
