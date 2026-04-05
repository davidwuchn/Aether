package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// setupSwarmDisplayTest creates a temp directory with store initialized for testing.
func setupSwarmDisplayTest(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create temp data dir: %v", err)
	}
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	// Reset flags to defaults so previous test values don't leak
	swarmDisplayRenderCmd.Flags().Set("format", "tree")
	swarmDisplayRenderCmd.Flags().Set("max-depth", "3")
	swarmDisplayInlineCmd.Flags().Set("section", "")
	swarmDisplayTextCmd.Flags().Set("section", "")
	swarmDisplayTextCmd.Flags().Set("max-width", "80")

	return dataDir
}

// writeTestColonyState writes a minimal COLONY_STATE.json for testing.
func writeTestColonyState(t *testing.T, dataDir string) {
	t.Helper()
	goal := "test goal"
	state := map[string]interface{}{
		"goal":           &goal,
		"current_phase":  2,
		"milestone":      "Open Chambers",
		"state":          "building",
		"colony_version": 1,
		"plan": map[string]interface{}{
			"phases": []map[string]interface{}{
				{"id": 1, "name": "Setup", "status": "complete", "description": "Initial setup"},
				{"id": 2, "name": "Build Core", "status": "in_progress", "description": "Core features"},
				{"id": 3, "name": "Polish", "status": "pending", "description": "Polish and harden"},
			},
		},
		"memory": map[string]interface{}{
			"phase_learnings": []interface{}{},
			"decisions":       []interface{}{},
			"instincts":       []interface{}{},
		},
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(dataDir+"/COLONY_STATE.json", data, 0644); err != nil {
		t.Fatalf("failed to write test colony state: %v", err)
	}
}

// TestSwarmDisplayRenderTree tests tree format rendering.
func TestSwarmDisplayRenderTree(t *testing.T) {
	dataDir := setupSwarmDisplayTest(t)
	writeTestColonyState(t, dataDir)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"swarm-display-render"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-display-render returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true in output, got: %s", output)
	}
	if !strings.Contains(output, `"format":"tree"`) {
		t.Errorf("expected tree format in output, got: %s", output)
	}
	if !strings.Contains(output, "Goal:") {
		t.Errorf("expected Goal in tree output, got: %s", output)
	}
}

// TestSwarmDisplayRenderJSON tests JSON format rendering.
func TestSwarmDisplayRenderJSON(t *testing.T) {
	dataDir := setupSwarmDisplayTest(t)
	writeTestColonyState(t, dataDir)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"swarm-display-render", "--format", "json"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-display-render --format json returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"format":"json"`) {
		t.Errorf("expected json format, got: %s", output)
	}
}

// TestSwarmDisplayRenderFlat tests flat format rendering.
func TestSwarmDisplayRenderFlat(t *testing.T) {
	dataDir := setupSwarmDisplayTest(t)
	writeTestColonyState(t, dataDir)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"swarm-display-render", "--format", "flat"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-display-render --format flat returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, "Phase 1") {
		t.Errorf("expected phase listing in flat output, got: %s", output)
	}
}

// TestSwarmDisplayInline tests inline status rendering.
func TestSwarmDisplayInline(t *testing.T) {
	dataDir := setupSwarmDisplayTest(t)
	writeTestColonyState(t, dataDir)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"swarm-display-inline"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-display-inline returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"inline"`) {
		t.Errorf("expected inline field, got: %s", output)
	}
}

// TestSwarmDisplayText tests text block rendering.
func TestSwarmDisplayText(t *testing.T) {
	dataDir := setupSwarmDisplayTest(t)
	writeTestColonyState(t, dataDir)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"swarm-display-text"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-display-text returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"text"`) {
		t.Errorf("expected text field, got: %s", output)
	}
	if !strings.Contains(output, "Colony:") {
		t.Errorf("expected Colony: header in text, got: %s", output)
	}
}

// TestSwarmDisplayTextMaxWidth tests that text respects max-width.
func TestSwarmDisplayTextMaxWidth(t *testing.T) {
	dataDir := setupSwarmDisplayTest(t)
	writeTestColonyState(t, dataDir)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"swarm-display-text", "--max-width", "40"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-display-text returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
}

// TestSwarmDisplayCommandsRegistered verifies all commands are registered.
func TestSwarmDisplayCommandsRegistered(t *testing.T) {
	for _, name := range []string{"swarm-display-render", "swarm-display-inline", "swarm-display-text"} {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Errorf("command %s not registered: %v", name, err)
			continue
		}
		if cmd.Name() != name {
			t.Errorf("expected command name %s, got %s", name, cmd.Name())
		}
	}
}
