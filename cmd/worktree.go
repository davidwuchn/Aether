package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// worktreeBaseDir is the base directory for managed worktrees.
const worktreeBaseDir = ".aether/worktrees"

// agentBranchRe matches agent-track branch names: phase-N/caste-name
// where N is a positive integer (1-9 followed by any digits) and caste-name
// is lowercase alphanumeric with hyphens.
var agentBranchRe = regexp.MustCompile(`^phase-[1-9]\d*/[a-z0-9-]+$`)

// humanBranchPrefixes lists allowed prefixes for human-track branch names.
var humanBranchPrefixes = []string{"feature/", "fix/", "experiment/", "colony/"}

// ---------------------------------------------------------------------------
// validateBranchName
// ---------------------------------------------------------------------------

// validateBranchName checks that a branch name follows the two-track naming convention.
// Agent track: phase-N/caste-name (e.g., "phase-2/builder-1")
// Human track: prefix/description where prefix is one of the allowed values.
func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid branch name \"\": branch name is required")
	}

	// Reject path traversal
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid branch name %q: path traversal detected", name)
	}

	// Agent track: phase-N/caste-name
	if agentBranchRe.MatchString(name) {
		return nil
	}

	// Human track: prefix/description
	for _, prefix := range humanBranchPrefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			return nil
		}
	}

	return fmt.Errorf("invalid branch name %q: must match phase-N/name or use prefix/name (prefix: feature, fix, experiment, colony)", name)
}

// ---------------------------------------------------------------------------
// sanitizeBranchPath
// ---------------------------------------------------------------------------

// sanitizeBranchPath converts a branch name to a safe filesystem path by
// replacing slashes with hyphens. This prevents directory structure issues
// with branch names like "phase-2/builder-1".
func sanitizeBranchPath(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

// ---------------------------------------------------------------------------
// generateWorktreeID
// ---------------------------------------------------------------------------

// generateWorktreeID produces a unique worktree identifier.
// Format: wt_<unix_timestamp>_<random_hex>
func generateWorktreeID() string {
	rnd := make([]byte, 4)
	rand.Read(rnd)
	return fmt.Sprintf("wt_%d_%s", time.Now().Unix(), hex.EncodeToString(rnd))
}

// ---------------------------------------------------------------------------
// isWorktreeOrphaned
// ---------------------------------------------------------------------------

// isWorktreeOrphaned checks whether a worktree's last commit time exceeds the
// given threshold duration from now. A zero commitAt is treated as orphaned
// (no commits ever made).
func isWorktreeOrphaned(commitAt time.Time, threshold time.Duration) bool {
	if commitAt.IsZero() {
		return true
	}
	return time.Since(commitAt) > threshold
}

// ---------------------------------------------------------------------------
// getLastCommitTime
// ---------------------------------------------------------------------------

// getLastCommitTime retrieves the timestamp of the most recent commit in a worktree.
func getLastCommitTime(worktreePath string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", worktreePath,
		"log", "-1", "--format=%ct").Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("git log: %w", err)
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse commit timestamp: %w", err)
	}
	return time.Unix(ts, 0), nil
}

// ---------------------------------------------------------------------------
// worktree-allocate command
// ---------------------------------------------------------------------------

