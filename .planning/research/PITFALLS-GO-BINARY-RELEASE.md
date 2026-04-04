# Pitfalls Research: Go Binary Release and Distribution

**Domain:** Adding goreleaser binary release, binary auto-install on update, and version-gated YAML wiring to an existing npm-distributed CLI tool (aether-colony)
**Researched:** 2026-04-04
**Confidence:** HIGH -- grounded in direct codebase inspection of goreleaser.yml, package.json, bin/cli.js update flow, UpdateTransaction class, 45 YAML command files, CI workflow, .npmignore, and Makefile

---

## Critical Pitfalls

### Pitfall 1: The Half-Updated State (Binary Installed but YAML Still Points to Shell)

**What goes wrong:**
The update flow installs the Go binary but simultaneously swaps the YAML commands to call `aether <subcommand>` instead of `bash .aether/aether-utils.sh <subcommand>`. If the binary download fails or the binary is corrupted, the YAML commands now call a non-existent or broken binary. The user runs `/ant:status` and gets a silent failure or a shell error instead of colony status.

**Why it happens:**
The temptation is to make the update atomic -- install binary AND wire YAML in one step. But these are two different failure domains: network download (binary) can fail independently of file sync (YAML). The current YAML commands already call `aether <subcommand>` (60 occurrences across 15 files), but those calls work today because the npm package's `bin` field maps `aether` to `bin/cli.js`. When a real Go binary lands on PATH, the name collision creates ambiguity about which `aether` gets called.

**Concrete failure modes:**

1. **Binary download succeeds, chmod fails**: The binary lands in `~/.aether/bin/aether` but isn't executable. YAML commands call `aether`, resolve to the broken binary, get "Permission denied". The npm-installed `aether` (bin/cli.js) is shadowed by the broken binary if PATH prioritizes `~/.aether/bin/` over npm global bin.

2. **Binary for wrong architecture**: The update script detects `darwin` but downloads `amd64` binary on an `arm64` machine (or vice versa). The binary exists, is executable, but crashes immediately with "exec format error". YAML commands now fail silently because they call a binary that crashes.

3. **Checksum mismatch goes unnoticed**: The download completes but the SHA doesn't match. If the code doesn't verify the checksum before wiring, a truncated or corrupted binary gets wired. Commands work for simple operations but fail unpredictably on complex ones.

4. **YAML wired before binary verified**: The update swaps 45 YAML command files to call the Go binary, then the binary fails a smoke test. Now 45 commands are broken and the working shell-based commands are gone. The user cannot even run `aether update` to fix it because that command itself is broken.

**How to avoid:**
Enforce a strict two-phase rollout:

- **Phase 1: Binary only.** `aether update` downloads and installs the Go binary to `~/.aether/bin/aether`. It does NOT change any YAML files. The npm-installed `aether` (bin/cli.js) continues to work as before. The binary coexists but is not called by anything yet. A `aether doctor` command can verify the binary works.

- **Phase 2: YAML wiring (version-gated).** Only after the binary is confirmed working (via a version check: `~/.aether/bin/aether version` matches the npm package version) does the update swap YAML commands. If the binary check fails, the update completes without wiring and the user gets a warning.

The version gate must be: "binary exists AND `aether version` succeeds AND output matches expected version". All three conditions, not just existence.

**Warning signs:**
- A test that only checks `fs.existsSync(binaryPath)` without actually executing the binary
- YAML generation that unconditionally calls `aether` without a fallback
- No `aether doctor` or `aether verify-binary` command in the update flow

**Phase to address:**
Phase 1 (binary install) must include the download + verify gate. Phase 2 (YAML wiring) must check the gate before swapping. The roadmap must have these as separate phases, never combined.

---

### Pitfall 2: PATH Collision Between npm `aether` and Binary `aether`

