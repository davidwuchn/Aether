#!/usr/bin/env node
/**
 * Binary Downloader for Aether Colony
 *
 * Downloads the correct platform Go binary from GitHub Releases during
 * `npm install -g aether-colony`. Verifies SHA-256 checksum before
 * atomic install to ~/.aether/bin/aether.
 *
 * Uses ONLY Node.js built-in modules. No external dependencies.
 */

const https = require('https');
const http = require('http');
const crypto = require('crypto');
const fs = require('fs');
const fsPromises = require('fs/promises');
const os = require('os');
const path = require('path');
const { execFile } = require('child_process');
const { pipeline } = require('stream/promises');
const { promisify } = require('util');

const execFileAsync = promisify(execFile);

// Platform detection maps process.platform + process.arch to goreleaser naming
const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows',
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
};

/**
 * Detect platform and architecture, mapped to goreleaser naming.
 * @returns {{os: string, arch: string}|null} Platform info or null if unsupported
 */
function getPlatformArch() {
  const goos = PLATFORM_MAP[process.platform];
  const goarch = ARCH_MAP[process.arch];
  if (!goos || !goarch) return null;
  return { os: goos, arch: goarch };
}

/**
 * Download a URL following HTTP redirects (GitHub releases always 302).
 * Node.js https.get() does NOT follow redirects automatically.
 *
 * @param {string} url - URL to download
 * @param {number} maxRedirects - Maximum redirect hops (default 5)
 * @returns {Promise<import('http').IncomingMessage>} Response stream on 200
 */
