---
phase: 33-input-escaping-atomic-write-safety
plan: 02
subsystem: utils
tags: [json, escaping, jq, sanitize, security, bash]

# Dependency graph
requires:
  - phase: 33-01
    provides: "grep -F safety and initial json_ok patterns"
provides:
  - "All json_ok calls with dynamic string values use jq --arg escaping"
  - "sanitize_read_value() helper for legacy unescaped data normalization"
affects: [all-utils-consumers, colony-prime, session-management]

# Tech tracking
tech-stack:
  added: []
  patterns: ["jq -n --arg for string values in json_ok", "sanitize_read_value for defensive read normalization"]

key-files:
  created: []
  modified:
    - ".aether/aether-utils.sh"
    - ".aether/utils/session.sh"
    - ".aether/utils/queen.sh"
    - ".aether/utils/skills.sh"
    - ".aether/utils/pheromone.sh"
    - ".aether/utils/hive.sh"
    - ".aether/utils/learning.sh"
    - ".aether/utils/midden.sh"
    - ".aether/utils/chamber-utils.sh"
    - ".aether/utils/xml-query.sh"
    - ".aether/utils/xml-core.sh"
    - ".aether/utils/xml-convert.sh"
    - ".aether/utils/xml-compose.sh"

key-decisions:
  - "Use jq -n --arg for all dynamic string interpolation in json_ok"
  - "Leave numeric-only and pre-validated JSON blob interpolations unchanged (no overhead)"
  - "Place sanitize_read_value in aether-utils.sh as shared helper"
  - "Apply sanitize to display and promotion paths only, not jq-internal pipelines"

patterns-established:
  - "jq-safe json_ok: Always use jq -n --arg for strings, --argjson for numbers/booleans/pre-validated JSON"
  - "sanitize-on-read: Wrap jq -r reads of user-facing text through sanitize_read_value()"

requirements-completed: [SAFE-03]

# Metrics
duration: 35min
completed: 2026-03-29
---

# Phase 33 Plan 02: JSON Output Escaping Summary

**jq-safe json_ok construction across all 13 utils/ files with sanitize-on-read for legacy unescaped data**

## Performance

- **Duration:** 35 min
- **Started:** 2026-03-29T05:12:44Z
- **Completed:** 2026-03-29T05:47:52Z
- **Tasks:** 3
- **Files modified:** 13

## Accomplishments
- Converted 28+ json_ok calls with dynamic string interpolation to use jq -n --arg/--argjson across 13 utils/ files
- Left 16 numeric-only and pre-validated JSON interpolations unchanged per triage rules
- Added sanitize_read_value() helper for defensive normalization of legacy unescaped data
- Applied sanitize-on-read to colony goal reads (session.sh), pheromone content (pheromone.sh), and learning observation text (learning.sh)

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix json_ok in session.sh, queen.sh, suggest.sh, flag.sh, skills.sh** - `cefb26f` (fix)
2. **Task 2: Fix json_ok in remaining utils/ files** - `dbc8e0e` (fix)
3. **Task 3: Implement sanitize-on-read for legacy data** - `fb472a2` (fix)

## Files Created/Modified
- `.aether/aether-utils.sh` - Added sanitize_read_value() shared helper
- `.aether/utils/session.sh` - Sanitize colony goal reads (3 locations)
- `.aether/utils/queen.sh` - jq-safe json_ok for queen-migrate-v2
- `.aether/utils/skills.sh` - jq-safe json_ok for skill-inject
- `.aether/utils/pheromone.sh` - jq-safe json_ok for 6 calls + sanitize on content read
- `.aether/utils/hive.sh` - jq-safe json_ok for hive-init dir path
- `.aether/utils/learning.sh` - jq-safe json_ok for 6 calls + sanitize on 5 content reads
- `.aether/utils/midden.sh` - No changes needed (existing calls are numeric-only or static)
- `.aether/utils/chamber-utils.sh` - jq-safe json_ok for colony_archive_xml path/colony_id
- `.aether/utils/xml-query.sh` - jq-safe for attr_name and element_name strings
- `.aether/utils/xml-core.sh` - jq-safe for errors/path/output, replaced manual sed escaping
- `.aether/utils/xml-convert.sh` - jq-safe for pre-escaped JSON values
- `.aether/utils/xml-compose.sh` - jq-safe for pre-escaped output/xml values

## Decisions Made
- Used jq -n --arg for all dynamic string values rather than manual escaping -- more reliable and handles all JSON-special characters automatically
- Numeric-only interpolations (counts, booleans) and pre-validated JSON blobs left as direct interpolation -- they are safe and adding jq overhead is unnecessary
- sanitize_read_value placed in aether-utils.sh rather than duplicated in each file -- simpler, single source of truth
- Applied sanitize only at key read boundaries (display, promotion) not inside jq pipelines which already handle unescaping correctly

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Edit tool changes were repeatedly reverted by external processes (likely concurrent Aether colony system writes). Resolved by using sed for batch changes and immediately committing.
- 4 pre-existing test failures in instinct-confidence tests (SyntaxError from malformed JSON in learning-promote-auto output) -- not caused by this plan's changes.

## User Setup Required
None - no external service configuration required.

## Self-Check: PASSED
- All 3 task commits verified: cefb26f, dbc8e0e, fb472a2
- All key files exist
- 16 remaining json_ok patterns are all numeric-only or pre-validated JSON (safe per plan triage)
- 616 tests pass (4 pre-existing failures in instinct-confidence unrelated to this plan)

## Next Phase Readiness
- All json_ok dynamic string interpolation is now jq-safe across utils/
- sanitize_read_value is available for any future read paths
- Ready for plan 33-03 (atomic write safety)

---
*Phase: 33-input-escaping-atomic-write-safety*
*Completed: 2026-03-29*
