# Project Research Summary

**Project:** Aether v5.5 Go Binary Release
**Domain:** Binary release, auto-install, and version-gated YAML wiring for existing npm-distributed Go CLI
**Researched:** 2026-04-04
**Confidence:** HIGH

## Executive Summary

Aether v5.4 completed a full shell-to-Go rewrite, producing a Go binary with 254+ Cobra commands that replaces the bash-based dispatcher. However, the binary has never been distributed to users -- it only exists locally via `make build`. The v5.5 milestone closes this gap: goreleaser produces cross-platform binaries on GitHub Releases, the npm install flow downloads the correct platform binary, `aether update` keeps it current, and a version gate prevents YAML commands from calling a binary that does not exist or is outdated. This is an integration project, not a greenfield build -- the YAML command files already call `aether <subcommand>`, and the Go binary already responds to all 254+ subcommands. The missing piece is distribution.

The recommended approach is minimal and safe. goreleaser v2 (already 90% configured) handles cross-platform builds and GitHub Release creation. A Node.js binary downloader (using only stdlib `https` + `crypto` + `tar` package) fetches the correct platform binary during `npm install` or `aether update`. The critical design decision is that the npm `aether` command becomes a thin shim that delegates to the Go binary when present and falls back to Node.js when it is not. This means binary download failure never breaks the user's workflow -- it is a progressive enhancement, not a hard requirement. The version gate is a simple three-condition check: binary exists, is executable, and `aether version` matches the expected version. Only when all three pass does the system consider the binary safe to use.

The key risks are: (1) the half-updated state where YAML is wired to Go but the binary is broken -- prevented by strict two-phase rollout (binary first, wiring second); (2) PATH collision between npm's `aether` shim and the Go binary -- prevented by hub-scoped install to `~/.aether/bin/` with the npm shim delegating to it; (3) non-atomic binary replacement leaving a corrupted binary -- prevented by download-to-temp, verify, then rename pattern. The research found that Aether's version-gated YAML wiring approach is genuinely novel among Go CLI tools distributed via npm (esbuild, turbo, biome use binary-or-bust patterns), providing a resilience advantage.

## Key Findings

### Recommended Stack

Minimal stack additions -- no new Go dependencies, only one new npm dependency (`tar`). All research is grounded in direct codebase inspection of existing files.

**Core technologies:**
- goreleaser v2.15.2: Cross-platform binary build + GitHub Release -- `.goreleaser.yml` already exists with 6-platform config, only needs universal binaries and release workflow
- goreleaser-action v7.0.0: Run goreleaser in GitHub Actions -- latest stable (2026-02-21), requires Node 24/ESM
- actions/checkout v6.0.2 + actions/setup-go v6.4.0: CI foundation -- prior research had stale versions (v4/v5), now corrected
- Node.js stdlib `https` + `crypto`: Binary download and SHA-256 checksum verification -- zero new dependencies for HTTP and hashing
- `tar` npm package v7.5.13: Extract `.tar.gz` archives -- already in package.json overrides, promote to direct dependency
- Custom semver comparison: 10-line function for version gating -- `semver` npm package is overkill for "is X >= Y"

**What NOT to use:** Cosign/SBOM (oversized for current scale), Docker/Snap/nfpm (wrong distribution model), bundled binaries in npm (100MB+ package), `node-fetch`/`got`/`axios` (stdlib handles GitHub redirects fine).

### Expected Features

**Must have (table stakes -- P1 for v5.5 launch):**
- goreleaser release pipeline -- tag-triggered CI producing 6 platform binaries (darwin/linux/windows x amd64/arm64) with checksums
- Binary download during `aether install` -- platform detection, download from GitHub Releases, install to `~/.aether/bin/`
- Binary download during `aether update` -- check binary version, refresh if outdated, non-blocking on failure
- Version gate logic -- three-condition check (exists + executable + version matches) before routing to Go binary
- npm shim delegation -- `bin/cli.js` delegates to Go binary when present, falls back to Node.js when not

**Should have (competitive -- P2 after validation):**
- Checksum verification on download -- fetch `checksums.txt`, verify SHA-256 against downloaded binary
- Homebrew tap distribution -- goreleaser `brews` section pushing to `calcosmic/homebrew-tap`
- Atomic binary swap with backup -- keep `aether.old` for rollback capability