**What goes wrong:**
The npm package already creates a `aether` command via the `bin` field in package.json (`"aether": "bin/cli.js"`). When the Go binary is also installed as `aether` (to `~/.aether/bin/aether` or similar), there are two executables with the same name. Which one gets called depends on PATH ordering, which varies by shell, OS, and install method.

**Why it happens:**
The current package.json has:
```json
"bin": {
  "aether": "bin/cli.js",
  "aether-colony": "bin/npx-entry.js"
}
```

Users install globally with `npm i -g aether-colony`, which puts `aether` (Node.js wrapper) on PATH. The new Go binary would be installed to a different location (`~/.aether/bin/aether`). If the user's PATH has `~/.aether/bin/` before the npm global bin, the Go binary shadows the Node one. If after, the Node one shadows the Go one. Neither is consistently correct.

**Concrete failure modes:**

1. **npm update breaks binary**: User has working Go binary. Runs `npm update -g aether-colony`. npm overwrites the `aether` shim, and depending on PATH, the old binary is now shadowed. User gets the Node CLI instead of the Go one without realizing.

2. **`npx aether-colony` always uses Node version**: `npx` creates a temporary shim that points to bin/cli.js. It never picks up the Go binary. Users who use npx instead of global install get inconsistent behavior compared to those with the binary.

3. **Different behavior in different terminals**: If `.zshrc` adds `~/.aether/bin/` to PATH but `.bash_profile` doesn't, switching terminals changes which `aether` runs. Debugging becomes a nightmare.

4. **Homebrew or system package manager conflicts**: If a future step adds a Homebrew formula, three `aether` commands compete. The user has no idea which one runs.

**How to avoid:**
Use a clear binary naming and location strategy:

- Install the Go binary to `~/.aether/bin/aether` (hub-scoped, not system-scoped)
- The npm `aether` command (bin/cli.js) should detect the Go binary and delegate to it when available: if `~/.aether/bin/aether` exists and is executable, exec it; otherwise fall back to Node implementation
- This means the npm package becomes a thin wrapper/shim that either delegates to the Go binary or handles the subset of commands it can handle (like `install`, `update`)
- Eventually the npm package's only job is: install the binary, then delegate everything to it

The key insight: the npm `aether` command should NOT be a separate implementation. It should be a router that prefers the Go binary. This eliminates the collision because both paths lead to the same binary.

**Warning signs:**
- Users reporting "aether version" returns different values in different contexts
- Tests that only work when run via `node bin/cli.js` but fail when the Go binary is on PATH
- CI pipeline where npm tests and Go tests give different results for the same command name

**Phase to address:**
Phase 1 (binary install) must include the npm-shim-delegates-to-binary pattern. If the npm command doesn't delegate, Phase 2 (YAML wiring) will be a mess because YAML commands call `aether` which may resolve to either implementation.

---

### Pitfall 3: goreleaser Config Missing Pieces for This Specific Setup

**What goes wrong:**
The current `.goreleaser.yml` is minimal (no homebrew tap, no universal binaries for macOS, no release name template, no extra_files for the shell scripts that still ship). It will produce working binaries, but the release artifacts won't include the supporting files (YAML commands, shell utilities, agent definitions) that the CLI needs to function.

**Why it happens:**
The goreleaser config only builds the Go binary. But `aether-colony` is not just a binary -- it's a binary PLUS 45 YAML command files, agent definitions, skills, templates, shell scripts (for the 11 remaining `bash .aether/` calls), and the update transaction system in Node.js. The goreleaser release will have binaries but no supporting infrastructure.

**Concrete failure modes:**

1. **Binary runs but has no commands to execute**: The Go binary expects YAML command files in `.aether/commands/`. goreleaser doesn't include these. The `aether update` flow syncs them from the npm package. But if the user installs via GitHub release binary only (no npm), they have no commands.

2. **macOS users get wrong architecture binary**: The goreleaser config builds separate `darwin_amd64` and `darwin_arm64` archives. macOS users on Apple Silicon need `arm64`, but if the download script detects `uname -m` incorrectly (reports `x86_64` under Rosetta), they get the wrong binary. No universal binary fallback exists.