var worktreeAllocateCmd = &cobra.Command{
	Use:   "worktree-allocate",
	Short: "Create worktree with enforced naming and state tracking",
	Long: "Creates a git worktree with branch name validation and lifecycle tracking. " +
		"Use --agent/--phase for agent-track names (phase-N/name) or --branch for human-track names.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		agentName, _ := cmd.Flags().GetString("agent")
		phaseNum, _ := cmd.Flags().GetInt("phase")
		branchName, _ := cmd.Flags().GetString("branch")

		// Determine branch name
		var branch string
		if agentName != "" && phaseNum > 0 {
			branch = fmt.Sprintf("phase-%d/%s", phaseNum, agentName)
		} else if branchName != "" {
			branch = branchName
		} else {
			outputError(1, "either --agent/--phase or --branch is required", nil)
			return nil
		}

		// Validate branch name
		if err := validateBranchName(branch); err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		// Check for duplicate in COLONY_STATE.json
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			// If state file doesn't exist, that's ok for allocate
			if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "read") {
				outputError(2, fmt.Sprintf("failed to load colony state: %v", err), nil)
				return nil
			}
		}

		// Check for existing non-merged worktree with same branch
		for _, wt := range state.Worktrees {
			if wt.Branch == branch && wt.Status != colony.WorktreeMerged {
				outputError(1, fmt.Sprintf("worktree %q already tracked with status %q", branch, wt.Status), nil)
				return nil
			}
		}

		// Create worktree directory path
		sanitized := sanitizeBranchPath(branch)
		worktreePath := worktreeBaseDir + "/" + sanitized

		// Create parent directory
		if err := os.MkdirAll(worktreeBaseDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create worktree directory: %v", err), nil)
			return nil
		}

		// Create git worktree: try creating new branch first, then reuse existing
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()

		addCmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", branch, worktreePath, "HEAD")
		if out, err := addCmd.CombinedOutput(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git worktree add timed out after %v", GitTimeout), nil)
				return nil
			}
			// Branch may already exist; try reusing it
			reuseCmd := exec.CommandContext(ctx, "git", "worktree", "add", worktreePath, branch)
			if out2, err2 := reuseCmd.CombinedOutput(); err2 != nil {
				outputError(2, fmt.Sprintf("failed to create worktree: %v: %s (also tried reusing branch: %v: %s)",
					err, string(out), err2, string(out2)), nil)
				return nil
			}
		}

		// Register in COLONY_STATE.json
		now := time.Now().UTC().Format(time.RFC3339)
		entry := colony.WorktreeEntry{
			ID:        generateWorktreeID(),
			Branch:    branch,
			Path:      worktreePath,
			Status:    colony.WorktreeAllocated,
			Phase:     phaseNum,
			Agent:     agentName,
			CreatedAt: now,
			UpdatedAt: now,
		}

		state.Worktrees = append(state.Worktrees, entry)

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			// Rollback: remove the worktree
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), GitTimeout)
			defer rollbackCancel()
			exec.CommandContext(rollbackCtx, "git", "worktree", "remove", worktreePath, "--force").Run()
			exec.CommandContext(rollbackCtx, "git", "worktree", "prune").Run()
			outputError(2, fmt.Sprintf("failed to save colony state: %v (worktree rolled back)", err), nil)
			return nil
		}

		// Audit log
		auditEntry := map[string]interface{}{
			"action":    "worktree-allocate",
			"branch":    branch,
			"path":      worktreePath,
			"id":        entry.ID,
			"timestamp": now,
		}
		store.AppendJSONL("state-changelog.jsonl", auditEntry)

		outputOK(map[string]interface{}{
			"ok":      true,
			"branch":  branch,
			"path":    worktreePath,
			"id":      entry.ID,
			"status":  string(entry.Status),
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// worktree-list command
// ---------------------------------------------------------------------------

var worktreeListCmd = &cobra.Command{
	Use:   "worktree-list",
	Short: "List tracked worktrees with lifecycle status",
	Long: "Shows all tracked worktrees from COLONY_STATE.json with on-disk cross-reference. " +
		"Use --status to filter by a specific status.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		statusFilter, _ := cmd.Flags().GetString("status")

		// Load colony state
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			// No state file means no worktrees
			outputOK(map[string]interface{}{
				"worktrees": []interface{}{},
			})
			return nil
		}

		// Get on-disk worktrees via git
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		gitOut, err := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain").Output()
		onDiskPaths := map[string]bool{}
		if err == nil {
			for _, p := range parseWorktreePaths(string(gitOut)) {
				onDiskPaths[p] = true
			}
		}

		// Build output with on_disk cross-reference
		type worktreeOutput struct {
			colony.WorktreeEntry
			OnDisk bool `json:"on_disk"`
		}

		var result []worktreeOutput
		for _, wt := range state.Worktrees {
			// Apply status filter
			if statusFilter != "" && string(wt.Status) != statusFilter {
				continue
			}

			// Check if on disk (check both absolute and relative path)
			onDisk := false
			if _, err := os.Stat(wt.Path); err == nil {
				onDisk = true
			} else if onDiskPaths[wt.Path] {
				onDisk = true
			}

			result = append(result, worktreeOutput{
				WorktreeEntry: wt,
				OnDisk:        onDisk,
			})
		}

		if result == nil {
			result = []worktreeOutput{}
		}

		outputOK(map[string]interface{}{
			"worktrees": result,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// worktree-orphan-scan command
// ---------------------------------------------------------------------------

var worktreeOrphanScanCmd = &cobra.Command{
	Use:   "worktree-orphan-scan",
	Short: "Detect orphaned or stale worktrees",
	Long: "Scans tracked worktrees for staleness (no commits in threshold hours) " +
		"and detects untracked worktrees on disk. Updates status to 'orphaned' for stale worktrees.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		thresholdHours, _ := cmd.Flags().GetInt("threshold")
		if thresholdHours <= 0 {
			thresholdHours = 48
		}
		threshold := time.Duration(thresholdHours) * time.Hour

		// Load colony state
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputOK(map[string]interface{}{
				"orphaned":  []interface{}{},
				"stale":     []interface{}{},
				"untracked": []interface{}{},
				"threshold": thresholdHours,
			})
			return nil
		}

		// Get on-disk worktrees
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		gitOut, err := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain").Output()
		onDiskPaths := map[string]bool{}
		if err == nil {
			for _, p := range parseWorktreePaths(string(gitOut)) {
				onDiskPaths[p] = true
			}
		}

		now := time.Now().UTC().Format(time.RFC3339)
		var orphaned []colony.WorktreeEntry
		var stale []colony.WorktreeEntry
		stateChanged := false

		// Scan tracked worktrees
		for i, wt := range state.Worktrees {
			// Skip already-merged worktrees
			if wt.Status == colony.WorktreeMerged {
				continue
			}

			// Check if on disk
			onDisk := false
			if _, err := os.Stat(wt.Path); err == nil {
				onDisk = true
			}

			if !onDisk {
				// Worktree in state but not on disk = stale state entry
				stale = append(stale, wt)
				continue
			}

			// Check last commit time
			var lastCommit time.Time
			commitTime, err := getLastCommitTime(wt.Path)
			if err != nil {
				// No commits or git error; use creation time as fallback
				createdAt, parseErr := time.Parse(time.RFC3339, wt.CreatedAt)
				if parseErr == nil {
					lastCommit = createdAt
				} else {
					lastCommit = time.Time{} // treat as orphaned
				}
			} else {
				lastCommit = commitTime
				state.Worktrees[i].LastCommitAt = commitTime.UTC().Format(time.RFC3339)
			}

			if isWorktreeOrphaned(lastCommit, threshold) {
				state.Worktrees[i].Status = colony.WorktreeOrphaned
				state.Worktrees[i].UpdatedAt = now
				orphaned = append(orphaned, wt)
				stateChanged = true

				// Audit log
				store.AppendJSONL("state-changelog.jsonl", map[string]interface{}{
					"action":    "worktree-orphaned",
					"branch":    wt.Branch,
					"path":      wt.Path,
					"id":        wt.ID,
					"timestamp": now,
				})
			}
		}

		// Find untracked worktrees (on disk but not in state)
		trackedPaths := map[string]bool{}
		for _, wt := range state.Worktrees {
			trackedPaths[wt.Path] = true
		}

		var untracked []map[string]interface{}
		for path := range onDiskPaths {
			if !trackedPaths[path] {
				untracked = append(untracked, map[string]interface{}{
					"path": path,
				})
			}
		}

		// Save state if any worktrees were marked as orphaned
		if stateChanged {
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				outputError(2, fmt.Sprintf("failed to save colony state: %v", err), nil)
				return nil
			}
		}

		if orphaned == nil {
			orphaned = []colony.WorktreeEntry{}
		}
		if stale == nil {
			stale = []colony.WorktreeEntry{}
		}
		if untracked == nil {
			untracked = []map[string]interface{}{}
		}

		outputOK(map[string]interface{}{
			"orphaned":  orphaned,
			"stale":     stale,
			"untracked": untracked,
			"threshold": thresholdHours,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// init: register commands
// ---------------------------------------------------------------------------

func init() {
	worktreeAllocateCmd.Flags().String("agent", "", "Agent name (for agent track: phase-N/name)")
	worktreeAllocateCmd.Flags().Int("phase", 0, "Phase number (for agent track)")
	worktreeAllocateCmd.Flags().String("branch", "", "Branch name (for human track)")

	worktreeListCmd.Flags().String("status", "", "Filter by status (allocated, in-progress, merged, orphaned)")

	worktreeOrphanScanCmd.Flags().Int("threshold", 48, "Staleness threshold in hours (default: 48)")

	for _, c := range []*cobra.Command{
		worktreeAllocateCmd, worktreeListCmd, worktreeOrphanScanCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
