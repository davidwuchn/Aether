package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var changelogAppendCmd = &cobra.Command{
	Use:   "changelog-append",
	Short: "Append an entry to CHANGELOG.md",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		date := mustGetString(cmd, "date")
		if date == "" {
			return nil
		}
		phase := mustGetString(cmd, "phase")
		if phase == "" {
			return nil
		}
		plan := mustGetString(cmd, "plan")
		if plan == "" {
			return nil
		}
		entry, _ := cmd.Flags().GetString("entry")

		// Check if CHANGELOG.md exists
		if _, err := os.Stat("CHANGELOG.md"); err != nil {
			outputError(1, "CHANGELOG.md not found", nil)
			return nil
		}

		data, err := os.ReadFile("CHANGELOG.md")
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read CHANGELOG.md: %v", err), nil)
			return nil
		}

		// Append entry
		var sb strings.Builder
		sb.Write(data)
		sb.WriteString(fmt.Sprintf("\n- [%s] Phase %s, Plan %s: %s\n", date, phase, plan, entry))

		if err := os.WriteFile("CHANGELOG.md", []byte(sb.String()), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write CHANGELOG.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"appended": true,
			"date":     date,
			"phase":    phase,
			"plan":     plan,
		})
		return nil
	},
}

var changelogCollectPlanDataCmd = &cobra.Command{
	Use:   "changelog-collect-plan-data",
	Short: "Extract plan data for changelog entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile := mustGetString(cmd, "plan-file")
		if planFile == "" {
			return nil
		}

		var data []byte
		var err error
		// Try store-relative first, then absolute path
		data, err = store.ReadFile(planFile)
		if err != nil {
			data, err = os.ReadFile(planFile)
		}
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read plan file: %v", err), nil)
			return nil
		}

		// Extract phase/plan from YAML frontmatter
		content := string(data)
		result := map[string]interface{}{
			"file": planFile,
		}

		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "phase:") {
				result["phase"] = strings.TrimSpace(strings.TrimPrefix(line, "phase:"))
			}
			if strings.HasPrefix(line, "plan:") {
				result["plan"] = strings.TrimSpace(strings.TrimPrefix(line, "plan:"))
			}
		}

		outputOK(result)
		return nil
	},
}

func init() {
	changelogAppendCmd.Flags().String("date", "", "Date string (required)")
	changelogAppendCmd.Flags().String("phase", "", "Phase number (required)")
	changelogAppendCmd.Flags().String("plan", "", "Plan number (required)")
	changelogAppendCmd.Flags().String("entry", "", "Changelog entry text")

	changelogCollectPlanDataCmd.Flags().String("plan-file", "", "Path to plan file (required)")

	rootCmd.AddCommand(changelogAppendCmd)
	rootCmd.AddCommand(changelogCollectPlanDataCmd)
}

// unused import suppression
var _ = json.Marshal
