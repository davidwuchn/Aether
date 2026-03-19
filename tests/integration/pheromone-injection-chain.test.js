/**
 * Pheromone Injection Chain & Lifecycle Integration Tests
 *
 * End-to-end tests verifying:
 * PHER-01: Signal injection chain (pheromone-write -> pheromones.json -> pheromone-prime -> colony-prime -> prompt_section)
 * PHER-02: Signal lifecycle (phase_end expiration, time-based expiration, midden archival, GC excludes from colony-prime)
 *
 * These tests prove that a user-emitted signal actually appears in the worker
 * spawn context (prompt_section) and that expired signals are correctly removed.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-chain-'));
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
// Test Group 1: Injection Chain (PHER-01)
// =============================================================================

test.serial('user-emitted FOCUS signal appears in colony-prime prompt_section', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Write a FOCUS signal via pheromone-write
    const writeResult = runAetherUtil(tmpDir, 'pheromone-write', [
      'FOCUS', 'security review',
      '--source', 'user',
      '--strength', '0.8',
      '--reason', 'User directed'
    ]);
    const writeJson = JSON.parse(writeResult);
    t.true(writeJson.ok, 'pheromone-write should return ok=true');

    // Call colony-prime --compact and check prompt_section
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true');

    const section = primeJson.result.prompt_section;
    t.truthy(section, 'prompt_section should not be empty');
    t.true(section.includes('security review'),
      'prompt_section should contain the signal content "security review"');
    t.true(section.includes('FOCUS') || section.includes('Pay attention'),
      'prompt_section should contain signal type label FOCUS or "Pay attention"');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('user-emitted REDIRECT signal appears in colony-prime with HARD CONSTRAINT label', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Write a REDIRECT signal
    const writeResult = runAetherUtil(tmpDir, 'pheromone-write', [
      'REDIRECT', 'no console.log',
      '--source', 'user',
      '--strength', '0.9'
    ]);
    const writeJson = JSON.parse(writeResult);
    t.true(writeJson.ok, 'pheromone-write should return ok=true');

    // Call colony-prime --compact
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true');

    const section = primeJson.result.prompt_section;
    t.truthy(section, 'prompt_section should not be empty');
    t.true(section.includes('no console.log'),
      'prompt_section should contain the signal content "no console.log"');
    t.true(section.includes('REDIRECT') || section.includes('HARD CONSTRAINT'),
      'prompt_section should contain REDIRECT or HARD CONSTRAINT label');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('multiple signals appear in colony-prime prompt_section with both contents present', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Write FOCUS signal with strength 0.8
    runAetherUtil(tmpDir, 'pheromone-write', [
      'FOCUS', 'performance optimization',
      '--source', 'user',
      '--strength', '0.8'
    ]);

    // Write REDIRECT signal with strength 0.9
    runAetherUtil(tmpDir, 'pheromone-write', [
      'REDIRECT', 'avoid global state',
      '--source', 'user',
      '--strength', '0.9'
    ]);

    // Call colony-prime --compact
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true');

    const section = primeJson.result.prompt_section;
    t.truthy(section, 'prompt_section should not be empty');

    // Both signal contents should appear
    t.true(section.includes('performance optimization'),
      'prompt_section should contain FOCUS signal content');
    t.true(section.includes('avoid global state'),
      'prompt_section should contain REDIRECT signal content');

    // Both type headers should be present
    t.true(section.includes('FOCUS'),
      'prompt_section should have FOCUS section header');
    t.true(section.includes('REDIRECT'),
      'prompt_section should have REDIRECT section header');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('decayed signal below 0.1 threshold does NOT appear in colony-prime', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Create a signal that was created 29 days ago with strength 0.8
    // FOCUS decay_days = 30, so effective = 0.8 * (1 - 29/30) = 0.8 * 0.033 = 0.027 < 0.1
    const now = new Date();
    const twentyNineDaysAgo = new Date(now.getTime() - 29 * 24 * 60 * 60 * 1000);
    const futureExpiry = new Date(now.getTime() + 60 * 24 * 60 * 60 * 1000);

    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_focus_decayed_001',
        type: 'FOCUS',
        priority: 'normal',
        source: 'user',
        created_at: twentyNineDaysAgo.toISOString(),
        expires_at: futureExpiry.toISOString(),
        active: true,
        strength: 0.8,
        reason: 'User directed',
        content: { text: 'decayed signal content xyz' }
      }]
    });

    // Call colony-prime --compact
    const primeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true');

    const section = primeJson.result.prompt_section;
    // The decayed signal should NOT appear in prompt_section
    t.false(
      (section || '').includes('decayed signal content xyz'),
      'prompt_section should NOT contain the decayed signal content'
    );
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Test Group 2: Signal Lifecycle (PHER-02)
// =============================================================================

test.serial('pheromone-expire --phase-end-only expires phase_end signals', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const now = new Date();
    const futureExpiry = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);

    await setupTestColony(tmpDir, {
      pheromoneSignals: [
        {
          id: 'sig_phase_end_001',
          type: 'FOCUS',
          priority: 'normal',
          source: 'user',
          created_at: now.toISOString(),
          expires_at: 'phase_end',
          active: true,
          strength: 0.8,
          reason: 'Phase-scoped focus',
          content: { text: 'phase-end signal' }
        },
        {
          id: 'sig_future_001',
          type: 'REDIRECT',
          priority: 'high',
          source: 'user',
          created_at: now.toISOString(),
          expires_at: futureExpiry.toISOString(),
          active: true,
          strength: 0.9,
          reason: 'Long-lived redirect',
          content: { text: 'future-dated signal' }
        }
      ]
    });

    // Call pheromone-expire --phase-end-only
    const expireResult = runAetherUtil(tmpDir, 'pheromone-expire', ['--phase-end-only']);
    const expireJson = JSON.parse(expireResult);
    t.true(expireJson.ok, 'pheromone-expire should return ok=true');
    t.is(expireJson.result.expired_count, 1, 'Should expire exactly 1 signal (the phase_end one)');

    // Read pheromones.json directly to verify signal states
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));

    const phaseEndSignal = pheromones.signals.find(s => s.id === 'sig_phase_end_001');
    t.is(phaseEndSignal.active, false, 'phase_end signal should have active: false');

    const futureSignal = pheromones.signals.find(s => s.id === 'sig_future_001');
    t.is(futureSignal.active, true, 'future-dated signal should still have active: true');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('pheromone-expire (no flag) expires time-expired signals', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const now = new Date();
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    const thirtyDaysFromNow = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);

    await setupTestColony(tmpDir, {
      pheromoneSignals: [
        {
          id: 'sig_expired_001',
          type: 'FOCUS',
          priority: 'normal',
          source: 'user',
          created_at: yesterday.toISOString(),
          expires_at: yesterday.toISOString(),
          active: true,
          strength: 0.8,
          reason: 'Already expired by time',
          content: { text: 'past-dated signal' }
        },
        {
          id: 'sig_active_001',
          type: 'REDIRECT',
          priority: 'high',
          source: 'user',
          created_at: now.toISOString(),
          expires_at: thirtyDaysFromNow.toISOString(),
          active: true,
          strength: 0.9,
          reason: 'Still active',
          content: { text: 'future-active signal' }
        }
      ]
    });

    // Call pheromone-expire (no flags)
    const expireResult = runAetherUtil(tmpDir, 'pheromone-expire');
    const expireJson = JSON.parse(expireResult);
    t.true(expireJson.ok, 'pheromone-expire should return ok=true');
    t.is(expireJson.result.expired_count, 1, 'Should expire exactly 1 signal (the past-dated one)');

    // Verify signal states in pheromones.json
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));

    const expiredSignal = pheromones.signals.find(s => s.id === 'sig_expired_001');
    t.is(expiredSignal.active, false, 'Past-dated signal should be expired (active: false)');

    const activeSignal = pheromones.signals.find(s => s.id === 'sig_active_001');
    t.is(activeSignal.active, true, 'Future-dated signal should remain active');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('expired signals are archived to midden', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const now = new Date();

    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_to_archive_001',
        type: 'FOCUS',
        priority: 'normal',
        source: 'user',
        created_at: now.toISOString(),
        expires_at: 'phase_end',
        active: true,
        strength: 0.7,
        reason: 'Will be archived on expire',
        content: { text: 'archivable signal' }
      }]
    });

    // Expire the phase_end signal
    const expireResult = runAetherUtil(tmpDir, 'pheromone-expire', ['--phase-end-only']);
    const expireJson = JSON.parse(expireResult);
    t.true(expireJson.ok, 'pheromone-expire should return ok=true');
    t.is(expireJson.result.expired_count, 1, 'Should expire 1 signal');

    // Read midden.json and verify the signal was archived
    const middenFile = path.join(tmpDir, '.aether', 'data', 'midden', 'midden.json');
    t.true(fs.existsSync(middenFile), 'midden.json should exist after expiration');

    const midden = JSON.parse(fs.readFileSync(middenFile, 'utf8'));
    t.true(midden.signals.length >= 1, 'midden signals array should have at least 1 entry');

    const archivedSignal = midden.signals.find(s => s.id === 'sig_to_archive_001');
    t.truthy(archivedSignal, 'midden should contain the expired signal by ID');
    t.is(archivedSignal.active, false, 'Archived signal should have active: false');
    t.truthy(archivedSignal.archived_at, 'Archived signal should have an archived_at timestamp');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('expired signal does not appear in colony-prime after garbage collection', async (t) => {
  const tmpDir = await createTempDir();

  try {
    const now = new Date();

    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_gc_test_001',
        type: 'FOCUS',
        priority: 'normal',
        source: 'user',
        created_at: now.toISOString(),
        expires_at: 'phase_end',
        active: true,
        strength: 0.8,
        reason: 'Will be GCed',
        content: { text: 'gc-test unique content' }
      }]
    });

    // First verify the signal DOES appear in colony-prime before expiration
    const beforeResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const beforeJson = JSON.parse(beforeResult);
    t.true(beforeJson.ok, 'colony-prime should return ok=true (before expire)');
    t.true(
      (beforeJson.result.prompt_section || '').includes('gc-test unique content'),
      'Signal should appear in prompt_section BEFORE expiration'
    );

    // Now expire the signal
    const expireResult = runAetherUtil(tmpDir, 'pheromone-expire', ['--phase-end-only']);
    const expireJson = JSON.parse(expireResult);
    t.is(expireJson.result.expired_count, 1, 'Should expire 1 signal');

    // Verify signal does NOT appear in colony-prime after expiration
    const afterResult = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const afterJson = JSON.parse(afterResult);
    t.true(afterJson.ok, 'colony-prime should return ok=true (after expire)');

    const afterSection = afterJson.result.prompt_section || '';
    t.false(
      afterSection.includes('gc-test unique content'),
      'Signal should NOT appear in prompt_section AFTER expiration'
    );
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
