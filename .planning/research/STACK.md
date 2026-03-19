# Stack Research: Pheromone Integration for Aether Colony System

**Domain:** Multi-agent CLI orchestration / signal-event system maintenance
**Researched:** 2026-03-19
**Confidence:** HIGH (existing stack is well-established; recommendations are evolutionary, not revolutionary)

## Context: Existing Stack (Not Changing)

Aether is already built. This research is about what to **add or upgrade** to make the pheromone system work end-to-end, not about rebuilding the foundation. The existing stack is:

| Layer | Technology | Status |
|-------|-----------|--------|
| Core logic | Bash (`aether-utils.sh`, ~10K lines, 150 subcommands) | Staying |
| CLI entry | Node.js + Commander.js v12 | Staying |
| JSON processing | `jq` (system dependency) | Staying |
| File safety | Custom `file-lock.sh` + `atomic-write.sh` | Staying |
| Testing (JS) | AVA v6 | Staying |
| Testing (bash) | Custom test harness (`test-aether-utils.sh`) | Staying |
| Distribution | npm package | Staying |
| State storage | JSON files (COLONY_STATE.json, pheromones.json, etc.) | Staying |

**The job is not new infrastructure. The job is wiring existing pieces together and hardening the seams.**

---

## Recommended Stack Additions

### Core Technologies (Upgrade/Add)

| Technology | Version | Purpose | Why Recommended | Confidence |
|------------|---------|---------|-----------------|------------|
| Commander.js | ^14.0.3 | CLI framework (upgrade from ^12) | Current version requires Node v20+, matches project's direction. v15 goes ESM-only in May 2026 -- stay on v14 for now to keep CJS compatibility with existing codebase | HIGH |
| Ajv | ^8.18.0 | JSON Schema validation for pheromone signals | Fastest validator at 14M ops/sec. Validates signal shape at write-time to prevent malformed pheromones from entering the system. Aether already uses JSON Schema concepts (XSD for XML exchange); Ajv standardizes this for JSON | HIGH |
| picocolors | ^1.1.1 (already installed) | Terminal colors | Already a dependency. No change needed | HIGH |

### Supporting Libraries (Add)

| Library | Version | Purpose | When to Use | Confidence |
|---------|---------|---------|-------------|------------|
| Ajv | ^8.18.0 | Validate pheromone signal JSON before write | Every `pheromone-write` call. Schema ensures `type`, `content.text`, `strength`, `expires_at` are valid before persisting | HIGH |
| chokidar | ^4.0.3 | File watching for live pheromone monitoring | Only if building a `--watch` mode for `/ant:pheromones`. Current `pheromone-display` is poll-based; chokidar enables reactive updates. Used in 30M+ repos, proven stable | MEDIUM |
| js-yaml | ^4.1.0 (already installed) | YAML parsing | Already a dependency. No change needed | HIGH |

### Development Tools (No Changes)

| Tool | Purpose | Notes |
|------|---------|-------|
| AVA ^6.0 | Unit tests (JS layer) | Current version 6.4.1 is latest stable. Supports Node 20+. No upgrade needed |
| ShellCheck | Bash linting | Already in lint scripts. Keep using for new bash code |
| bats-core 1.13.0 | Bash test framework | **Recommended addition** for structured bash testing instead of the custom harness. TAP-compliant, supports setup/teardown. However, the existing custom harness works -- adopt only if adding significant new bash test coverage |
| sinon ^19.0.5 | JS test mocking | Already installed. No change needed |
| proxyquire ^2.1.3 | Module stubbing | Already installed. No change needed |

---

## What the Pheromone Integration Actually Needs (Stack Perspective)

The gap is not missing libraries. The gap is **missing wiring**. Here is what each integration point needs from a technology standpoint:

### 1. Signal Validation (Ajv -- NEW)

**Problem:** `pheromone-write` accepts any JSON shape. Malformed signals cause silent failures downstream when `pheromone-read` or `pheromone-prime` tries to process them.

**Solution:** Add Ajv-based JSON Schema validation in the Node.js CLI layer. Define a schema for the pheromone signal shape and validate before the bash layer writes.

**Why Ajv over alternatives:**
- Ajv: 14M ops/sec, JSON Schema standard, shareable across languages
- Zod: Great for TypeScript, but Aether is CJS JavaScript -- no TypeScript benefit
- Joi: Slower, heavier API, no JSON Schema standard compatibility

**Implementation pattern:**
```javascript
// bin/lib/pheromone-schema.js
const Ajv = require('ajv');
const ajv = new Ajv({ allErrors: true });

const signalSchema = {
  type: 'object',
  required: ['type', 'content'],
  properties: {
    type: { enum: ['FOCUS', 'REDIRECT', 'FEEDBACK'] },
    content: {
      type: 'object',
      required: ['text'],
      properties: { text: { type: 'string', minLength: 1 } }
    },
    strength: { type: 'number', minimum: 0, maximum: 1 },
    expires_at: { type: 'string' },
  }
};

const validate = ajv.compile(signalSchema);
module.exports = { validate, signalSchema };
```

