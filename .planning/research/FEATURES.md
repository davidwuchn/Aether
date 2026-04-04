# Feature Research: Go Binary Release and Distribution (v5.5)

**Domain:** Binary release, auto-install, and version-gated YAML wiring for an existing npm-distributed Go CLI tool
**Researched:** 2026-04-04
**Confidence:** MEDIUM-HIGH (direct codebase analysis of existing goreleaser.yml, npm distribution, update flow, and YAML wiring system; goreleaser and binary distribution patterns are well-established domain knowledge; web search rate-limited so no external verification)

---

## Context

### Current State

Aether is distributed as an npm package (`aether-colony`) that installs a Node.js CLI (`bin/cli.js`) which manages hub setup, command sync, and repo updates. The v5.4 milestone added a Go binary (`cmd/aether/main.go`) with 254+ Cobra commands that replaces the shell-based `aether-utils.sh` dispatcher. The Go binary is currently built via `make build` but is NOT distributed through npm or GitHub Releases.

The YAML command system works as follows:
1. 87 YAML source files in `.aether/commands/*.yaml` define slash commands
2. `bin/generate-commands.js` converts YAML to markdown for both Claude Code (`.claude/commands/ant/*.md`) and OpenCode (`.opencode/commands/ant/*.md`)
3. The generated markdown contains `aether <subcommand>` calls that execute via the Go binary (or fall back to shell)
4. 11 playbook files in `.aether/docs/command-playbooks/` follow the same pattern

### What v5.5 Changes

Three interconnected features:
1. **goreleaser releases** -- produce downloadable binaries for darwin/linux/windows, amd64/arm64
2. **Binary auto-install on update** -- `aether update` downloads the binary if missing from PATH
3. **Version-gated YAML wiring** -- only swap Go-wired YAML commands when the binary is confirmed working

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist for a CLI tool with a Go binary. Missing these = the binary release feels broken or unusable.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Cross-platform binary builds | Users on macOS (Intel + Apple Silicon), Linux, Windows expect `aether` to work on their machine | LOW | goreleaser config already exists with the right goos/goarch matrix. The `.goreleaser.yml` is 90% complete -- needs release pipeline and possibly archive tweaks. |
| GitHub Release with checksums | Users expect verifiable downloads; security-conscious users check checksums | LOW | goreleaser generates checksums automatically. Just needs `checksum.name_template` which is already set. |
| Tag-triggered release workflow | npm packages publish on tag push; Go releases should follow the same pattern | LOW | Add goreleaser step to CI or create a separate release workflow triggered on `v*` tags. Existing tags follow semver-ish pattern (v5.4, v5.3.3). |
| Binary downloads the correct platform binary | Users should not need to know their OS/arch; the tool handles it | MEDIUM | Need a download script (Node.js in `postinstall` or a standalone script) that detects `process.platform` + `process.arch` and downloads from GitHub Releases. Pattern used by esbuild, turbo, biome, etc. |
| Version command works on binary | `aether version` must report the correct version, not "0.0.0-dev" | LOW | Already implemented via ldflags: `-X github.com/aether-colony/aether/cmd.Version={{.Version}}` in goreleaser.yml. Just needs to be verified in the release pipeline. |
| Existing npm install still works | Current users doing `npm install -g aether-colony` must not break | MEDIUM | npm package stays as-is. The Go binary is an addition, not a replacement. The `bin/cli.js` remains the npm entry point. Binary gets downloaded during `aether install` or `aether update`. |
| Update command remains functional | `aether update` must still sync hub files, plus now handle binary | MEDIUM | The update command in `bin/cli.js` (line 1427) already does transactional updates. Binary download adds a step after the existing sync logic. |

### Differentiators (Competitive Advantage)