3. **Windows binary has no `.exe` extension in archive name**: The config has `format_overrides` for zip, but the binary name inside the archive may not have `.exe`. The `builds` section doesn't set the binary name, so goreleaser uses the directory name. On Windows, the binary needs `.exe` appended.

4. **`go mod tidy` in before hooks can fail**: The `before: hooks: - go mod tidy` runs before every build. If `go.mod` has a dependency that's only in the local cache (not pushed), `go mod tidy` removes it and the build fails during goreleaser but passes locally.

**How to avoid:**
- Add `universal_binaries` section for macOS to produce a single fat binary
- Test the download script under Rosetta (`arch -x86_64`) to verify architecture detection
- Ensure the binary name includes `.exe` on windows via `binary: aether.exe` in the builds config
- Remove `go mod tidy` from goreleaser before hooks -- run it in CI separately before goreleaser
- Plan for a future where the binary is self-contained (embed YAML files with `go:embed`), but don't try it in v5.5 -- that's a separate milestone
- For v5.5, the npm package remains the source of YAML files. The binary is an accelerator, not a replacement

**Warning signs:**
- goreleaser config doesn't mention `universal_binaries`
- Download script uses `uname -m` without checking for Rosetta (`sysctl -n sysctl_proc_translated`)
- CI workflow doesn't run `goreleaser check` or `goreleaser release --snapshot`

**Phase to address:**
Phase 1 (goreleaser setup) must include: universal binaries, `.exe` naming, removing `go mod tidy` from hooks, and adding a goreleaser CI step. The download script (Phase 1 or 2) must handle architecture detection correctly.

---

### Pitfall 4: The Update Flow Downloads a Binary But Cannot Roll Back

**What goes wrong:**
The existing `UpdateTransaction` class implements a two-phase commit for file sync: checkpoint, sync, verify, update version. But downloading a binary is a fundamentally different operation than syncing files. If the binary download corrupts the existing working binary, the update transaction's rollback mechanism (file restore from checkpoint) cannot undo a binary replacement because the binary is not a managed file in the update transaction.

**Why it happens:**
The update transaction works on managed files within the repo's `.aether/` directory. The Go binary will live in `~/.aether/bin/` (hub-level, outside the repo). The current update flow has no concept of hub-level binary management. The `setupHub()` function syncs files to `~/.aether/system/` but doesn't manage binaries.

**Concrete failure modes:**

1. **Download replaces working binary with broken one**: The update downloads a new binary over the existing one. The download is interrupted. The binary file is truncated. The user now has no working `aether` at all. The update transaction can rollback the YAML files, but the binary is already corrupted.

2. **Disk full during download**: The binary is 10-30MB. If the disk is nearly full, the download fails mid-write. The partial file is left in place. Next update attempt sees the file exists (wrong version) and skips download. User is stuck with a corrupted binary.

3. **Old binary deleted before new binary verified**: The update removes the old binary, downloads the new one, and the new one fails verification. The old binary is gone. The user has no working `aether` command.

**How to avoid:**
Implement atomic binary replacement:

1. Download new binary to a temp file (`~/.aether/bin/aether.new`)
2. Verify checksum of the temp file
3. Run `~/.aether/bin/aether.new version` to verify it executes and returns expected version
4. Only if all checks pass: rename `aether` to `aether.old`, rename `aether.new` to `aether`
5. Verify `~/.aether/bin/aether version` works
6. Delete `aether.old`
7. If any step fails, restore `aether.old` to `aether`

This is a different transaction model than the file-sync transaction. It needs its own implementation, not bolted onto `UpdateTransaction`.

**Warning signs:**
- Binary download code that writes directly to the final path
- No temp file or rename-based atomic replacement
- No pre-delete backup of the existing binary
- No verification step between download and activation

