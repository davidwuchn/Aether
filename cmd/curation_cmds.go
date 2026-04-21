package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/agent/curation"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

var curationArchivistThreshold float64

// curationSentinelCmd runs the sentinel curation ant.
var curationSentinelCmd = &cobra.Command{
	Use:   "curation-sentinel",
	Short: "Run sentinel curation ant (data integrity check)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		s := curation.NewSentinel(store)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := s.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("sentinel failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationNurseCmd runs the nurse curation ant.
var curationNurseCmd = &cobra.Command{
	Use:   "curation-nurse",
	Short: "Run nurse curation ant (trust score recalculation)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		n := curation.NewNurse(store)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := n.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("nurse failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationCriticCmd runs the critic curation ant.
var curationCriticCmd = &cobra.Command{
	Use:   "curation-critic",
	Short: "Run critic curation ant (contradiction detection)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		c := curation.NewCritic(store)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := c.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("critic failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationHeraldCmd runs the herald curation ant.
var curationHeraldCmd = &cobra.Command{
	Use:   "curation-herald",
	Short: "Run herald curation ant (high-confidence instinct promotion)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		h := curation.NewHerald(store)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := h.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("herald failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationJanitorCmd runs the janitor curation ant.
var curationJanitorCmd = &cobra.Command{
	Use:   "curation-janitor",
	Short: "Run janitor curation ant (expired event cleanup)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		bus := events.NewBus(store, events.DefaultConfig())
		j := curation.NewJanitor(store, bus)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := j.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("janitor failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationArchivistCmd runs the archivist curation ant.
var curationArchivistCmd = &cobra.Command{
	Use:   "curation-archivist",
	Short: "Run archivist curation ant (low-trust instinct archival)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		threshold, _ := cmd.Flags().GetFloat64("threshold")
		a := curation.NewArchivistWithThreshold(store, threshold)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := a.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("archivist failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationLibrarianCmd runs the librarian curation ant.
var curationLibrarianCmd = &cobra.Command{
	Use:   "curation-librarian",
	Short: "Run librarian curation ant (inventory statistics)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		bus := events.NewBus(store, events.DefaultConfig())
		l := curation.NewLibrarian(store, bus)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := l.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("librarian failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationScribeCmd runs the scribe curation ant.
var curationScribeCmd = &cobra.Command{
	Use:   "curation-scribe",
	Short: "Run scribe curation ant (report generation)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Scribe does not require a store, but we still check for consistency
		// with the overall CLI pattern and store initialization from PersistentPreRunE.

		dryRun := mustGetBool(cmd, "dry-run")
		s := curation.NewScribe()

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		sr, err := s.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("scribe failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"name":    sr.Name,
			"success": sr.Success,
			"summary": sr.Summary,
			"dry_run": dryRun,
		})
		return nil
	},
}

// curationRunCmd runs all 8 curation ants via the orchestrator.
var curationRunCmd = &cobra.Command{
	Use:   "curation-run",
	Short: "Run all 8 curation ants in sequence",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun := mustGetBool(cmd, "dry-run")
		bus := events.NewBus(store, events.DefaultConfig())
		o := curation.NewOrchestrator(store, bus)

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()
		result, err := o.Run(ctx, dryRun)
		if err != nil {
			outputError(2, fmt.Sprintf("curation run failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"steps":       result.Steps,
			"succeeded":   result.Succeeded,
			"failed":      result.Failed,
			"skipped":     result.Skipped,
			"dry_run":     result.DryRun,
			"duration_ms": result.DurationMs,
		})
		return nil
	},
}

func init() {
	curationArchivistCmd.Flags().Float64Var(&curationArchivistThreshold, "threshold", 0.30, "Archive threshold for active instincts")
}

func init() {
	curationSentinelCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationNurseCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationCriticCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationHeraldCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationJanitorCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationArchivistCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationLibrarianCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationScribeCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")
	curationRunCmd.Flags().Bool("dry-run", false, "Dry run mode (no mutations)")

	rootCmd.AddCommand(curationSentinelCmd)
	rootCmd.AddCommand(curationNurseCmd)
	rootCmd.AddCommand(curationCriticCmd)
	rootCmd.AddCommand(curationHeraldCmd)
	rootCmd.AddCommand(curationJanitorCmd)
	rootCmd.AddCommand(curationArchivistCmd)
	rootCmd.AddCommand(curationLibrarianCmd)
	rootCmd.AddCommand(curationScribeCmd)
	rootCmd.AddCommand(curationRunCmd)
}
