# Phase 3: Build Depth Controls - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 03-build-depth-controls
**Areas discussed:** Depth levels, Token budget scaling, Depth persistence model, Depth validation enforcement

---

## Depth levels

| Option | Description | Selected |
|--------|-------------|----------|
| 3 levels (match requirements) | Keep light/standard/deep only. Remove 'full' from code. Move chaos gating to 'deep'. | |
| 4 levels (keep full) | Keep all 4 levels. Update requirements to note 'full' as extra level. | ✓ |
| You decide | Pick whichever makes sense. | |

**User's choice:** 4 levels (keep full)
**Notes:** The existing playbook depth checks already work with 4 levels. Keeping 'full' means no restructuring of the chaos gating that's currently on 'full'.

---

## Token budget scaling

| Option | Description | Selected |
|--------|-------------|----------|
| Linear (4/8/12/16K) | Light 4K, Standard 8K, Deep 12K, Full 16K. Linear scaling. | |
| Progressive (4/8/16/24K) | Light 4K, Standard 8K, Deep 16K, Full 24K. Non-linear — deeper gets disproportionately more. | ✓ |
| Light-reduced only (4/8/8/8K) | Only light gets reduced budget. Standard+ all use 8K. | |
| You decide | Pick the right scaling model. | |

**User's choice:** Progressive (4/8/16/24K)
**Notes:** Deeper builds spawn oracle + architect who need richer context, so non-linear scaling makes sense. Skills budget also scales but less aggressively.

---

## Depth persistence model

| Option | Description | Selected |
|--------|-------------|----------|
| Persistent (current) | /ant:build --depth persists the setting. Simple, already implemented. | ✓ |
| Per-build override | --depth only affects that one build. More intuitive for one-off thorough builds. | |
| You decide | Pick whichever is simpler. | |

**User's choice:** You decide → Claude chose Persistent
**Notes:** Persistent is simpler, already implemented in playbooks, and aligns with how other CLI tools work (git config pattern). /ant:init --depth light gives users a clean way to set depth at colony creation.

---

## Depth validation enforcement

| Option | Description | Selected |
|--------|-------------|----------|
| Go enum type (Recommended) | Create ColonyDepth type with constants. Compile-time safety. | ✓ |
| Validation function | Keep string field, add validation before writes. Simpler but less safe. | |
| You decide | Pick the right approach. | |

**User's choice:** Go enum type
**Notes:** Follows the existing State type pattern in pkg/colony/colony.go. Prevents invalid values from state-mutate at compile time.

---

## Claude's Discretion

- Depth persistence model — Claude chose "Persistent" based on simplicity and existing implementation
- Token budget exact values, implementation approach, error messages, backward compatibility handling

## Deferred Ideas

None.
