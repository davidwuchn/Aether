package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var stateMutateCmd = &cobra.Command{
	Use:   "state-mutate",
	Short: "Atomically mutate a field in COLONY_STATE.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		field := mustGetString(cmd, "field")
		if field == "" {
			return nil
		}
		value := mustGetString(cmd, "value")

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		switch field {
		case "goal":
			state.Goal = &value
		case "state":
			newState := colony.State(value)
			if err := colony.Transition(state.State, newState); err != nil {
				outputError(1, fmt.Sprintf("invalid transition %s -> %s: %v", state.State, newState, err), nil)
				return nil
			}
			state.State = newState
		case "current_phase":
			// Validate phase exists
			phaseNum := 0
			if _, err := fmt.Sscanf(value, "%d", &phaseNum); err != nil {
				outputError(1, fmt.Sprintf("invalid phase number %q", value), nil)
				return nil
			}
			if phaseNum > 0 && phaseNum <= len(state.Plan.Phases) {
				state.Plan.Phases[phaseNum-1].Status = colony.PhaseInProgress
			}
			state.CurrentPhase = phaseNum
		case "milestone":
			state.Milestone = value
		case "colony_depth":
			state.ColonyDepth = value
		case "colony_name":
			state.ColonyName = &value
		default:
			// Try nested field via JSON manipulation
			if updated, err := setNestedField(state, field, value); err == nil {
				state = updated.(colony.ColonyState)
			} else {
				outputError(1, fmt.Sprintf("unknown field %q", field), nil)
				return nil
			}
		}

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"updated": true,
			"field":   field,
			"value":   value,
		})
		return nil
	},
}

var loadStateCmd = &cobra.Command{
	Use:   "load-state",
	Short: "Load COLONY_STATE.json and return it",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		outputOK(state)
		return nil
	},
}

var unloadStateCmd = &cobra.Command{
	Use:   "unload-state",
	Short: "Release state lock (placeholder)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(map[string]interface{}{"unloaded": true})
		return nil
	},
}

var validateStateCmd = &cobra.Command{
	Use:   "validate-state",
	Short: "Validate COLONY_STATE.json structure",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, fmt.Sprintf("failed to load state: %v", err), nil)
			return nil
		}

		issues := []string{}

		if state.Version == "" {
			issues = append(issues, "missing version")
		}
		if state.Goal == nil || *state.Goal == "" {
			issues = append(issues, "missing goal")
		}
		if state.State == "" {
			issues = append(issues, "missing state")
		}

		valid := len(issues) == 0

		outputOK(map[string]interface{}{
			"valid":  valid,
			"issues": issues,
			"version": state.Version,
		})
		return nil
	},
}

var stateReadCmd = &cobra.Command{
	Use:   "state-read",
	Short: "Read COLONY_STATE.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		outputOK(state)
		return nil
	},
}

var stateReadFieldCmd = &cobra.Command{
	Use:   "state-read-field",
	Short: "Read a specific field from COLONY_STATE.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		field := mustGetString(cmd, "field")
		if field == "" {
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		var result interface{}
		switch field {
		case "goal":
			result = state.Goal
		case "state":
			result = string(state.State)
		case "current_phase":
			result = state.CurrentPhase
		case "milestone":
			result = state.Milestone
		case "colony_depth":
			result = state.ColonyDepth
		case "colony_name":
			result = state.ColonyName
		case "session_id":
			result = state.SessionID
		default:
			outputError(1, fmt.Sprintf("unknown field %q", field), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"field": field,
			"value": result,
		})
		return nil
	},
}

func init() {
	stateMutateCmd.Flags().String("field", "", "Field to mutate (required)")
	stateMutateCmd.Flags().String("value", "", "New value (required)")

	stateReadFieldCmd.Flags().String("field", "", "Field to read (required)")

	rootCmd.AddCommand(stateMutateCmd)
	rootCmd.AddCommand(loadStateCmd)
	rootCmd.AddCommand(unloadStateCmd)
	rootCmd.AddCommand(validateStateCmd)
	rootCmd.AddCommand(stateReadCmd)
	rootCmd.AddCommand(stateReadFieldCmd)
}

// setNestedField sets a dot-separated nested field on a struct via JSON manipulation.
func setNestedField(obj interface{}, fieldPath, value string) (interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	parts := strings.Split(fieldPath, ".")
	current := m
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			next, ok := current[part].(map[string]interface{})
			if !ok {
				next = make(map[string]interface{})
				current[part] = next
			}
			current = next
		}
	}

	// Re-marshal back to the target type
	result, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	// Return as generic map (caller re-types if needed)
	var result2 map[string]interface{}
	json.Unmarshal(result, &result2)
	return result2, nil
}
