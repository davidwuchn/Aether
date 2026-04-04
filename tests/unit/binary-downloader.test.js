/**
 * Binary Downloader Unit Tests
 *
 * Tests for platform detection, checksum parsing, redirect following,
 * and the full downloadBinary flow using ava + sinon + proxyquire.
 *
 * Uses test.serial() for tests that stub built-in modules.
 * All tests mock network and filesystem -- no real I/O.
 */

const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

let sandbox;

test.before(() => {
  sandbox = sinon.createSandbox();
});

test.afterEach(() => {
  sandbox.restore();
});

// ============================================================
// A. Platform Detection Tests
// ============================================================

function withPlatform(platform, arch, fn) {
  const origPlatform = Object.getOwnPropertyDescriptor(process, 'platform');
  const origArch = Object.getOwnPropertyDescriptor(process, 'arch');
  Object.defineProperty(process, 'platform', { value: platform });
  Object.defineProperty(process, 'arch', { value: arch });
  try {
    return fn();
  } finally {
    Object.defineProperty(process, 'platform', origPlatform);
    Object.defineProperty(process, 'arch', origArch);
  }
}

test.serial('getPlatformArch returns darwin/arm64 for macOS Apple Silicon', (t) => {
  const result = withPlatform('darwin', 'arm64', () => {
    // Re-require to pick up changed platform
    delete require.cache[require.resolve('../../bin/lib/binary-downloader')];
    const { getPlatformArch } = require('../../bin/lib/binary-downloader');
    return getPlatformArch();
  });
  t.deepEqual(result, { os: 'darwin', arch: 'arm64' });
});

test.serial('getPlatformArch returns linux/amd64 for Linux x64', (t) => {
  const result = withPlatform('linux', 'x64', () => {
    delete require.cache[require.resolve('../../bin/lib/binary-downloader')];
    const { getPlatformArch } = require('../../bin/lib/binary-downloader');
    return getPlatformArch();
  });
  t.deepEqual(result, { os: 'linux', arch: 'amd64' });
});

test.serial('getPlatformArch returns windows/amd64 for Windows x64', (t) => {
  const result = withPlatform('win32', 'x64', () => {
    delete require.cache[require.resolve('../../bin/lib/binary-downloader')];
    const { getPlatformArch } = require('../../bin/lib/binary-downloader');
    return getPlatformArch();
  });
  t.deepEqual(result, { os: 'windows', arch: 'amd64' });
});

test.serial('getPlatformArch returns null for unsupported platform (freebsd)', (t) => {
  const result = withPlatform('freebsd', 'x64', () => {
    delete require.cache[require.resolve('../../bin/lib/binary-downloader')];
    const { getPlatformArch } = require('../../bin/lib/binary-downloader');
    return getPlatformArch();
  });
  t.is(result, null);
});

test.serial('getPlatformArch returns null for unsupported arch (ia32)', (t) => {
  const result = withPlatform('darwin', 'ia32', () => {
    delete require.cache[require.resolve('../../bin/lib/binary-downloader')];
    const { getPlatformArch } = require('../../bin/lib/binary-downloader');
    return getPlatformArch();
  });
  t.is(result, null);
});

// ============================================================
// B. Checksum Parsing Tests
// ============================================================

test('_findChecksum parses standard sha256sum format', (t) => {
  const { _findChecksum } = require('../../bin/lib/binary-downloader');
  const content = [
    'abc123def456  aether_5.3.3_darwin_arm64.tar.gz',
    '789xyz123456  aether_5.3.3_linux_amd64.tar.gz',
  ].join('\n');
  const result = _findChecksum(content, 'aether_5.3.3_darwin_arm64.tar.gz');
  t.is(result, 'abc123def456');
});

test('_findChecksum returns null for filename not in checksums', (t) => {
  const { _findChecksum } = require('../../bin/lib/binary-downloader');
  const content = 'abc123def456  aether_5.3.3_darwin_arm64.tar.gz\n';
  const result = _findChecksum(content, 'nonexistent.tar.gz');
  t.is(result, null);
});

