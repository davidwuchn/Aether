/**
 * Colony Isolation Integration Tests (Phase 34 Plan 05)
 *
 * Proves that cross-colony isolation works end-to-end:
 *
 * SAFE-02: Per-colony data directories, auto-migration, lock tagging, backwards compat
 *
 * Tests cover:
 * 1. COLONY_DATA_DIR resolves to colonies/{sanitized-name}/ when colony exists
 * 2. COLONY_DATA_DIR falls back to DATA_DIR when no colony exists (pre-init)
 * 3. Auto-migration moves flat files to colony subdirectory on first access
 * 4. COLONY_STATE.json stays at DATA_DIR root after migration
 * 5. Colony name sanitization handles spaces, special characters, mixed case
 * 6. Lock files include colony name tag (acquire_lock_at)
 * 7. Partial migration recovery (files in both locations)
 * 8. No session_id splitting anywhere in codebase
 * 9. Backward compatibility: single-colony user sees no change
 *
 * Uses real aether-utils.sh subcommands with temp directory isolation.
 */

const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const AETHER_ROOT = path.join(__dirname, '../..');
const AETHER_UTILS = path.join(AETHER_ROOT, '.aether/aether-utils.sh');

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-isolation-'));
}

function cleanupTempDir(tmpDir) {
  try {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  } catch {
    // Ignore cleanup errors
  }
}

/**
 * Run an aether-utils subcommand in an isolated temp environment.
 * Returns parsed JSON output.
 */
function run(tmpDir, cmd, args = [], opts = {}) {
  const quotedArgs = args.map(a => `'${a.replace(/'/g, "'\\''")}'`).join(' ');
  const full = `bash "${AETHER_UTILS}" ${cmd} ${quotedArgs}`;
  const output = execSync(full, {
    cwd: tmpDir,
    encoding: 'utf8',
    timeout: 15000,
    env: {
      ...process.env,
      AETHER_ROOT: tmpDir,
      DATA_DIR: path.join(tmpDir, '.aether', 'data'),
      LOCK_DIR: path.join(tmpDir, '.aether', 'locks'),
      HOME: tmpDir,
      AETHER_STALE_LOCK_MODE: 'auto',
      ...opts.env
    },
    stdio: ['pipe', 'pipe', 'pipe']
  });
  return JSON.parse(output.trim());
}

/**
 * Run subcommand and return raw output (for non-JSON or error cases).
 */
function runRaw(tmpDir, cmd, args = [], opts = {}) {
  const quotedArgs = args.map(a => `'${a.replace(/'/g, "'\\''")}'`).join(' ');
  const full = `bash "${AETHER_UTILS}" ${cmd} ${quotedArgs}`;
  return execSync(full, {
    cwd: tmpDir,
    encoding: 'utf8',
    timeout: 15000,
    env: {
      ...process.env,
      AETHER_ROOT: tmpDir,
      DATA_DIR: path.join(tmpDir, '.aether', 'data'),
      LOCK_DIR: path.join(tmpDir, '.aether', 'locks'),
      HOME: tmpDir,
      AETHER_STALE_LOCK_MODE: 'auto',
      ...opts.env
    },
    stdio: ['pipe', 'pipe', 'pipe']
  });
}

/**
 * Set up a minimal colony with a given colony_name in a temp directory.
 */
