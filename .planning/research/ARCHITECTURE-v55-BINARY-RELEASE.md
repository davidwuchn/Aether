# Architecture Research: v5.5 Go Binary Release

**Domain:** Binary distribution, auto-install, and version-gated YAML wiring for existing npm+Go CLI
**Researched:** 2026-04-04
**Confidence:** HIGH (based on direct codebase analysis: .goreleaser.yml, cmd/ directory, package.json, bin/cli.js, update-transaction.js, .github/workflows/ci.yml, 87 YAML command files, 11 playbooks)

---

## Executive Summary

The Aether project has completed a shell-to-Go rewrite (v5.4) that produced a Go binary with 254+ Cobra commands. The YAML command files (87 in `.claude/commands/ant/`, 87 in `.opencode/commands/ant/`) and 11 playbooks (275 Go calls) already invoke `aether <subcommand>` in Bash blocks -- they are wired to call a binary named `aether` on PATH. The npm package (`aether-colony`) distributes these YAML files plus a Node.js CLI that handles install, update, and hub management. The Go binary is built locally via `make build` but has never been shipped to users.

The v5.5 milestone connects these pieces: goreleaser produces platform binaries attached to GitHub Releases, the npm `install` flow downloads the correct binary, `aether update` checks for newer binaries, and a version gate prevents YAML files from calling a Go binary that does not exist or is too old. This is an integration project, not a greenfield architecture -- the constraint is fitting into existing flows without breaking the 616+ tests or the npm distribution that currently works.

**Key architectural insight:** The YAML wiring already exists. Every YAML command file calls `aether <subcommand>` in Bash blocks. The Go binary already responds to all 254+ subcommands. The gap is distribution: users get the YAML files from npm but have no `aether` binary on PATH. The v5.5 work is about closing that gap safely.

---

## System Overview

### Current Distribution Flow (v5.4)

```
npm install -g aether-colony
       |
       v
  bin/cli.js install
       |
       +---> Copies YAML commands to ~/.claude/commands/ant/
       +---> Copies agents to ~/.claude/agents/ant/
       +---> Copies OpenCode equivalents to ~/.opencode/
       +---> setupHub() copies .aether/ -> ~/.aether/system/
       |
       v
  User runs /ant:init in Claude Code
       |
       v
  YAML file tells Claude to run: `aether <subcommand>`
       |
       v
  FAILS: "aether: command not found" (binary not distributed)
```

### Target Distribution Flow (v5.5)

```
npm install -g aether-colony
       |
       v
  bin/cli.js install
       |
       +---> [EXISTING] Copies YAML/agents/OpenCode files
       +---> [NEW] Downloads platform binary to ~/.aether/bin/aether
       +---> [NEW] Adds ~/.aether/bin to PATH (or symlinks)
       +---> [NEW] Runs `aether version` to verify binary works
       +---> [NEW] Records binary version in ~/.aether/version.json
       |
       v
  User runs /ant:init in Claude Code
       |
       v
  YAML file tells Claude to run: `aether <subcommand>`
       |
       v
  WORKS: binary on PATH, 254+ commands available
```

### Architecture Diagram

