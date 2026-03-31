/**
 * Interactive Installer Tests
 *
 * Tests for:
 *   - bin/lib/banner.js (BANNER export)
 *   - bin/lib/interactive-setup.js (environment detection, menu logic)
 *   - bin/lib/init.js setupOnly option
 *   - bin/npx-entry.js delegation logic
 */

const test = require('ava');
const path = require('path');
const sinon = require('sinon');
const proxyquire = require('proxyquire').noCallThru();
const os = require('os');

// --- banner.js ---

test('banner module exports BANNER string', (t) => {
  const { BANNER } = require('../../bin/lib/banner');
  t.is(typeof BANNER, 'string');
  t.true(BANNER.includes('█'));
});

// --- init.js setupOnly ---

const { createMockFs, resetMockFs, setupMockFiles } = require('./helpers/mock-fs');

let mockFs;
let init;

test.before(() => {
  mockFs = createMockFs();
  init = proxyquire('../../bin/lib/init', {
    fs: mockFs
  });
});

test.beforeEach(() => {
  resetMockFs(mockFs);
});

test.serial('initializeRepo with setupOnly skips COLONY_STATE.json creation', async (t) => {
  const repoPath = '/test/repo-setup-only';
  const homeDir = os.homedir();

  setupMockFiles(mockFs, {
    [homeDir + '/.aether']: null,
    [homeDir + '/.aether/system']: null,
    [homeDir + '/.aether/version.json']: JSON.stringify({ version: '5.3.0' }),
    [homeDir + '/.aether/registry.json']: JSON.stringify({ schema_version: 1, repos: [] })
  });

  const result = await init.initializeRepo(repoPath, { setupOnly: true, quiet: true });

  t.true(result.success);
  t.is(result.stateFile, null, 'stateFile should be null when setupOnly is true');

  // Verify COLONY_STATE.json was NOT written
  const stateFilePath = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');
  const writeCalls = mockFs.writeFileSync.args.map(args => args[0]);
  t.false(
    writeCalls.some(p => p === stateFilePath),
    'COLONY_STATE.json should not be written in setupOnly mode'
  );
});

test.serial('initializeRepo without setupOnly writes COLONY_STATE.json', async (t) => {
  const repoPath = '/test/repo-full-init';
  const homeDir = os.homedir();

  setupMockFiles(mockFs, {
    [homeDir + '/.aether']: null,
    [homeDir + '/.aether/system']: null,
    [homeDir + '/.aether/version.json']: JSON.stringify({ version: '5.3.0' }),
    [homeDir + '/.aether/registry.json']: JSON.stringify({ schema_version: 1, repos: [] })
  });

  const result = await init.initializeRepo(repoPath, { goal: 'Test Goal', quiet: true });

  t.true(result.success);

  const stateFilePath = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');
  const writeCalls = mockFs.writeFileSync.args.map(args => args[0]);
  t.true(
    writeCalls.some(p => p === stateFilePath),
    'COLONY_STATE.json should be written in normal init mode'
  );
});

// --- interactive-setup.js environment detection ---

function makeSetupModule(overrides = {}) {
  const defaultMockFs = {
    existsSync: sinon.stub().returns(false),
  };
  const mockCli = {
    performGlobalInstall: sinon.stub().resolves(),
  };
  const mockInitModule = {
    initializeRepo: sinon.stub().resolves({ success: true, filesCopied: 42 }),
  };

  return proxyquire('../../bin/lib/interactive-setup', {
    fs: { ...defaultMockFs, ...(overrides.fs || {}) },
    path: path,
    os: os,
    './banner': { BANNER: 'TEST_BANNER' },
    '../cli': { ...mockCli, ...(overrides.cli || {}) },
    './init': { ...mockInitModule, ...(overrides.init || {}) },
    readline: overrides.readline || require('readline'),
    ...overrides.extra,
  });
}

test('detectEnvironment returns hubInstalled=false when version.json missing', (t) => {
  const mockFsOverride = {
    existsSync: sinon.stub().returns(false),
  };
  const { detectEnvironment } = makeSetupModule({ fs: mockFsOverride });
  const env = detectEnvironment();
  t.false(env.hubInstalled);
});

test('detectEnvironment returns hubInstalled=true when version.json exists', (t) => {
  const homeDir = os.homedir();
  const hubVersionPath = path.join(homeDir, '.aether', 'version.json');
  const mockFsOverride = {
    existsSync: sinon.stub().callsFake(p => p === hubVersionPath),
  };
  const { detectEnvironment } = makeSetupModule({ fs: mockFsOverride });
  const env = detectEnvironment();
  t.true(env.hubInstalled);
});

test('detectEnvironment returns hasAether=true when .aether/aether-utils.sh exists', (t) => {
  const aetherUtilsPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const mockFsOverride = {
    existsSync: sinon.stub().callsFake(p => p === aetherUtilsPath),
  };
  const { detectEnvironment } = makeSetupModule({ fs: mockFsOverride });
  const env = detectEnvironment();
  t.true(env.hasAether);
});

test('detectEnvironment returns isProjectDir=true when package.json exists', (t) => {
  const pkgPath = path.join(process.cwd(), 'package.json');
  const mockFsOverride = {
    existsSync: sinon.stub().callsFake(p => p === pkgPath),
  };
  const { detectEnvironment } = makeSetupModule({ fs: mockFsOverride });
  const env = detectEnvironment();
  t.true(env.isProjectDir);
});

test('detectEnvironment returns isProjectDir=true when .git exists', (t) => {
  const gitPath = path.join(process.cwd(), '.git');
  const mockFsOverride = {
    existsSync: sinon.stub().callsFake(p => p === gitPath),
  };
  const { detectEnvironment } = makeSetupModule({ fs: mockFsOverride });
  const env = detectEnvironment();
  t.true(env.isProjectDir);
});

test('getDefaultOption returns 1 (full setup) when no hub and is project dir', (t) => {
  const { getDefaultOption } = makeSetupModule();
  const def = getDefaultOption({ hubInstalled: false, isProjectDir: true, hasAether: false });
  t.is(def, 1);
});

test('getDefaultOption returns 2 (global only) when no hub and not project dir', (t) => {
  const { getDefaultOption } = makeSetupModule();
  const def = getDefaultOption({ hubInstalled: false, isProjectDir: false, hasAether: false });
  t.is(def, 2);
});

test('getDefaultOption returns 3 (repo only) when hub installed but no .aether/', (t) => {
  const { getDefaultOption } = makeSetupModule();
  const def = getDefaultOption({ hubInstalled: true, isProjectDir: true, hasAether: false });
  t.is(def, 3);
});

// --- npx-entry.js delegation ---

test('npx-entry delegates to cli when subcommand arg provided', (t) => {
  // We can't easily test process.argv, but we can verify the module structure
  // by checking the file exists and reading its logic
  const fs = require('fs');
  const entryPath = path.join(__dirname, '../../bin/npx-entry.js');
  t.true(fs.existsSync(entryPath), 'npx-entry.js must exist');
  const content = fs.readFileSync(entryPath, 'utf8');
  t.true(content.includes('require(\'./cli.js\')'), 'must delegate to cli.js for subcommands');
  t.true(content.includes('interactive-setup'), 'must require interactive-setup for no-args case');
});
