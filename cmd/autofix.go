package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var autofixCheckpointCmd = &cobra.Command{
	Use:   "autofix-checkpoint",
	Short: "Create a checkpoint before autofix attempt",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		issue := mustGetString(cmd, "issue")
		if issue == "" {
			return nil
		}

		// Read current COLONY_STATE.json
		data, err := store.ReadFile("COLONY_STATE.json")
		if err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		// Create checkpoint filename with timestamp
		timestamp := time.Now().UTC().Format("20060102T150405Z")
		checkpointName := fmt.Sprintf("autofix-%s", timestamp)
		checkpointPath := filepath.Join("checkpoints", checkpointName+".json")

		// Create checkpoint with metadata
		checkpoint := map[string]interface{}{
			"checkpoint": checkpointName,
			"issue":      issue,
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"data":       string(data),
		}

		if err := store.SaveJSON(checkpointPath, checkpoint); err != nil {
			outputError(2, fmt.Sprintf("failed to create checkpoint: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"checkpoint": checkpointName,
			"path":       checkpointPath,
			"issue":      issue,
		})
		return nil
	},
}

var autofixRollbackCmd = &cobra.Command{
	Use:   "autofix-rollback",
	Short: "Rollback to a checkpoint",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		checkpointID := mustGetString(cmd, "checkpoint-id")
		if checkpointID == "" {
			return nil
		}

		// Read checkpoint file
		checkpointPath := filepath.Join("checkpoints", checkpointID+".json")
		data, err := store.ReadFile(checkpointPath)
		if err != nil {
			outputError(1, fmt.Sprintf("checkpoint %q not found", checkpointID), nil)
			return nil
		}

		// Overwrite COLONY_STATE.json with the checkpoint data
		if err := store.AtomicWrite("COLONY_STATE.json", data); err != nil {
			outputError(2, fmt.Sprintf("failed to rollback: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"rolled_back": true,
			"checkpoint":  checkpointID,
		})
		return nil
	},
}

func init() {
	autofixCheckpointCmd.Flags().String("issue", "", "Description of the issue being fixed (required)")

	autofixRollbackCmd.Flags().String("checkpoint-id", "", "Checkpoint name to rollback to (required)")

	rootCmd.AddCommand(autofixCheckpointCmd)
	rootCmd.AddCommand(autofixRollbackCmd)
}