function downloadWithRedirects(url, maxRedirects = 5) {
  return new Promise((resolve, reject) => {
    function attempt(currentUrl, redirectsLeft) {
      const client = currentUrl.startsWith('https') ? https : http;
      client.get(currentUrl, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          if (redirectsLeft <= 0) {
            res.resume();
            return reject(new Error('Too many redirects'));
          }
          res.resume(); // Drain response body before next request
          return attempt(res.headers.location, redirectsLeft - 1);
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode}`));
        }
        resolve(res);
      }).on('error', reject);
    }
    attempt(url, maxRedirects);
  });
}

/**
 * Download a URL and return the full response body as a string.
 *
 * @param {string} url - URL to download
 * @returns {Promise<string>} Response body text
 */
async function downloadText(url) {
  const response = await downloadWithRedirects(url);
  return new Promise((resolve, reject) => {
    let data = '';
    response.on('data', (chunk) => { data += chunk; });
    response.on('end', () => resolve(data));
    response.on('error', reject);
  });
}

/**
 * Parse goreleaser checksums.txt to find the hash for a specific filename.
 * Format: <sha256_hex>  <filename> (two-space separator)
 *
 * @param {string} checksumsContent - Full checksums.txt content
 * @param {string} filename - Archive filename to find
 * @returns {string|null} SHA-256 hex digest or null if not found
 */
function findChecksum(checksumsContent, filename) {
  const lines = checksumsContent.split('\n');
  for (const line of lines) {
    const parts = line.split('  ');
    if (parts.length >= 2 && parts[1] === filename) {
      return parts[0];
    }
  }
  return null;
}

/**
 * Download an archive to a temp file while computing SHA-256 hash.
 * Hashing happens during the stream -- no extra I/O pass.
 *
 * @param {string} url - Archive URL
 * @param {string} tmpPath - Temp file path for the download
 * @returns {Promise<{hash: string, tmpPath: string}>} Computed hash and temp file path
 */
async function downloadAndHash(url, tmpPath) {
  const hash = crypto.createHash('sha256');
  const fileStream = fs.createWriteStream(tmpPath);

  const response = await downloadWithRedirects(url);

  // Hash while streaming -- zero extra I/O
  response.on('data', (chunk) => hash.update(chunk));

  await pipeline(response, fileStream);

  return { hash: hash.digest('hex'), tmpPath };
}

/**
 * Extract binary from archive using system tar command.
 * tar is available on macOS, Linux, and Windows 10+.
 *
 * @param {string} archivePath - Path to the archive file
 * @param {string} destDir - Directory to extract into
 * @param {string} goos - Goreleaser OS name (darwin, linux, windows)
 * @returns {Promise<string>} Path to the extracted binary
 */
async function extractBinary(archivePath, destDir, goos) {
  const binaryName = goos === 'windows' ? 'aether.exe' : 'aether';

  await fsPromises.mkdir(destDir, { recursive: true });

  await execFileAsync('tar', [
    '-xf', archivePath,
    '-C', destDir,
    '--strip-components', '1',
    binaryName,
  ]);

  return path.join(destDir, binaryName);
}

/**
 * Atomically install a binary by renaming from temp to final path.
 * fs.rename() is atomic on both POSIX and Windows (Node 14+).
 *
 * @param {string} extractedBinary - Path to the extracted binary
 * @param {string} targetPath - Final install path
 */
async function atomicInstall(extractedBinary, targetPath) {
  await fsPromises.mkdir(path.dirname(targetPath), { recursive: true });
  await fsPromises.rename(extractedBinary, targetPath);
  // Set executable permission on Unix
  if (process.platform !== 'win32') {
    await fsPromises.chmod(targetPath, 0o755);
  }
}

/**
 * Download and install the correct platform Go binary from GitHub Releases.
 *
 * This function NEVER throws. On any error it returns {success: false, reason: string}.
 *
 * @param {string} version - Version string (e.g. "5.3.3")
 * @param {object} [options] - Download options
 * @param {boolean} [options.quiet=false] - Suppress progress output
 * @param {number} [options.timeout=30000] - Download timeout in milliseconds
 * @returns {Promise<{success: boolean, reason?: string, path?: string}>} Result
 */
async function downloadBinary(version, options = {}) {
  const { quiet = false, timeout = 30000 } = options;

  try {
    // 1. Platform detection
    const platform = getPlatformArch();
    if (!platform) {
      return { success: false, reason: `Unsupported platform: ${process.platform}/${process.arch}` };
    }

    // 2. Construct URLs
    const archiveExt = platform.os === 'windows' ? '.zip' : '.tar.gz';
    const archiveFilename = `aether_${version}_${platform.os}_${platform.arch}${archiveExt}`;
    const baseUrl = `https://github.com/calcosmic/Aether/releases/download/v${version}`;
    const archiveUrl = `${baseUrl}/${archiveFilename}`;
    const checksumsUrl = `${baseUrl}/checksums.txt`;

    // 3. Download checksums.txt
    let checksumsContent;
    try {
      checksumsContent = await downloadText(checksumsUrl);
    } catch (err) {
      return { success: false, reason: `Failed to download checksums: ${err.message}` };
    }

    // 4. Parse expected hash
    const expectedHash = findChecksum(checksumsContent, archiveFilename);

    // 5. Download archive and compute hash
    const tmpArchive = path.join(os.tmpdir(), `aether-download-${Date.now()}.tmp`);

    // Race download against timeout, clearing timer on success
    let timeoutId;
    const downloadPromise = downloadAndHash(archiveUrl, tmpArchive);
    const timeoutPromise = new Promise((_, reject) => {
      timeoutId = setTimeout(() => reject(new Error('Download timed out')), timeout);
    });
    const downloadResult = await Promise.race([downloadPromise, timeoutPromise])
      .finally(() => clearTimeout(timeoutId));

    const { hash: actualHash } = downloadResult;

    // 6. Verify checksum
    if (expectedHash && actualHash !== expectedHash) {
      await fsPromises.unlink(tmpArchive).catch(() => {});
      return { success: false, reason: `Checksum mismatch for ${archiveFilename}` };
    }

    // 7. Extract binary from archive
    const tmpBinaryDir = path.join(os.tmpdir(), `aether-extract-${Date.now()}`);
    const extractedBinary = await extractBinary(tmpArchive, tmpBinaryDir, platform.os);

    // 8. Atomic install
    const targetPath = path.join(
      os.homedir(), '.aether', 'bin',
      process.platform === 'win32' ? 'aether.exe' : 'aether'
    );
    await atomicInstall(extractedBinary, targetPath);

    // 9. Cleanup temp files (best effort)
    await fsPromises.unlink(tmpArchive).catch(() => {});
    await fsPromises.rm(tmpBinaryDir, { recursive: true }).catch(() => {});

    return { success: true, path: targetPath };
  } catch (err) {
    return { success: false, reason: err.message };
  }
}

module.exports = {
  downloadBinary,
  getPlatformArch,
  // Internal helpers exported for testing (prefixed with _)
  _findChecksum: findChecksum,
  _downloadWithRedirects: downloadWithRedirects,
  _downloadText: downloadText,
  _downloadAndHash: downloadAndHash,
  _extractBinary: extractBinary,
  _atomicInstall: atomicInstall,
};
