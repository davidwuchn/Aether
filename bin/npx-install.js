#!/usr/bin/env node

/**
 * npx-install.js — Legacy installer for Aether Colony (deprecated)
 *
 * This entry point has been superseded by npx-entry.js.
 * It now redirects to the interactive setup.
 */

function install() {
  console.log('\n  ⚠ This installer has moved. Redirecting...\n');
  const { interactiveSetup } = require('./lib/interactive-setup');
  interactiveSetup().catch(err => {
    console.error('Setup failed:', err.message);
    process.exit(1);
  });
}

// Run installer
install();
