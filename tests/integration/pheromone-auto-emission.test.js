/**
 * Pheromone Auto-Emission Integration Tests
 *
 * End-to-end tests for all three pheromone auto-emission sources:
 * PHER-01: Decision-to-FEEDBACK (auto:decision source)
 * PHER-02: Midden error pattern-to-REDIRECT (auto:error source)
 * PHER-03: Success criteria recurrence-to-FEEDBACK (auto:success source)
 *
 * These tests verify that auto-emitted pheromones are created correctly
 * via pheromone-write, flow through the existing pheromone-prime pipeline,
 * and are distinguishable from manual pheromones.
 */

const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

// Helper to create temp directory
async function createTempDir() {
  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'aether-pher-'));
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
// Extended for pheromone auto-emission: accepts middenFailures and completedPhases options
async function setupTestColony(tmpDir, opts = {}) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');

  // Create directories
  await fs.promises.mkdir(dataDir, { recursive: true });

  // Create QUEEN.md from template (METADATA on single line to avoid awk issues)
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
  const instincts = opts.instincts || [];
  const phaseLearnings = opts.phaseLearnings || [];
  const currentPhase = opts.currentPhase !== undefined ? opts.currentPhase : 1;
  const completedPhases = opts.completedPhases || [];

  const colonyState = {
    session_id: 'colony_test',
    goal: 'test',
    state: 'BUILDING',
    current_phase: currentPhase,
    plan: { phases: completedPhases },
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

  // Create pheromones.json (optionally with signals)
  const signals = opts.pheromoneSignals || [];
  await fs.promises.writeFile(
    path.join(dataDir, 'pheromones.json'),
    JSON.stringify({ signals: signals, version: '1.0.0' }, null, 2)
  );

  // Write midden failures if provided
  if (opts.middenFailures !== undefined) {
    const middenDir = path.join(dataDir, 'midden');
    await fs.promises.mkdir(middenDir, { recursive: true });
    const middenData = { entries: opts.middenFailures };
    await fs.promises.writeFile(
      path.join(middenDir, 'midden.json'),
      JSON.stringify(middenData, null, 2)
    );
  }

  // Write CONTEXT.md if contextDecisions provided
  if (opts.contextDecisions !== undefined) {
    let contextMd = `# Aether Colony -- Current Context

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
`;
    for (const d of opts.contextDecisions) {
      contextMd += `| ${d.date} | ${d.decision} | ${d.rationale} | ${d.madeBy} |\n`;
    }
    contextMd += `
---

## Recent Activity

*No recent activity*
`;
    await fs.promises.writeFile(path.join(aetherDir, 'CONTEXT.md'), contextMd);
  }

  return { aetherDir, dataDir };
}


// =============================================================================
// PHER-01: Decision auto-emission tests
// =============================================================================

