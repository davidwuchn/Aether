package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var verifyClaimsCmd = &cobra.Command{
	Use:   "verify-claims",
	Short: "Verify claims in COLONY_STATE.json are consistent with actual file state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		data, err := store.ReadFile("COLONY_STATE.json")
		if err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		var state map[string]interface{}
		if err := json.Unmarshal(data, &state); err != nil {
			outputError(1, "failed to parse COLONY_STATE.json", nil)
			return nil
		}

		type claimCheck struct {
			Name   string
			Passed bool
			Detail string
		}

		var checks []claimCheck
		issues := []string{}
		allValid := true

		// Check 1: If state says "EXECUTING", verify build_started_at is set
		stateVal, _ := state["state"].(string)
		if stateVal == "EXECUTING" {
			if _, ok := state["build_started_at"]; !ok {
				checks = append(checks, claimCheck{
					Name:   "executing_has_build_started",
					Passed: false,
					Detail: "state is EXECUTING but build_started_at is not set",
				})
				issues = append(issues, "EXECUTING without build_started_at")
				allValid = false
			} else {
				checks = append(checks, claimCheck{
					Name:   "executing_has_build_started",
					Passed: true,
					Detail: "build_started_at is set for EXECUTING state",
				})
			}
		}

		// Check 2: If milestone is set, verify milestone_updated_at is set
		milestone, hasMilestone := state["milestone"].(string)
		if hasMilestone && milestone != "" {
			if _, ok := state["milestone_updated_at"]; !ok {
				checks = append(checks, claimCheck{
					Name:   "milestone_has_updated_at",
					Passed: false,
					Detail: "milestone is set but milestone_updated_at is not set",
				})
				issues = append(issues, "milestone without milestone_updated_at")
				allValid = false
			} else {
				checks = append(checks, claimCheck{
					Name:   "milestone_has_updated_at",
					Passed: true,
					Detail: "milestone_updated_at is set",
				})
			}
		}

		// Check 3: If current_phase > 0, verify plan.phases has that many entries
		phaseRaw, hasPhase := state["current_phase"]
		if hasPhase {
			// JSON numbers are float64 in Go
			phaseFloat, ok := phaseRaw.(float64)
			if ok && phaseFloat > 0 {
				plan, hasPlan := state["plan"].(map[string]interface{})
				if hasPlan {
					phasesRaw, hasPhases := plan["phases"]
					if hasPhases {
						phases, ok := phasesRaw.([]interface{})
						if ok && int(phaseFloat) > len(phases) {
							checks = append(checks, claimCheck{
								Name:   "phase_in_range",
								Passed: false,
								Detail: "current_phase exceeds number of plan phases",
							})
							issues = append(issues, "current_phase out of range")
							allValid = false
						} else {
							checks = append(checks, claimCheck{
								Name:   "phase_in_range",
								Passed: true,
								Detail: "current_phase is within plan phases range",
							})
						}
					}
				}
			}
		}

		// Check 4: Verify version is present
		version, hasVersion := state["version"].(string)
		if !hasVersion || version == "" {
			checks = append(checks, claimCheck{
				Name:   "version_present",
				Passed: false,
				Detail: "version is missing or empty",
			})
			issues = append(issues, "missing version")
			allValid = false
		} else {
			checks = append(checks, claimCheck{
				Name:   "version_present",
				Passed: true,
				Detail: "version is present",
			})
		}

		// Convert checks to []interface{} for outputOK
		checksIface := make([]interface{}, len(checks))
		for i, c := range checks {
			checksIface[i] = map[string]interface{}{
				"name":   c.Name,
				"passed": c.Passed,
				"detail": c.Detail,
			}
		}

		outputOK(map[string]interface{}{
			"valid":  allValid,
			"checks": checksIface,
			"issues": issues,
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyClaimsCmd)
}
