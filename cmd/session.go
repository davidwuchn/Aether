package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var activityLogCmd = &cobra.Command{
	Use:   "activity-log",
	Short: "Append an entry to the activity log",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		command := mustGetString(cmd, "command")
		if command == "" {
			return nil
		}
		phase := mustGetInt(cmd, "phase")
		result, _ := cmd.Flags().GetString("result")
		details, _ := cmd.Flags().GetString("details")

		entry := map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"command":   command,
			"phase":     phase,
		}
		if result != "" {
			entry["result"] = result
		}
		if details != "" {
			entry["details"] = details
		}

		if err := store.AppendJSONL("activity-log.jsonl", entry); err != nil {
			outputError(2, fmt.Sprintf("failed to append activity log: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"logged":  true,
			"command": command,
			"phase":   phase,
		})
		return nil
	},
}

var activityLogInitCmd = &cobra.Command{
	Use:   "activity-log-init",
	Short: "Initialize an empty activity log",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Write empty JSONL file
		if err := store.AtomicWrite("activity-log.jsonl", []byte("")); err != nil {
			outputError(2, fmt.Sprintf("failed to create activity log: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"initialized": true})
		return nil
	},
}

var activityLogReadCmd = &cobra.Command{
	Use:   "activity-log-read",
	Short: "Read all entries from the activity log",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		entries, err := store.ReadJSONL("activity-log.jsonl")
		if err != nil {
			outputOK(map[string]interface{}{
				"entries": []json.RawMessage{},
				"count":   0,
			})
			return nil
		}

		if entries == nil {
			entries = []json.RawMessage{}
		}

		outputOK(map[string]interface{}{
			"entries": entries,
			"count":   len(entries),
		})
		return nil
	},
}

func init() {
	activityLogCmd.Flags().String("command", "", "Command name (required)")
	activityLogCmd.Flags().Int("phase", 0, "Phase number (required)")
	activityLogCmd.Flags().String("result", "", "Result status")
	activityLogCmd.Flags().String("details", "", "Additional details")

	rootCmd.AddCommand(activityLogCmd)
	rootCmd.AddCommand(activityLogInitCmd)
	rootCmd.AddCommand(activityLogReadCmd)
}
