package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}
	if rootCmd.Use != "aether" {
		t.Errorf("rootCmd.Use = %q, want \"aether\"", rootCmd.Use)
	}
}

func TestVersionFlag(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "aether") {
		t.Errorf("--version output %q does not contain \"aether\"", output)
	}
	if !strings.Contains(output, Version) {
		t.Errorf("--version output %q does not contain version %q", output, Version)
	}
}

func TestHelpFlag(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	output := buf.String()
	// Cobra uses "Available Commands" in help output
	if !strings.Contains(output, "Usage") && !strings.Contains(output, "Available Commands") {
		t.Errorf("--help output does not contain usage information: %q", output)
	}
}

func TestPersistentPreRunStoreInit(t *testing.T) {
	// Create a temp directory with .aether/data/
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create temp data dir: %v", err)
	}

	// Set environment to point to temp directory
	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	// Reset store for test isolation
	store = nil

	// Create a test command that requires store
	testCmd := &cobra.Command{
		Use: "test-store-init",
		RunE: func(cmd *cobra.Command, args []string) error {
			if store == nil {
				return errStoreNil
			}
			return nil
		},
	}
	_ = cobra.Command{}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	rootCmd.SetArgs([]string{"test-store-init"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	// The test command should succeed if store was initialized
	if err != nil {
		t.Errorf("store init test command failed: %v", err)
	}

	if store == nil {
		t.Error("store was not initialized by PersistentPreRunE")
	}

	if store != nil && store.BasePath() != dataDir {
		t.Errorf("store.BasePath() = %q, want %q", store.BasePath(), dataDir)
	}
}

// errStoreNil is a sentinel error for testing.
var errStoreNil = func() error {
	return os.ErrNotExist
}()
