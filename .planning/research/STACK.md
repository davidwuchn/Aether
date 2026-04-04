# Technology Stack: Go Binary Release and Distribution

**Project:** Aether v5.5 Go Binary Release
**Researched:** 2026-04-04
**Scope:** goreleaser binary releases, binary install on update, version-gated YAML wiring
**Predecessor:** `.planning/research/STACK-GO-SHIPPING.md` (v5.4 era -- superseded by this document)

---

## Executive Summary

Aether v5.4 shipped 254+ Go commands via Cobra CLI. The Go binary works locally (`make build` produces `./aether`). Now the goal is: (1) publish cross-platform binaries via goreleaser on GitHub Releases, (2) make `aether update` download and install the Go binary automatically, and (3) only wire YAML commands to the Go binary once it is confirmed present and working.

The stack additions are minimal: goreleaser v2 for release automation, a Node.js binary downloader as npm postinstall, and a version-gate check in the npm CLI wrapper. No new Go dependencies are needed -- version comparison is trivial with semver strings and stdlib.

**Critical correction from prior research:** goreleaser-action is now at v7 (not v6 as STACK-GO-SHIPPING.md stated). setup-go is at v6 (not v5). actions/checkout is at v6 (not v4). The prior research had stale action versions.

---

## Recommended Stack

### Release Automation

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| goreleaser | v2.15.2 | Cross-platform binary build + release | Industry standard for Go releases. `.goreleaser.yml` already exists with basic config. Handles cross-compilation, checksums, archives, Homebrew taps, and changelogs in one tool. |
| goreleaser/goreleaser-action | v7.0.0 | Run goreleaser in GitHub Actions | Official GitHub Action. v7 (released 2026-02-21) requires Node 24 and ESM. Must use v7 since GitHub Actions runners have Node 24+. |

### GitHub Actions (Updated Versions)

| Action | Version | Purpose | Why |
|--------|---------|---------|-----|
| actions/checkout | v6.0.2 | Clone repository in CI | Latest stable. Prior research referenced v4 which is outdated. |
| actions/setup-go | v6.4.0 | Install Go in CI | Latest stable with `cache: true` support. Prior research referenced v5. |
| goreleaser/goreleaser-action | v7.0.0 | Run goreleaser in CI | Latest stable. Breaking change from v6: requires ESM/Node 24. |

### Binary Download (npm Integration)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Node.js `https` module | stdlib | HTTP download from GitHub Releases | Zero dependencies. Follows redirects for GitHub release downloads. |
| `tar` npm package | v7.5.13 | Extract `.tar.gz` archives | Already in `package.json` overrides at v7.5.11. Must promote to direct dependency. Supports `tar.x()` stream extraction. Requires Node >=18 (already the constraint from `engines`). |
| Node.js `crypto` module | stdlib | SHA-256 checksum verification | Verify downloaded binary against goreleaser `checksums.txt`. Zero dependencies. |
| Node.js `child_process` module | stdlib | Spawn Go binary from npm wrapper | `spawn(binaryPath, args, { stdio: 'inherit' })` -- zero overhead process proxy. |

### Version Gating (No New Dependencies)

| Technology | Purpose | Why |
|------------|---------|-----|
| Node.js semver comparison (custom) | Check binary version >= required version | Simple string split + numeric compare. No need for the `semver` npm package. A 10-line function is sufficient. |
| Go `Version` ldflags | Embed version in binary at build time | Already implemented in `cmd/root.go` via `-X github.com/aether-colony/aether/cmd.Version={{.Version}}`. Already working. |
| `aether version` command | Runtime version check | Already implemented in `cmd/version.go`. Outputs the version string. Already working. |

### Homebrew Distribution (Deferred -- Not v5.5)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| goreleaser `brews` section | v2 | Auto-publish Homebrew formula | goreleaser pushes formula to `calcosmic/homebrew-tap` repo automatically. Requires creating that repo once. **Defer until npm download is working.** Not needed for v5.5. |

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Release automation | goreleaser v2 | Manual `go build` + `gh release create` | goreleaser handles 6 platform targets, checksums, archives, and changelogs automatically. Manual approach is error-prone and tedious per release. |
| Release automation | goreleaser v2 | nfpm (deb/rpm/apk) | Aether is a developer CLI, not a system package. npm + direct download covers the audience. |
| Binary download | postinstall script | Platform-specific npm packages (`aether-darwin-arm64`, etc.) | Requires publishing 5+ packages per release. Turborepo and esbuild do this at massive scale. Aether does not need this complexity. Single package + download is simpler. |
| Binary download | postinstall script | Bundle binaries in npm package | 6 platform binaries would be ~30-50MB. npm install becomes slow for all users. Download-on-demand means only the needed platform binary is fetched. |
| Binary download | postinstall script | Compile from source during npm install | Requires Go toolchain on the user's machine. Most Aether users are AI-assisted developers, not Go developers. |
| Binary download | `https` stdlib | `node-fetch` or `got` | `https.get()` with redirect following handles GitHub release downloads fine. No need for a dependency for a single GET request with redirects. |
| Version gate | Custom semver compare function | `semver` npm package | Aether only needs "is X >= Y" for dotted version strings. A 10-line function does this. The `semver` package (v7.x) adds 30KB of node_modules for a trivial comparison. |
| Version gate | Custom semver compare function | `hashicorp/go-version` (Go side) | The version gate runs in Node.js (the npm wrapper), not in Go. No Go-side dependency needed. |

