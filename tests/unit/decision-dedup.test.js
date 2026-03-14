/**
 * Decision Dedup Integration Tests (DEC-01)
 *
 * Verifies that the decision-to-pheromone emission format is aligned
 * between context-update decision (real-time path) and continue-advance
 * Step 2.1b (batch path), ensuring deduplication works reliably.
 *
 * Tests:
 * 1. context-update decision emits [decision] format pheromone
 * 2. Both emission paths produce consistent signals (context-update always emits)
 * 3. Step 2.1b dedup jq query catches auto:decision signals
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-dedup-'));
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

// Helper to setup test colony structure
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md from template
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.MD --- Colony Wisdom

> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## Philosophies

*No philosophies recorded yet*

---

## Patterns

*No patterns recorded yet*

---

## Redirects

*No redirects recorded yet*

---

## Stack Wisdom

*No stack wisdom recorded yet*

---

## Decrees

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
    current_phase: 1,
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

  // Create pheromones.json (optionally with pre-populated signals)
  const signals = opts.pheromoneSignals || [];
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: signals, version: '1.0.0' }, null, 2)
  );

  // Write CONTEXT.md for context-update decision to work
  const contextMd = `# Aether Colony -- Current Context

| Field | Value |
|-------|-------|
| **Colony** | test |
| **Goal** | test |
| **Current Phase** | 1 |
| **Last Updated** | ${isoDate} |
| **Safe to Clear?** | Yes |

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|

---

## Recent Activity

*No recent activity*
`;
  await fs.promises.writeFile(path.join(aetherDir, 'CONTEXT.md'), contextMd);

  return { aetherDir, dataDir };
}


// =============================================================================
// Test 1: context-update decision emits [decision] format pheromone
// =============================================================================

test.serial('context-update decision emits [decision] format pheromone', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Run context-update decision
    runAetherUtil(tmpDir, 'context-update', [
      'decision', 'Use awk for parsing', 'More reliable than sed'
    ]);

    // Read pheromones.json and verify the signal format
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));

    const signal = pheromones.signals.find(s =>
      s.source === 'auto:decision' && s.content.text.includes('[decision]')
    );
    t.truthy(signal, 'Should find signal with source auto:decision and [decision] format');
    t.is(signal.content.text, '[decision] Use awk for parsing',
      'Signal content should be "[decision] Use awk for parsing"');
    t.is(signal.source, 'auto:decision', 'Source should be auto:decision');
    t.is(signal.strength, 0.6, 'Strength should be 0.6');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 2: context-update always emits (dedup is external, in Step 2.1b)
// =============================================================================

test.serial('dedup catches existing auto:decision pheromone', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Pre-populate with an existing auto:decision pheromone
    const now = new Date();
    const expiresAt = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_feedback_existing_001',
        type: 'FEEDBACK',
        priority: 'low',
        source: 'auto:decision',
        created_at: now.toISOString(),
        expires_at: expiresAt.toISOString(),
        active: true,
        strength: 0.6,
        reason: 'Auto-emitted from architectural decision',
        content: { text: '[decision] Use awk for parsing' }
      }]
    });

    // Run context-update decision with the same decision text
    runAetherUtil(tmpDir, 'context-update', [
      'decision', 'Use awk for parsing', 'reason'
    ]);

    // context-update always emits (it does NOT dedup), so we should see 2 signals
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    const autoDecisionSignals = pheromones.signals.filter(
      s => s.source === 'auto:decision' &&
           s.content.text.includes('[decision] Use awk for parsing')
    );

    // context-update always appends -- dedup is Step 2.1b's responsibility
    t.is(autoDecisionSignals.length, 2,
      'context-update always emits (2 signals), confirming dedup must be external (Step 2.1b)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test 3: Step 2.1b dedup jq query finds auto:decision signals
// =============================================================================

test.serial('Step 2.1b dedup jq query finds auto:decision signals', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Pre-populate with a [decision] format pheromone with auto:decision source
    const now = new Date();
    const expiresAt = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_feedback_dedup_001',
        type: 'FEEDBACK',
        priority: 'low',
        source: 'auto:decision',
        created_at: now.toISOString(),
        expires_at: expiresAt.toISOString(),
        active: true,
        strength: 0.6,
        reason: 'Auto-emitted from architectural decision',
        content: { text: '[decision] Use awk for parsing' }
      }]
    });

    // Run the exact jq dedup query from Step 2.1b
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const result = execSync(
      `jq -r --arg text "Use awk for parsing" '[.signals[] | select(.active == true and (.source == "auto:decision" or .source == "system:decision") and (.content.text | contains($text)))] | length' "${pherFile}"`,
      { encoding: 'utf8' }
    ).trim();

    t.is(result, '1',
      'Dedup query should find 1 existing signal, meaning Step 2.1b would skip emission');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
