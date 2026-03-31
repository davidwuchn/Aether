# Phase 44: Release Hygiene & Ship - Context

**Gathered:** 2026-03-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Published package is clean of dev artifacts, all tests pass, end-to-end workflow works smoothly, and v2.7.0 ships to npm with a GitHub release. Package cleanliness, NPX install flow, smoke testing, CLAUDE.md accuracy, README polish, and CHANGELOG update are all in scope.

</domain>

<decisions>
## Implementation Decisions

### Version Strategy
- **D-01:** npm version bumps to 5.3.0 (minor bump, keeps 5.x semver lineage). CLAUDE.md project version bumps to v2.7.0. Two version systems, both updated.
- **D-02:** package.json version field changes from "5.2.1" to "5.3.0".

### Test Bar & Quality
- **D-03:** The goal is not a test count — it's end-to-end reliability. The system must work seamlessly like it did in v2.5. 509 tests currently pass with zero failures. Add tests where they serve reliability, not to hit a number.
- **D-04:** Verify the "colony confusion" regression — colonies sometimes think all phases are done on a fresh init. If it still happens, fix it before shipping. If already resolved, move on.

### Package Cleanliness
- **D-05:** Exclude these dev-only files from the package (add to .npmignore):
  - `.aether/scripts/incident-test-add.sh` — dev scaffolding utility, never called at runtime
  - `.aether/scripts/weekly-audit.sh` — manual health check, never called at runtime
  - `.aether/docs/ci-context-assembly-design.md` — internal design doc
  - `.aether/docs/pheromone-propagation-design.md` — internal design doc
  - `.aether/docs/midden-collection-design.md` — internal design doc
  - `.aether/docs/state-contract-design.md` — internal design doc
  - `.aether/docs/plans/pheromone-display-plan.md` — completed plan doc
  - `.aether/schemas/example-prompt-builder.xml` — example file, not a real schema
- **D-06:** Keep `bin/sync-to-runtime.sh` (CI still calls it), `bin/generate-commands.sh`, and `bin/generate-commands.js` (used by `npm run lint:sync`).
- **D-07:** Run `npm pack --dry-run` after exclusions and verify the file list is clean. validate-package.sh must pass with zero warnings.

### NPX Install Flow
- **D-08:** Test and fix the `npx aether-colony` flow end-to-end on a clean environment. The website promotes NPX as the primary install method — it must work flawlessly.
- **D-09:** Current bin setup: `aether` → cli.js, `aether-colony` → npx-install.js. Verify both entry points work correctly.

### Ship Process
- **D-10:** Ship means: npm publish (aether-colony@5.3.0) + GitHub release (tagged) + CHANGELOG update. User will update website text separately.
- **D-11:** CHANGELOG.md must be updated with all v2.7 changes before publishing.

### End-to-End Smoke Test
- **D-12:** Full lifecycle smoke test: npx install → aether init → plan → build → continue → seal. The entire journey must work on a clean machine.
- **D-13:** Specifically verify: context prompts telling user when to clear, colony not getting confused about progress, emojis and text formatting look polished and clear. "Fun to build with."

### Regression Check vs v2.5
- **D-14:** v2.5 was the gold standard for UX smoothness. Key things that worked: clear context prompts, colony never confused about what it's working on, polished emoji usage, structured text output. Test the same workflows and verify no regression.
- **D-15:** The PR workflow features added in v2.7 (clash detection, worktree utils, pheromone propagation, midden collection) must be tested to ensure they integrate cleanly without disrupting the core experience.

### README
- **D-16:** README should match the website vibe — professional, fun, clear. Keep existing badges (npm downloads, license, stars, sponsor). Keep the sponsor section at the bottom. Good explanation of how the system works, what makes it unique, both approachable AND technically comprehensive.
- **D-17:** User will provide a logo to replace the current header image (to match aetherantcolony.com branding). Placeholder: keep current image, swap when logo is provided.

### CLAUDE.md
- **D-18:** Full accuracy audit of CLAUDE.md. Go through every section — agent count, command count, test count, architecture description, version references — and make sure it all matches current reality. Bump from v2.7-dev to v2.7.0.

### Claude's Discretion
- Exact order of release steps (clean → test → bump → publish → tag → release)
- Whether to run validate-package.sh as a pre-publish check or rely on the existing prepublishOnly hook
- How to structure the GitHub release notes (full changelog vs summary)
- Whether to add the design docs to .npmignore individually or via a wildcard pattern
- Smoke test implementation details (script vs manual checklist)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Package & Distribution
- `bin/validate-package.sh` — Pre-packaging validation, required file checks, content-aware scanning
- `bin/npx-install.js` — NPX installer, creates hub at ~/.aether/ with commands and agents
- `bin/cli.js` — Main CLI entry point, all aether commands
- `.npmignore` — Package exclusion rules
- `package.json` — Package metadata, bin entries, files whitelist, scripts

### Release Criteria (from ROADMAP.md Phase 44)
- `.planning/ROADMAP.md` §Phase 44 — Success criteria: pack clean, validate passes, tests pass, install works, CLAUDE.md updated

### Current State
- `CLAUDE.md` — Needs full accuracy audit and version bump to v2.7.0
- `README.md` — Needs polish pass to match website vibe
- `CHANGELOG.md` — Needs v2.7 changes added before publish

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `bin/validate-package.sh` — Already validates required files, .npmignore coverage, and content patterns. prepublishOnly hook runs it automatically.
- `bin/npx-install.js` — Full installer with banner, hub creation, command/agent copying. Already functional.
- `bin/cli.js` — Commander-based CLI with install, update, init, status, and more.

### Established Patterns
- `prepublishOnly` script in package.json runs validate-package.sh before every publish
- `npm run lint:sync` validates command/agent parity across Claude and OpenCode
- `.npmignore` + `files` field in package.json work together for inclusion/exclusion

### Integration Points
- `npm publish` triggers prepublishOnly → validate-package.sh
- `npx aether-colony` runs npx-install.js which copies from package to ~/.aether/, ~/.claude/, ~/.opencode/
- `aether init` (cli.js) initializes a colony in the current repo
- GitHub release creation via `gh release create`

</code_context>

<specifics>
## Specific Ideas

- "It's about ensuring all these things work. It's for a user, they understand what's going on and with the emojis and the way the text is structured and looks good and everything's super clear to them. It's just fun for them to build with it."
- v2.5 is the UX quality benchmark — that's when the system felt seamless
- Colony confusion bug (thinking phases are done on fresh init) must be verified/fixed
- Website at aetherantcolony.com promotes `npx aether init` — user will update this text. Our package uses `npx aether-colony`.
- README should keep sponsor section, badges, and professional appearance. Logo swap coming later.

</specifics>

<deferred>
## Deferred Ideas

- Website text update for install command — user handling separately
- Logo swap in README — user will provide the logo file
- Requirement IDs (REL-01, REL-02, REL-03, TEST-01, TEST-02) referenced in roadmap but not defined in REQUIREMENTS.md — should be added to REQUIREMENTS.md in a future maintenance pass

</deferred>

---

*Phase: 44-release-hygiene-ship*
*Context gathered: 2026-03-31*
