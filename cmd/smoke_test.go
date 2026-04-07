package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// skipSmokeCommands lists commands that are long-running servers or otherwise
// unsuitable for a no-arg invocation in a test loop. These are tested
// individually elsewhere.
var skipSmokeCommands = map[string]bool{
	"serve": true, // starts HTTP server, blocks on ListenAndServe
}

// TestSmokeCommands verifies that every registered subcommand can run without
// panicking on a fresh install (no colony state). Commands may produce errors
// (missing required flags, no store data), but must not crash.
func TestSmokeCommands(t *testing.T) {
	commands := rootCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("no commands registered on rootCmd")
	}

	for _, cmd := range commands {
		cmd := cmd // capture range variable
		name := cmd.Name()
		if skipSmokeCommands[name] {
			t.Run(name, func(t *testing.T) {
				t.Skipf("skipping long-running command: %s", name)
			})
			continue
		}
		t.Run(name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			// Capture output
			var outBuf, errBuf bytes.Buffer
			stdout = &outBuf
			stderr = &errBuf
			defer func() {
				stdout = os.Stdout
				stderr = os.Stderr
			}()

			// Isolated temp directory with .aether/data subdirectory
			tmpDir := t.TempDir()
			dataDir := filepath.Join(tmpDir, ".aether", "data")
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				t.Fatalf("failed to create test data dir: %v", err)
			}

			// Point AETHER_ROOT to temp directory so PersistentPreRunE uses it
			origRoot := os.Getenv("AETHER_ROOT")
			t.Setenv("AETHER_ROOT", tmpDir)
			_ = origRoot // t.Setenv handles cleanup

			// Create a store so commands that check store != nil don't bail
			s, err := createTestStore(dataDir)
			if err != nil {
				t.Fatalf("failed to create test store: %v", err)
			}
			store = s

			// Catch panics -- commands must not crash
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PANIC running %q: %v", cmd.Name(), r)
				}
			}()

			// Execute with no arguments (simplest invocation)
			rootCmd.SetArgs([]string{cmd.Name()})
			_ = rootCmd.Execute()

			// Verify some output was produced (help, error, or result)
			output := outBuf.String() + errBuf.String()
			if output == "" && !cmd.HasSubCommands() {
				t.Logf("command %q produced no output (may be acceptable for help-only commands)", cmd.Name())
			}
		})
	}
}