```
+------------------------------------------------------------------+
|                     GITHUB RELEASE (goreleaser)                    |
|   aether_5.5.0_darwin_amd64.tar.gz                                |
|   aether_5.5.0_darwin_arm64.tar.gz                                |
|   aether_5.5.0_linux_amd64.tar.gz     ...checksums.txt            |
|   aether_5.5.0_linux_arm64.tar.gz      ...checksums.txt.sig       |
+------------------------------------------------------------------+
         | tag push v5.5.0           | npm publish
         v                           v
+------------------+    +-------------------------+
| GitHub Release   |    | npm registry            |
| (binary assets)  |    | aether-colony@5.5.0     |
+------------------+    |  - bin/cli.js           |
         |              |  - .claude/commands/ant/ |
         |              |  - .opencode/commands/   |
         |              |  - .aether/ (public)     |
         |              +-------------------------+
         |                        |
         +------ USER MACHINE ----+
                    |              |
                    v              v
         +----------------+  +------------------+
         | npm install -g |  | bin/cli.js       |
         | aether-colony  |  |  install command |
         +----------------+  +------------------+
                                |           |
                 +--------------+           |
                 | [NEW] Binary            | [EXISTING] YAML/agent
                 | Downloader              | file sync
                 v                         v
         +------------------+    +---------------------+
         | ~/.aether/bin/   |    | ~/.claude/commands/ |
         |   aether (bin)   |    |   ant/*.md (87)     |
         +------------------+    +---------------------+
                 |                        |
                 v                        v
         +------------------+    +---------------------+
         | PATH:            |    | YAML files call:     |
         |  aether <cmd>    |    |  `aether <subcmd>`   |
         +------------------+    +---------------------+
                                            |
                                            v
                                 +---------------------+
                                 | Go binary executes   |
                                 | 254+ Cobra commands  |
                                 +---------------------+
```

---

## Component Inventory

### Existing Components (modified)

| Component | File(s) | Change Type | What Changes |
|-----------|---------|-------------|-------------|
| npm install flow | `bin/cli.js` performGlobalInstall() | MODIFY | Add binary download step after hub setup |
| npm update flow | `bin/cli.js` update command handler | MODIFY | Add binary version check + download |
| CI workflow | `.github/workflows/ci.yml` | MODIFY | Add goreleaser snapshot validation |
| package.json | `package.json` | MODIFY | Bump version, add binary-download dependency or script |
| validate-package.sh | `bin/validate-package.sh` | MODIFY | Verify .goreleaser.yml consistency |

### New Components

| Component | File(s) | Responsibility |
|-----------|---------|---------------|
| Binary downloader | `bin/lib/binary-downloader.js` | Detect platform, download from GitHub Release, verify checksum, install to ~/.aether/bin/ |
| Version gate | `bin/lib/version-gate.js` | Check `aether version` matches minimum required, return pass/fail |
| Release workflow | `.github/workflows/release.yml` | goreleaser action on tag push |
| Binary shim (Windows) | `bin/aether.cmd` | Windows PATH compatibility |

### Unchanged Components

| Component | Why Unchanged |
|-----------|---------------|
| Go binary source | `cmd/*.go` -- already works, 254+ commands complete |
| YAML source files | `.aether/commands/*.yaml` -- already call `aether <subcommand>` |
| Generated YAML files | `.claude/commands/ant/*.md` -- already contain `aether <subcommand>` calls |
| Playbook files | `.aether/docs/command-playbooks/*.md` -- already contain 275 Go calls |
| Hub setup | `setupHub()` -- copies .aether/ to hub, unrelated to binary |
| UpdateTransaction | `bin/lib/update-transaction.js` -- handles file sync rollback, not binary |
| Go storage layer | `pkg/storage/` -- binary distribution does not change storage |

---

## Integration Point 1: goreleaser + GitHub Releases

### Current State

The `.goreleaser.yml` already exists with correct configuration:
- Builds from `./cmd/aether/` (confirmed: `cmd/aether/main.go` exists)
- Injects version via ldflags: `-X github.com/aether-colony/aether/cmd.Version={{.Version}}`
- Cross-compiles: linux/darwin/windows x amd64/arm64 (6 platforms)
- Produces tar.gz archives (zip for Windows)
- Generates checksums.txt
- Module path: `github.com/aether-colony/aether`

### What Needs to Happen

1. **New GitHub Actions workflow** (`.github/workflows/release.yml`):
   - Triggers on tag push (`v*` pattern)
   - Uses `goreleaser/goreleaser-action@v6`
   - Sets `GITHUB_TOKEN` with `contents: write` permission
   - Go version should match `go.mod`: `1.26.1`

2. **Tag format alignment**: Existing tags use `v5.4`, `v5.3.3` etc. These are valid semver for goreleaser. The `Version` variable in `cmd/version.go` is injected via ldflags, so goreleaser's `{{.Version}}` (derived from git tag) flows directly.

