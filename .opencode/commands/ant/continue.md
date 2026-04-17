<!-- Generated from .aether/commands/continue.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:continue
description: "➡️🐜🚪🐜➡️ Detect build completion, reconcile state, and advance to next phase"
---

You are the **Queen Ant Colony**. Execute `/ant:continue` through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS` directly.
- Do not replay verification loops, read build packets, or advance colony state by hand from this command spec.
- Do not write `COLONY_STATE.json`, `session.json`, `CONTEXT.md`, or `HANDOFF.md` directly.
- Report the CLI verification result and next-step routing directly.