Features that make the binary distribution smooth and professional. Not strictly required, but valuable for adoption and reliability.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Version-gated YAML wiring | Prevents broken commands if binary download fails; only switches YAML to Go-mode after confirmed working binary | MEDIUM-HIGH | This is the key innovation. Each YAML command currently calls `aether <subcommand>`. If the binary is missing, these calls fail silently (the `2>/dev/null || true` pattern). Version gating means: only generate YAML that calls the Go binary once the binary is installed and its version matches the expected version. Before that, fall back to shell. Implementation: track binary version in hub `version.json`, regenerate YAML only when binary version >= minimum. |
| Atomic binary swap | Binary update never leaves a broken intermediate state; old binary works until new one is verified | MEDIUM | Download new binary to temp file, verify checksum, then `mv` (atomic on same filesystem). Pattern: download to `~/.aether/bin/aether.new`, verify, rename to `~/.aether/bin/aether`. |
| Checksum verification on download | Users can trust the binary is what was published | LOW | goreleaser publishes checksums.txt. Download script fetches checksums.txt, verifies SHA256 of downloaded binary. ~20 lines of Node.js. |
| Homebrew tap distribution | macOS users can `brew install aether-colony/tap/aether` | MEDIUM | goreleaser has native `brews` support. Requires a separate `homebrew-tap` repository. Deferrable -- npm + binary download covers this initially. |
| Binary self-update | `aether update` can update itself without npm | HIGH | The binary checks GitHub Releases API for latest version, downloads replacement. Complex edge cases (in-place replacement, permissions, PATH management). Defer to v5.6+. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems for this specific project.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Replace npm package entirely with binary-only distribution | Simpler distribution model, one artifact | Breaks existing users; npm is how Claude Code and OpenCode discover and install tools; npm handles hub setup (commands, agents, system files); the Go binary only handles CLI subcommands, not the full install flow | Keep npm as the primary distribution channel. Binary is an optimization inside npm, not a replacement. |
| Bundle binary inside npm package | Simpler postinstall, no runtime download | npm package size balloons (6 platforms x 2 archs = 12 binaries x ~15MB = ~180MB); most users download 11 unused binaries; npm install becomes slow | Download correct binary on-demand during `aether install` or `aether update`. Only the matching platform binary is fetched (~15MB). |
| Auto-wire YAML to Go binary on every update | Ensures latest commands always use Go binary | If binary download fails or is corrupted, ALL 87 commands break simultaneously; no fallback; user sees confusing errors across every slash command | Version-gate the wiring. Only switch YAML to Go-mode after binary is confirmed working. Keep shell fallback YAML available. |
| Sign binaries with GPG/code-signing certificates | Professional release hygiene | Requires paid Apple Developer certificate ($99/year), Windows code signing ($200+/year), key management complexity; goreleaser supports it but the setup is non-trivial for a solo developer | Start with checksum verification only. Add code signing later if users request it. goreleaser makes adding signing incremental. |
| Nightly/snapshot builds | Bleeding-edge users get latest fixes | Adds CI complexity, creates support burden (users report bugs on unreleased versions), tag-based releases are sufficient for current cadence | Use `goreleaser release --snapshot` locally for testing. Ship tagged releases when ready. |

---

## Feature Dependencies

```
[GitHub Release Pipeline]
    |
    v
[Binary Download Script]
    |
    v
[Binary Install on Update] ──requires──> [GitHub Release Pipeline]
    |                                         (needs release artifacts to download)
    |
    v
[Version Check in Hub]
    |
    v
[Version-Gated YAML Wiring] ──requires──> [Binary Install on Update]
    |                                          (needs installed binary to verify)
    |                                     ──requires──> [Binary Download Script]
    |                                          (needs download capability)
    v
[YAML Regeneration with Go-mode] ──enhances──> [Version-Gated YAML Wiring]
    |                                               (generates correct YAML for current state)
    v
[Full End-to-End: install -> update -> commands work]

[Homebrew Tap] ──independent──> [GitHub Release Pipeline]
                                  (needs releases, but can be added anytime after)

[Binary Self-Update] ──conflicts──> [npm-based update flow]
                                    (two update paths create confusion;
                                     defer until npm is deprecated)
```

### Dependency Notes

- **Binary Download Script requires GitHub Release Pipeline:** The download script fetches binaries from GitHub Releases. Without published releases, there is nothing to download. The release pipeline must be working first.
- **Version-Gated YAML Wiring requires Binary Install on Update:** The wiring decision (shell vs Go) depends on whether a working binary exists on the system. The install-on-update feature must reliably place the binary before wiring can switch.
- **Binary Self-Update conflicts with npm-based update flow:** Having two independent update mechanisms (npm `aether update` and binary self-update) creates version drift and confusion. The npm flow remains authoritative for now; the binary is installed and managed by the npm package, not independently.
- **Homebrew Tap is independent:** It only needs GitHub Releases to exist. Can be added at any point after the release pipeline works. Does not block or depend on any other feature.
- **YAML Regeneration enhances Version-Gated YAML Wiring:** The existing `generate-commands.js` already generates YAML-based markdown. The enhancement adds a mode flag: generate with `aether <subcommand>` calls (Go mode) or with `bash .aether/aether-utils.sh <subcommand>` calls (shell mode). The version gate determines which mode to use.

