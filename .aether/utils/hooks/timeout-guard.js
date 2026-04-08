#!/usr/bin/env node
// Timeout Guard PreToolUse Hook
//
// Runs before every tool call in Claude Code when active.
// Checks if the current agent has exceeded its spawn timeout.
// If timed out, blocks the tool call and forces escalation.
//
// Exit codes:
//   0 - Allow the operation (not timed out or no tracking)
//   2 - Block the operation (timeout exceeded)
//
// Design principle: fail-open. If tracking data is unavailable or
// parsing fails, allow the operation. The goal is preventing runaway
// agents, not blocking all work.

const { execSync } = require('child_process');

const CHECK_TIMEOUT = 3000; // ms

let input = '';
const stdinTimeout = setTimeout(() => process.exit(0), CHECK_TIMEOUT);
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => input += chunk);
process.stdin.on('end', () => {
  clearTimeout(stdinTimeout);
  try {
    const data = JSON.parse(input);

    // Extract agent context from environment or tool input
    // Claude Code passes the session context but not directly the agent name
    // We check if there's a spawn-track.json with an active agent
    const cwd = data.cwd || process.cwd();

    // Try to read spawn track
    const fs = require('fs');
    const path = require('path');
    const trackFile = path.join(cwd, '.aether', 'data', 'spawn-track.json');

    if (!fs.existsSync(trackFile)) {
      process.exit(0); // No tracking file = no enforcement
    }

    const trackData = JSON.parse(fs.readFileSync(trackFile, 'utf8'));

    // Check if timeout is set and exceeded
    if (!trackData.timeout || trackData.timeout <= 0) {
      process.exit(0); // No timeout configured
    }

    const elapsed = Math.floor(Date.now() / 1000) - trackData.start;
    if (elapsed <= trackData.timeout) {
      process.exit(0); // Within timeout
    }

    // TIMEOUT EXCEEDED — block the operation
    const output = {
      decision: 'block',
      reason: `Agent "${trackData.agent}" exceeded timeout of ${trackData.timeout}s ` +
        `(elapsed: ${elapsed}s). Escalate to Queen or clear with: ` +
        `aether spawn-track --action clear`
    };
    process.stderr.write(JSON.stringify(output) + '\n');
    process.exit(2);
  } catch (e) {
    // Fail-open on any error
    process.exit(0);
  }
});
