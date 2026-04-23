package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	versionCmd.Flags().Bool("check", false, "Verify binary version matches hub version")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print aether version",
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		binaryVersion := resolveVersion()

		if check {
			hubVersion := readInstalledHubVersion()
			if hubVersion == "" {
				return fmt.Errorf("version check failed: no hub version found (is Aether installed?)")
			}
			if binaryVersion != hubVersion {
				return fmt.Errorf("version mismatch: binary=%s hub=%s", binaryVersion, hubVersion)
			}
			outputWorkflow(map[string]interface{}{
				"ok":      true,
				"message": fmt.Sprintf("Version check passed: binary and hub both at %s", binaryVersion),
				"version": binaryVersion,
			}, "")
			return nil
		}

		outputOK(binaryVersion)
		return nil
	},
}
