package cmd

import (
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var normalizeArgsCmd = &cobra.Command{
	Use:   "normalize-args [args...]",
	Short: "Normalize arguments from environment or positional params",
	Long:  "Outputs a single normalized argument string. Reads from ARGUMENTS env var first, then falls back to positional args. Collapses whitespace.",
	RunE: func(cmd *cobra.Command, args []string) error {
		normalized := ""

		// Try ARGUMENTS env var first (Claude Code style)
		if envArgs := os.Getenv("ARGUMENTS"); envArgs != "" {
			normalized = envArgs
		} else if len(args) > 0 {
			// Fall back to positional params (OpenCode style)
			normalized = strings.Join(args, " ")
		}

		// Collapse whitespace: replace runs of whitespace with single space, trim edges
		if normalized != "" {
			re := regexp.MustCompile(`\s+`)
			normalized = strings.TrimSpace(re.ReplaceAllString(normalized, " "))
		}

		outputOK(normalized)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(normalizeArgsCmd)
}