**Defer (v5.6+):**
- Binary self-update (`aether update-self`) -- conflicts with npm update flow, too complex for current scope
- Code signing (Apple Developer + Windows) -- cost/benefit does not justify for current user base
- Remove shell fallback entirely -- shell fallback is insurance against binary issues, keep it indefinitely
- `go:embed` for self-contained binary -- separate milestone, not distribution

### Architecture Approach

This is an integration project into an existing system. The Go binary (254+ commands), YAML command files (87 files with 275 `aether <subcommand>` calls), playbooks (11 files with 275 Go calls), and npm distribution (`bin/cli.js` with install/update/hub management) all already exist. The v5.5 work adds four components that connect these pieces.

**Major components:**
1. **Release workflow** (`.github/workflows/release.yml`) -- goreleaser action triggered on `v*` tag push, produces platform binaries uploaded to GitHub Releases. Separate from existing `ci.yml`.
2. **Binary downloader** (`bin/lib/binary-downloader.js`) -- Detects platform, downloads from GitHub Releases, verifies checksum, installs to `~/.aether/bin/aether` with atomic rename (download to `.new`, verify, rename).
3. **Version gate** (`bin/lib/version-gate.js`) -- Runs `aether version`, compares against minimum required version, returns pass/fail. Used by npm shim and update flow.
4. **npm shim enhancement** (`bin/cli.js` modification) -- `aether` entry point delegates to Go binary when version gate passes, falls back to Node.js CLI when it does not.

**Key patterns to follow:**
- Non-blocking binary download -- failures never block install/update; YAML/agent sync is primary distribution
- Single source of truth for version -- `package.json` is canonical; Makefile reads it; goreleaser reads git tag; CI asserts they match
- Hub-local binary storage -- `~/.aether/bin/aether`, not npm global bin (npm may clean its directory)
- Checksum verification before install -- every downloaded binary verified against published checksums.txt
- Binary download runs OUTSIDE UpdateTransaction -- file sync transaction handles repo files; binary is a separate concern with simpler rollback (delete file)

### Critical Pitfalls

1. **The Half-Updated State** -- Installing binary AND swapping YAML in one step creates a window where a failed binary download leaves 45 commands broken simultaneously. Prevention: strict two-phase rollout -- binary first (Phase 2), YAML wiring only after binary confirmed working (Phase 4). The version gate must check all three conditions (exists + executable + version), not just existence.

2. **PATH Collision Between npm `aether` and Binary `aether`** -- Two executables with the same name on different PATH entries create unpredictable behavior across shells, terminals, and install methods. Prevention: install binary to `~/.aether/bin/` (hub-scoped), have npm shim delegate to it when available. Both paths lead to the same binary, eliminating the collision.

3. **Non-Atomic Binary Replacement** -- Writing directly to the final path means network interruption or disk-full leaves a corrupted binary with no way to recover. Prevention: download to `aether.new`, verify checksum, verify executable, rename `aether` to `aether.old`, rename `aether.new` to `aether`, verify, then delete `aether.old`. This is the single most important safety mechanism for the milestone.

4. **Version Drift Between npm and Binary** -- Two independent versioning channels (npm registry vs git tags) can desync, causing YAML commands to use flags the binary does not have. Prevention: single source of truth (package.json), CI assertion that git tag matches package.json version, binary self-reports version via ldflags injection, update flow compares versions before proceeding.

5. **goreleaser Config Gaps** -- The existing config lacks universal binaries for macOS, `.exe` naming for Windows, and has `go mod tidy` in before-hooks (which can fail in CI). Prevention: add `universal_binaries` section, set `binary: aether.exe` for Windows, move `go mod tidy` to a separate CI step.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: goreleaser Release Pipeline
**Rationale:** Foundation for everything else. Must produce downloadable binaries before any download logic can be tested. Independent of npm changes -- zero risk to existing users. Pure CI addition.
**Delivers:** `.github/workflows/release.yml` triggered on `v*` tags, completed `.goreleaser.yml` (universal binaries, `.exe` naming, strip flags), goreleaser snapshot validation in CI, first GitHub Release with 6 platform archives + checksums
**Addresses:** STACK goreleaser v2.15.2, ARCHITECTURE Integration Point 1
**Avoids:** PITFALLS #3 (goreleaser config gaps), #5 (version drift -- CI version alignment)