### 2. Signal Reading in Workers (No New Tech -- Bash)

**Problem:** Workers (Builder, Watcher, Chaos, etc.) receive `prompt_section` from `colony-prime` but don't parse or act on individual signals programmatically.

**Solution:** This is a prompt engineering + bash wiring problem. The `pheromone-prime` output already produces structured text blocks. Workers need to be updated to read and respond to these blocks. No new library needed.

### 3. Signal Auto-Emission During Builds (No New Tech -- Bash)

**Problem:** Auto-emission in `continue-advance.md` Steps 2.1a-2.1d is well-defined but fragile. Failures in pheromone writes silently swallow errors.

**Solution:** Add structured error reporting in the existing bash layer. The `memory-capture` function already handles auto-pheromone emission. The wiring gap is in the build playbooks, not in missing technology.

### 4. Signal Lifecycle Management (No New Tech -- jq)

**Problem:** Decay calculation in `pheromone-read` uses approximate epoch math (30-day months). Expired signals accumulate in pheromones.json.

**Solution:** Fix the jq decay math and add a cleanup step. No new library -- this is a jq logic fix.

### 5. Cross-Session Signal Persistence (No New Tech -- File I/O)

**Problem:** Signals with `expires_at: "phase_end"` are expired by `pheromone-expire --phase-end-only`, but there is no mechanism to carry high-value signals across colony sessions.

**Solution:** The `eternal-init` and `pheromone-export-eternal` subcommands already exist. The gap is that they are not called at the right lifecycle points. This is a playbook wiring fix, not a technology gap.

---

## Installation

