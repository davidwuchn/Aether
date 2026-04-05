package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

const viewStateFile = "view-state.json"

// viewStateRead reads the view-state.json file, returning an empty map if not found.
func viewStateRead() map[string]interface{} {
	data, err := store.ReadFile(viewStateFile)
	if err != nil {
		return make(map[string]interface{})
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string]interface{})
	}
	if m == nil {
		return make(map[string]interface{})
	}
	return m
}

var viewStateInitCmd = &cobra.Command{
	Use:   "view-state-init",
	Short: "Initialize view state file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		m := viewStateRead()

		if err := store.SaveJSON(viewStateFile, m); err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"initialized": true,
		})
		return nil
	},
}

var viewStateGetCmd = &cobra.Command{
	Use:   "view-state-get",
	Short: "Get a view state value",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		key := mustGetString(cmd, "key")
		if key == "" {
			return nil
		}

		m := viewStateRead()

		val, found := m[key]
		if !found {
			outputOK(map[string]interface{}{
				"key":   key,
				"value": nil,
				"found": false,
			})
			return nil
		}

		outputOK(map[string]interface{}{
			"key":   key,
			"value": val,
			"found": true,
		})
		return nil
	},
}

var viewStateSetCmd = &cobra.Command{
	Use:   "view-state-set",
	Short: "Set a view state value",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		key := mustGetString(cmd, "key")
		if key == "" {
			return nil
		}
		value := mustGetString(cmd, "value")
		if value == "" {
			return nil
		}

		m := viewStateRead()

		m[key] = value

		if err := store.SaveJSON(viewStateFile, m); err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"key":  key,
			"set":  true,
		})
		return nil
	},
}

var viewStateToggleCmd = &cobra.Command{
	Use:   "view-state-toggle",
	Short: "Toggle a boolean view state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		key := mustGetString(cmd, "key")
		if key == "" {
			return nil
		}

		m := viewStateRead()

		current, _ := m[key].(bool)

		toggled := !current
		m[key] = toggled

		if err := store.SaveJSON(viewStateFile, m); err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"key":   key,
			"value": toggled,
		})
		return nil
	},
}

var viewStateExpandCmd = &cobra.Command{
	Use:   "view-state-expand",
	Short: "Expand a collapsed section",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		section := mustGetString(cmd, "section")
		if section == "" {
			return nil
		}

		m := viewStateRead()
		m["expanded_"+section] = true

		if err := store.SaveJSON(viewStateFile, m); err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"section":  section,
			"expanded": true,
		})
		return nil
	},
}

var viewStateCollapseCmd = &cobra.Command{
	Use:   "view-state-collapse",
	Short: "Collapse a section",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		section := mustGetString(cmd, "section")
		if section == "" {
			return nil
		}

		m := viewStateRead()
		m["expanded_"+section] = false

		if err := store.SaveJSON(viewStateFile, m); err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"section":  section,
			"expanded": false,
		})
		return nil
	},
}

func init() {
	viewStateGetCmd.Flags().String("key", "", "Key to get (required)")
	viewStateSetCmd.Flags().String("key", "", "Key to set (required)")
	viewStateSetCmd.Flags().String("value", "", "Value to set (required)")
	viewStateToggleCmd.Flags().String("key", "", "Key to toggle (required)")
	viewStateExpandCmd.Flags().String("section", "", "Section to expand (required)")
	viewStateCollapseCmd.Flags().String("section", "", "Section to collapse (required)")

	rootCmd.AddCommand(viewStateInitCmd)
	rootCmd.AddCommand(viewStateGetCmd)
	rootCmd.AddCommand(viewStateSetCmd)
	rootCmd.AddCommand(viewStateToggleCmd)
	rootCmd.AddCommand(viewStateExpandCmd)
	rootCmd.AddCommand(viewStateCollapseCmd)
}