test('_findChecksum handles trailing newline', (t) => {
  const { _findChecksum } = require('../../bin/lib/binary-downloader');
  const content = 'abc123def456  aether_5.3.3_darwin_arm64.tar.gz\n\n';
  const result = _findChecksum(content, 'aether_5.3.3_darwin_arm64.tar.gz');
  t.is(result, 'abc123def456');
});

// ============================================================
// C. Redirect Following Tests
// ============================================================

function createMockResponse(statusCode, headers = {}, body = null) {
  const listeners = {};
  const res = {
    statusCode,
    headers,
    on: (event, fn) => {
      listeners[event] = listeners[event] || [];
      listeners[event].push(fn);
    },
    resume: sinon.stub(),
    emit: (event, ...args) => {
      (listeners[event] || []).forEach((fn) => fn(...args));
    },
  };
  return res;
}

function createMockGet(url, res) {
  return (reqUrl, callback) => {
    const request = {
      on: (event, fn) => {
        if (event === 'error' && res._error) fn(res._error);
      },
    };
    process.nextTick(() => callback(res));
    return request;
  };
}

test.serial('_downloadWithRedirects follows 302 redirect to final URL', async (t) => {
  const { _downloadWithRedirects } = require('../../bin/lib/binary-downloader');

  const finalRes = createMockResponse(200, {});
  const redirectRes = createMockResponse(302, { location: 'https://objects.githubusercontent.com/final' });

  const httpsStub = {
    get: sinon.stub(),
  };

  // First call returns redirect, second call returns 200
  httpsStub.get.onFirstCall().callsFake(createMockGet('url1', redirectRes));
  httpsStub.get.onSecondCall().callsFake(createMockGet('url2', finalRes));

  const mod = proxyquire('../../bin/lib/binary-downloader', {
    https: httpsStub,
    http: { get: sinon.stub() },
  });

  const result = await mod._downloadWithRedirects('https://github.com/release');
  t.is(result, finalRes);
  t.is(redirectRes.resume.callCount, 1);
});

test.serial('_downloadWithRedirects rejects after 5 redirects', async (t) => {
  const { _downloadWithRedirects } = require('../../bin/lib/binary-downloader');

  const redirectRes = createMockResponse(302, { location: 'https://next' });
  const httpsStub = {
    get: sinon.stub().callsFake((url, cb) => {
      process.nextTick(() => cb(redirectRes));
      return { on: () => {} };
    }),
  };

  const mod = proxyquire('../../bin/lib/binary-downloader', {
    https: httpsStub,
    http: { get: sinon.stub() },
  });

  await t.throwsAsync(
    () => mod._downloadWithRedirects('https://github.com/release'),
    { message: 'Too many redirects' }
  );
});

test.serial('_downloadWithRedirects rejects on non-200, non-redirect status', async (t) => {
  const { _downloadWithRedirects } = require('../../bin/lib/binary-downloader');

  const errorRes = createMockResponse(404, {});

  const httpsStub = {
    get: sinon.stub().callsFake((url, cb) => {
      process.nextTick(() => cb(errorRes));
      return { on: () => {} };
    }),
  };

  const mod = proxyquire('../../bin/lib/binary-downloader', {
    https: httpsStub,
    http: { get: sinon.stub() },
  });

  await t.throwsAsync(
    () => mod._downloadWithRedirects('https://github.com/release'),
    { message: 'HTTP 404' }
  );
});

// ============================================================
// D. Full downloadBinary Integration Tests
// ============================================================

test.serial('downloadBinary returns failure for unsupported platform', async (t) => {
  const result = await withPlatform('freebsd', 'x64', async () => {
    delete require.cache[require.resolve('../../bin/lib/binary-downloader')];
    const { downloadBinary } = require('../../bin/lib/binary-downloader');
    return downloadBinary('5.3.3');
  });
  t.false(result.success);
  t.true(result.reason.includes('Unsupported platform'));
});