3. **No changes to .goreleaser.yml needed** -- it is production-ready as-is. The config already handles:
   - `go mod tidy` as a pre-hook
   - `CGO_ENABLED=0` for static binaries
   - Archive naming with OS/arch templates
   - Changelog generation with exclusion filters

### Risks

- The `version` field in `.goreleaser.yml` is `version: 2` which requires goreleaser v2+. Pin to `~> v2` in the action.
- The GitHub repo is `calcosmic/Aether` but the Go module is `github.com/aether-colony/aether`. goreleaser uses the git remote for the release, not the Go module path. No conflict, but worth noting.

---

## Integration Point 2: Binary Download on npm Install

### Where It Fits

The `performGlobalInstall()` function in `bin/cli.js` (line 1322) runs during `npm install -g aether-colony` (via postinstall hook: `node bin/cli.js install --quiet`). It currently does:

1. Sync YAML commands to `~/.claude/commands/ant/`
2. Sync agents to `~/.claude/agents/ant/`
3. Sync OpenCode equivalents
4. Call `setupHub()` to copy `.aether/` to `~/.aether/system/`
5. Print success message

**Binary download inserts between step 4 and step 5.**

### Binary Download Logic

```
NEW: binary-downloader.js

Input:  version from package.json (e.g., "5.5.0")
        platform from process.platform + process.arch
Output: installed binary at ~/.aether/bin/aether
        version recorded in ~/.aether/version.json

Steps:
1. Detect platform:
   - darwin + arm64 -> aether_{version}_darwin_arm64.tar.gz
   - darwin + x64   -> aether_{version}_darwin_amd64.tar.gz
   - linux + arm64  -> aether_{version}_linux_arm64.tar.gz
   - linux + x64    -> aether_{version}_linux_amd64.tar.gz
   - win32 + x64    -> aether_{version}_windows_amd64.zip
   - win32 + arm64  -> aether_{version}_windows_arm64.zip

2. Construct download URL:
   https://github.com/calcosmic/Aether/releases/download/v{version}/aether_{version}_{os}_{arch}.tar.gz

3. Download to temp file

4. Verify checksum against checksums.txt from same release

5. Extract binary to ~/.aether/bin/aether
   - chmod +x on non-Windows
   - On Windows: aether.exe to ~/.aether/bin/aether.exe

6. Update ~/.aether/version.json:
   { "version": "5.5.0", "binaryVersion": "5.5.0", "binaryPlatform": "darwin_arm64" }

7. Verify: run `~/.aether/bin/aether version` and check output matches expected version
```

### PATH Management

Two approaches, recommendation first:

**Recommended: Symlink to global npm bin directory**
- npm already manages a global bin directory (where `aether` npm entry point lives)
- Create symlink from `{npm_global_bin}/aether-go` -> `~/.aether/bin/aether`
- Or better: the npm package already has `"bin": { "aether": "bin/cli.js" }` -- the Go binary can be `~/.aether/bin/aether` and PATH setup can be done in the shell profile
- Simplest: `~/.aether/bin/` added to PATH via a profile script that npm install appends to `.bashrc`/`.zshrc`
- Risk: modifying shell profiles is invasive and error-prone

**Alternative: Rename binary, distribute alongside npm**
- Go binary named `aether-bin` or `aether-go` to avoid collision with npm `aether` entry point
- YAML files would need to call `aether-bin` instead of `aether`
- Problem: 275 Go calls across 98 files would all need updating

**Best approach: Shadow the npm `aether` command**
- The npm `bin.aether` currently points to `bin/cli.js` (Node CLI)
- The Node CLI handles install, update, setup, and a few other commands
- The Go binary handles all 254+ operational commands
- Strategy: keep npm `bin.aether` for install/update, but have the Node CLI forward unknown commands to the Go binary
- This way YAML files call `aether <subcommand>` and it works whether the Go binary is primary or the Node CLI forwards

Actually, the simplest approach given the existing architecture:

