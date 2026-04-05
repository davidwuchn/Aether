package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Clash detection prevents file conflicts between worktrees.

// --- clash-check ---

var clashCheckCmd = &cobra.Command{
	Use:   "clash-check",
	Short: "Check if file is modified in another worktree",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		file := mustGetString(cmd, "file")
		if file == "" {
			return nil
		}

		// List all worktrees and check if the file is modified in any of them
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain").Output()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git worktree list timed out after %v", GitTimeout), nil)
				return nil
			}
			outputOK(map[string]interface{}{"clash": false, "reason": "not a git worktree repo"})
			return nil
		}

		clashingWorktrees := []string{}

		// Check for modifications in each worktree
		worktreePaths := parseWorktreePaths(string(out))
		for _, wtPath := range worktreePaths {
			// Check if file has changes in that worktree
			diffCtx, diffCancel := context.WithTimeout(context.Background(), GitTimeout)
			defer diffCancel()
			diffCmd := exec.CommandContext(diffCtx, "git", "-C", wtPath, "diff", "--name-only", "HEAD", "--", file)
			diffOut, diffErr := diffCmd.Output()
			if diffErr == nil && strings.TrimSpace(string(diffOut)) != "" {
				clashingWorktrees = append(clashingWorktrees, wtPath)
			}
		}

		if len(clashingWorktrees) > 0 {
			outputOK(map[string]interface{}{
				"clash":     true,
				"file":      file,
				"worktrees": clashingWorktrees,
				"count":     len(clashingWorktrees),
			})
		} else {
			outputOK(map[string]interface{}{"clash": false, "file": file})
		}
		return nil
	},
}

func parseWorktreePaths(porcelain string) []string {
	var paths []string
	lines := strings.Split(porcelain, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if path != "" {
				paths = append(paths, path)
			}
		}
	}
	return paths
}

// --- clash-setup ---

var clashSetupCmd = &cobra.Command{
	Use:   "clash-setup",
	Short: "Install clash detection hooks and merge driver",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set up git merge driver for clash detection
		driverCmd := "aether clash-check --file %A"
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		if err := exec.CommandContext(ctx, "git", "config", "--local", "merge.aether-clash.name", "Aether Clash Detection").Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git config timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("failed to set merge driver name: %v", err), nil)
			return nil
		}
		if err := exec.CommandContext(ctx, "git", "config", "--local", "merge.aether-clash.driver", driverCmd).Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git config timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("failed to set merge driver: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"setup": true, "driver": "aether-clash"})
		return nil
	},
}

// --- worktree-create ---

var worktreeCreateCmd = &cobra.Command{
	Use:   "worktree-create",
	Short: "Create worktree with pheromone injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := mustGetString(cmd, "branch")
		if branch == "" {
			return nil
		}

		// Create worktree
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "git", "worktree", "add", branch, "-b", branch).CombinedOutput()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git worktree add timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("failed to create worktree: %v: %s", err, string(out)), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"created": true,
			"branch":  branch,
			"path":    branch,
		})
		return nil
	},
}

// --- worktree-cleanup ---

var worktreeCleanupCmd = &cobra.Command{
	Use:   "worktree-cleanup",
	Short: "Clean up worktree after merge",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := mustGetString(cmd, "branch")
		if branch == "" {
			return nil
		}

		// Remove worktree
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "git", "worktree", "remove", branch, "--force").CombinedOutput()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git worktree remove timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("failed to remove worktree: %v: %s", err, string(out)), nil)
			return nil
		}

		// Prune stale references
		pruneCtx, pruneCancel := context.WithTimeout(context.Background(), GitTimeout)
		defer pruneCancel()
		exec.CommandContext(pruneCtx, "git", "worktree", "prune").Run()

		outputOK(map[string]interface{}{
			"cleaned": true,
			"branch":  branch,
		})
		return nil
	},
}

func init() {
	clashCheckCmd.Flags().String("file", "", "File path to check (required)")
	worktreeCreateCmd.Flags().String("branch", "", "Branch name (required)")
	worktreeCleanupCmd.Flags().String("branch", "", "Branch name (required)")

	for _, c := range []*cobra.Command{
		clashCheckCmd, clashSetupCmd, worktreeCreateCmd, worktreeCleanupCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