---

## What NOT to Use

| Avoid | Why |
|-------|-----|
| Cosign / Fulcio / SBOM | Aether is not in supply-chain-sensitive environments. `checksums.txt` from goreleaser is sufficient for current scale. |
| Docker image distribution | Aether runs in the user's terminal alongside Claude Code. Docker makes no sense for a CLI tool. |
| Snap packages | Linux-specific, requires Snapcraft account. npm covers Linux users. |
| nfpm (deb/rpm) | Developer CLI, not a system package. No one installs CLI dev tools via apt. |
| `semver` npm package | 30KB dependency for what a 10-line function handles. Overkill. |
| `node-fetch` / `got` / `axios` | `https` stdlib handles GitHub release downloads with redirect following. Zero dependencies. |
| goreleaser-action v6 | v7 is the current release (2026-02-21). v6 is not getting updates. v7's breaking change is internal (ESM migration) -- workflow usage is identical. |

---

## Detailed Configuration

### 1. goreleaser.yml (Updated from existing)

The existing `.goreleaser.yml` needs these additions: binary id, strip flags, universal binaries for macOS, and archive files list.

**Current state** (verified 2026-04-04):
```yaml
version: 2
before:
  hooks:
    - go mod tidy
builds:
  - main: ./cmd/aether/
    ldflags:
      - -X github.com/aether-colony/aether/cmd.Version={{.Version}}
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
```

**Required changes:**
- Add `id: aether` and `binary: aether` to the build (required for `universal_binaries` reference)
- Add `-s -w` to ldflags (strip debug info, 20-30% size reduction -- the binary is a CLI tool, no one debugs it with gdb)
- Add `universal_binaries` section for macOS fat binary (one download works on Intel and Apple Silicon)
- Add `ignore` for `windows/arm64` (poor Go toolchain support, negligible user base)
- Add `files` to archives section to include LICENSE

No Homebrew tap in v5.5 -- defer to a later milestone. The core goal is npm binary download first.

### 2. GitHub Actions: Add Release Workflow

Create a new `.github/workflows/release.yml` file. Do NOT modify the existing `ci.yml` -- the CI workflow stays as-is for PR testing. The release workflow triggers only on version tags.

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
          cache: true
      - run: go test -race -count=1 ./...
      - run: go vet ./...
      - run: go build -ldflags "-X github.com/aether-colony/aether/cmd.Version=${{ github.ref_name }}" ./cmd/aether/
      - run: ./aether version

  release:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
          cache: true
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Why separate release.yml, not added to ci.yml:** The release workflow only triggers on tags. CI triggers on PRs and pushes to main. Mixing them creates conditional job complexity. Separate files are cleaner and easier to debug.

**Why goreleaser-action v7 not v6:** v7 is the current release (2026-02-21). v6 is not getting updates. v7's breaking change is internal (ESM migration) -- usage in workflows is identical to v6.

**Why actions/setup-go v6:** Latest release (2026-03-30). Supports `cache: true` for Go module caching. Prior research referenced v5 which is outdated.

### 3. npm Binary Download Script

Two new files in `bin/`:

**bin/download-binary.js** -- Downloads platform-correct binary from GitHub Releases.

Key design decisions:
- Use `https` stdlib with redirect following (GitHub Releases uses 302 redirects to CDN)
- Use `tar` package (v7.5.13) for `.tar.gz` extraction, `zlib.createGunzip()` for decompression
- macOS universal binary: download `_darwin_all.tar.gz` regardless of arch
- Linux/Windows: download `_linux_amd64.tar.gz` or `_windows_amd64.zip` etc.
- Checksum verification against `checksums.txt` from the same release
- Non-fatal: if download fails, the Node.js CLI still works
- Idempotent: if binary already exists and version matches, skip download

**bin/run.js** -- Thin wrapper that proxies to Go binary if present, falls back to Node.js CLI.

Key design decisions:
- `spawn(binaryPath, args, { stdio: 'inherit' })` -- zero overhead process proxy
- Version gate: run `aether version` and compare against `MIN_BINARY_VERSION` before routing to binary
- If binary missing or version too old: fall back to `require('./cli.js')`
- Binary search order: (1) local `bin/aether`, (2) hub `~/.aether/bin/aether`, (3) PATH `aether`

**package.json changes:**
```json
{
  "bin": {
    "aether": "bin/run.js",
    "aether-colony": "bin/npx-entry.js"
  },
  "scripts": {
    "postinstall": "node bin/download-binary.js"
  },
  "dependencies": {
    "tar": "^7.5.13"
  }
}
```

