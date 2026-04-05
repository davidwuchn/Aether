package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// surveyFiles lists the expected survey JSON files.
var surveyFiles = []string{
	"blueprint",
	"chambers",
	"disciplines",
	"provisions",
	"pathogens",
}

var surveyLoadCmd = &cobra.Command{
	Use:   "survey-load",
	Short: "Load survey results from territory survey",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		surveyDir := filepath.Join(store.BasePath(), "survey")

		// Check if survey directory exists
		info, err := os.Stat(surveyDir)
		if err != nil || !info.IsDir() {
			outputOK(map[string]interface{}{
				"loaded": false,
				"files":  map[string]interface{}{},
				"data":   nil,
			})
			return nil
		}

		files := make(map[string]interface{})
		data := make(map[string]interface{})

		for _, name := range surveyFiles {
			filePath := filepath.Join(surveyDir, name+".json")
			content, err := os.ReadFile(filePath)
			if err != nil {
				files[name] = false
				continue
			}

			var parsed interface{}
			if err := json.Unmarshal(content, &parsed); err != nil {
				files[name] = false
				continue
			}

			files[name] = true
			data[name] = parsed
		}

		outputOK(map[string]interface{}{
			"loaded": true,
			"files":  files,
			"data":   data,
		})
		return nil
	},
}

var surveyVerifyCmd = &cobra.Command{
	Use:   "survey-verify",
	Short: "Verify survey data integrity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		surveyDir := filepath.Join(store.BasePath(), "survey")
		issues := []string{}
		allValid := true

		type fileCheck struct {
			Name      string `json:"name"`
			Exists    bool   `json:"exists"`
			ValidJSON bool   `json:"valid_json"`
		}

		var checks []fileCheck

		for _, name := range surveyFiles {
			filePath := filepath.Join(surveyDir, name+".json")
			check := fileCheck{Name: name}

			content, err := os.ReadFile(filePath)
			if err != nil {
				check.Exists = false
				check.ValidJSON = false
				allValid = false
				issues = append(issues, name+".json: file not found")
			} else {
				check.Exists = true
				if !json.Valid(content) {
					check.ValidJSON = false
					allValid = false
					issues = append(issues, name+".json: invalid JSON")
				} else {
					check.ValidJSON = true
				}
			}

			checks = append(checks, check)
		}

		// Convert to []interface{} for outputOK compatibility
		checksIface := make([]interface{}, len(checks))
		for i, c := range checks {
			checksIface[i] = map[string]interface{}{
				"name":       c.Name,
				"exists":     c.Exists,
				"valid_json": c.ValidJSON,
			}
		}

		outputOK(map[string]interface{}{
			"valid":  allValid,
			"files":  checksIface,
			"issues": issues,
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(surveyLoadCmd)
	rootCmd.AddCommand(surveyVerifyCmd)
}
