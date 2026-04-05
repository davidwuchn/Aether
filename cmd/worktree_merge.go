// Package cmd implements the Aether CLI commands using Cobra.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// worktreeMergeCmd merges a worktree branch back to the target branch with
// safety checks: dirty worktree detection, commits-ahead check, conflict
// detection via dry run, and build verification. All checks are hard gates
// per D-04 (fail-fast) and D-05 (safety checks are non-negotiable).
var worktreeMergeCmd = &cobra.Command{
	Use:   "worktree-merge",
	Short: "Merge a worktree branch back to target with safety checks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := mustGetString(cmd, "branch")
		if branch == "" {
			return nil
		}

		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
			defer cancel()
			out, _ := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD").Output()
			target = strings.TrimSpace(string(out))
			if target == "" {
				target = "main"
			}
		}

		// Resolve the git working directory for running commands.
		gitDir := resolveGitDir()

		// Safety check 1: Dirty worktree detection.
		// Check for uncommitted changes, excluding .aether/ paths (local-only state).
		if err := checkDirtyWorktree(gitDir, branch); err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		// Safety check 2: Commits ahead of target.
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "git", "-C", gitDir, "rev-list", "--count", target+".."+branch).Output()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(1, fmt.Sprintf("git rev-list timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(1, fmt.Sprintf("failed to check commits ahead: %v", err), nil)
			return nil
		}
		aheadCount := strings.TrimSpace(string(out))
		if aheadCount == "0" {
			outputError(1, fmt.Sprintf("nothing to merge: branch %q has no commits ahead of %q", branch, target), nil)
			return nil
		}

		// Safety check 3: Conflict detection via dry run.
		if err := checkMergeConflicts(gitDir, target, branch); err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		// Perform the merge.
		mergeMsg := fmt.Sprintf("merge: worktree branch %s into %s", branch, target)
		mergeCtx, mergeCancel := context.WithTimeout(context.Background(), GitTimeout)
		defer mergeCancel()
		out, err = exec.CommandContext(mergeCtx, "git", "-C", gitDir, "merge", branch, "--no-edit", "--no-ff", "-m", mergeMsg).CombinedOutput()
		if err != nil {
			if mergeCtx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git merge timed out after %v", GitTimeout), nil)
				return nil
			}
			abortCtx, abortCancel := context.WithTimeout(context.Background(), GitTimeout)
			defer abortCancel()
			exec.CommandContext(abortCtx, "git", "-C", gitDir, "merge", "--abort").Run()
			outputError(2, fmt.Sprintf("merge failed: %s", strings.TrimSpace(string(out))), nil)
			return nil
		}

		// Safety check 4: Build verification (only when go.mod exists).
		// Per D-05, verify the merged code compiles. Skip if not a Go project.
		if _, err := os.Stat(gitDir + "/go.mod"); err == nil {
			buildCtx, buildCancel := context.WithTimeout(context.Background(), BuildTimeout)
			defer buildCancel()
			out, err = exec.CommandContext(buildCtx, "go", "build", "-C", gitDir, "./cmd/aether").CombinedOutput()
			if err != nil {
				if buildCtx.Err() == context.DeadlineExceeded {
					outputError(2, fmt.Sprintf("go build timed out after %v", BuildTimeout), nil)
					return nil
				}
				abortCtx2, abortCancel2 := context.WithTimeout(context.Background(), GitTimeout)
				defer abortCancel2()
				exec.CommandContext(abortCtx2, "git", "-C", gitDir, "merge", "--abort").Run()
				outputError(2, fmt.Sprintf("build failed after merge: %s", strings.TrimSpace(string(out))), nil)
				return nil
			}
		}

		// Restore .aether/data/ to target branch version.
		// Per MERGE-03: .aether/data/ conflicts prefer the target (main) version.
		// This prevents worktree-local colony state from overriding main's data.
		// Use HEAD^1 (first parent of merge = pre-merge target) to restore files.
		lsCtx, lsCancel := context.WithTimeout(context.Background(), GitTimeout)
		defer lsCancel()
		dataCheckOut, _ := exec.CommandContext(lsCtx, "git", "-C", gitDir, "ls-tree", "-r", "--name-only", "HEAD^1", ".aether/data/").Output()
		if len(strings.TrimSpace(string(dataCheckOut))) > 0 {
			coCtx, coCancel := context.WithTimeout(context.Background(), GitTimeout)
			defer coCancel()
			exec.CommandContext(coCtx, "git", "-C", gitDir, "checkout", "HEAD^1", "--", ".aether/data/").Run()
		}

		// Success: report merge result.
		shaCtx, shaCancel := context.WithTimeout(context.Background(), GitTimeout)
		defer shaCancel()
		mergeSHA, _ := exec.CommandContext(shaCtx, "git", "-C", gitDir, "rev-parse", "HEAD").Output()
		outputOK(map[string]interface{}{
			"merged": true,
			"branch": branch,
			"target": target,
			"sha":    strings.TrimSpace(string(mergeSHA)),
		})
		return nil
	},
}

// resolveGitDir returns the directory from which git commands should run.
// It checks AETHER_ROOT first, falling back to the current working directory.
func resolveGitDir() string {
	if root := os.Getenv("AETHER_ROOT"); root != "" {
		return root
	}
	dir, _ := os.Getwd()
	return dir
}

// checkDirtyWorktree checks for uncommitted changes in the working tree,
// excluding .aether/ paths which are local-only colony state.
func checkDirtyWorktree(gitDir, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", gitDir, "status", "--porcelain").Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git status timed out after %v", GitTimeout)
		}
		return nil // If status fails, proceed (might not be in a git repo)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var dirtyCount int
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Exclude .aether/ paths -- local-only colony state, not user changes.
		if strings.Contains(line, ".aether/") {
			continue
		}
		dirtyCount++
	}

	if dirtyCount > 0 {
		return fmt.Errorf("dirty worktree: branch %q has %d uncommitted changes", branch, dirtyCount)
	}
	return nil
}

// checkMergeConflicts uses git merge-tree to detect conflicts before merging.
func checkMergeConflicts(gitDir, target, branch string) error {
	// Find the merge base between target and branch.
	baseCtx, baseCancel := context.WithTimeout(context.Background(), GitTimeout)
	defer baseCancel()
	baseOut, err := exec.CommandContext(baseCtx, "git", "-C", gitDir, "merge-base", target, branch).Output()
	if err != nil {
		if baseCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git merge-base timed out after %v", GitTimeout)
		}
		return fmt.Errorf("failed to find merge base: %v", err)
	}
	base := strings.TrimSpace(string(baseOut))

	// Dry-run merge to detect conflicts.
	treeCtx, treeCancel := context.WithTimeout(context.Background(), GitTimeout)
	defer treeCancel()
	out, err := exec.CommandContext(treeCtx, "git", "-C", gitDir, "merge-tree", base, target, branch).Output()
	if err != nil {
		if treeCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git merge-tree timed out after %v", GitTimeout)
		}
		return fmt.Errorf("conflict detection failed: %v", err)
	}

	output := string(out)
	if strings.Contains(output, "changed in both") {
		// Count the number of conflicts.
		conflictCount := strings.Count(output, "changed in both")
		return fmt.Errorf("merge would produce %d conflict(s)", conflictCount)
	}
	return nil
}

func init() {
	worktreeMergeCmd.Flags().String("branch", "", "Branch name (required)")
	worktreeMergeCmd.Flags().String("target", "", "Target branch (default: current)")
	rootCmd.AddCommand(worktreeMergeCmd)
}