**Revised recommendation: `~/.aether/bin/aether` with PATH prepend in postinstall**
- The postinstall script already runs `node bin/cli.js install --quiet`
- After downloading the binary to `~/.aether/bin/aether`, add `~/.aether/bin` to PATH via:
  1. Check if `~/.aether/bin` is already in PATH
  2. If not, append `export PATH="$HOME/.aether/bin:$PATH"` to `~/.bashrc` or `~/.zshrc`
  3. For current session: also set `process.env.PATH`
- The `aether` name is unique -- no collision because npm's `bin.aether` goes to `bin/cli.js` which is the Node CLI, and the Go binary at `~/.aether/bin/aether` takes precedence when PATH is set up correctly
- On Windows: add to user PATH via `setx`

**Even simpler: just check `~/.aether/bin` exists and prepend it at runtime**
- `bin/cli.js` can prepend `~/.aether/bin` to PATH in its own process
- But YAML files run Bash commands independently, so they need the PATH to be set in the shell session
- Claude Code and OpenCode spawn Bash shells that source `.bashrc`/`.zshrc`

**Final recommendation: profile PATH injection with idempotency guard**

```bash
# In postinstall, after binary download:
if ! echo "$PATH" | grep -q "$HOME/.aether/bin"; then
  echo 'export PATH="$HOME/.aether/bin:$PATH"' >> ~/.bashrc 2>/dev/null || true
  echo 'export PATH="$HOME/.aether/bin:$PATH"' >> ~/.zshrc 2>/dev/null || true
fi
```

---

## Integration Point 3: Binary Auto-Install on `aether update`

### Current Update Flow

The `aether update` command in `bin/cli.js` (line 1427):
1. Reads `~/.aether/version.json` for hub version
2. For each registered repo, calls `updateRepo()` which:
   - Creates checkpoint (two-phase commit via UpdateTransaction)
   - Syncs system files from hub to repo's `.aether/`
   - Syncs commands and agents
   - Updates version tracking

### Where Binary Check Fits

After step 2 completes (file sync), before reporting success:

```
NEW STEP: Check binary version
1. Run `aether version 2>/dev/null` to get installed binary version
2. Compare to version.json binaryVersion (or package version)
3. If binary missing or outdated:
   a. Download new binary using same binary-downloader.js
   b. Verify checksum
   c. Update version.json
4. If download fails: log warning, do NOT block the update (non-blocking)
```

This keeps the update flow resilient: YAML files and agents always update even if binary download fails. The binary is a progressive enhancement.

### Version Check Logic

```javascript
// version-gate.js
function getInstalledBinaryVersion() {
  try {
    const result = execSync('aether version', { encoding: 'utf8', timeout: 5000 });
    // Go binary outputs: "v5.5.0" or via JSON: {"ok":true,"result":"0.0.0-dev"}
    const match = result.match(/v?(\d+\.\d+\.\d+)/);
    return match ? match[1] : null;
  } catch {
    return null; // binary not found or broken
  }
}

function checkVersionGate(requiredVersion) {
  const installed = getInstalledBinaryVersion();
  if (!installed) return { pass: false, reason: 'binary not found' };
  if (semver.lt(installed, requiredVersion)) return { pass: false, reason: `binary ${installed} < required ${requiredVersion}` };
  return { pass: true, version: installed };
}
```

---

## Integration Point 4: Version-Gated YAML Wiring

### The Problem

YAML files currently call `aether <subcommand>` unconditionally. If the binary is missing or too old, the Bash command fails silently or with an error that the LLM cannot interpret. This degrades the user experience.

### The Gate

The version gate is NOT about rewriting YAML files conditionally. The YAML files are already written and call `aether` commands. The gate is about ensuring the binary is present before the YAML commands are invoked.

**Strategy: Gate at install time, not at runtime**

1. During `npm install`, download the binary. If download fails, the YAML commands will fail but the system still works for install/update (Node CLI handles those).
2. During `aether update`, check and refresh the binary. Non-blocking if it fails.
3. Add a pre-flight check in critical YAML commands (init, build, continue) that verifies the binary exists:

