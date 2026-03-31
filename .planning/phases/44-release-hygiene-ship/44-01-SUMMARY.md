---
phase: 44-release-hygiene-ship
plan: 01
subsystem: infra
tags: [npm, npmignore, packaging, npx-install]

# Dependency graph
requires: []
provides:
  - "Clean npm package excluding 8 dev-only files via .npmignore patterns"
  - "Fixed npx-install.js agent source from .aether/agents-claude/ instead of .opencode/agents/"
affects: [44-02, 44-03]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Wildcard .npmignore patterns for dev-file categories"]

key-files:
  created: []
  modified:
    - ".aether/.npmignore"
    - "bin/npx-install.js"

key-decisions:
  - "Used wildcard patterns (scripts/, docs/*-design.md, docs/plans/) instead of listing each file individually"
  - "Left unused OPENCODE_COMMANDS_SRC constant in npx-install.js as harmless and potentially useful for future OpenCode install step"

patterns-established:
  - "Dev scripts excluded via directory-level .npmignore pattern rather than per-file"

requirements-completed: [REL-01, REL-02, TEST-02]

# Metrics
duration: 2min
completed: 2026-03-31
---

# Phase 44 Plan 01: Package Hygiene Summary

**Excluded 8 dev-only files from npm tarball via .npmignore patterns and fixed npx-install.js to copy Claude agents from the correct source directory**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-31T06:03:46Z
- **Completed:** 2026-03-31T06:05:21Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added 4 exclusion patterns to .aether/.npmignore covering all 8 dev-only files (scripts/, docs/*-design.md, docs/plans/, schemas/example-prompt-builder.xml)
- Fixed npx-install.js to source Claude agents from .aether/agents-claude/ instead of incorrectly using .opencode/agents/
- Package file count reduced from 364 to 352 (12 fewer files)
- validate-package.sh passes, 509 tests pass with zero failures

## Task Commits

Each task was committed atomically:

1. **Task 1: Add dev-file exclusions to .aether/.npmignore and fix npx-install.js agent source** - `11d29ee` (feat)
2. **Task 2: Run validate-package.sh and npm test to confirm package health** - verification-only, no new commit

## Files Created/Modified
- `.aether/.npmignore` - Added 4 exclusion patterns for dev scripts, design docs, plans, and example schema
- `bin/npx-install.js` - Changed OPENCODE_AGENTS_SRC to CLAUDE_AGENTS_SRC pointing at .aether/agents-claude/

## Decisions Made
- Used wildcard patterns (e.g., `docs/*-design.md`) instead of listing individual files -- fewer lines, catches future additions
- Left the unused `OPENCODE_COMMANDS_SRC` constant as-is per plan instructions -- harmless, may be used later

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Package is clean of dev artifacts and ready for version bump and publish steps in Plan 02/03
- npx-install.js agent copying bug fixed, ready for end-to-end smoke test verification

## Self-Check: PASSED

- FOUND: .aether/.npmignore
- FOUND: bin/npx-install.js
- FOUND: 44-01-SUMMARY.md
- FOUND: commit 11d29ee

---
*Phase: 44-release-hygiene-ship*
*Completed: 2026-03-31*
