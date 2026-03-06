/**
 * Learnings Injection Integration Tests
 *
 * End-to-end tests for the learnings injection pipeline:
 * phase_learnings in COLONY_STATE.json -> colony-prime -> prompt_section
 *
 * These tests verify that LEARN-01 and LEARN-04 work together correctly:
 * validated claims from previous phases reach builder prompts, while
 * hypothesis/disproven claims and current/future phase learnings are excluded.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-learnings-'));
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
// Extended for learnings injection: accepts phaseLearnings and currentPhase options
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md from template (METADATA on single line to avoid awk issues)
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md \u2014 Colony Wisdom

> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0

---

## \u{1F4DC} Philosophies

Core beliefs that guide all colony work.

*No philosophies recorded yet*

---

## \u{1F9ED} Patterns

Validated approaches that consistently work.

*No patterns recorded yet*

---

## \u{26A0}\u{FE0F} Redirects

Anti-patterns to avoid.

*No redirects recorded yet*

---

## \u{1F527} Stack Wisdom

Technology-specific insights.

*No stack wisdom recorded yet*

---

## \u{1F3DB}\u{FE0F} Decrees

User-mandated rules.

*No decrees recorded yet*

---

## \u{1F4CA} Evolution Log

| Date | Colony | Change | Details |
|------|--------|--------|---------|

---

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);

  // Create COLONY_STATE.json
  const instincts = opts.instincts || [];
  const phaseLearnings = opts.phaseLearnings || [];
  const currentPhase = opts.currentPhase !== undefined ? opts.currentPhase : 1;

  const colonyState = {
    session_id: 'colony_test',
    goal: 'test',
    state: 'BUILDING',
    current_phase: currentPhase,
    plan: { phases: [] },
    memory: {
      instincts: instincts,
      phase_learnings: phaseLearnings,
      decisions: []
    },
    errors: { flagged_patterns: [] },
    events: []
  };
  await fs.promises.writeFile(
    path.join(dataDir, 'COLONY_STATE.json'),
    JSON.stringify(colonyState, null, 2)
  );

  // Create pheromones.json
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: [], version: '1.0.0' }, null, 2)
  );

  return { aetherDir, dataDir };
}


test.serial('colony-prime includes validated learnings from previous phases', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 3,
      phaseLearnings: [
        {
          id: 'pl_1',
          phase: 1,
          phase_name: 'foundation',
          learnings: [
            { claim: 'Use barrel exports for clean imports', status: 'validated' },
            { claim: 'Avoid circular dependencies', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        },
        {
          id: 'pl_2',
          phase: 2,
          phase_name: 'integration',
          learnings: [
            { claim: 'Integration tests catch wiring bugs', status: 'validated' },
            { claim: 'Might need caching layer', status: 'hypothesis' }
          ],
          timestamp: new Date().toISOString()
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Validated claims should appear
    t.true(section.includes('Use barrel exports for clean imports'),
      'Should include validated claim from phase 1');
    t.true(section.includes('Avoid circular dependencies'),
      'Should include second validated claim from phase 1');
    t.true(section.includes('Integration tests catch wiring bugs'),
      'Should include validated claim from phase 2');

    // Hypothesis claim should NOT appear
    t.false(section.includes('Might need caching layer'),
      'Should NOT include hypothesis claim');

    // Header should be present
    t.true(section.includes('PHASE LEARNINGS'),
      'Should contain PHASE LEARNINGS header');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime excludes learnings from current and future phases', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      phaseLearnings: [
        {
          id: 'pl_past',
          phase: 1,
          phase_name: 'setup',
          learnings: [
            { claim: 'Past phase claim should appear', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        },
        {
          id: 'pl_current',
          phase: 2,
          phase_name: 'current',
          learnings: [
            { claim: 'Current phase claim should NOT appear', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        },
        {
          id: 'pl_future',
          phase: 3,
          phase_name: 'future',
          learnings: [
            { claim: 'Future phase claim should NOT appear', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Only phase 1 claims should appear
    t.true(section.includes('Past phase claim should appear'),
      'Should include claim from past phase');
    t.false(section.includes('Current phase claim should NOT appear'),
      'Should NOT include claim from current phase');
    t.false(section.includes('Future phase claim should NOT appear'),
      'Should NOT include claim from future phase');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime includes inherited learnings', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      phaseLearnings: [
        {
          id: 'pl_inherited',
          phase: 'inherited',
          phase_name: '',
          learnings: [
            { claim: 'Always validate input at boundaries', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        },
        {
          id: 'pl_phase1',
          phase: 1,
          phase_name: 'bootstrap',
          learnings: [
            { claim: 'TDD catches regressions early', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Both inherited and phase 1 claims should appear
    t.true(section.includes('Always validate input at boundaries'),
      'Should include inherited learning');
    t.true(section.includes('TDD catches regressions early'),
      'Should include phase 1 learning');

    // Inherited group label should be present
    t.true(section.includes('Inherited'),
      'Should contain Inherited group label');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime omits section when no validated learnings exist', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      phaseLearnings: [
        {
          id: 'pl_hypo',
          phase: 1,
          phase_name: 'exploration',
          learnings: [
            { claim: 'This is just a hypothesis', status: 'hypothesis' },
            { claim: 'This was disproven', status: 'disproven' }
          ],
          timestamp: new Date().toISOString()
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // No validated learnings -> no PHASE LEARNINGS section
    t.false(section.includes('PHASE LEARNINGS'),
      'Should NOT contain PHASE LEARNINGS when no validated learnings exist');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime omits section when no previous phases have learnings', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Phase 1 with no previous phases -> no learnings to inject
    await setupTestColony(tmpDir, {
      currentPhase: 1,
      phaseLearnings: []
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    t.false(section.includes('PHASE LEARNINGS'),
      'Should NOT contain PHASE LEARNINGS at phase 1 with no learnings');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime respects compact mode cap', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Create 10 validated claims across phases 1 and 2
    await setupTestColony(tmpDir, {
      currentPhase: 3,
      phaseLearnings: [
        {
          id: 'pl_many1',
          phase: 1,
          phase_name: 'first',
          learnings: [
            { claim: 'Compact claim A1', status: 'validated' },
            { claim: 'Compact claim A2', status: 'validated' },
            { claim: 'Compact claim A3', status: 'validated' },
            { claim: 'Compact claim A4', status: 'validated' },
            { claim: 'Compact claim A5', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        },
        {
          id: 'pl_many2',
          phase: 2,
          phase_name: 'second',
          learnings: [
            { claim: 'Compact claim B1', status: 'validated' },
            { claim: 'Compact claim B2', status: 'validated' },
            { claim: 'Compact claim B3', status: 'validated' },
            { claim: 'Compact claim B4', status: 'validated' },
            { claim: 'Compact claim B5', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        }
      ]
    });

    // Run with --compact flag
    const result = runAetherUtil(tmpDir, 'colony-prime', ['--compact']);
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Extract the PHASE LEARNINGS section
    const learningsStart = section.indexOf('PHASE LEARNINGS');
    const learningsEnd = section.indexOf('END PHASE LEARNINGS');
    t.true(learningsStart !== -1, 'Should have PHASE LEARNINGS section');
    t.true(learningsEnd !== -1, 'Should have END PHASE LEARNINGS marker');

    const learningsSection = section.substring(learningsStart, learningsEnd);

    // Count claim bullets (lines starting with "  - ")
    const claimLines = learningsSection.split('\n').filter(line => line.trimStart().startsWith('- '));
    t.true(claimLines.length <= 5,
      `Compact mode should cap at 5 claims, found ${claimLines.length}`);
    t.true(claimLines.length > 0,
      'Should have at least 1 claim in compact mode');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime log_line includes learning count', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 2,
      phaseLearnings: [
        {
          id: 'pl_log',
          phase: 1,
          phase_name: 'counted',
          learnings: [
            { claim: 'First countable learning', status: 'validated' },
            { claim: 'Second countable learning', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const logLine = resultJson.result.log_line;
    t.true(logLine.includes('learnings'),
      'Log line should mention learnings');
    t.true(logLine.includes('2 learnings'),
      'Log line should report correct count of 2 learnings');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('colony-prime formats learnings as actionable text grouped by phase', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir, {
      currentPhase: 3,
      phaseLearnings: [
        {
          id: 'pl_fmt1',
          phase: 1,
          phase_name: 'setup',
          learnings: [
            { claim: 'Setup phase claim for formatting', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        },
        {
          id: 'pl_fmt2',
          phase: 2,
          phase_name: 'wiring',
          learnings: [
            { claim: 'Wiring phase claim for formatting', status: 'validated' }
          ],
          timestamp: new Date().toISOString()
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;

    // Phase group headers should be present
    t.true(section.includes('Phase 1'),
      'Should contain Phase 1 group header');
    t.true(section.includes('Phase 2'),
      'Should contain Phase 2 group header');

    // Phase names should appear in headers
    t.true(section.includes('setup'),
      'Should include phase_name "setup" in header');
    t.true(section.includes('wiring'),
      'Should include phase_name "wiring" in header');

    // Claims should be indented with "  - " prefix
    t.true(section.includes('  - Setup phase claim for formatting'),
      'Claims should be indented with "  - " prefix');
    t.true(section.includes('  - Wiring phase claim for formatting'),
      'Claims should be indented with "  - " prefix');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