---

## MVP Definition

### Launch With (v5.5)

Minimum viable product -- what is needed to ship working binary distribution.

- [ ] **goreleaser release pipeline** -- Tag-triggered CI workflow producing 6 platform binaries (darwin/linux/windows x amd64/arm64) uploaded to GitHub Releases with checksums. The `.goreleaser.yml` is already 90% configured.
- [ ] **Binary download during `aether install`** -- The `performGlobalInstall()` function in `bin/cli.js` (line 1322) gains a step that downloads the correct platform binary from GitHub Releases to `~/.aether/bin/aether` and adds it to PATH (or verifies it is on PATH).
- [ ] **Binary download during `aether update`** -- The `updateRepo()` function (line 1207) checks if the Go binary is present and current. If missing or outdated, downloads the matching version from the GitHub Release corresponding to the npm package version.
- [ ] **Version gate logic** -- A function that checks: (a) binary exists at `~/.aether/bin/aether` or on PATH, (b) `aether version` returns the expected version, (c) binary is executable. Returns boolean for "safe to wire Go mode."
- [ ] **Conditional YAML generation** -- `generate-commands.js` gains a `--go-mode` flag. When set, generates commands with `aether <subcommand>` calls. When not set, generates commands with `bash .aether/aether-utils.sh <subcommand>` calls (current behavior). The `aether install` and `aether update` commands check the version gate and regenerate YAML accordingly.

### Add After Validation (v5.6)

Features to add once core binary distribution is working.

- [ ] **Homebrew tap** -- Add `brews` section to `.goreleaser.yml` and create a `homebrew-tap` repository. Trigger: when macOS users ask for brew install.
- [ ] **Checksum verification on download** -- Fetch `checksums.txt` from release, verify SHA256 of downloaded binary. Trigger: first security review or user request.
- [ ] **Atomic binary swap with backup** -- Keep previous binary as `~/.aether/bin/aether.bak`, allow rollback. Trigger: if binary update ever breaks users.

### Future Consideration (v6+)

Features to defer until binary distribution is mature.

- [ ] **Binary self-update** -- `aether update-self` checks latest release and replaces in-place. Defer: npm flow is sufficient; self-update adds complexity without clear benefit until npm is deprecated.
- [ ] **Code signing** -- Apple Developer + Windows code signing. Defer: cost/benefit does not justify for current user base.
- [ ] **Remove shell fallback entirely** -- Once binary is stable and widely deployed, remove `aether-utils.sh` dependency. Defer: shell fallback is insurance against binary issues.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| goreleaser release pipeline | HIGH -- enables all other features | LOW -- goreleaser config is 90% done, CI workflow is ~30 lines | P1 |
| Binary download during install | HIGH -- binary actually gets on user machines | MEDIUM -- platform detection, download logic, PATH setup (~100-150 lines Node.js) | P1 |
| Binary download during update | HIGH -- keeps binary current across versions | MEDIUM -- extends existing update flow, same download logic as install | P1 |
| Version gate logic | HIGH -- prevents broken commands, enables safe YAML wiring | LOW -- boolean check function (~30 lines) | P1 |
| Conditional YAML generation | HIGH -- switches commands to Go binary when safe | MEDIUM -- modify generate-commands.js to support two modes, add regeneration step to install/update | P1 |
| Checksum verification | MEDIUM -- security best practice | LOW -- SHA256 verification is ~20 lines | P2 |
| Homebrew tap | LOW-MEDIUM -- convenience for macOS users | MEDIUM -- requires separate repo, formula maintenance | P2 |
| Atomic binary swap with backup | MEDIUM -- reliability improvement | LOW -- rename + backup is ~10 lines | P2 |
| Binary self-update | LOW -- npm handles updates | HIGH -- in-place replacement edge cases, permission issues | P3 |
| Code signing | LOW -- trust signal for enterprise users | HIGH -- certificate management, CI complexity | P3 |
| Remove shell fallback | LOW -- cleanup, no user-facing change | MEDIUM -- requires confidence in binary stability | P3 |