**Concrete work items:**
- Complete `.goreleaser.yml`: add `id: aether`, `binary: aether`/`aether.exe`, `-s -w` ldflags, `universal_binaries` for macOS, `ignore: windows/arm64`, LICENSE in archives
- Remove `go mod tidy` from before-hooks, run in CI separately
- Create `.github/workflows/release.yml` with test job + goreleaser release job
- Add `goreleaser check` to existing `ci.yml`
- Pin action versions: checkout@v6, setup-go@v6, goreleaser-action@v7
- Test by pushing `v5.5.0-beta.1` tag

### Phase 2: Binary Downloader + npm Install Integration
**Rationale:** Depends on Phase 1 (needs release artifacts to download). This is the core delivery -- users get the Go binary on their machines for the first time.
**Delivers:** `bin/lib/binary-downloader.js` (platform detection, download, checksum verify, atomic install), modified `performGlobalInstall()` in `bin/cli.js`, `~/.aether/bin/` creation, PATH management, version recording in `~/.aether/version.json`
**Addresses:** FEATURES P1 (binary download during install), ARCHITECTURE Integration Point 2
**Avoids:** PITFALLS #3 (non-atomic replacement), #1 (half-updated state -- binary verified before anything depends on it)

**Concrete work items:**
- Create `bin/lib/binary-downloader.js` with platform detection (process.platform + process.arch mapping, Rosetta check via sysctl)
- Implement atomic download: fetch to `.new`, verify checksum, verify executable, rename
- Add `tar` to package.json dependencies (promote from overrides)
- Modify `performGlobalInstall()` to call binary downloader after hub setup
- Implement PATH management (idempotent profile injection for `~/.aether/bin/`)
- Update `~/.aether/version.json` schema with binaryVersion, binaryPlatform, binaryInstalledAt, binaryChecksum
- Update `.npmignore` to exclude Go source files

### Phase 3: Update Flow Binary Refresh
**Rationale:** Depends on Phase 2 (needs binary-downloader.js). Keeps the binary current across npm updates. Can run in parallel with Phase 4.
**Delivers:** `bin/lib/version-gate.js`, modified update command handler, non-blocking binary refresh after file sync
**Addresses:** FEATURES P1 (binary download during update), ARCHITECTURE Integration Point 3
**Avoids:** PITFALLS #4 (version drift -- update compares binary vs npm version)

**Concrete work items:**
- Create `bin/lib/version-gate.js` with custom semver comparison (no npm dependency)
- Modify update command handler to check binary version after file sync
- Non-blocking download: warn on failure, never block the update
- Version comparison: if binary < npm, download new binary; if binary > npm, warn about drift

### Phase 4: npm Shim Delegation + Version Gate
**Rationale:** Depends on Phase 2 (needs version gate logic). This is the final piece that routes YAML commands through the Go binary safely. Can run in parallel with Phase 3.
**Delivers:** Modified `bin/run.js` or `bin/cli.js` entry point that delegates to Go binary when version gate passes, falls back to Node.js CLI when it does not. The 87 YAML files and 275 Go calls require zero changes -- they already call `aether <subcommand>`.
**Addresses:** FEATURES P1 (version gate logic, npm shim delegation), ARCHITECTURE Integration Point 4
**Avoids:** PITFALLS #1 (half-updated state -- wiring only happens when binary confirmed), #2 (PATH collision -- shim delegates, does not compete)

**Concrete work items:**
- Implement shim delegation: check `~/.aether/bin/aether` exists and is executable, run `aether version`, compare against `MIN_BINARY_VERSION`
- If version gate passes: `spawn(binaryPath, args, { stdio: 'inherit' })`
- If version gate fails: fall back to existing Node.js CLI implementation
- Binary search order: (1) hub `~/.aether/bin/aether`, (2) PATH `aether`
- Add pre-flight binary check to critical YAML commands (init.yaml, build.yaml, continue.yaml)
- Update `package.json` bin entry if needed

### Phase Ordering Rationale

- Phase 1 is independent and zero-risk -- pure CI addition, no user-facing changes
- Phase 2 depends on Phase 1 -- needs a GitHub Release with downloadable binaries
- Phases 3 and 4 both depend on Phase 2 only -- they can run in parallel
- Phase ordering avoids the two most dangerous pitfalls: half-updated state (binary and YAML never change in the same step) and PATH collision (npm shim delegates to binary, does not compete with it)

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2:** Binary downloader has platform-specific edge cases (Rosetta detection on macOS, Windows PATH management via `setx`, corporate proxy handling). The research identified these but did not produce implementation-ready code for each platform.
- **Phase 4:** The npm shim delegation pattern has an unresolved question about how to handle commands that only exist in the Node.js CLI (like `install`, `update`, `setupHub`). These must stay in Node.js even when the Go binary is active. A command routing table is needed.

