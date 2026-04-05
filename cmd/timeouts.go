package cmd

import "time"

// Timeout constants for different command categories.
// Git operations: 60s (version control, worktree, merge)
// Build verification: 120s (compilation, test runs)
// General commands: 30s (quick checks, config reads)
var (
	GitTimeout     = 60 * time.Second
	BuildTimeout   = 120 * time.Second
	GeneralTimeout = 30 * time.Second
)
