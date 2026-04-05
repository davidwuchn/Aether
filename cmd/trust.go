package cmd

import (
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/spf13/cobra"
)

var (
	trustSourceType string
	trustEvidence   string
	trustDaysSince  int
	trustDecayScore float64
	trustDecayDays  int
	trustTierScore  float64
)

var trustScoreComputeCmd = &cobra.Command{
	Use:   "trust-score-compute",
	Short: "Compute a trust score from source, evidence, and days-since inputs",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		input := memory.TrustInput{
			SourceType: mustGetString(cmd, "source-type"),
			Evidence:   mustGetString(cmd, "evidence"),
			DaysSince:  mustGetInt(cmd, "days-since"),
		}
		if input.SourceType == "" || input.Evidence == "" {
			return nil
		}
		result := memory.Calculate(input)
		outputOK(map[string]interface{}{
			"score":          result.Score,
			"source_score":   result.SourceScore,
			"evidence_score": result.EvidenceScore,
			"activity_score": result.ActivityScore,
			"tier":           result.Tier,
			"tier_index":     result.TierIndex,
		})
		return nil
	},
}

var trustScoreDecayCmd = &cobra.Command{
	Use:   "trust-score-decay",
	Short: "Apply half-life decay to an existing trust score",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		score, err := cmd.Flags().GetFloat64("score")
		if err != nil {
			outputError(1, "missing flag --score", nil)
			return nil
		}
		days := mustGetInt(cmd, "days")
		decayed := memory.Decay(score, days)
		outputOK(map[string]interface{}{
			"original_score": score,
			"days":           days,
			"decayed_score":  decayed,
		})
		return nil
	},
}

var trustTierCmd = &cobra.Command{
	Use:   "trust-tier",
	Short: "Return the trust tier name and index for a given score",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		score, err := cmd.Flags().GetFloat64("score")
		if err != nil {
			outputError(1, "missing flag --score", nil)
			return nil
		}
		tierName, tierIndex := memory.Tier(score)
		outputOK(map[string]interface{}{
			"score":      score,
			"tier":       tierName,
			"tier_index": tierIndex,
		})
		return nil
	},
}

func init() {
	// trust-score-compute
	rootCmd.AddCommand(trustScoreComputeCmd)
	trustScoreComputeCmd.Flags().StringVar(&trustSourceType, "source-type", "", "Source type (user_feedback, error_resolution, success_pattern, observation, heuristic)")
	trustScoreComputeCmd.Flags().StringVar(&trustEvidence, "evidence", "", "Evidence type (test_verified, multi_phase, single_phase, anecdotal)")
	trustScoreComputeCmd.Flags().IntVar(&trustDaysSince, "days-since", 0, "Days since the observation was first seen")

	// trust-score-decay
	rootCmd.AddCommand(trustScoreDecayCmd)
	trustScoreDecayCmd.Flags().Float64Var(&trustDecayScore, "score", 0, "Current trust score")
	trustScoreDecayCmd.Flags().IntVar(&trustDecayDays, "days", 0, "Days of decay to apply")

	// trust-tier
	rootCmd.AddCommand(trustTierCmd)
	trustTierCmd.Flags().Float64Var(&trustTierScore, "score", 0, "Trust score to classify")
}