test.serial('downloadBinary returns failure when checksums download fails', async (t) => {
  const httpsStub = {
    get: sinon.stub().callsFake((url, cb) => {
      const errRes = createMockResponse(404, {});
      process.nextTick(() => cb(errRes));
      return { on: () => {} };
    }),
  };

  const mod = proxyquire('../../bin/lib/binary-downloader', {
    https: httpsStub,
    http: { get: sinon.stub() },
  });

  const result = await mod.downloadBinary('5.3.3');
  t.false(result.success);
  t.true(result.reason.includes('checksums'));
});

test.serial('downloadBinary returns failure on checksum mismatch', async (t) => {
  const checksumsContent = 'expectedhash000  aether_5.3.3_darwin_arm64.tar.gz\n';

  // Mock responses for different URLs
  const checksumsRes = createMockResponse(200, {}, checksumsContent);
  checksumsRes.on = (event, fn) => {
    if (event === 'data') fn(checksumsContent);
    if (event === 'end') fn();
  };

  const archiveRes = createMockResponse(200, {});
  const redirectRes = createMockResponse(302, { location: 'https://objects.githubusercontent.com/checksums' });
  const redirectRes2 = createMockResponse(302, { location: 'https://objects.githubusercontent.com/archive' });

  // Track call count to serve different responses
  let getCallCount = 0;
  const httpsStub = {
    get: sinon.stub().callsFake((url, cb) => {
      getCallCount++;
      if (url.includes('checksums')) {
        // First request: redirect to final checksums URL
        if (getCallCount === 1) {
          const res = createMockResponse(302, { location: 'https://objects.githubusercontent.com/checksums-final' });
          process.nextTick(() => cb(res));
          return { on: () => {} };
        }
        // Final checksums response
        const res = createMockResponse(200, {});
        res.on = (event, fn) => {
          if (event === 'data') fn(checksumsContent);
          if (event === 'end') fn();
        };
        process.nextTick(() => cb(res));
        return { on: () => {} };
      }
      // Archive request: redirect then 200
      if (getCallCount === 3) {
        const res = createMockResponse(302, { location: 'https://objects.githubusercontent.com/archive-final' });
        process.nextTick(() => cb(res));
        return { on: () => {} };
      }
      // Final archive response
      const res = createMockResponse(200, {});
      res.on = (event, fn) => {
        if (event === 'data') fn(Buffer.from('fake archive data'));
        if (event === 'end') fn();
      };
      process.nextTick(() => cb(res));
      return { on: () => {} };
    }),
  };

  const unlinkStub = sinon.stub().resolves();
  const mkdirStub = sinon.stub().resolves();

  const mod = proxyquire('../../bin/lib/binary-downloader', {
    https: httpsStub,
    http: { get: sinon.stub() },
    fs: {
      createWriteStream: sinon.stub().returns({
        on: () => {},
        emit: () => {},
      }),
    },
    'fs/promises': {
      unlink: unlinkStub,
      mkdir: mkdirStub,
    },
    'stream/promises': {
      pipeline: sinon.stub().resolves(),
    },
  });

  const result = await mod.downloadBinary('5.3.3');
  t.false(result.success);
  t.true(result.reason.includes('Checksum mismatch'));
});