Phases with standard patterns (skip research-phase):
- **Phase 1:** goreleaser + GitHub Actions is a well-documented, widely-used pattern. The `.goreleaser.yml` is already 90% complete. The release workflow is ~30 lines of standard YAML.
- **Phase 3:** Version gate comparison is trivial (string split + numeric compare). The update flow modification is a well-understood enhancement to existing code.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All recommendations grounded in direct codebase inspection. goreleaser config verified against existing `.goreleaser.yml`. Action versions checked against current GitHub releases. Only gap: web search was rate-limited, so goreleaser v2.15.2 version is from training data, not live verification. |
| Features | HIGH | Feature landscape grounded in direct analysis of 87 YAML command files, 11 playbooks, and competitor analysis (esbuild, turbo, biome). P1 features are tightly scoped and have clear implementation paths. |
| Architecture | HIGH | All four integration points identified with specific file locations, line numbers, and data flow diagrams. The key insight (YAML already wired, gap is distribution) is verified by pattern analysis across all command files. |
| Pitfalls | HIGH | 6 critical pitfalls identified from direct codebase inspection of goreleaser.yml, package.json, bin/cli.js, UpdateTransaction class, and 45 YAML command files. Each has concrete failure modes and prevention strategies. |

**Overall confidence:** HIGH

### Gaps to Address

- **goreleaser-action v7 ESM requirement:** The research notes v7 requires Node 24 and ESM, but the exact impact on workflow syntax was not verified against live documentation. Check during Phase 1 planning that the workflow YAML is compatible with v7's ESM migration.

- **macOS Rosetta architecture detection:** The research identifies that `uname -m` reports `x86_64` under Rosetta on Apple Silicon, but the exact `sysctl` command (`sysctl -n sysctl_proc_translated`) needs platform testing. The universal binary approach (single `_darwin_all.tar.gz`) mitigates this by not needing arch detection for macOS at all -- but it needs verification.

- **npm shim command routing table:** The research identifies that some commands (install, update, setupHub) must stay in Node.js while operational commands route to Go. A specific command routing table (which commands stay Node.js vs delegate to Go) needs to be defined during Phase 4 planning.

- **GitHub API rate limits for binary download:** The research notes that unauthenticated GitHub API access is rate-limited to 60 requests/hour. GitHub Releases CDN serves assets without rate limits for public repos, so this is likely a non-issue, but it should be confirmed during Phase 2.

- **Windows PATH management:** The research mentions `setx` for adding `~/.aether/bin` to PATH on Windows but does not provide implementation details. Windows users are a secondary audience but the download script must handle `win32` platform correctly.

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection: `.goreleaser.yml`, `package.json`, `bin/cli.js`, `bin/lib/update-transaction.js`, `cmd/root.go`, `cmd/version.go`, `cmd/aether/main.go`, `Makefile`, `.npmignore`, `.github/workflows/ci.yml`
- Direct analysis: 87 YAML command files (133 `aether <subcommand>` calls), 11 playbook files (275 Go binary calls), 4 YAML files with 11 remaining `bash .aether/...` calls
- PROJECT.md v5.5 scope definition and milestone history
- Existing git tag history (v1.1.2 through v5.4) confirming tag-based release convention

### Secondary (MEDIUM confidence)
- goreleaser documentation (training data -- HIGH confidence given config is already written and working)
- goreleaser-action v7, actions/setup-go v6, actions/checkout v6 GitHub release pages (version numbers verified via training data, not live API calls)
- npm binary distribution patterns from esbuild, turbo, biome, prisma (well-established patterns from training data)
- `tar` npm package v7.5.13 API (already in project overrides)

### Tertiary (LOW confidence)
- Homebrew tap distribution via goreleaser `brews` section -- deferred to v5.6+, not researched in depth
- macOS Gatekeeper behavior for unsigned binaries -- acceptable for v5.5 with `xattr -cr` workaround
- Windows SmartScreen behavior for unsigned binaries -- acceptable for v5.5

---
*Research completed: 2026-04-04*
*Ready for roadmap: yes*
