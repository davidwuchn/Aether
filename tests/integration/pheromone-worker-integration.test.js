/**
 * Pheromone Worker Integration Tests
 *
 * End-to-end tests verifying:
 * PHER-04: Cross-phase signal influence (auto-emitted signals appear in subsequent colony-prime prompt_section
 *          AND agent definitions contain explicit pheromone_protocol instructions)
 * PHER-05: Midden threshold auto-REDIRECT (3+ failures in same category creates REDIRECT,
 *          appears in prompt_section, deduplication works, below-threshold is ignored)
 *
 * These tests prove the behavioral loop is structurally closed:
 * failure/learning -> auto-emit signal -> signal in prompt_section -> agent has instructions to act on it
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-worker-'));
  return tmpDir;
}

// Helper to cleanup temp directory
async function cleanupTempDir(tmpDir) {
  try {
    await fs.promises.rm(tmpDir, { recursive: true, force: true });
  } catch (err) {
    // Ignore cleanup errors
  }
}

// Helper to run aether-utils.sh commands
function runAetherUtil(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

// Helper to setup test colony structure with COLONY_STATE.json and pheromones.json
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md (required by colony-prime)
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md --- Colony Wisdom

> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## Philosophies

Core beliefs that guide all colony work.

*No philosophies recorded yet*

---

## Patterns

Validated approaches that consistently work.

*No patterns recorded yet*

---

## Redirects

Anti-patterns to avoid.

*No redirects recorded yet*

---

## Stack Wisdom

Technology-specific insights.

*No stack wisdom recorded yet*

---

## Decrees

User-mandated rules.

*No decrees recorded yet*

---

## Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);

  // Create COLONY_STATE.json
  const colonyState = {
    session_id: 'colony_test',
    goal: 'test',
    state: 'BUILDING',
    current_phase: opts.currentPhase !== undefined ? opts.currentPhase : 1,
    plan: { phases: [] },
    memory: {
      instincts: [],
      phase_learnings: [],
      decisions: []
    },
    errors: { flagged_patterns: [] },
    events: []
  };
  await fs.promises.writeFile(
    path.join(dataDir, 'COLONY_STATE.json'),
    JSON.stringify(colonyState, null, 2)
  );

  // Create pheromones.json (optionally with signals)
  const signals = opts.pheromoneSignals || [];
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: signals, version: '1.0.0' }, null, 2)
  );

  return { aetherDir, dataDir };
}


// =============================================================================
// Test Group 1: Cross-Phase Signal Influence (PHER-04)
// =============================================================================

test.serial('PHER-04: auto-emitted failure REDIRECT appears in subsequent prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Simulate what happens during a build: a failure event triggers memory-capture
    // which auto-emits a REDIRECT pheromone
    const captureResult = runAetherUtil(tmpDir, 'memory-capture', [
      'failure',
      'Builder failed on parsing task: null pointer in JSON handler',
      'failure',
      'worker:builder'
    ]);
    const captureJson = JSON.parse(captureResult);
    t.true(captureJson.ok, 'memory-capture should return ok=true');
    t.true(captureJson.result.pheromone_created, 'memory-capture should auto-create a pheromone');

    // Now simulate the subsequent build phase: call colony-prime --compact
    // to get the prompt_section that would be injected into the next worker
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true');

    const section = primeJson.result.prompt_section;
    t.truthy(section, 'prompt_section should not be empty');

    // The auto-emitted REDIRECT from the failure should appear in prompt_section
    t.true(
      section.includes('null pointer in JSON handler') || section.includes('parsing task'),
      'prompt_section should contain the failure description from the auto-emitted REDIRECT'
    );
    t.true(
      section.includes('REDIRECT') || section.includes('HARD CONSTRAINT'),
      'prompt_section should contain REDIRECT label for the failure signal'
    );
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('PHER-04: auto-emitted learning FEEDBACK appears in subsequent prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Simulate a learning event via memory-capture (auto-emits FEEDBACK)
    const captureResult = runAetherUtil(tmpDir, 'memory-capture', [
      'learning',
      'awk is faster than sed for multi-field extraction',
      'pattern',
      'worker:builder'
    ]);
    const captureJson = JSON.parse(captureResult);
    t.true(captureJson.ok, 'memory-capture should return ok=true');
    t.true(captureJson.result.pheromone_created, 'memory-capture should auto-create a FEEDBACK pheromone');

    // Call colony-prime --compact to get prompt_section for subsequent build
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true');

    const section = primeJson.result.prompt_section;
    t.truthy(section, 'prompt_section should not be empty');

    // The auto-emitted FEEDBACK from the learning should appear
    t.true(
      section.includes('awk is faster than sed'),
      'prompt_section should contain the learning content from the auto-emitted FEEDBACK'
    );
    t.true(
      section.includes('FEEDBACK') || section.includes('Flexible guidance'),
      'prompt_section should contain FEEDBACK label for the learning signal'
    );
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('PHER-04: multiple auto-emitted signals from different events all appear in prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Emit a REDIRECT from a failure event
    runAetherUtil(tmpDir, 'memory-capture', [
      'failure',
      'Security vulnerability in auth handler',
      'failure',
      'worker:builder'
    ]);

    // Emit a FEEDBACK from a learning event
    runAetherUtil(tmpDir, 'memory-capture', [
      'learning',
      'Input validation should precede business logic',
      'pattern',
      'worker:watcher'
    ]);

    // Call colony-prime --compact to get combined prompt_section
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true');

    const section = primeJson.result.prompt_section;
    t.truthy(section, 'prompt_section should not be empty');

    // Both auto-emitted signals should appear
    t.true(
      section.includes('Security vulnerability') || section.includes('auth handler'),
      'prompt_section should contain the failure REDIRECT content'
    );
    t.true(
      section.includes('Input validation') || section.includes('business logic'),
      'prompt_section should contain the learning FEEDBACK content'
    );

    // Both signal type headers should be present
    t.true(section.includes('REDIRECT'), 'prompt_section should have REDIRECT section');
    t.true(section.includes('FEEDBACK'), 'prompt_section should have FEEDBACK section');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('PHER-04: agent definitions contain pheromone_protocol with signal handling instructions', async (t) => {
  // Structural verification: read the three agent definition files and verify
  // they contain pheromone_protocol sections with instructions for all signal types.
  // This proves that workers spawned with these definitions will receive both
  // the signal text (via prompt_section) AND instructions to act on it.

  const agentFiles = [
    path.join(process.cwd(), '.claude', 'agents', 'ant', 'aether-builder.md'),
    path.join(process.cwd(), '.claude', 'agents', 'ant', 'aether-watcher.md'),
    path.join(process.cwd(), '.claude', 'agents', 'ant', 'aether-scout.md')
  ];

  for (const agentFile of agentFiles) {
    const agentName = path.basename(agentFile, '.md');
    const content = fs.readFileSync(agentFile, 'utf8');

    // Verify pheromone_protocol section exists
    t.true(
      content.includes('pheromone_protocol'),
      `${agentName} should contain pheromone_protocol section`
    );

    // Verify instructions for each signal type
    t.true(
      content.includes('REDIRECT'),
      `${agentName} should contain REDIRECT handling instructions`
    );
    t.true(
      content.includes('FOCUS'),
      `${agentName} should contain FOCUS handling instructions`
    );
    t.true(
      content.includes('FEEDBACK'),
      `${agentName} should contain FEEDBACK handling instructions`
    );
  }
});