**Phase to address:**
Phase 1 (binary install on update) must implement atomic binary replacement. This is the single most important safety mechanism for the entire milestone.

---

### Pitfall 5: Version Drift Between npm Package and Go Binary

**What goes wrong:**
The npm package version (in `package.json`) and the Go binary version (injected via ldflags from goreleaser) can drift apart. The npm package is at `5.3.3`, goreleaser uses git tags for versioning. If a user updates npm but not the binary, or vice versa, the versions mismatch. YAML commands may expect features from one version while the binary is another.

**Why it happens:**
There are two independent versioning channels:
- npm: `package.json` version field, published to npm registry
- goreleaser: git tag (e.g., `v5.4.0`), published to GitHub releases

These can get out of sync during development. The npm package might be at `5.3.3` while the binary is built from a `v5.4.0` tag. Or the npm package gets patched to `5.3.4` but no new binary release is cut.

**Concrete failure modes:**

1. **YAML command uses new flag, binary doesn't have it**: The npm package updates a YAML command to pass `--new-flag` to the Go binary. The user has the old binary. The command fails with "unknown flag: --new-flag". There's no version check before the YAML command executes.

2. **Binary reports wrong version**: The ldflags injection `-X github.com/aether-colony/aether/cmd.Version={{.Version}}` works in goreleaser but not in `make build` (the Makefile uses `$(VERSION)` from package.json). If a developer builds locally with `make build`, the binary reports the npm version. If goreleaser builds from a tag, it reports the tag version. These can differ.

3. **Update loop**: The user runs `aether update`. The npm package is at 5.4.0. The binary download gets 5.3.3 (old release). The version check says "binary is outdated, please update". But the update flow doesn't re-download because the binary already exists. User is stuck in a loop.

**How to avoid:**
- **Single source of truth for version**: The `package.json` version must always match the git tag when cutting a release. The release process should: (1) update package.json version, (2) commit, (3) tag, (4) goreleaser release, (5) npm publish. All from the same commit.
- **Binary self-reports version**: `aether version` must work and return the ldflags-injected value. The npm wrapper must also report version. If they disagree, warn the user.
- **Version gate check**: The update flow must compare the binary version against the npm package version. If binary < npm, download new binary. If binary > npm, warn about drift (user may have a dev build).
- **The Makefile `VERSION` extraction from package.json is correct** for local builds. For goreleaser, the `{{.Version}}` template uses the git tag. Ensure these are always in sync via the release process.

**Warning signs:**
- `make build` produces a binary with a different version than `goreleaser release`
- No `aether version` command in the Go binary (actually there IS one -- cmd/version.go exists)
- Update flow doesn't compare binary version to npm version before proceeding
- CI doesn't fail when package.json version doesn't match the git tag

**Phase to address:**
Phase 1 must establish the version synchronization contract. The release CI workflow must verify version alignment. Phase 2 (YAML wiring) must check version compatibility before swapping.

---

### Pitfall 6: 11 Remaining Shell Script Calls Break When Binary Is Primary

**What goes wrong:**
There are still 11 occurrences of `bash .aether/...` calls across 4 YAML command files (tunnels.yaml, swarm.yaml, init.yaml, watch.yaml). When the Go binary becomes the primary CLI, these shell script calls become a second code path that bypasses the binary entirely. They may break because:
- The shell scripts expect environment variables that the Go binary sets but direct bash calls don't
- The shell scripts read files in formats that the Go binary writes differently
- The shell scripts use utility functions that depend on other shell scripts being sourced first

**Why it happens:**
The v5.4 milestone ported 254 commands to Go but left a handful of utility scripts unported. These are in `tunnels.yaml` (chamber-compare.sh), `swarm.yaml` (swarm-display.sh), `init.yaml` (various setup calls), and `watch.yaml` (watch-spawn-tree.sh, colorize-log.sh). They work today because the shell environment is fully set up when these YAML commands run. If the binary becomes primary but these shell scripts aren't ported, they become fragile edge cases.

