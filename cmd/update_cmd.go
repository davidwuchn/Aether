package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/calcosmic/Aether/pkg/downloader"
)

// updateCmd implements "aether update" which syncs companion files from the
// hub and optionally downloads a new binary.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Aether companion files and optionally the binary",
	Long: `Update Aether by syncing companion files from the distribution hub
(~/.aether/system/) to the local .aether/ directory.

This updates slash commands, agent definitions, skills, templates, and docs.
Local user data (COLONY_STATE.json, pheromones, etc.) is never overwritten.

Use --download-binary to also fetch the latest Go binary from GitHub Releases.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

var (
	updateDownloadBinary bool
	updateBinaryVersion  string
	updateDryRun         bool
)

func init() {
	updateCmd.Flags().Bool("download-binary", false, "Also download the latest binary from GitHub Releases")
	updateCmd.Flags().String("binary-version", "", "Binary version to download (default: latest)")
	updateCmd.Flags().Bool("dry-run", false, "Show what would be updated without making changes")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Check hub exists
	hubDir := filepath.Join(homeDir, ".aether")
	hubVersionFile := filepath.Join(hubDir, "version.json")
	if _, err := os.Stat(hubVersionFile); os.IsNotExist(err) {
		outputErrorMessage("Aether hub not installed. Run \"aether install\" first.")
		return nil
	}

	// Read hub version for comparison
	hubVersion := readHubVersion(hubVersionFile)

	// Get repo directory
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	// Sync companion files from hub
	if dryRun {
		outputOK(map[string]interface{}{
			"message": "Dry run — would sync companion files from hub",
			"hub_version": hubVersion,
			"local_version": Version,
			"actions": []string{
				"Sync .aether/ system files (commands, agents, skills, templates, docs)",
				"Sync .claude/commands/ant/",
				"Sync .claude/agents/ant/",
				"Sync .opencode/commands/ant/",
				"Sync .opencode/agents/",
			},
		})
	} else {
		syncResult := runUpdateSync(hubDir, repoDir)
		outputOK(map[string]interface{}{
			"message": fmt.Sprintf("Updated: %d files copied, %d unchanged", syncResult.copied, syncResult.skipped),
			"hub_version":   hubVersion,
			"local_version": Version,
			"details":       syncResult.details,
		})
	}

	// Download binary if requested
	downloadBinary, _ := cmd.Flags().GetBool("download-binary")
	if downloadBinary && !dryRun {
		version, _ := cmd.Flags().GetString("binary-version")
		if version == "" {
			version = "latest"
		}

		destDir := filepath.Join(homeDir, downloader.DefaultDestSubdir())
		outputOK(map[string]interface{}{
			"message": fmt.Sprintf("Downloading aether %s binary...", version),
		})

		result, err := downloader.DownloadBinary(version, destDir)
		if err != nil {
			return fmt.Errorf("file sync succeeded but binary download failed: %w", err)
		}

		outputOK(map[string]interface{}{
			"message": fmt.Sprintf("Binary installed to %s", result.Path),
			"path":    result.Path,
			"version": result.Version,
		})
	} else if downloadBinary && dryRun {
		outputOK(map[string]interface{}{
			"message": "Would download binary from GitHub Releases",
		})
	}

	return nil
}

// updateSyncResult holds the result of an update sync.
type updateSyncResult struct {
	copied  int
	skipped int
	details []map[string]interface{}
}

// runUpdateSync syncs companion files from hub to local repo.
func runUpdateSync(hubDir, repoDir string) updateSyncResult {
	result := updateSyncResult{}

	hubSystem := filepath.Join(hubDir, "system")
	localAether := filepath.Join(repoDir, ".aether")

	syncPairs := []struct {
		srcRel  string
		destRel string
		label   string
	}{
		{".", ".", "System files"},
		{"commands/claude", "../.claude/commands/ant", "Commands (claude)"},
		{"commands/opencode", "../.opencode/commands/ant", "Commands (opencode)"},
		{"agents", "../.opencode/agents", "Agents (opencode)"},
		{"agents-claude", "../.claude/agents/ant", "Agents (claude)"},
	}

	for _, pair := range syncPairs {
		srcDir := filepath.Join(hubSystem, filepath.FromSlash(pair.srcRel))
		destDir := filepath.Join(localAether, filepath.FromSlash(pair.destRel))

		syncRes := setupSyncDir(srcDir, destDir)
		result.details = append(result.details, map[string]interface{}{
			"label":   pair.label,
			"copied":  syncRes.copied,
			"skipped": syncRes.skipped,
		})
		result.copied += syncRes.copied
		result.skipped += syncRes.skipped
	}

	return result
}

// readHubVersion reads the version from the hub's version.json.
func readHubVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return "unknown"
	}
	return v.Version
}