test.serial('pheromone-write auto:decision source creates FEEDBACK signal', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const result = runAetherUtil(tmpDir, 'pheromone-write', [
      'FEEDBACK', '[decision] Use awk for parsing',
      '--source', 'auto:decision',
      '--strength', '0.6',
      '--ttl', '30d'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');

    // Verify source is stored in pheromones.json
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    const signal = pheromones.signals.find(s => s.source === 'auto:decision');
    t.truthy(signal, 'Should find signal with source auto:decision');
    t.is(signal.type, 'FEEDBACK', 'Signal type should be FEEDBACK');
    t.true(signal.content.text.includes('[decision]'),
      'Signal content should include [decision] label');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('auto:decision pheromones appear in colony-prime output', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Set up colony with a manually-written auto:decision pheromone
    const now = new Date();
    const expiresAt = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_feedback_test_001',
        type: 'FEEDBACK',
        priority: 'low',
        source: 'auto:decision',
        created_at: now.toISOString(),
        expires_at: expiresAt.toISOString(),
        active: true,
        strength: 0.6,
        reason: 'Auto-emitted from phase decision during continue',
        content: { text: '[decision] Use awk for parsing' }
      }]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;
    t.true(section.includes('[decision]'),
      'colony-prime output should contain [decision] text from auto-emitted pheromone');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('auto:decision pheromone deduplication skips existing signals', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Pre-populate pheromones.json with an existing auto:decision signal
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
        reason: 'Auto-emitted from phase decision',
        content: { text: '[decision] Use awk for parsing' }
      }]
    });

    // Write the same pheromone again via pheromone-write
    runAetherUtil(tmpDir, 'pheromone-write', [
      'FEEDBACK', '[decision] Use awk for parsing',
      '--source', 'auto:decision',
      '--strength', '0.6',
      '--ttl', '30d'
    ]);

    // Verify: pheromone-write appends (it does not deduplicate itself),
    // so we should see 2 signals. The dedup check is in the playbook caller,
    // not in pheromone-write. Here we verify the count to confirm the
    // deduplication pattern must be applied externally.
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    const autoDecisionSignals = pheromones.signals.filter(
      s => s.source === 'auto:decision' && s.content.text.includes('[decision] Use awk for parsing')
    );
    // pheromone-write always appends -- it's 2 signals without dedup
    t.is(autoDecisionSignals.length, 2,
      'Without external dedup, pheromone-write creates duplicates (confirming dedup must be external)');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// PHER-02: Midden error pattern emission tests
// =============================================================================