**Priority key:**
- P1: Must have for launch -- these are the five features that constitute the v5.5 milestone
- P2: Should have, add when possible -- quick wins after core is working
- P3: Nice to have, future consideration -- defer to later milestones

---

## Competitor Feature Analysis

| Feature | esbuild | Turborepo | Biome | Aether (v5.5 plan) |
|---------|---------|-----------|-------|---------------------|
| Binary distribution via npm postinstall | Yes -- downloads platform binary | Yes -- platform-specific optional deps | Yes -- downloads platform binary | Yes -- download during `aether install` |
| Cross-platform builds | Yes (10+ platforms) | Yes (6 platforms) | Yes (6 platforms) | Yes (6 platforms: darwin/linux/windows x amd64/arm64) |
| Checksum verification | Yes | Partial (npm handles) | Yes | Yes (P2) |
| Fallback if binary missing | Compiles from source (slow) | Platform-specific deps graceful fail | Shows error | Shell fallback (existing aether-utils.sh) |
| Self-update | No (use npm) | No (use npm) | No (use npm) | No (use npm -- defer to v6+) |
| goreleaser | No (custom) | No (custom) | No (custom) | Yes -- goreleaser for release pipeline |
| Version-gated wiring | N/A | N/A | N/A | Yes -- unique to Aether; only switch to Go-mode after binary confirmed |

### Key Insight

Aether's version-gated YAML wiring is genuinely novel among these tools. Most Go/Rust CLI tools distributed via npm use a simple binary-or-bust approach. Aether's dual-mode YAML (shell vs Go) with a safety gate is more resilient. The shell fallback means a failed binary download does not break the user's workflow -- they just get the slower shell-based commands until the binary is available.

---

## Existing Infrastructure Dependencies

These are the existing systems that v5.5 features depend on.

### npm Distribution (`bin/cli.js`)
- `performGlobalInstall()` (line 1322) -- handles `aether install`, syncs commands/agents to hub
- `updateRepo()` (line 1207) -- handles `aether update`, transactional with rollback
- `setupHub()` -- creates `~/.aether/system/` with all system files
- **Binary download will be added as a step inside `performGlobalInstall()` and `updateRepo()`**

### YAML Command System (`bin/generate-commands.js`)
- Reads 87 YAML files from `.aether/commands/`
- Generates markdown for Claude Code and OpenCode
- `{{TOOL_PREFIX}}` template generates platform-specific run instructions
- **Conditional YAML generation will add a `--go-mode` flag that changes the `aether <subcommand>` prefix behavior**

### goreleaser Configuration (`.goreleaser.yml`)
- Already configured for 6 platforms (darwin/linux/windows, amd64/arm64)
- ldflags set for version injection
- CGO_ENABLED=0 for static binaries
- **Needs: release workflow, possibly `extra_files` or archive adjustments**

### Version Management
- `cmd/root.go` has `var Version = "0.0.0-dev"` (overridden by ldflags at build)
- `package.json` has `"version": "5.3.3"` (npm package version)
- `~/.aether/version.json` tracks hub version
- **Need: alignment between npm package version, Go binary version, and GitHub Release tag**

---

## Sources

- Direct codebase analysis: `.goreleaser.yml`, `bin/cli.js`, `bin/generate-commands.js`, `cmd/version.go`, `cmd/aether/main.go`, `.github/workflows/ci.yml`, `package.json`, `.npmignore`, `.claude/commands/ant/update.md`, `.aether/commands/status.yaml`
- goreleaser documentation: https://goreleaser.com/ (training data, MEDIUM confidence -- not verified against current docs due to search rate limit)
- npm binary download patterns: esbuild, turbo, biome, prisma postinstall scripts (training data, MEDIUM confidence -- well-established patterns)
- Git tag history: existing tags v1.1.2 through v5.4 confirm tag-based release convention

---
*Feature research for: Go Binary Release and Distribution (v5.5)*
*Researched: 2026-04-04*
