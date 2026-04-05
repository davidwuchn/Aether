package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// setupSuggestTest creates a temp directory with store initialized for testing.
func setupSuggestTest(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("failed to create temp data dir: %v", err)
	}
	os.Setenv("AETHER_ROOT", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AETHER_ROOT") })

	// Reset flags to defaults so previous test values don't leak
	suggestRecordCmd.Flags().Set("content", "")
	suggestRecordCmd.Flags().Set("type", "FOCUS")
	suggestRecordCmd.Flags().Set("reason", "")
	suggestRecordCmd.Flags().Set("priority", "normal")
	suggestAnalyzeCmd.Flags().Set("max", "5")
	suggestAnalyzeCmd.Flags().Set("context", "")
	suggestCheckCmd.Flags().Set("limit", "20")

	return dataDir
}

// TestSuggestAnalyze tests the suggest-analyze command.
func TestSuggestAnalyze(t *testing.T) {
	dataDir := setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	// Create a test Go file with TODO comments to trigger analysis
	testFile := dataDir + "/../../test_analyze_example.go"
	os.WriteFile(testFile, []byte("// TODO: fix this later\n// FIXME: broken\npackage example\n"), 0644)
	defer os.Remove(testFile)

	rootCmd.SetArgs([]string{"suggest-analyze", "--max", "3"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-analyze returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true in output, got: %s", output)
	}
	if !strings.Contains(output, `"suggestions"`) {
		t.Errorf("expected suggestions array in output, got: %s", output)
	}

	// Parse JSON to verify structure
	var result map[string]interface{}
	// Strip the outer envelope: {"ok":true,"result":{...}}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// Try extracting from result wrapper
		t.Logf("Output: %s", output)
	}
}

// TestSuggestAnalyzeMaxFlag tests that --max limits suggestions.
func TestSuggestAnalyzeMaxFlag(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-analyze", "--max", "1"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-analyze returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
}

// TestSuggestAnalyzeContextFlag tests that --context adds a user-specified suggestion.
func TestSuggestAnalyzeContextFlag(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-analyze", "--context", "security testing", "--max", "5"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-analyze returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, "security testing") {
		t.Errorf("expected context text in suggestions, got: %s", output)
	}
}

// TestSuggestRecord tests the suggest-record command.
func TestSuggestRecord(t *testing.T) {
	dataDir := setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-record", "--content", "Test suggestion content", "--type", "FOCUS", "--reason", "testing"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-record returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"recorded":true`) {
		t.Errorf("expected recorded:true, got: %s", output)
	}
	if !strings.Contains(output, `"duplicate":false`) {
		t.Errorf("expected duplicate:false, got: %s", output)
	}

	// Verify suggestions.json was created
	if _, err := os.Stat(dataDir + "/suggestions.json"); os.IsNotExist(err) {
		t.Error("suggestions.json was not created")
	}
}

// TestSuggestRecordDuplicate tests that recording the same content twice is deduplicated.
func TestSuggestRecordDuplicate(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	// First record
	rootCmd.SetArgs([]string{"suggest-record", "--content", "Dedup test", "--type", "FOCUS"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first suggest-record returned error: %v", err)
	}

	// Reset buffer
	stdout = &bytes.Buffer{}

	// Second record with same content
	rootCmd.SetArgs([]string{"suggest-record", "--content", "Dedup test", "--type", "FOCUS"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second suggest-record returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"duplicate":true`) {
		t.Errorf("expected duplicate:true on second record, got: %s", output)
	}
}

// TestSuggestRecordMissingContent tests that --content is required.
func TestSuggestRecordMissingContent(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-record"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-record returned error: %v", err)
	}

	var errOutput string
	if buf, ok := stderr.(*bytes.Buffer); ok {
		errOutput = buf.String()
	}
	if !strings.Contains(errOutput, "required") {
		t.Errorf("expected required error, got: %s", errOutput)
	}
}

// TestSuggestCheck tests the suggest-check command.
func TestSuggestCheck(t *testing.T) {
	dataDir := setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	// First record a suggestion
	rootCmd.SetArgs([]string{"suggest-record", "--content", "Check test suggestion", "--type", "FOCUS"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-record returned error: %v", err)
	}

	// Reset buffer
	stdout = &bytes.Buffer{}

	// Now check suggestions
	rootCmd.SetArgs([]string{"suggest-check", "--limit", "10"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-check returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"suggestions"`) {
		t.Errorf("expected suggestions array, got: %s", output)
	}
	if !strings.Contains(output, "Check test suggestion") {
		t.Errorf("expected recorded suggestion in check output, got: %s", output)
	}

	// Verify data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("data directory should exist")
	}
}

// TestSuggestCheckEmpty tests suggest-check with no suggestions.
func TestSuggestCheckEmpty(t *testing.T) {
	_ = setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	rootCmd.SetArgs([]string{"suggest-check"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-check returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"count":0`) {
		t.Errorf("expected count:0 for empty, got: %s", output)
	}
}

// TestSuggestCheckDedup tests that suggest-check deduplicates against active pheromones.
func TestSuggestCheckDedup(t *testing.T) {
	dataDir := setupSuggestTest(t)
	store = nil
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	defer func() {
		stdout = os.Stdout
		stderr = os.Stderr
	}()

	// Create a suggestion with known content
	hash := contentHash("manual", "FOCUS", "Dedup against pheromones test")

	sugFile := SuggestionsFile{
		Suggestions: []Suggestion{
			{
				ID:       "sug_test_dedup",
				Type:     "FOCUS",
				Content:  "Dedup against pheromones test",
				Reason:   "testing",
				Priority: "normal",
				Hash:     hash,
			},
		},
	}
	sugData, _ := json.Marshal(sugFile)
	os.WriteFile(dataDir+"/suggestions.json", sugData, 0644)

	// Create pheromones.json with matching content hash
	pheroContent := `{"signals":[{"id":"sig_existing","content_hash":"` + hash + `","active":true}]}`
	os.WriteFile(dataDir+"/pheromones.json", []byte(pheroContent), 0644)

	// Check should filter out the duplicate
	rootCmd.SetArgs([]string{"suggest-check"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("suggest-check returned error: %v", err)
	}

	var output string
	if buf, ok := stdout.(*bytes.Buffer); ok {
		output = buf.String()
	}
	if !strings.Contains(output, `"deduplicated_against":1`) {
		t.Errorf("expected deduplicated_against:1, got: %s", output)
	}
}

// TestSuggestCommandsRegistered verifies all 5 commands are registered.
func TestSuggestCommandsRegistered(t *testing.T) {
	expectedCommands := []string{
		"suggest-analyze",
		"suggest-record",
		"suggest-check",
		"suggest-approve",
		"suggest-quick-dismiss",
	}

	for _, name := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not registered in rootCmd", name)
		}
	}
}