```bash
# Version gate check in YAML files (top of critical commands only)
if ! command -v aether &>/dev/null; then
  echo "ERROR: aether binary not found. Run: npm install -g aether-colony"
  exit 1
fi
```

4. For the full YAML fleet: no changes needed. The 275 `aether <subcommand>` calls already produce clear error messages when the binary is missing. The gate is at install time.

### What NOT To Do

- Do NOT add version checks to every `aether <subcommand>` call in every YAML file. That is 275 insertions across 98 files -- unmaintainable.
- Do NOT try to conditionally swap YAML content based on binary presence. The YAML files are static markdown served to the LLM.
- Do NOT make the binary mandatory for npm install. Some users may only want the YAML/agent files without the Go binary.

---

## Data Flow Changes

### New State in ~/.aether/version.json

```json
{
  "version": "5.5.0",
  "binaryVersion": "5.5.0",
  "binaryPlatform": "darwin_arm64",
  "binaryInstalledAt": "2026-04-04T12:00:00Z",
  "binaryChecksum": "sha256:abc123..."
}
```

### New Files on User Machine

| Path | Created By | Purpose |
|------|-----------|---------|
| `~/.aether/bin/aether` | binary-downloader.js | Go binary executable |
| `~/.aether/bin/aether.exe` | binary-downloader.js | Windows variant |
| `~/.aether/cache/aether_*.tar.gz` | binary-downloader.js | Download cache (temporary) |

### Modified Files on User Machine

| Path | Change |
|------|--------|
| `~/.bashrc` or `~/.zshrc` | PATH prepend for `~/.aether/bin` |
| `~/.aether/version.json` | New fields: binaryVersion, binaryPlatform, binaryInstalledAt, binaryChecksum |

---

## Build Order (Dependency-Aware)

The phases must be ordered so that each phase's prerequisites are met.

### Phase 1: CI + goreleaser Validation (no user-facing changes)

**Goal:** Verify goreleaser config works without shipping anything.

**Changes:**
- Add `.github/workflows/release.yml` (goreleaser action on tag push)
- Add goreleaser snapshot validation to `.github/workflows/ci.yml` (PR test)
- No changes to npm package or binary distribution

**Dependencies:** None. Pure CI addition.
**Test:** Push a `v5.5.0-beta.1` tag, verify GitHub Release is created with 6 platform archives.

### Phase 2: Binary Downloader (npm-side)

**Goal:** npm install can download and install the Go binary.

**Changes:**
- New `bin/lib/binary-downloader.js`
- Modify `performGlobalInstall()` in `bin/cli.js` to call binary downloader after hub setup
- PATH management in postinstall
- Update `package.json` version to match goreleaser version

**Dependencies:** Phase 1 must produce a GitHub Release with binaries (need a tag).
**Test:** `npm install -g .` should download binary, verify it works, and `aether version` should return the correct version.

### Phase 3: Update Flow Binary Refresh

**Goal:** `aether update` checks and refreshes the binary.

**Changes:**
- New `bin/lib/version-gate.js`
- Modify update command handler in `bin/cli.js` to check binary version after file sync
- Non-blocking: update succeeds even if binary download fails

**Dependencies:** Phase 2 (binary-downloader.js must exist).
**Test:** With an outdated binary, run `aether update`, verify binary is refreshed.

### Phase 4: Version Gate in Critical YAML Commands

**Goal:** Critical commands fail fast with a clear message if binary is missing.

**Changes:**
- Add binary existence check to `init.yaml`, `build.yaml`, `continue.yaml` (3 files)
- Regenerate Claude and OpenCode command files via `npm run generate`
- Update `validate-package.sh` to verify generated files contain the check

**Dependencies:** Phase 2 (binary must be downloadable for the check to make sense).
**Test:** Remove binary from PATH, run `/ant:init`, verify clear error message.

### Phase Ordering Rationale

