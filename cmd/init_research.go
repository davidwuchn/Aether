package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// projectDetectors maps a marker file to a project type description.
var projectDetectors = []struct {
	file    string
	typ     string
	frameworks []string
}{
	{"package.json", "node", []string{"node"}},
	{"go.mod", "go", []string{"go"}},
	{"Cargo.toml", "rust", []string{"rust"}},
	{"pyproject.toml", "python", []string{"python"}},
	{"requirements.txt", "python", []string{"python"}},
	{"Gemfile", "ruby", []string{"ruby"}},
	{"pom.xml", "java", []string{"java", "maven"}},
	{"build.gradle", "java", []string{"java", "gradle"}},
	{"build.sbt", "scala", []string{"scala"}},
	{"mix.exs", "elixir", []string{"elixir"}},
	{"composer.json", "php", []string{"php"}},
	{"Makefile", "make", []string{"make"}},
}

var initResearchCmd = &cobra.Command{
	Use:   "init-research",
	Short: "Perform initial research for colony setup",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := mustGetString(cmd, "goal")
		if goal == "" {
			return nil
		}

		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			target = "."
		}

		languages := []string{}
		frameworks := []string{}
		detected := ""

		entries, err := os.ReadDir(target)
		if err != nil {
			outputError(1, "failed to read directory", nil)
			return nil
		}

		entryNames := make(map[string]bool)
		for _, e := range entries {
			if !e.IsDir() {
				entryNames[e.Name()] = true
			}
		}

		seenTypes := make(map[string]bool)
		seenFrameworks := make(map[string]bool)

		for _, det := range projectDetectors {
			if entryNames[det.file] {
				if !seenTypes[det.typ] {
					languages = append(languages, det.typ)
					seenTypes[det.typ] = true
				}
				if detected == "" {
					detected = det.typ
				}
				for _, fw := range det.frameworks {
					if !seenFrameworks[fw] {
						frameworks = append(frameworks, fw)
						seenFrameworks[fw] = true
					}
				}
			}
		}

		// Normalize detected type
		if detected == "" {
			detected = "unknown"
		}

		outputOK(map[string]interface{}{
			"detected_type": detected,
			"languages":     languages,
			"frameworks":    frameworks,
			"goal":          goal,
		})
		return nil
	},
}

func init() {
	initResearchCmd.Flags().String("goal", "", "Colony goal (required)")
	initResearchCmd.Flags().String("target", "", "Directory to scan (default: current directory)")

	rootCmd.AddCommand(initResearchCmd)
}

// hasSuffix checks if s has any of the given suffixes.
func hasSuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// resolveFileList expands a glob pattern and returns matching file paths.
func resolveFileList(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}
