package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aether-colony/aether/pkg/colony"
)

// TestPheromoneExportXMLHelp verifies the pheromone-export-xml command exists and shows help.
func TestPheromoneExportXMLHelp(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	// Capture rootCmd output
	origOut := rootCmd.OutOrStdout()
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(origOut)

	rootCmd.SetArgs([]string{"pheromone-export-xml", "--help"})
	defer rootCmd.SetArgs([]string{})

	// --help causes a return error in cobra, which is fine
	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "pheromone-export-xml") {
		t.Errorf("expected help output to contain 'pheromone-export-xml', got: %s", output)
	}
}

// TestAllAliasCommandsExist verifies all 7 alias commands are registered.
func TestAllAliasCommandsExist(t *testing.T) {
	aliases := []string{
		"pheromone-export-xml",
		"pheromone-import-xml",
		"wisdom-export-xml",
		"wisdom-import-xml",
		"registry-export-xml",
		"registry-import-xml",
		"colony-archive-xml",
	}

	for _, name := range aliases {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name || strings.HasPrefix(cmd.Use, name+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("alias command %q not registered in rootCmd", name)
		}
	}
}

// TestPheromoneDisplayEmpty verifies pheromone-display works with no signals.
func TestPheromoneDisplayEmpty(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-display"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-display returned error: %v", err)
	}

	// Find the JSON envelope in output (may have display text before it)
	output := buf.String()
	var envelope map[string]interface{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &envelope); err == nil {
				break
			}
		}
	}

	if envelope == nil {
		t.Fatalf("no JSON envelope found in output: %q", output)
	}

	if ok, _ := envelope["ok"].(bool); !ok {
		t.Errorf("expected ok:true, got: %v", envelope)
	}

	result, _ := envelope["result"].(map[string]interface{})
	count, _ := result["count"].(float64)
	if count != 0 {
		t.Errorf("expected count 0 for empty pheromones, got: %v", count)
	}
}

// TestPheromoneDisplayWithSignals verifies pheromone-display shows signals in a table.
func TestPheromoneDisplayWithSignals(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Write test pheromone signals
	content, _ := json.Marshal(map[string]string{"text": "Focus on testing"})
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:       "sig-1",
				Type:     "FOCUS",
				Priority: "normal",
				Active:   true,
				Content:  content,
			},
			{
				ID:       "sig-2",
				Type:     "REDIRECT",
				Priority: "high",
				Active:   false,
				Content:  content,
			},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"pheromone-display"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-display returned error: %v", err)
	}

	output := buf.String()

	// Should contain the table header
	if !strings.Contains(output, "TYPE") || !strings.Contains(output, "CONTENT") {
		t.Errorf("expected table header in output, got: %s", output)
	}

	// Should show the active FOCUS signal
	if !strings.Contains(output, "FOCUS") {
		t.Errorf("expected FOCUS signal in output, got: %s", output)
	}

	// Should NOT show inactive REDIRECT (active-only defaults true)
	if strings.Contains(output, "REDIRECT") {
		t.Errorf("expected REDIRECT to be filtered out (inactive), got: %s", output)
	}
}
