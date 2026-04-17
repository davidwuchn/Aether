<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:build
description: "🔨🐜🏗️🐜🔨 Build a phase through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS` directly.
- Do not mutate `COLONY_STATE.json`, `constraints.json`, `pheromones.json`, or handoff files manually.
- If the runtime says no colony or no plan exists, relay that exact guidance.
- If docs and runtime disagree, runtime wins.

If `$ARGUMENTS` is empty, show:

```text
Usage: /ant:build <phase_number>
```