function setupColony(tmpDir, colonyName = 'Test Colony') {
  const dataDir = path.join(tmpDir, '.aether', 'data');
  const middenDir = path.join(dataDir, 'midden');
  const locksDir = path.join(tmpDir, '.aether', 'locks');
  const eternalDir = path.join(tmpDir, '.aether', 'eternal');

  fs.mkdirSync(dataDir, { recursive: true });
  fs.mkdirSync(middenDir, { recursive: true });
  fs.mkdirSync(locksDir, { recursive: true });
  fs.mkdirSync(eternalDir, { recursive: true });

  // COLONY_STATE.json with colony_name field
  fs.writeFileSync(path.join(dataDir, 'COLONY_STATE.json'), JSON.stringify({
    colony_name: colonyName,
    session_id: 'isolation_test',
    goal: 'colony isolation test',
    state: 'BUILDING',
    current_phase: 1,
    colony_version: 1,
    plan: { phases: [] },
    memory: { instincts: [], phase_learnings: [], decisions: [] },
    errors: { flagged_patterns: [] },
    events: []
  }, null, 2));

  // Minimal pheromones.json
  fs.writeFileSync(path.join(dataDir, 'pheromones.json'), JSON.stringify({
    signals: [],
    version: '1.0.0'
  }, null, 2));

  // Minimal flags.json
  fs.writeFileSync(path.join(dataDir, 'flags.json'), JSON.stringify({
    flags: [],
    version: '1.0.0'
  }, null, 2));

  // Minimal midden.json
  fs.writeFileSync(path.join(middenDir, 'midden.json'), JSON.stringify({
    version: '1.0.0',
    entries: []
  }, null, 2));

  // Minimal eternal memory
  fs.writeFileSync(path.join(eternalDir, 'memory.json'), JSON.stringify({
    version: '1.0.0',
    entries: [],
    high_value_signals: [],
    stats: { total_entries: 0, total_promotions: 0 }
  }, null, 2));

  // QUEEN.md (required by various subcommands)
  const isoDate = new Date().toISOString();
  fs.writeFileSync(path.join(tmpDir, '.aether', 'QUEEN.md'), `# QUEEN.md --- Colony Wisdom

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

<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","colonies_contributed":[],"promotion_thresholds":{"philosophy":1,"pattern":1,"redirect":1,"stack":1,"decree":0},"stats":{"total_philosophies":0,"total_patterns":0,"total_redirects":0,"total_stack_entries":0,"total_decrees":0}} -->`);

  return { dataDir, middenDir, locksDir };
}

/**
 * Set up a pre-init environment (no COLONY_STATE.json).
 */
function setupPreInit(tmpDir) {
  const dataDir = path.join(tmpDir, '.aether', 'data');
  const locksDir = path.join(tmpDir, '.aether', 'locks');

  fs.mkdirSync(dataDir, { recursive: true });
  fs.mkdirSync(locksDir, { recursive: true });

  // Minimal pheromones.json at data dir root
  fs.writeFileSync(path.join(dataDir, 'pheromones.json'), JSON.stringify({
    signals: [],
    version: '1.0.0'
  }, null, 2));

  // Minimal flags.json at data dir root
  fs.writeFileSync(path.join(dataDir, 'flags.json'), JSON.stringify({
    flags: [],
    version: '1.0.0'
  }, null, 2));

  return { dataDir, locksDir };
}

/**
 * Sanitize colony name the same way aether-utils.sh does.
 */
