package cmd

import (
	"testing"
)

// TestCommandCount verifies that the CLI registers at least 145 commands.
// This serves as a regression guard ensuring no commands are accidentally
// dropped during development. If the count falls below 145, the test fails
// and prints all registered command names for debugging.
func TestCommandCount(t *testing.T) {
	commands := rootCmd.Commands()

	if len(commands) < 145 {
		t.Errorf("expected >= 145 commands registered, got %d", len(commands))
		t.Log("Registered commands:")
		for _, c := range commands {
			t.Logf("  - %s", c.Name())
		}
	}
}
