package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
)

var traceReplayCmd = &cobra.Command{
	Use:   "trace-replay",
	Short: "Replay trace entries for a run",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		runID, _ := cmd.Flags().GetString("run-id")
		if runID == "" {
			outputError(1, "--run-id is required", nil)
			return nil
		}

		levelFilter, _ := cmd.Flags().GetString("level")
		sinceStr, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 100
		}

		var since time.Time
		if sinceStr != "" {
			var err error
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				outputError(1, fmt.Sprintf("invalid --since format: %v", err), nil)
				return nil
			}
		}

		levels := map[string]bool{}
		if levelFilter != "" {
			for _, l := range strings.Split(levelFilter, ",") {
				levels[strings.TrimSpace(l)] = true
			}
		}

		lines, err := store.ReadJSONL("trace.jsonl")
		if err != nil {
			outputOK(map[string]interface{}{
				"entries": []trace.TraceEntry{},
				"count":   0,
				"run_id":  runID,
			})
			return nil
		}

		var results []trace.TraceEntry
		for _, line := range lines {
			var entry trace.TraceEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}
			if entry.RunID != runID {
				continue
			}
			if len(levels) > 0 && !levels[string(entry.Level)] {
				continue
			}
			if !since.IsZero() {
				entryTime, err := time.Parse(time.RFC3339, entry.Timestamp)
				if err == nil && entryTime.Before(since) {
					continue
				}
			}
			results = append(results, entry)
			if len(results) >= limit {
				break
			}
		}

		outputOK(map[string]interface{}{
			"entries": results,
			"count":   len(results),
			"run_id":  runID,
		})
		return nil
	},
}

var traceExportCmd = &cobra.Command{
	Use:   "trace-export",
	Short: "Export trace entries for a run to JSON",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		runID, _ := cmd.Flags().GetString("run-id")
		if runID == "" {
			outputError(1, "--run-id is required", nil)
			return nil
		}

		outputPath, _ := cmd.Flags().GetString("output")
		levelFilter, _ := cmd.Flags().GetString("level")
		sinceStr, _ := cmd.Flags().GetString("since")

		var since time.Time
		if sinceStr != "" {
			var err error
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				outputError(1, fmt.Sprintf("invalid --since format: %v", err), nil)
				return nil
			}
		}

		levels := map[string]bool{}
		if levelFilter != "" {
			for _, l := range strings.Split(levelFilter, ",") {
				levels[strings.TrimSpace(l)] = true
			}
		}

		lines, err := store.ReadJSONL("trace.jsonl")
		if err != nil {
			outputOK(map[string]interface{}{
				"entries": []trace.TraceEntry{},
				"count":   0,
				"run_id":  runID,
			})
			return nil
		}

		var results []trace.TraceEntry
		for _, line := range lines {
			var entry trace.TraceEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}
			if entry.RunID != runID {
				continue
			}
			if len(levels) > 0 && !levels[string(entry.Level)] {
				continue
			}
			if !since.IsZero() {
				entryTime, err := time.Parse(time.RFC3339, entry.Timestamp)
				if err == nil && entryTime.Before(since) {
					continue
				}
			}
			results = append(results, entry)
		}

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			outputError(2, fmt.Sprintf("failed to marshal export: %v", err), nil)
			return nil
		}

		if outputPath != "" {
			if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
				outputError(2, fmt.Sprintf("failed to write export: %v", err), nil)
				return nil
			}
			outputOK(map[string]interface{}{
				"exported": true,
				"count":    len(results),
				"run_id":   runID,
				"output":   outputPath,
			})
		} else {
			fmt.Fprintln(stdout, string(data))
		}
		return nil
	},
}

func init() {
	traceReplayCmd.Flags().String("run-id", "", "Filter by run ID (required)")
	traceReplayCmd.Flags().String("level", "", "Filter by level (comma-separated)")
	traceReplayCmd.Flags().String("since", "", "Filter by timestamp (RFC3339)")
	traceReplayCmd.Flags().Int("limit", 100, "Max entries (default 100)")

	traceExportCmd.Flags().String("run-id", "", "Filter by run ID (required)")
	traceExportCmd.Flags().String("level", "", "Filter by level (comma-separated)")
	traceExportCmd.Flags().String("since", "", "Filter by timestamp (RFC3339)")
	traceExportCmd.Flags().String("output", "", "Output file path (default stdout)")

	rootCmd.AddCommand(traceReplayCmd)
	rootCmd.AddCommand(traceExportCmd)
}