function sanitizeColonyName(rawName) {
  return rawName
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

// ===========================================================================
// Test 1: COLONY_DATA_DIR resolves to colonies/{sanitized-name}/
// ===========================================================================

test.serial('COLONY_DATA_DIR resolves to colonies/{sanitized-name}/ when colony exists', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'Test Colony';
    const { dataDir } = setupColony(tmpDir, colonyName);
    const expectedSubdir = sanitizeColonyName(colonyName); // "test-colony"

    // colony-name subcommand returns the name from COLONY_STATE.json
    const nameResult = run(tmpDir, 'colony-name');
    t.true(nameResult.ok, 'colony-name should succeed');
    t.is(nameResult.result.name, colonyName, 'colony-name should return the colony name');

    // After any subcommand, the colonies/ subdirectory should be created
    const colonyDir = path.join(dataDir, 'colonies', expectedSubdir);
    t.true(fs.existsSync(colonyDir), `Colony directory should exist: ${expectedSubdir}`);

    // pheromone-read should look in the colony subdirectory
    // Write a pheromone signal into the colony dir to prove it reads from there
    fs.writeFileSync(path.join(colonyDir, 'pheromones.json'), JSON.stringify({
      signals: [{
        type: 'FOCUS',
        content: { text: 'test isolation signal' },
        strength: 0.9,
        created_at: new Date().toISOString(),
        expires_at: 'phase_end',
        active: true
      }],
      version: '1.0.0'
    }, null, 2));

    const readResult = run(tmpDir, 'pheromone-read');
    t.true(readResult.ok, 'pheromone-read should succeed');
    t.true(Array.isArray(readResult.result.signals), 'pheromone-read should return signals array');
    t.true(readResult.result.signals.length >= 1, 'Should read pheromones from colony dir');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 2: COLONY_DATA_DIR falls back to DATA_DIR when no colony exists
// ===========================================================================

test.serial('COLONY_DATA_DIR falls back to DATA_DIR when no colony exists (pre-init)', (t) => {
  const tmpDir = createTempDir();
  try {
    const { dataDir } = setupPreInit(tmpDir);

    // flag-list should work reading from DATA_DIR (no colony subdirectory)
    const flagResult = run(tmpDir, 'flag-list');
    t.true(flagResult.ok, 'flag-list should succeed in pre-init state');
    t.is(flagResult.result.count, 0, 'Should have 0 flags');

    // No colonies/ subdirectory should exist
    t.false(fs.existsSync(path.join(dataDir, 'colonies')), 'No colonies/ dir should exist');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 3: Auto-migration moves flat files to colony subdirectory
// ===========================================================================

test.serial('Auto-migration moves flat files to colony subdirectory', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'Migration Test';
    const { dataDir } = setupColony(tmpDir, colonyName);
    const expectedSubdir = sanitizeColonyName(colonyName); // "migration-test"
    const colonyDir = path.join(dataDir, 'colonies', expectedSubdir);

    // Place flat files at DATA_DIR root (simulating old-style colony)
    const flatPheromones = { signals: [{ type: 'FOCUS', content: { text: 'old signal' }, strength: 0.8, created_at: new Date().toISOString(), active: true }], version: '1.0.0' };
    fs.writeFileSync(path.join(dataDir, 'pheromones.json'), JSON.stringify(flatPheromones, null, 2));
    fs.writeFileSync(path.join(dataDir, 'flags.json'), JSON.stringify({ flags: [{ id: 'f1', title: 'test flag' }], version: '1.0.0' }, null, 2));
    fs.writeFileSync(path.join(dataDir, 'session.json'), JSON.stringify({ session_id: 'old_session', started_at: new Date().toISOString() }, null, 2));

    // Verify flat files exist at root
    t.true(fs.existsSync(path.join(dataDir, 'pheromones.json')), 'pheromones.json should start at root');
    t.true(fs.existsSync(path.join(dataDir, 'flags.json')), 'flags.json should start at root');
    t.true(fs.existsSync(path.join(dataDir, 'session.json')), 'session.json should start at root');

    // Run a subcommand that triggers COLONY_DATA_DIR resolution and migration
    const flagResult = run(tmpDir, 'flag-list');
    t.true(flagResult.ok, 'flag-list should trigger migration and succeed');

    // Verify files were moved to colony subdirectory
    t.true(fs.existsSync(path.join(colonyDir, 'pheromones.json')), 'pheromones.json should be in colony dir after migration');
    t.true(fs.existsSync(path.join(colonyDir, 'flags.json')), 'flags.json should be in colony dir after migration');
    t.true(fs.existsSync(path.join(colonyDir, 'session.json')), 'session.json should be in colony dir after migration');

    // Verify originals gone from root
    t.false(fs.existsSync(path.join(dataDir, 'pheromones.json')), 'pheromones.json should be gone from root');
    t.false(fs.existsSync(path.join(dataDir, 'flags.json')), 'flags.json should be gone from root');
    t.false(fs.existsSync(path.join(dataDir, 'session.json')), 'session.json should be gone from root');

    // Verify the migrated data is still readable
    const migratedResult = run(tmpDir, 'flag-list');
    t.true(migratedResult.ok, 'flag-list should work after migration');
    t.is(migratedResult.result.count, 1, 'Migrated flag should be readable');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 4: COLONY_STATE.json stays at root after migration
// ===========================================================================

test.serial('COLONY_STATE.json stays at DATA_DIR root after migration', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'State Anchor';
    const { dataDir } = setupColony(tmpDir, colonyName);
    const expectedSubdir = sanitizeColonyName(colonyName); // "state-anchor"
    const colonyDir = path.join(dataDir, 'colonies', expectedSubdir);

    // Place flat files to trigger migration
    fs.writeFileSync(path.join(dataDir, 'pheromones.json'), JSON.stringify({ signals: [], version: '1.0.0' }));

    // Run migration trigger
    run(tmpDir, 'flag-list');

    // COLONY_STATE.json must remain at root
    t.true(fs.existsSync(path.join(dataDir, 'COLONY_STATE.json')), 'COLONY_STATE.json must stay at DATA_DIR root');

    // COLONY_STATE.json must NOT be in colony subdirectory
    t.false(fs.existsSync(path.join(colonyDir, 'COLONY_STATE.json')), 'COLONY_STATE.json must NOT be in colony dir');

    // Verify the state file is still valid
    const state = JSON.parse(fs.readFileSync(path.join(dataDir, 'COLONY_STATE.json'), 'utf8'));
    t.is(state.colony_name, colonyName, 'COLONY_STATE.json should still have correct colony_name');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 5: Colony name sanitization handles edge cases
// ===========================================================================

test.serial('Colony name sanitization handles spaces and special characters', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'My Project (v2)';
    const { dataDir } = setupColony(tmpDir, colonyName);
    const expectedSubdir = sanitizeColonyName(colonyName); // "my-project-v2"

    // Run a subcommand to trigger COLONY_DATA_DIR resolution
    run(tmpDir, 'colony-name');

    const colonyDir = path.join(dataDir, 'colonies', expectedSubdir);
    t.true(fs.existsSync(colonyDir), `Colony dir should use sanitized name: ${expectedSubdir}`);
    // Verify no spaces or special chars in the directory name
    const dirName = path.basename(colonyDir);
    t.false(dirName.includes(' '), 'Dir name should have no spaces');
    t.false(dirName.includes('('), 'Dir name should have no parentheses');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

test.serial('Colony name sanitization handles mixed case', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'UPPERCASE Colony';
    const { dataDir } = setupColony(tmpDir, colonyName);
    const expectedSubdir = sanitizeColonyName(colonyName); // "uppercase-colony"

    run(tmpDir, 'colony-name');

    const colonyDir = path.join(dataDir, 'colonies', expectedSubdir);
    t.true(fs.existsSync(colonyDir), `Colony dir should be lowercase: ${expectedSubdir}`);
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 6: Lock files include colony name tag
// ===========================================================================

test.serial('Lock files include colony name tag via acquire_lock_at', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'Lock Test Colony';
    const { dataDir, locksDir } = setupColony(tmpDir, colonyName);
    const expectedSubdir = sanitizeColonyName(colonyName); // "lock-test-colony"

    // Source file-lock.sh to get acquire_lock_at
    const fileLockPath = path.join(AETHER_ROOT, '.aether/utils/file-lock.sh');

    // Create a dummy file to lock
    const dummyFile = path.join(dataDir, 'test-target.json');
    fs.writeFileSync(dummyFile, '{}');

    // Use acquire_lock_at with a colony tag via a bash one-liner
    // Verify lock file name and existence within the same process (EXIT trap cleans up on exit)
    const lockScript = `
      source "${fileLockPath}"
      acquire_lock_at "${dummyFile}" "${locksDir}" "${expectedSubdir}"
      echo "LOCK_AT_FILE=$LOCK_AT_FILE"
      echo "LOCK_EXISTS=$(test -f "$LOCK_AT_FILE" && echo yes || echo no)"
    `;
    const output = execSync(lockScript, {
      cwd: tmpDir,
      encoding: 'utf8',
      timeout: 10000,
      env: {
        ...process.env,
        HOME: tmpDir,
        AETHER_STALE_LOCK_MODE: 'auto'
      },
      stdio: ['pipe', 'pipe', 'pipe']
    });

    // Parse the lock file path
    const match = output.match(/LOCK_AT_FILE=(.+)/);
    t.truthy(match, 'Should output LOCK_AT_FILE');

    const lockFilePath = match[1].trim();
    const lockFileName = path.basename(lockFilePath);

    // Lock file should include the colony tag
    t.true(lockFileName.includes(expectedSubdir), `Lock file name should contain colony tag: ${lockFileName}`);
    t.true(lockFileName.endsWith('.lock'), 'Lock file should end with .lock');

    // Verify lock existed within the bash process (before EXIT trap)
    const existsMatch = output.match(/LOCK_EXISTS=(.+)/);
    t.is(existsMatch[1].trim(), 'yes', 'Lock file should exist within the bash process');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 7: Partial migration recovery (files in both locations)
// ===========================================================================

test.serial('Partial migration: already-migrated colony dir is not re-migrated', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'Partial Migrate';
    const { dataDir } = setupColony(tmpDir, colonyName);
    const expectedSubdir = sanitizeColonyName(colonyName); // "partial-migrate"
    const colonyDir = path.join(dataDir, 'colonies', expectedSubdir);

    // Simulate a colony that was already migrated: colony dir has key files
    fs.mkdirSync(colonyDir, { recursive: true });
    const existingPheromones = { signals: [{ type: 'REDIRECT', content: { text: 'already migrated' }, strength: 0.9, created_at: new Date().toISOString(), active: true }], version: '1.0.0' };
    fs.writeFileSync(path.join(colonyDir, 'pheromones.json'), JSON.stringify(existingPheromones, null, 2));
    fs.writeFileSync(path.join(colonyDir, 'flags.json'), JSON.stringify({ flags: [{ id: 'f1', title: 'existing flag' }], version: '1.0.0' }, null, 2));

    // A stray file appears at root (e.g., from a backup restore or manual copy)
    // Migration should NOT move it because presence detection sees colony dir files
    fs.writeFileSync(path.join(dataDir, 'flags.json'), JSON.stringify({ flags: [{ id: 'f2', title: 'stray root flag' }], version: '1.0.0' }, null, 2));

    // Run a subcommand -- migration should be skipped
    run(tmpDir, 'flag-list');

    // Existing colony dir files should be preserved (not overwritten)
    const pheromonesInColony = JSON.parse(fs.readFileSync(path.join(colonyDir, 'pheromones.json'), 'utf8'));
    t.is(pheromonesInColony.signals[0].content.text, 'already migrated', 'Existing colony dir file should be preserved');

    // Colony dir flags should be the original, not overwritten by root file
    const flagsInColony = JSON.parse(fs.readFileSync(path.join(colonyDir, 'flags.json'), 'utf8'));
    t.is(flagsInColony.flags[0].title, 'existing flag', 'Colony dir flags should not be overwritten');

    // Root stray file should still be there (not moved)
    t.true(fs.existsSync(path.join(dataDir, 'flags.json')), 'Stray root file should not be moved when colony is already migrated');

    // Subcommands should read from colony dir (not root)
    const flagResult = run(tmpDir, 'flag-list');
    t.true(flagResult.ok, 'flag-list should work');
    t.is(flagResult.result.count, 1, 'Should have exactly 1 flag from colony dir');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 8: No session_id splitting anywhere in codebase
// ===========================================================================

test('No session_id splitting anywhere in codebase', (t) => {
  // Search for session_id.*split pattern in shell scripts and playbooks
  const searchDirs = [
    '.aether/',
    '.opencode/',
    '.claude/'
  ];
  for (const dir of searchDirs) {
    if (!fs.existsSync(dir)) continue;
    try {
      const result = execSync(
        `grep -rn 'session_id.*split' ${dir} --include='*.sh' --include='*.md' 2>/dev/null || true`,
        { cwd: AETHER_ROOT, encoding: 'utf8', timeout: 10000 }
      );
      // If any matches, check they are not in actual code (could be comments/docs)
      const lines = result.trim().split('\n').filter(l => l.length > 0);
      for (const line of lines) {
        // Skip documentation references (comments, plan files, summaries)
        const isComment = line.includes('#') && (line.indexOf('#') < line.indexOf('split'));
        const isPlanFile = line.includes('PLAN.md') || line.includes('SUMMARY.md');
        const isResearch = line.includes('RESEARCH.md') || line.includes('CONTEXT.md');
        const isTest = line.includes('.test.js');

        if (!isComment && !isPlanFile && !isResearch && !isTest) {
          // This is an actual code reference -- fail the test
          t.fail(`Found session_id.split in code: ${line}`);
        }
      }
    } catch {
      // grep returned no matches -- that is good
    }
  }
  t.pass('No session_id splitting found in codebase code');
});

// ===========================================================================
// Test 9: Backward compatibility -- single-colony user sees no change
// ===========================================================================

test.serial('Backward compat: single-colony workflow produces valid JSON', (t) => {
  const tmpDir = createTempDir();
  try {
    const colonyName = 'Aether Colony';
    const { dataDir } = setupColony(tmpDir, colonyName);

    // Place flat files (old-style) -- migration should handle them transparently
    fs.writeFileSync(path.join(dataDir, 'pheromones.json'), JSON.stringify({
      signals: [],
      version: '1.0.0'
    }, null, 2));
    fs.writeFileSync(path.join(dataDir, 'flags.json'), JSON.stringify({
      flags: [],
      version: '1.0.0'
    }, null, 2));
    fs.writeFileSync(path.join(dataDir, 'session.json'), JSON.stringify({
      session_id: 'compat_session',
      started_at: new Date().toISOString()
    }, null, 2));

    // Standard workflow commands should all work and return valid JSON
    const pheromoneResult = run(tmpDir, 'pheromone-read');
    t.true(pheromoneResult.ok, 'pheromone-read should return valid JSON');
    t.true(Array.isArray(pheromoneResult.result.signals), 'pheromone-read should return signals array');

    const flagResult = run(tmpDir, 'flag-list');
    t.true(flagResult.ok, 'flag-list should return valid JSON');
    t.true(Array.isArray(flagResult.result.flags), 'flag-list should return array');

    const nameResult = run(tmpDir, 'colony-name');
    t.true(nameResult.ok, 'colony-name should return valid JSON');
    t.is(nameResult.result.name, colonyName, 'colony-name should return correct name');

    // Write a pheromone and read it back
    run(tmpDir, 'pheromone-write', [
      'FOCUS', 'compatibility test',
      '--source', 'user',
      '--strength', '0.7'
    ]);

    const readBack = run(tmpDir, 'pheromone-read');
    t.true(readBack.ok, 'pheromone-read after write should work');
    t.true(readBack.result.signals.length > 0, 'Should have at least one signal after write');

    // Add a flag and read it back
    run(tmpDir, 'flag-add', ['note', 'compat flag', 'test note', 'builder']);
    const flagsAfter = run(tmpDir, 'flag-list');
    t.true(flagsAfter.ok, 'flag-list after add should work');
    t.is(flagsAfter.result.count, 1, 'Should have exactly 1 flag');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 10: Two colonies in same temp root get separate data directories
// ===========================================================================

test.serial('Two colonies get separate COLONY_DATA_DIR paths', (t) => {
  const tmpDir = createTempDir();
  try {
    // Colony A
    const colonyAName = 'Colony Alpha';
    const { dataDir: dataDirA } = setupColony(tmpDir, colonyAName);

    // Colony B (separate data dir within same temp)
    const dataDirB = path.join(tmpDir, '.aether', 'data-b');
    const middenDirB = path.join(dataDirB, 'midden');
    fs.mkdirSync(dataDirB, { recursive: true });
    fs.mkdirSync(middenDirB, { recursive: true });
    fs.writeFileSync(path.join(dataDirB, 'COLONY_STATE.json'), JSON.stringify({
      colony_name: 'Colony Beta',
      session_id: 'colony_b_session',
      goal: 'beta colony',
      state: 'BUILDING',
      current_phase: 1,
      colony_version: 1,
      plan: { phases: [] },
      memory: { instincts: [], phase_learnings: [], decisions: [] },
      errors: { flagged_patterns: [] },
      events: []
    }, null, 2));
    fs.writeFileSync(path.join(dataDirB, 'pheromones.json'), JSON.stringify({ signals: [], version: '1.0.0' }));

    // Run colony-name for colony A
    const nameA = run(tmpDir, 'colony-name', [], {
      env: { DATA_DIR: dataDirA }
    });
    t.is(nameA.result.name, colonyAName, 'Colony A should resolve to its name');

    // Run colony-name for colony B
    const nameB = run(tmpDir, 'colony-name', [], {
      env: { DATA_DIR: dataDirB }
    });
    t.is(nameB.result.name, 'Colony Beta', 'Colony B should resolve to its name');

    // Verify different colony directories are created
    const colonyADir = path.join(dataDirA, 'colonies', sanitizeColonyName(colonyAName));
    const colonyBDir = path.join(dataDirB, 'colonies', sanitizeColonyName('Colony Beta'));

    t.true(fs.existsSync(colonyADir), 'Colony A dir should exist');
    t.true(fs.existsSync(colonyBDir), 'Colony B dir should exist');
    t.not(colonyADir, colonyBDir, 'Colony A and B dirs should be different');
  } finally {
    cleanupTempDir(tmpDir);
  }
});

// ===========================================================================
// Test 11: Empty colony name after sanitization fails loudly
// ===========================================================================

test.serial('Empty colony name after sanitization produces error', (t) => {
  const tmpDir = createTempDir();
  try {
    const dataDir = path.join(tmpDir, '.aether', 'data');
    const locksDir = path.join(tmpDir, '.aether', 'locks');
    fs.mkdirSync(dataDir, { recursive: true });
    fs.mkdirSync(locksDir, { recursive: true });

    // COLONY_STATE.json with colony_name that sanitizes to empty
    fs.writeFileSync(path.join(dataDir, 'COLONY_STATE.json'), JSON.stringify({
      colony_name: '---',
      session_id: 'test',
      goal: 'test',
      state: 'BUILDING',
      current_phase: 1,
      colony_version: 1,
      plan: { phases: [] },
      memory: { instincts: [], phase_learnings: [], decisions: [] },
      errors: { flagged_patterns: [] },
      events: []
    }, null, 2));

    // Any subcommand should fail with an error about empty sanitization
    try {
      run(tmpDir, 'colony-name');
      t.fail('Should have thrown an error for empty sanitized name');
    } catch (e) {
      t.truthy(e.stderr || e.message, 'Should have an error message');
      const msg = (e.stderr || e.message || '').toLowerCase();
      t.true(msg.includes('sanitizes') || msg.includes('error') || msg.includes('empty'),
        'Error should mention sanitization problem');
    }
  } finally {
    cleanupTempDir(tmpDir);
  }
});
