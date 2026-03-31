# Phase 44: Release Hygiene & Ship - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-31
**Phase:** 44-release-hygiene-ship
**Areas discussed:** Version strategy, Test gap, Package cleanliness, Ship process, NPX flow, End-to-end smoke test, Regression check vs v2.5, README, CLAUDE.md

---

## Version Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Bump to 5.3.0 | Keep npm semver lineage (5.x). Minor bump since v2.7 adds features. | ✓ |
| Reset to 2.7.0 | Align npm version with project version. Breaking for 5.x users. | |
| You decide | Let Claude pick whichever makes sense. | |

**User's choice:** Bump to 5.3.0
**Notes:** CLAUDE.md says v2.7.0 but package.json stays in 5.x lineage.

---

## Test Gap

| Option | Description | Selected |
|--------|-------------|----------|
| Ship at 509 | All passing, zero failures. Don't block on count. | |
| Add tests for new v2.7 code | Write tests for new modules. ~560-580 target. | |
| Hit 620+ before shipping | Original target. More thorough but delays release. | |

**User's choice:** (Free text) The most important thing is end-to-end reliability, not a test count. v2.5 was the quality benchmark — everything worked seamlessly. Focus on making the system work well, not metrics.
**Notes:** User emphasized: context prompts, colony not getting confused, emojis and style, fun to build with.

---

## Package Cleanliness

| Option | Description | Selected |
|--------|-------------|----------|
| Exclude all flagged items | Add all 6 categories to .npmignore. Saves ~120KB. | |
| Exclude dev scripts only | Remove scripts, keep design docs. | |
| You decide | Claude makes the call per item. | |

**User's choice:** (Free text) Extensively review every file. Don't delete anything needed. Check everything thoroughly.
**Notes:** Spawned deep audit agent. Found 8 files safe to exclude, 3 must keep. ~120KB savings.

---

## Ship Process

| Option | Description | Selected |
|--------|-------------|----------|
| npm publish | Publish aether-colony@5.3.0 to npm. | ✓ |
| Fix website install cmd | Website says 'npx aether init' but package is 'aether-colony'. | |
| GitHub release | Tagged release with changelog. | ✓ |
| CHANGELOG update | Update with v2.7 changes before publish. | ✓ |

**User's choice:** npm publish + GitHub release + CHANGELOG update. User will update website text separately.

---

## NPX Install Flow

| Option | Description | Selected |
|--------|-------------|----------|
| Test and fix npx flow | Ensure `npx aether-colony` works end-to-end cleanly. | ✓ |
| It already works, just ship | Don't overthink, focus on publish. | |
| You decide | Check and fix what needs fixing. | |

**User's choice:** Test and fix npx flow.

---

## End-to-End Smoke Test

| Option | Description | Selected |
|--------|-------------|----------|
| Full lifecycle | npx install → init → plan → build → continue → seal. Full journey. | ✓ |
| Happy path only | Install → init → build one phase. Minimum viable. | |
| You decide scope | Define minimum viable smoke test. | |

**User's choice:** Full lifecycle.

---

## Regression Check vs v2.5

| Option | Description | Selected |
|--------|-------------|----------|
| It just felt smoother | Can't pinpoint specifics. Test same workflows. | ✓ |
| Specific issues I've seen | Describe what's felt off. | |
| Compare outputs side by side | Run commands on v2.5 and current, compare. | |

**User's choice:** It just felt smoother. Key issues: colony confusion bug (thinking all phases done on fresh init), context clarity, emoji/text quality.

---

## Colony Confusion Bug

| Option | Description | Selected |
|--------|-------------|----------|
| Fix it in this phase | Include the fix if needed. | |
| It's a known issue, separate fix | Track but don't block release. | |
| Verify it first | Test whether it still happens. Fix if yes, move on if no. | ✓ |

**User's choice:** Verify first, might already be fixed.

---

## README

| Option | Description | Selected |
|--------|-------------|----------|
| Match website vibe | Clean, simple, fun, not corporate. | |
| Technical and comprehensive | Full feature list, architecture, all agents. | |
| Keep what's there | Current README is fine, just update versions. | |

**User's choice:** (Free text) Match website vibe AND be technically comprehensive. Keep badges (npm, license, stars, sponsor). Keep sponsor section. Professional and fun. User will provide logo to swap header image later.

---

## CLAUDE.md

| Option | Description | Selected |
|--------|-------------|----------|
| Version bump + counts update | Quick pass on version and stats. | |
| Full accuracy audit | Every section verified against reality. | ✓ |
| You decide | Fix what's wrong without asking. | |

**User's choice:** Full accuracy audit.

---

## Claude's Discretion

- Exact order of release steps
- validate-package.sh pre-publish strategy
- GitHub release notes format
- .npmignore pattern strategy for design docs
- Smoke test implementation approach

## Deferred Ideas

- Website install command text update — user handling separately
- Logo swap in README — user will provide file
- REQUIREMENTS.md missing v2.7 requirement IDs — future maintenance
