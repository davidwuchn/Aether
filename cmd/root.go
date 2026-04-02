// Package cmd implements the Aether CLI commands using Cobra.
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/aether-colony/aether/pkg/storage"
	"github.com/spf13/cobra"
)

// Version is set via -ldflags at build time.
var Version = "0.0.0-dev"

func init() {
	// Override Cobra's default version template to print "aether v<version>"
	// instead of "aether version v<version>"
	rootCmd.SetVersionTemplate("aether {{ .Version }}\n")
}

// store is the shared storage instance initialized by PersistentPreRunE.
// Commands that need data access should check this variable.
var store *storage.Store

// stdout and stderr are package-level writers that tests can override.
var stdout io.Writer = os.Stdout
var stderr io.Writer = os.Stderr

// rootCmd is the root Cobra command for the aether CLI.
var rootCmd = &cobra.Command{
	Use:   "aether",
	Short: "Aether Colony Utility Layer",
	// SilenceUsage and SilenceErrors prevent Cobra from printing
	// usage/error text automatically -- we control output ourselves.
	SilenceUsage:  true,
	SilenceErrors: true,
	// Custom version printer to match expected format "aether v<version>"
	Version:       "v" + Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip store initialization for commands that don't need it.
		if skipStoreInit(cmd) {
			return nil
		}

		dataDir := storage.ResolveDataDir()
		s, err := storage.NewStore(dataDir)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		store = s
		return nil
	},
}

// skipStoreInit returns true for commands that don't require a store
// (completion, version, help).
func skipStoreInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "completion", "version", "help":
			return true
		}
	}
	return false
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

// ExitWithError prints the error to stderr and exits with code 1.
func ExitWithError(err error) {
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err.Error())
	}
	os.Exit(1)
}