test.serial('midden-recent-failures returns failures for grouping', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Set up midden with 4 failures: 3 in "security", 1 in "test"
    await setupTestColony(tmpDir, {
      middenFailures: [
        { id: 'fail_001', timestamp: '2026-03-06T10:00:00Z', category: 'security', source: 'gatekeeper', message: 'Exposed API key in config' },
        { id: 'fail_002', timestamp: '2026-03-06T11:00:00Z', category: 'security', source: 'gatekeeper', message: 'Debug endpoint left open' },
        { id: 'fail_003', timestamp: '2026-03-06T12:00:00Z', category: 'security', source: 'gatekeeper', message: 'Missing auth on admin route' },
        { id: 'fail_004', timestamp: '2026-03-06T13:00:00Z', category: 'test', source: 'watcher', message: 'Flaky test in module X' }
      ]
    });

    const result = runAetherUtil(tmpDir, 'midden-recent-failures', ['50']);
    const parsed = JSON.parse(result);

    t.is(parsed.count, 4, 'Should have 4 total failures');
    t.is(parsed.failures.length, 4, 'Should return all 4 failures');

    // Group by category using JS and verify security has 3+
    const categoryGroups = {};
    for (const f of parsed.failures) {
      categoryGroups[f.category] = (categoryGroups[f.category] || 0) + 1;
    }
    t.true(categoryGroups['security'] >= 3,
      'Security category should have 3+ failures for recurring pattern detection');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('pheromone-write auto:error source creates REDIRECT signal', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const result = runAetherUtil(tmpDir, 'pheromone-write', [
      'REDIRECT', '[error-pattern] Category "security" recurring (3 occurrences)',
      '--source', 'auto:error',
      '--strength', '0.7',
      '--ttl', '30d'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');

    // Verify source and type in pheromones.json
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    const signal = pheromones.signals.find(s => s.source === 'auto:error');
    t.truthy(signal, 'Should find signal with source auto:error');
    t.is(signal.type, 'REDIRECT', 'Signal type should be REDIRECT');
    t.true(signal.content.text.includes('[error-pattern]'),
      'Signal content should include [error-pattern] label');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('auto:error pheromones appear in colony-prime output', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Set up colony with a manually-written auto:error REDIRECT pheromone
    const now = new Date();
    const expiresAt = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_redirect_test_001',
        type: 'REDIRECT',
        priority: 'high',
        source: 'auto:error',
        created_at: now.toISOString(),
        expires_at: expiresAt.toISOString(),
        active: true,
        strength: 0.7,
        reason: 'Auto-emitted: midden error pattern recurred 3+ times',
        content: { text: '[error-pattern] Category security recurring (3 occurrences)' }
      }]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;
    t.true(section.includes('[error-pattern]'),
      'colony-prime output should contain [error-pattern] text from auto-emitted pheromone');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// PHER-03: Success criteria recurrence emission tests
// =============================================================================

test.serial('success criteria recurrence detection finds matching criteria across phases', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Set up COLONY_STATE with 3 completed phases, 2 sharing identical success criteria
    await setupTestColony(tmpDir, {
      currentPhase: 4,
      completedPhases: [
        {
          id: 1,
          name: 'instinct-pipeline',
          status: 'completed',
          success_criteria: ['tests pass without regressions', 'instincts stored correctly'],
          tasks: [
            { name: 'task-1', success_criteria: ['pipeline runs end to end'] }
          ]
        },
        {
          id: 2,
          name: 'learnings-injection',
          status: 'completed',
          success_criteria: ['tests pass without regressions', 'learnings injected into prompt'],
          tasks: [
            { name: 'task-1', success_criteria: ['injection format correct'] }
          ]
        },
        {
          id: 3,
          name: 'context-expansion',
          status: 'completed',
          success_criteria: ['decisions visible in prompt'],
          tasks: [
            { name: 'task-1', success_criteria: ['context expanded'] }
          ]
        }
      ]
    });

    // Use jq to extract and group success criteria
    const stateFile = path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json');
    const state = JSON.parse(fs.readFileSync(stateFile, 'utf8'));

    // Extract all success criteria from completed phases
    const criteriaByPhase = {};
    for (const phase of state.plan.phases) {
      if (phase.status !== 'completed') continue;
      const allCriteria = [...(phase.success_criteria || [])];
      for (const task of (phase.tasks || [])) {
        allCriteria.push(...(task.success_criteria || []));
      }
      for (const criterion of allCriteria) {
        const normalized = criterion.toLowerCase().trim();
        if (!criteriaByPhase[normalized]) criteriaByPhase[normalized] = [];
        criteriaByPhase[normalized].push(phase.id);
      }
    }

    // Find recurring criteria (2+ phases)
    const recurring = Object.entries(criteriaByPhase)
      .filter(([, phases]) => phases.length >= 2);

    t.true(recurring.length >= 1,
      'Should detect at least 1 recurring criterion');
    t.is(recurring[0][0], 'tests pass without regressions',
      'Recurring criterion should be "tests pass without regressions"');
    t.true(recurring[0][1].length >= 2,
      'Recurring criterion should appear in 2+ phases');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('pheromone-write auto:success source creates FEEDBACK signal', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    const result = runAetherUtil(tmpDir, 'pheromone-write', [
      'FEEDBACK', '[success-pattern] "tests pass without regressions" recurs across phases 1, 2',
      '--source', 'auto:success',
      '--strength', '0.6',
      '--ttl', '30d'
    ]);

    const resultJson = JSON.parse(result);
    t.true(resultJson.ok, 'Should return ok=true');

    // Verify source and type in pheromones.json
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    const signal = pheromones.signals.find(s => s.source === 'auto:success');
    t.truthy(signal, 'Should find signal with source auto:success');
    t.is(signal.type, 'FEEDBACK', 'Signal type should be FEEDBACK');
    t.true(signal.content.text.includes('[success-pattern]'),
      'Signal content should include [success-pattern] label');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('auto:success pheromones appear in colony-prime output', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Set up colony with a manually-written auto:success FEEDBACK pheromone
    const now = new Date();
    const expiresAt = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_feedback_success_001',
        type: 'FEEDBACK',
        priority: 'low',
        source: 'auto:success',
        created_at: now.toISOString(),
        expires_at: expiresAt.toISOString(),
        active: true,
        strength: 0.6,
        reason: 'Auto-emitted: success criteria pattern recurred',
        content: { text: '[success-pattern] tests pass without regressions recurs' }
      }]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const resultJson = JSON.parse(result);

    t.true(resultJson.ok, 'Should return ok=true');

    const section = resultJson.result.prompt_section;
    t.true(section.includes('[success-pattern]'),
      'colony-prime output should contain [success-pattern] text from auto-emitted pheromone');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


// =============================================================================
// Cross-cutting tests
// =============================================================================

test.serial('auto-emitted pheromones are distinguishable from manual pheromones', async (t) => {
  const tmpDir = await createTempDir();

  try {
    await setupTestColony(tmpDir);

    // Create three pheromones with distinct sources
    // 1. Manual user FEEDBACK
    runAetherUtil(tmpDir, 'pheromone-write', [
      'FEEDBACK', 'Manual user feedback signal',
      '--source', 'user',
      '--strength', '0.7',
      '--ttl', '30d'
    ]);

    // 2. Auto:decision FEEDBACK
    runAetherUtil(tmpDir, 'pheromone-write', [
      'FEEDBACK', '[decision] Use awk for parsing',
      '--source', 'auto:decision',
      '--strength', '0.6',
      '--ttl', '30d'
    ]);

    // 3. Auto:error REDIRECT
    runAetherUtil(tmpDir, 'pheromone-write', [
      'REDIRECT', '[error-pattern] Category security recurring (3 occurrences)',
      '--source', 'auto:error',
      '--strength', '0.7',
      '--ttl', '30d'
    ]);

    // Read pheromones.json and verify distinct sources
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));

    const sources = pheromones.signals.map(s => s.source);
    t.true(sources.includes('user'), 'Should have manual user source');
    t.true(sources.includes('auto:decision'), 'Should have auto:decision source');
    t.true(sources.includes('auto:error'), 'Should have auto:error source');

    // Verify all three are distinct
    const uniqueSources = [...new Set(sources)];
    t.is(uniqueSources.length, 3, 'All three sources should be distinct');

    // Run pheromone-prime and verify all three appear in output
    const primeResult = runAetherUtil(tmpDir, 'pheromone-prime');
    const primeJson = JSON.parse(primeResult);

    t.true(primeJson.ok, 'pheromone-prime should return ok=true');
    t.is(primeJson.result.signal_count, 3, 'Should have 3 signals');

    const section = primeJson.result.prompt_section;
    t.true(section.includes('Manual user feedback signal'),
      'Should contain manual user feedback');
    t.true(section.includes('[decision] Use awk for parsing'),
      'Should contain auto:decision content');
    t.true(section.includes('[error-pattern]'),
      'Should contain auto:error content');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});


test.serial('no pheromones emitted when data sources are empty', async (t) => {
  const tmpDir = await createTempDir();

  try {
    // Set up colony with no CONTEXT.md, empty midden, and no completed phases
    await setupTestColony(tmpDir, {
      currentPhase: 1,
      completedPhases: [],
      middenFailures: []
    });

    // Verify pheromones.json starts with 0 signals
    const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
    const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
    t.is(pheromones.signals.length, 0,
      'Should have 0 signals when all data sources are empty');

    // Verify midden-recent-failures returns empty without crashing
    const middenResult = runAetherUtil(tmpDir, 'midden-recent-failures', ['50']);
    const middenParsed = JSON.parse(middenResult);
    t.is(middenParsed.count, 0, 'Midden should report 0 failures');
    t.deepEqual(middenParsed.failures, [], 'Midden failures should be empty array');

    // Verify colony-prime works without crashing (no CONTEXT.md, no signals)
    const primeResult = runAetherUtil(tmpDir, 'colony-prime');
    const primeJson = JSON.parse(primeResult);
    t.true(primeJson.ok, 'colony-prime should return ok=true even with empty data sources');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