The `tar` package moves from `overrides` to `dependencies` since the download script needs it directly.

### 4. Version-Gated YAML Wiring

The version gate lives in `bin/run.js` and works like this:

```
1. Check if Go binary exists at bin/aether (or bin/aether.exe)
2. If exists: spawn `aether version`, capture output
3. Parse version string (strip leading 'v', split by '.', compare numerically)
4. If version >= MIN_BINARY_VERSION: proxy to Go binary
5. If version < MIN_BINARY_VERSION or binary missing: fall back to Node.js CLI
```

`MIN_BINARY_VERSION` is a constant in `bin/run.js` set to `"5.5.0"` -- the first release that includes the Go binary.

This means:
- Existing users with v5.4 npm package: YAML commands still route through Node.js CLI (no behavior change)
- Users who `npm update` to v5.5+: postinstall downloads the Go binary, YAML commands route through it
- Users on air-gapped networks: Go binary download fails silently, Node.js CLI still works
- If the Go binary is somehow corrupted: version check fails, Node.js CLI still works

The YAML command files (`.claude/commands/ant/*.md`) do NOT need changes. They already call `aether <subcommand>` which resolves to the `bin` entry point in `package.json`. The `run.js` wrapper handles routing.

### 5. Update Flow: `aether update` Enhances to Download Binary

The existing `aether update` command in `bin/cli.js` syncs system files from hub to repo. For v5.5, it gains an additional step:

```
After syncing system files:
1. Check if Go binary exists in ~/.aether/bin/ (hub-level binary)
2. If not: download from GitHub Releases using same logic as postinstall
3. Verify binary runs and version matches
4. Report binary status in update summary
```

This is an enhancement to the existing `performGlobalInstall()` function in `bin/cli.js`, not a new command. The install flow becomes:

```
performGlobalInstall():
  1. Sync commands (existing)
  2. Sync agents (existing)
  3. Setup hub (existing)
  4. Download Go binary to ~/.aether/bin/aether (NEW)
  5. Verify binary runs and version matches (NEW)
```

The binary lives at `~/.aether/bin/aether` (hub level) so all repos share one binary. The `bin/run.js` wrapper checks this location as a fallback if the local `bin/aether` does not exist.

---

## .npmignore Updates

Add these to `.npmignore` to exclude Go source code from the npm package (the binary is downloaded, not shipped as source):

```
# Go source (binary is downloaded, not shipped)
cmd/
pkg/
go.mod
go.sum
.goreleaser.yml
Makefile
*.exe
aether
```

This reduces npm package size significantly. The Go binary is ~15-25MB compiled; the source code is unnecessary in the npm package since the binary is downloaded at install time.

---

## Installation (After v5.5)

```bash
# Option 1: npm (existing users, seamless upgrade)
npm install -g aether-colony
# postinstall downloads Go binary automatically; falls back to Node.js CLI

# Option 2: Direct download from GitHub Releases
# Download from https://github.com/calcosmic/Aether/releases/latest
# Extract and place on PATH

# Option 3: go install (Go developers)
go install github.com/aether-colony/aether/cmd/aether@latest
```

---

## Sources

- `.goreleaser.yml` -- Existing config, verified 2026-04-04. Uses `version: 2` schema. (HIGH confidence)
- `go.mod` -- Go 1.26.1 with Cobra v1.10.2, go-pretty v6.7.8. (HIGH confidence)
- `package.json` -- npm package `aether-colony` v5.3.3, bin entries, `tar` in overrides. (HIGH confidence)
- `.github/workflows/ci.yml` -- Existing CI pipeline with Go test job already present. (HIGH confidence)
- `cmd/root.go`, `cmd/version.go` -- Version injection via ldflags, `aether version` command. (HIGH confidence)
- `bin/cli.js` -- Existing install/update commands, `performGlobalInstall()`. (HIGH confidence)
- `.npmignore` -- Current exclusion rules, no Go source exclusions. (HIGH confidence)
- goreleaser GitHub releases API -- goreleaser v2.15.2 (latest, 2026-03-31). (HIGH confidence)
- goreleaser-action GitHub releases API -- v7.0.0 (latest, 2026-02-21). Breaking change from v6: ESM/Node 24. (HIGH confidence)
- actions/setup-go GitHub releases API -- v6.4.0 (latest, 2026-03-30). (HIGH confidence)
- actions/checkout GitHub releases API -- v6.0.2 (latest, 2026-01-09). (HIGH confidence)
- npm registry API -- `tar` package v7.5.13 (latest), requires Node >=18. (HIGH confidence)
- `bin/npx-entry.js`, `bin/npx-install.js` -- Entry point routing, verified unchanged. (HIGH confidence)
- YAML commands (`.claude/commands/ant/*.md`) -- 133 `aether <subcommand>` calls across 42 files, already calling `aether` directly. (HIGH confidence)

---

*Stack research for: Aether v5.5 Go Binary Release and Distribution*
*Researched: 2026-04-04*