**How to avoid:**
- **Option A (recommended for v5.5):** Leave these 11 calls as-is. They call specific utility scripts that don't need the Go binary. The shell scripts are independent utilities that work standalone. Don't port them during the binary release milestone -- that's scope creep.

- **Option B:** Port them to Go before wiring. This is risky because it adds work to the binary release milestone and delays shipping.

The right call is Option A. The 11 remaining shell calls are fine. They're utility scripts, not CLI commands. The binary release milestone should focus on distribution, not completing the port.

**Warning signs:**
- Attempting to port utility scripts as part of the binary release milestone
- CI failures on the 4 affected YAML commands after wiring changes
- Shell scripts that import `aether-utils.sh` expecting the full dispatcher environment

**Phase to address:**
Not addressed in this milestone. The 11 remaining calls are explicitly out of scope. Document them as known shell dependencies for a future milestone.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip checksum verification on binary download | Faster update, simpler code | Corrupted binary silently accepted; random failures in production | Never |
| Write binary directly to final path (no atomic rename) | Simpler download code | Corrupted binary on disk full or network interruption | Never |
| Bundle all platform binaries in npm package | Users get binary without download | npm package balloons from ~2MB to ~100MB+; npm install becomes slow | Never |
| Hardcode GitHub release URL | Works immediately for goreleaser output | Breaks if repo moves, GitHub changes URL format, or enterprise mirrors are needed | Only in first iteration, replace with config |
| Skip universal binary for macOS | Simpler goreleaser config, smaller releases | Apple Silicon users might download wrong arch; Rosetta users definitely get wrong arch | Never for darwin |
| Leave version as string comparison | Quick version gate implementation | Semver comparison is harder than string comparison (5.10.0 > 5.9.0 but "5.10.0" < "5.9.0" lexicographically) | Only if using semver library |
| Use npm postinstall for binary download | Binary installed automatically with npm install | Users who run `npm install --ignore-scripts` get no binary; CI environments often use --ignore-scripts | Acceptable as fallback, not primary mechanism |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| goreleaser + GitHub Actions | Not adding `contents: write` permission to GITHUB_TOKEN in release workflow | Add `permissions: contents: write` to the goreleaser job in the release workflow |
| goreleaser + git tags | Tagging with version that doesn't match package.json | CI step that asserts `git tag version == package.json version` before goreleaser runs |
| Binary download + GitHub API | Using unauthenticated GitHub API (rate limited to 60 requests/hour) | For public repos, 60/hr is usually fine for CLI tool updates. Monitor and add auth if needed |
| Binary + npm bin collision | Installing binary to same location as npm shim | Install binary to `~/.aether/bin/` (hub-scoped), never to npm's global bin directory |
| goreleaser + CGO_ENABLED=0 | Forgetting that some dependencies need CGO (e.g., sqlite3) | Current codebase uses CGO_ENABLED=0 and has no CGO dependencies. Keep it that way. If CGO is needed later, that's a major change |
| macOS binary + codesigning | Distributing unsigned binary | macOS Gatekeeper will block unsigned binaries. Either codesign or instruct users to `xattr -cr` the binary. For v5.5, the `xattr` workaround is acceptable |
| Windows binary + SmartScreen | Distributing unsigned Windows binary | Windows SmartScreen warns about unsigned binaries. For v5.5, acceptable. Add signing in a future milestone |
| Binary download + corporate proxy | Using HTTPS without proxy awareness | Node.js `https` module respects `HTTP_PROXY`/`HTTPS_PROXY` env vars. Ensure the download code doesn't bypass these |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Binary download blocking the update | `aether update` takes 30+ seconds on slow connections | Download binary asynchronously or show progress; make binary download optional (skip if already at correct version) | First user on slow WiFi |
| goreleaser building all 6 platform binaries on every push | CI takes 10+ minutes for the release job | Only run goreleaser on tag pushes, not on every commit to main | CI costs spiral |
| Large binary size (debug symbols included) | Binary is 30-50MB instead of 5-10MB | Add `-s -w` to ldflags to strip debug symbols and DWARF info | User disk space on constrained systems |
| Checksum file fetched on every update check | `aether update` makes HTTP request even when up to date | Cache the latest version locally; only fetch checksum + binary when version actually changed | Corporate networks with slow external access |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Download binary over HTTP | Man-in-the-middle can inject malicious binary | Always use HTTPS for GitHub release URLs |
| Skip checksum verification | Corrupted or tampered binary executes with user privileges | Always verify SHA-256 checksum from checksums.txt against downloaded binary |
| Don't verify HTTPS certificate | Allows MITM attacks on binary download | Use default TLS verification in Node.js; never set `rejectUnauthorized: false` |
| Store binary in world-writable directory | Another user can replace the binary | Install to `~/.aether/bin/` (user-owned, not world-writable); set mode 0o755 |
| Trust GitHub release artifact without verifying tag signature | Compromised GitHub account can push malicious release | For v5.5, trusting GitHub releases is acceptable. Future: verify git tag signature |
| Download script runs arbitrary code from internet | Supply chain attack | The download script is part of the npm package (reviewed), not fetched dynamically. Keep it that way |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silent binary download failure | User thinks update succeeded; next command fails mysteriously | Show clear success/failure message: "Go binary installed (v5.4.0)" or "Binary download failed -- using Node.js fallback" |
| No progress indicator during binary download | User thinks update is frozen; Ctrl-C interrupts it | Show download progress (percentage or spinner) |
| Error message says "binary" without context | Non-technical user has no idea what "binary" means | Say "Aether performance engine" or "native runtime" instead of "binary" |
| Requiring manual PATH modification | User installs Aether but `aether` command not found | Add `~/.aether/bin/` to PATH automatically via `.zshrc`/`.bashrc` or provide the npm-shim delegation pattern |
| Breaking existing `npx aether-colony` flow | Users who use npx instead of global install lose functionality | Ensure `npx aether-colony` still works (it uses the npm package, which delegates to binary if available) |

