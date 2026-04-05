package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// GraveEntry represents a failed agent record stored in graveyard.json.
type GraveEntry struct {
	Agent   string `json:"agent"`
	Reason  string `json:"reason"`
	Phase   string `json:"phase,omitempty"`
	Created string `json:"created_at"`
}

const graveyardFile = "graveyard.json"

// graveRead reads the graveyard.json file, returning an empty slice if not found.
func graveRead() []GraveEntry {
	data, err := store.ReadFile(graveyardFile)
	if err != nil {
		return []GraveEntry{}
	}
	var entries []GraveEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []GraveEntry{}
	}
	if entries == nil {
		return []GraveEntry{}
	}
	return entries
}

var graveAddCmd = &cobra.Command{
	Use:   "grave-add",
	Short: "Record a grave entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		agent := mustGetString(cmd, "agent")
		if agent == "" {
			return nil
		}
		reason := mustGetString(cmd, "reason")
		if reason == "" {
			return nil
		}
		phase, _ := cmd.Flags().GetString("phase")

		entries := graveRead()

		entry := GraveEntry{
			Agent:   agent,
			Reason:  reason,
			Phase:   phase,
			Created: time.Now().UTC().Format(time.RFC3339),
		}

		entries = append(entries, entry)

		if err := store.SaveJSON(graveyardFile, entries); err != nil {
			outputError(2, fmt.Sprintf("failed to save graveyard: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"agent":  agent,
			"buried": true,
		})
		return nil
	},
}

var graveCheckCmd = &cobra.Command{
	Use:   "grave-check",
	Short: "Check if agent has a grave entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		agent := mustGetString(cmd, "agent")
		if agent == "" {
			return nil
		}

		entries := graveRead()

		matching := []GraveEntry{}
		for _, e := range entries {
			if e.Agent == agent {
				matching = append(matching, e)
			}
		}

		found := len(matching) > 0

		outputOK(map[string]interface{}{
			"agent":   agent,
			"found":   found,
			"entries": matching,
		})
		return nil
	},
}

func init() {
	graveAddCmd.Flags().String("agent", "", "Agent name (required)")
	graveAddCmd.Flags().String("reason", "", "Failure reason (required)")
	graveAddCmd.Flags().String("phase", "", "Phase name (optional)")

	graveCheckCmd.Flags().String("agent", "", "Agent name (required)")

	rootCmd.AddCommand(graveAddCmd)
	rootCmd.AddCommand(graveCheckCmd)
}