```
Phase 1 (CI/goreleaser)     -- independent, no risk
     |
     v
Phase 2 (binary downloader) -- depends on Phase 1 release artifacts
     |
     v
Phase 3 (update refresh)    -- depends on Phase 2 downloader
     |
     v
Phase 4 (version gates)     -- depends on Phase 2 downloader
```

Phases 3 and 4 can run in parallel since they both only depend on Phase 2.

---

## Architectural Patterns

### Pattern 1: Non-Blocking Binary Download

**What:** Binary download failures never block the install/update flow.
**When:** Always. The YAML/agent sync is the primary distribution; the binary is a performance enhancement.
**Trade-off:** Users may have stale YAML files calling a missing binary. Mitigated by Phase 4's gate in critical commands.

```
// Pattern: try/catch with warning, never throw
try {
  await downloadBinary(version, platform);
  console.log('  Binary: installed');
} catch (err) {
  console.warn('  Binary: download failed (non-critical)');
  console.warn('  Run "aether update" later to retry');
}
```

### Pattern 2: Single Source of Truth for Version

**What:** `package.json` version is the canonical version. goreleaser reads it via the Makefile. The npm package ships it. The Go binary gets it via ldflags.
**When:** During release.
**Trade-off:** Requires package.json, Makefile VERSION, and git tag to all agree. The Makefile already reads from package.json, so this is already consistent.

```
// Makefile (existing)
VERSION := $(shell node -e "console.log(require('./package.json').version)")
LDFLAGS := -X github.com/aether-colony/aether/cmd.Version=$(VERSION)
```

### Pattern 3: Checksum Verification Before Install

**What:** Every downloaded binary is verified against the published checksums.txt before being placed on PATH.
**When:** During binary download (install and update).
**Trade-off:** Requires fetching checksums.txt alongside the binary archive. Small overhead for high security.

### Pattern 4: Hub-Local Binary Storage

**What:** Binary lives at `~/.aether/bin/aether`, not in the npm global bin directory.
**When:** Always. The npm bin directory is managed by npm and may be cleaned during npm operations.
**Trade-off:** Requires PATH management. But `~/.aether/` is already a managed directory (hub), so this is consistent.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Shipping Binary in npm Package

**What people do:** Include the pre-built Go binary in the npm tarball.
**Why it is wrong:** npm packages are platform-agnostic JavaScript. A 10MB+ platform-specific binary inflates the package size for all users regardless of platform. npm publish would need to build 6 platform variants. npm's `optionalDependencies` pattern for native binaries is fragile.
**Do this instead:** Download binary from GitHub Releases at install time, matching the user's platform.

### Anti-Pattern 2: Making Binary Mandatory

**What people do:** Fail npm install if binary download fails.
**Why it is wrong:** Network issues, GitHub API rate limits, air-gapped environments, or CI environments without GitHub access should not prevent installing the YAML commands and agents. The YAML system can degrade gracefully.
**Do this instead:** Non-blocking download with warning. Critical commands fail fast with clear instructions.

### Anti-Pattern 3: Version-Specific YAML Files

**What people do:** Generate different YAML files based on available Go binary version.
**Why it is wrong:** The YAML files are LLM prompts stored in git and distributed via npm. Generating different versions per user creates a support nightmare. The YAML files already call `aether <subcommand>` -- they should stay static.
**Do this instead:** Keep YAML files static. Add a single binary-existence check at the top of critical commands.

### Anti-Pattern 4: Replacing npm CLI with Go Binary

**What people do:** Make the Go binary the sole entry point, removing the Node CLI.
**Why it is wrong:** The Node CLI handles npm integration (postinstall hooks, hub setup, YAML sync). The Go binary handles operational commands (pheromone-write, state-mutate, etc.). They serve different purposes. Removing the Node CLI would break `npm install -g aether-colony` workflow.
**Do this instead:** Keep both. Node CLI for distribution lifecycle. Go binary for operational commands.

---

## Platform Support Matrix