```bash
# New dependency (only addition)
npm install ajv@^8.18.0

# Optional: upgrade commander (currently ^12, recommend ^14)
npm install commander@^14.0.3

# Optional: add file watching for live pheromone display
npm install chokidar@^4.0.3

# Optional: structured bash testing
brew install bats-core  # or: npm install -g bats
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Ajv (JSON Schema) | Zod | If project migrates to TypeScript. Zod's type inference is wasted in plain JS |
| Ajv (JSON Schema) | Joi | If you need complex custom validation beyond schema (e.g., conditional required fields with async checks). Joi is 40x slower but more expressive for edge cases |
| Keep bash for signal logic | Migrate signal logic to Node.js | If aether-utils.sh exceeds ~15K lines and jq operations become the bottleneck. Currently at ~10K lines -- still manageable |
| jq for JSON processing | Node.js native JSON | If concurrent write contention becomes a real problem (multiple agents writing pheromones.json simultaneously). Node.js has better locking primitives. Current file-lock.sh is adequate for sequential builds |
| Custom bash test harness | bats-core 1.13.0 | If adding 20+ new bash test cases for pheromone integration. bats provides proper test isolation, TAP output, and setup/teardown. For fewer tests, the existing harness is fine |
| chokidar for file watching | Node.js native fs.watch | If targeting Node 22+ only. fs.watch is now more reliable on modern Node but still has cross-platform quirks. chokidar handles edge cases |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Event sourcing frameworks (reSolve, EventSourcing.NodeJS) | Massive overkill. Aether's pheromone system is a simple signal store with decay, not a CQRS event log. Adding event sourcing infrastructure would triple complexity for zero benefit | JSON files with atomic writes (already have this) |
| Redis / SQLite for signal storage | Adds a runtime dependency to a CLI tool distributed via npm. Users would need Redis running or SQLite binaries. JSON files are portable, human-readable, and sufficient at Aether's scale (dozens of signals, not millions) | pheromones.json with file locking (already have this) |
| RxJS / event emitter libraries | Signals are not real-time streams. They are written during builds and read during builds. There is no "push" use case. Adding reactive programming adds conceptual overhead with no benefit | Direct file read/write with jq (already have this) |
| Multi-agent orchestration frameworks (LangGraph, CrewAI, AutoGen) | These are for LLM-to-LLM agent orchestration. Aether's agents are prompt-driven Claude Code subagents, not autonomous LLM chains. Different paradigm entirely | Current slash-command + Task tool spawning (already have this) |
| TypeScript migration | The codebase is CJS Node.js + bash. A TypeScript migration would touch every JS file, break the existing test suite, and add a build step -- all for a system that is prompt-driven, not logic-heavy. The Node.js layer is thin (CLI entry, file operations, validation). The real logic lives in bash and markdown playbooks | Keep CJS JavaScript. Add JSDoc type annotations if type safety is desired |
| Zod | Only beneficial for TypeScript projects. In plain CJS JavaScript, Zod's type inference provides no benefit over Ajv, and Ajv is 7x faster | Ajv |
| Complex pub/sub (NATS, RabbitMQ) | CLI tool, not a distributed service. Agents run in the same process tree. File-based signaling is the right abstraction for this scale | File-based pheromones.json (already have this) |

---

## Stack Patterns by Variant

**If adding new pheromone subcommands (bash):**
- Follow existing pattern in `aether-utils.sh`: case branch, `json_ok`/`json_err` return, file-lock + atomic-write for mutations
- Use `jq` for all JSON processing (no inline node calls from bash)
- Add corresponding test in `tests/bash/test-aether-utils.sh`

**If adding new validation/schema logic (Node.js):**
- Add to `bin/lib/` directory following existing patterns (`errors.js`, `model-profiles.js`)
- Use Ajv for schema compilation at module load time (compile once, validate many)
- Add AVA tests in `tests/unit/`

**If modifying build/continue playbooks (markdown):**
- Playbooks are in `.aether/docs/command-playbooks/`
- Changes here are prompt-engineering, not code -- no library dependency
- Test by running the actual command in a colony session

**If adding cross-session signal persistence:**
- Use existing `eternal-init` + `pheromone-export-eternal` subcommands
- Storage goes to `~/.aether/eternal/pheromones.xml`
- No new technology needed -- just call existing subcommands at right lifecycle points

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| commander@^14.0.3 | Node.js >=20 | v15 (May 2026) will be ESM-only. Stay on v14 for CJS compatibility |
| ajv@^8.18.0 | Node.js >=16 | Supports both CJS and ESM. Safe to use with current `"engines": {"node": ">=16.0.0"}` |
| chokidar@^4.0.3 | Node.js >=14 | v5 is ESM-only. Stay on v4 if adding this dependency |
| ava@^6.0 | Node.js >=20 | Already installed. No compatibility concerns |
| jq | Any version >= 1.5 | System dependency. Most systems have 1.6+. No concerns |
| bash | 3.2+ | macOS ships 3.2 (GPLv2). All existing code works with 3.2. Avoid bash 4+ features (associative arrays, `${var,,}` lowercasing) |

---

## Key Insight: The Stack is Not the Problem

The existing stack (bash + jq + Node.js + JSON files) is well-suited for Aether's pheromone system. The integration gap is **behavioral, not technological**:

1. **Workers receive signals but do not act on them** -- this is a prompt/playbook problem
2. **Auto-emission exists but is fragile** -- this is error-handling in bash, not a missing library
3. **Signals do not carry across sessions** -- the infrastructure exists (`eternal-*`), it is just not wired into lifecycle
4. **No validation at write time** -- Ajv fixes this (the one actual technology gap)
5. **Decay math is approximate** -- this is a jq formula fix

Adding Ajv for schema validation is the only genuine technology addition. Everything else is wiring, error handling, and prompt engineering within the existing stack.

---

## Sources

- [Commander.js npm page](https://www.npmjs.com/package/commander) -- verified v14.0.3 is latest stable, v15 ESM-only in May 2026 (HIGH confidence)
- [Ajv npm page](https://www.npmjs.com/package/ajv) -- verified v8.18.0 is latest stable (HIGH confidence)
- [Ajv official docs](https://ajv.js.org/) -- JSON Schema draft-2020-12 support confirmed (HIGH confidence)
- [AVA GitHub](https://github.com/avajs/ava) -- v6.4.1 confirmed latest, Node 20+ required (HIGH confidence)
- [bats-core GitHub](https://github.com/bats-core/bats-core) -- v1.13.0 latest, Bash 3.2+ compatible (HIGH confidence)
- [chokidar GitHub](https://github.com/paulmillr/chokidar) -- v4 confirmed CJS-compatible (MEDIUM confidence)
- [Validation library comparison](https://www.bitovi.com/blog/comparing-schema-validation-libraries-ajv-joi-yup-and-zod) -- Ajv 14M ops/sec vs Joi 322K vs Zod 1.9M benchmarks (MEDIUM confidence)
- [jq issue #2152](https://github.com/jqlang/jq/issues/2152) -- confirmed in-place write hazard requiring temp file pattern (HIGH confidence)
- [Stigmergy patterns in multi-agent systems](https://www.rodriguez.today/articles/emergent-coordination-without-managers) -- validated pheromone/stigmergy as established coordination pattern (MEDIUM confidence)
- [TTL best practices](https://www.imperva.com/learn/performance/time-to-live-ttl/) -- decay/expiration patterns verified against existing Aether implementation (MEDIUM confidence)
- Existing codebase analysis: `aether-utils.sh`, `pheromones.json`, `file-lock.sh`, `atomic-write.sh`, build/continue playbooks (HIGH confidence -- primary source)

---
*Stack research for: Aether pheromone integration milestone*
*Researched: 2026-03-19*