test.serial('downloadBinary returns success on full happy path', async (t) => {
  const checksumsContent = 'abc123def456789  aether_5.3.3_darwin_arm64.tar.gz\n';

  // Create a fake hash that matches
  const crypto = require('crypto');
  const fakeArchiveData = 'fake archive content for hash test';
  const expectedHash = crypto.createHash('sha256').update(fakeArchiveData).digest('hex');

  const checksumsWithRealHash = `${expectedHash}  aether_5.3.3_darwin_arm64.tar.gz\n`;

  let getCallCount = 0;
  const httpsStub = {
    get: sinon.stub().callsFake((url, cb) => {
      getCallCount++;
      if (url.includes('checksums')) {
        if (getCallCount === 1) {
          // Redirect for checksums
          const res = createMockResponse(302, { location: 'https://objects.githubusercontent.com/checksums-final' });
          process.nextTick(() => cb(res));
          return { on: () => {} };
        }
        // Final checksums response
        const res = createMockResponse(200, {});
        res.on = (event, fn) => {
          if (event === 'data') fn(checksumsWithRealHash);
          if (event === 'end') fn();
        };
        process.nextTick(() => cb(res));
        return { on: () => {} };
      }
      // Archive requests
      if (getCallCount === 3) {
        const res = createMockResponse(302, { location: 'https://objects.githubusercontent.com/archive-final' });
        process.nextTick(() => cb(res));
        return { on: () => {} };
      }
      // Final archive response with data that hashes correctly
      const res = createMockResponse(200, {});
      res.on = (event, fn) => {
        if (event === 'data') fn(Buffer.from(fakeArchiveData));
        if (event === 'end') fn();
      };
      process.nextTick(() => cb(res));
      return { on: () => {} };
    }),
  };

  const execFileStub = sinon.stub().callsArgWith(2, null, '', '');
  const renameStub = sinon.stub().resolves();
  const chmodStub = sinon.stub().resolves();
  const unlinkStub = sinon.stub().resolves();
  const rmStub = sinon.stub().resolves();
  const mkdirStub = sinon.stub().resolves();

  const mod = proxyquire('../../bin/lib/binary-downloader', {
    https: httpsStub,
    http: { get: sinon.stub() },
    fs: {
      createWriteStream: sinon.stub().returns({
        on: () => {},
        emit: () => {},
      }),
    },
    'fs/promises': {
      rename: renameStub,
      chmod: chmodStub,
      unlink: unlinkStub,
      rm: rmStub,
      mkdir: mkdirStub,
    },
    'child_process': {
      execFile: execFileStub,
    },
    'stream/promises': {
      pipeline: sinon.stub().resolves(),
    },
  });

  const result = await mod.downloadBinary('5.3.3');
  t.true(result.success);
  t.true(result.path.includes('aether'));
  t.false(result.path.includes('.exe')); // darwin, not windows
  t.true(renameStub.calledOnce);
  t.true(chmodStub.calledOnce);
});

test.serial('downloadBinary never throws even when internal functions throw', async (t) => {
  const mod = proxyquire('../../bin/lib/binary-downloader', {
    https: {
      get: sinon.stub().callsFake(() => {
        throw new Error('Unexpected synchronous error');
      }),
    },
    http: { get: sinon.stub() },
  });

  // Should not throw -- returns failure object instead
  const result = await mod.downloadBinary('5.3.3');
  t.false(result.success);
  t.truthy(result.reason);
});

// ============================================================
// E. Integration Contract Tests (cli.js wiring verification)
// ============================================================

const fs = require('fs');
const pathModule = require('path');

test.serial('downloadBinary returns non-throwing result on all failure paths', async (t) => {
  // Verify the contract: downloadBinary never throws, always returns {success, reason}
  const mod = proxyquire('../../bin/lib/binary-downloader', {
    'https': { get: () => ({ on: () => {} }) },
    'http': {},
    'fs': { createWriteStream: () => ({ on: () => {} }) },
    'fs/promises': { mkdir: async () => {}, rename: async () => {}, unlink: async () => {}, chmod: async () => {} },
    'os': { tmpdir: () => '/tmp', homedir: () => '/home/test' },
    'child_process': { execFile: () => {} },
  });
  // Unsupported platform should return failure, not throw
  const originalPlatform = process.platform;
  const originalArch = process.arch;
  Object.defineProperty(process, 'platform', { value: 'freebsd' });
  Object.defineProperty(process, 'arch', { value: 'x64' });
  const result = await mod.downloadBinary('1.0.0');
  t.false(result.success);
  t.truthy(result.reason);
  Object.defineProperty(process, 'platform', { value: originalPlatform });
  Object.defineProperty(process, 'arch', { value: originalArch });
});

test('cli.js performGlobalInstall contains downloadBinary wiring', (t) => {
  // Verify the wiring exists in cli.js source
  const cliSource = fs.readFileSync(pathModule.join(__dirname, '../../bin/cli.js'), 'utf8');
  t.true(cliSource.includes("require('./lib/binary-downloader')"));
  t.true(cliSource.includes('downloadBinary(VERSION)'));
  t.true(cliSource.includes('try'));
  t.true(cliSource.includes('c.warning'));
});