| OS | Arch | Archive Format | Binary Name | Notes |
|----|------|---------------|-------------|-------|
| darwin | arm64 | tar.gz | aether | Apple Silicon (M1+) |
| darwin | amd64 | tar.gz | aether | Intel Macs |
| linux | arm64 | tar.gz | aether | ARM servers, Raspberry Pi |
| linux | amd64 | tar.gz | aether | Standard servers |
| windows | arm64 | zip | aether.exe | ARM Windows |
| windows | amd64 | zip | aether.exe | Standard Windows |

The `.goreleaser.yml` already covers all 6 combinations. The `binary-downloader.js` must map `process.platform` + `process.arch` to the correct archive name.

Node.js platform/arch mapping:
- `process.platform`: `darwin`, `linux`, `win32`
- `process.arch`: `arm64`, `x64` (Node uses `x64`, goreleaser uses `amd64`)

**Important mapping:** `x64` in Node -> `amd64` in goreleaser archive names.

---

## Scalability Considerations

| Concern | Current (0-1k users) | Growth (1k-10k) | Scale (10k+) |
|---------|---------------------|-----------------|-------------|
| GitHub Release bandwidth | Free, unlimited | Free, unlimited | Free, unlimited |
| Download reliability | Direct GitHub URL | Direct GitHub URL | Consider CDN or mirror |
| Checksum verification | SHA256 from checksums.txt | Same | Same |
| Binary size | ~15-20MB per platform | Same | Same |
| Install time | ~2s (YAML) + ~3s (binary download) | Same | May need parallel download |

**First bottleneck:** GitHub API rate limits for unauthenticated releases. GitHub Releases are served from a CDN with no rate limit for public repos, so this is unlikely to be an issue.

**Second bottleneck:** Binary size as Go binary grows. Mitigated by `CGO_ENABLED=0` (static, no libc dependency) and UPX compression if needed.

---

## Rollback Strategy

If a binary release is broken:

1. **npm level:** Pin `package.json` to a known-good version. The npm package continues to distribute YAML files correctly.
2. **Binary level:** Delete `~/.aether/bin/aether`. The system degrades gracefully -- YAML commands fail but install/update still works.
3. **GitHub Release level:** Delete or mark the broken release as a pre-release. The next `aether update` will fail to download, which is non-blocking.

---

## Integration with Existing Update Transaction System

The `UpdateTransaction` class in `bin/lib/update-transaction.js` implements a two-phase commit with checkpoint, sync, verify, and rollback. Binary download should NOT be part of this transaction because:

1. Binary download is platform-specific and may legitimately fail (wrong arch, network issue).
2. The transaction handles file sync within a repo, not global binary management.
3. Binary rollback is simple: delete the file. No checkpoint needed.

**Recommendation:** Binary download runs OUTSIDE the UpdateTransaction, after it completes. If the transaction fails and rolls back, the binary download simply does not happen. If the transaction succeeds, the binary download runs and failure is non-blocking.

---

## Sources

- `.goreleaser.yml` -- existing goreleaser configuration (analyzed directly)
- `cmd/aether/main.go`, `cmd/root.go`, `cmd/version.go` -- Go binary entry points (analyzed directly)
- `bin/cli.js` -- npm CLI with install and update flows (analyzed directly)
- `bin/lib/update-transaction.js` -- two-phase commit for file updates (analyzed directly)
- `.github/workflows/ci.yml` -- existing CI pipeline (analyzed directly)
- `package.json` -- npm package configuration (analyzed directly)
- `Makefile` -- build configuration reading from package.json (analyzed directly)
- `.claude/commands/ant/*.md` -- 87 YAML command files with `aether <subcommand>` calls (pattern analyzed)
- `.aether/docs/command-playbooks/*.md` -- 11 playbooks with 275 Go binary calls (pattern analyzed)
- goreleaser documentation (training data -- HIGH confidence given the config is already written)
- goreleaser/goreleaser-action (training data -- standard GitHub Action pattern)

---
*Architecture research for: v5.5 Go Binary Release*
*Researched: 2026-04-04*
