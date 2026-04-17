<!-- Generated from .aether/commands/resume.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:resume
description: "Resume previous session through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether resume` directly.
- This is the quick restore path. For the fuller paused-session recovery view, use `AETHER_OUTPUT_MODE=visual aether resume-colony`.
- Do not reconstruct state manually from `session.json`, `COLONY_STATE.json`, or `.aether/HANDOFF.md`.
- If docs and runtime disagree, runtime wins.