## "Looks Done But Isn't" Checklist

- [ ] **Binary download:** Often missing atomic replacement (temp file + rename) -- verify download writes to `.new`, verifies, then renames
- [ ] **Binary permissions:** Often missing `chmod +x` after download -- verify execute bit is set
- [ ] **Architecture detection:** Often misses Rosetta detection on macOS -- verify `sysctl -n sysctl_proc_translated` is checked
- [ ] **Version alignment:** Often missing CI assertion that package.json version == git tag -- verify CI fails on mismatch
- [ ] **goreleaser CI:** Often missing `goreleaser check` step -- verify config is validated in CI
- [ ] **Fallback path:** Often missing "what if binary fails" fallback -- verify npm shim can still function without binary
- [ ] **Windows path:** Often uses forward slashes that break on Windows -- verify path.join() is used, not string concatenation
- [ ] **Checksum verification:** Often downloads checksums.txt but doesn't actually verify against the binary -- verify the SHA-256 comparison is performed
- [ ] **Clean old binary:** Often leaves old binary after update -- verify cleanup of `aether.old` after successful replacement
- [ ] **Universal binary for macOS:** Often produces separate arm64/amd64 but no universal binary -- verify goreleaser `universal_binaries` section

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Corrupted binary after failed download | LOW | Delete `~/.aether/bin/aether`, run `aether update` again. npm shim still works. |
| Wrong architecture binary | LOW | Delete `~/.aether/bin/aether`, manually download correct arch from GitHub releases, or run `aether update --force-binary` |
| Version drift between npm and binary | MEDIUM | Run `aether doctor` to detect drift. Follow recommendation: update npm package or re-download binary. |
| YAML commands calling broken binary (if wired too early) | HIGH | Must restore pre-wiring YAML files from checkpoint. Requires the npm package to ship both Go-wired and shell-wired YAML sets, or a `aether unwire` command to revert. This is why Phase 2 must be separate and gated. |
| npm postinstall skipped (--ignore-scripts) | LOW | Document the manual step: `aether install-binary`. The binary is optional; the npm CLI works without it. |
| goreleaser release fails in CI | MEDIUM | Check git tag format (must start with `v`), check GITHUB_TOKEN permissions, check dirty git state. Re-tag and push. |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Half-updated state (P1) | Phase 1: Binary install includes verify gate; Phase 2: YAML wiring checks gate | Run update on clean install, kill download mid-transfer, verify YAML commands still work |
| PATH collision (P2) | Phase 1: npm shim delegates to binary when available | Run `which aether` and `aether version` after both npm install and binary install; verify same binary runs |
| goreleaser config gaps (P3) | Phase 1: Complete goreleaser.yml with universal binaries, .exe naming, CI step | Run `goreleaser check` in CI; test download on macOS arm64, macOS amd64 (Rosetta), Linux amd64, Windows |
| No binary rollback (P4) | Phase 1: Atomic binary replacement with temp file + rename | Simulate disk full, network interrupt, and wrong checksum during download |
| Version drift (P5) | Phase 1: Version sync in CI; Phase 2: Version gate before wiring | CI step that asserts package.json version matches git tag |
| Shell script calls break (P6) | Out of scope for this milestone | Verify the 4 affected YAML commands still work after wiring changes |
| Corrupted binary (P4 variant) | Phase 1: Checksum verification in download flow | Download binary, modify one byte, run update, verify it rejects the corrupted binary |

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| goreleaser.yml setup | Missing universal_binaries for macOS | Add `universal_binaries` section; test download on both Apple Silicon and Intel Macs |
| Binary download in update flow | Writing directly to final path instead of atomic rename | Implement temp-file-then-rename pattern from Pitfall 4 |
| Architecture detection | Rosetta misreporting as x86_64 | Check `sysctl -n sysctl_proc_translated` in addition to `uname -m` |
| Version gate implementation | String comparison instead of semver comparison | Use a semver comparison function, not `===` or `>` on version strings |
| YAML wiring swap | Swapping all 45 files atomically (some may fail) | Use UpdateTransaction (already exists) for YAML file sync; verify all files after swap |
| npm package shim | Forgetting to handle the case where binary exists but isn't executable | Check both `fs.existsSync` and `fs.constants.X_OK` access before delegating |
| CI release workflow | Not adding a release workflow (only CI exists) | Create `.github/workflows/release.yml` triggered on tag push, separate from `ci.yml` |
| GitHub release URL format | Hardcoding URL that breaks when repo name changes | Use `repository.url` from package.json to construct the download URL dynamically |

## Sources

- Direct codebase inspection: `.goreleaser.yml`, `package.json`, `bin/cli.js`, `bin/lib/update-transaction.js`, `.aether/commands/*.yaml`, `.github/workflows/ci.yml`, `cmd/version.go`, `cmd/root.go`, `Makefile`, `.npmignore`
- goreleaser documentation: https://goreleaser.com/customization/build/ (build configuration)
- goreleaser documentation: https://goreleaser.com/customization/universalbinaries/ (macOS universal binaries)
- goreleaser documentation: https://goreleaser.com/customization/archive/ (archive naming)
- goreleaser CI integration: https://goreleaser.com/ci/actions/ (GitHub Actions setup)
- npm binary distribution patterns: esbuild (per-platform optional dependencies), node-semver (version comparison)
- Training knowledge: goreleaser common mistakes, macOS codesigning requirements, Windows SmartScreen behavior

---
*Pitfalls research for: Go Binary Release and Distribution (v5.5 milestone)*
*Researched: 2026-04-04*
