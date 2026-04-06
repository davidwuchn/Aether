package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestGenerateAntNameBuilder tests that generate-ant-name produces a valid name for builder caste.
// The Go command returns the name as a bare string in .result (matching shell parity).
func TestGenerateAntNameBuilder(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-ant-name", "builder"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-ant-name returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true in output, got: %s", output)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}

	name, ok := result["result"].(string)
	if !ok {
		t.Fatalf("expected result to be a string, got: %T", result["result"])
	}

	if !antNamePattern.MatchString(name) {
		t.Errorf("name %q does not match expected pattern ^[A-Z][a-z]+-\\d{1,2}$", name)
	}
}

// TestGenerateAntNameAllCastes tests that every caste produces a valid name.
// The Go command returns the name as a bare string in .result (matching shell parity).
func TestGenerateAntNameAllCastes(t *testing.T) {
	castes := []string{
		"builder", "watcher", "scout", "colonizer", "architect",
		"prime", "chaos", "archaeologist", "oracle", "ambassador",
		"auditor", "chronicler", "gatekeeper", "guardian", "includer",
		"keeper", "measurer", "probe", "tracker", "weaver",
	}

	for _, caste := range castes {
		t.Run(caste, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			stdout = &bytes.Buffer{}
			stderr = &bytes.Buffer{}
			store = nil

			rootCmd.SetArgs([]string{"generate-ant-name", caste})

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("generate-ant-name %s returned error: %v", caste, err)
			}

			output := stdout.(*bytes.Buffer).String()
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("failed to parse output: %v", err)
			}

			name, _ := result["result"].(string)

			if !antNamePattern.MatchString(name) {
				t.Errorf("caste %s produced invalid name %q", caste, name)
			}
		})
	}
}

// TestGenerateAntNameUnknownCaste tests that an unknown caste uses default prefixes.
func TestGenerateAntNameUnknownCaste(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-ant-name", "nonexistent"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-ant-name with unknown caste returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	name, _ := result["result"].(string)

	if !antNamePattern.MatchString(name) {
		t.Errorf("unknown caste produced invalid name %q", name)
	}
}

// TestGenerateAntNameDefaultCaste tests that no args still produces a valid name
// (defaults to builder prefixes internally).
func TestGenerateAntNameDefaultCaste(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-ant-name"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-ant-name with no args returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	name, _ := result["result"].(string)

	// Verify it produces a valid builder-prefixed name (default caste is builder)
	if !antNamePattern.MatchString(name) {
		t.Errorf("expected valid builder name, got %q", name)
	}
}

// TestGenerateAntNameWithSeed tests that --seed produces deterministic output.
func TestGenerateAntNameWithSeed(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-ant-name", "builder", "--seed", "42"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-ant-name --seed returned error: %v", err)
	}

	output1 := stdout.(*bytes.Buffer).String()
	var result1 map[string]interface{}
	json.Unmarshal([]byte(output1), &result1)
	name1, _ := result1["result"].(string)

	// Reset and run again with same seed
	stdout = &bytes.Buffer{}
	resetRootCmd(t)
	rootCmd.SetArgs([]string{"generate-ant-name", "builder", "--seed", "42"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second generate-ant-name --seed returned error: %v", err)
	}

	output2 := stdout.(*bytes.Buffer).String()
	var result2 map[string]interface{}
	json.Unmarshal([]byte(output2), &result2)
	name2, _ := result2["result"].(string)

	if name1 != name2 {
		t.Errorf("expected deterministic names with same seed, got %q and %q", name1, name2)
	}
}

// TestGenerateCommitMessage tests the commit message formatter.
func TestGenerateCommitMessage(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-commit-message", "--type", "feat", "--subject", "add new feature"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-commit-message returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)
	inner, _ := result["result"].(map[string]interface{})
	message, _ := inner["message"].(string)

	if message != "feat: add new feature" {
		t.Errorf("expected 'feat: add new feature', got %q", message)
	}
}

// TestGenerateCommitMessageWithScope tests commit message with scope.
func TestGenerateCommitMessageWithScope(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-commit-message", "--type", "fix", "--scope", "auth", "--subject", "fix login"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-commit-message returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)
	inner, _ := result["result"].(map[string]interface{})
	message, _ := inner["message"].(string)

	if message != "fix(auth): fix login" {
		t.Errorf("expected 'fix(auth): fix login', got %q", message)
	}
}

// TestGenerateCommitMessageWithBody tests commit message with body.
func TestGenerateCommitMessageWithBody(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-commit-message", "--type", "docs", "--subject", "update readme", "--body", "Added installation instructions"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-commit-message returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)
	inner, _ := result["result"].(map[string]interface{})
	message, _ := inner["message"].(string)

	expected := "docs: update readme\n\nAdded installation instructions"
	if message != expected {
		t.Errorf("expected %q, got %q", expected, message)
	}
}

// TestGenerateCommitMessageInvalidType tests that invalid type is rejected.
func TestGenerateCommitMessageInvalidType(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-commit-message", "--type", "invalid", "--subject", "test"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-commit-message returned error: %v", err)
	}

	errOutput := stderr.(*bytes.Buffer).String()
	if !strings.Contains(errOutput, "invalid type") {
		t.Errorf("expected invalid type error, got: %s", errOutput)
	}
}

// TestGenerateCommitMessageColonyTypes tests that colony-specific commit types
// (seal, milestone, pause, contextual) produce valid commit messages.
func TestGenerateCommitMessageColonyTypes(t *testing.T) {
	colonyTypes := []struct {
		typeVal string
		subject string
		scope   string
		want    string
	}{
		{"seal", "colony sealed at Crowned Anthill v1", "", "seal: colony sealed at Crowned Anthill v1"},
		{"milestone", "reached Open Chambers", "", "milestone: reached Open Chambers"},
		{"pause", "colony paused for replan", "", "pause: colony paused for replan"},
		{"contextual", "user preference captured", "prefs", "contextual(prefs): user preference captured"},
	}

	for _, tc := range colonyTypes {
		t.Run(tc.typeVal, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)

			stdout = &bytes.Buffer{}
			stderr = &bytes.Buffer{}
			store = nil

			args := []string{"generate-commit-message", "--type", tc.typeVal, "--subject", tc.subject}
			if tc.scope != "" {
				args = append(args, "--scope", tc.scope)
			}
			rootCmd.SetArgs(args)

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("generate-commit-message --type %s returned error: %v", tc.typeVal, err)
			}

			output := stdout.(*bytes.Buffer).String()
			if !strings.Contains(output, `"ok":true`) {
				errOut := stderr.(*bytes.Buffer).String()
				t.Errorf("expected ok:true for type %q, got stdout: %s, stderr: %s", tc.typeVal, output, errOut)
			}

			var result map[string]interface{}
			json.Unmarshal([]byte(output), &result)
			inner, _ := result["result"].(map[string]interface{})
			message, _ := inner["message"].(string)

			if message != tc.want {
				t.Errorf("type %q: expected %q, got %q", tc.typeVal, tc.want, message)
			}
		})
	}
}

// TestGenerateProgressBarCmd tests the progress bar command.
func TestGenerateProgressBarCmd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-progress-bar", "--current", "5", "--total", "10"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-progress-bar returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)
	inner, _ := result["result"].(map[string]interface{})

	bar, _ := inner["bar"].(string)
	pct, _ := inner["percentage"].(float64)

	if pct != 50 {
		t.Errorf("expected percentage 50, got %v", pct)
	}

	expectedBar := "[" + strings.Repeat("#", 15) + strings.Repeat("-", 15) + "]"
	if bar != expectedBar {
		t.Errorf("expected bar %q, got %q", expectedBar, bar)
	}
}

// TestGenerateProgressBarCmdFull tests progress bar at 100%.
func TestGenerateProgressBarCmdFull(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-progress-bar", "--current", "10", "--total", "10"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-progress-bar returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)
	inner, _ := result["result"].(map[string]interface{})

	bar, _ := inner["bar"].(string)
	pct, _ := inner["percentage"].(float64)

	if pct != 100 {
		t.Errorf("expected percentage 100, got %v", pct)
	}
	if !strings.Contains(bar, strings.Repeat("#", 30)) {
		t.Errorf("expected fully filled bar, got %q", bar)
	}
}

// TestGenerateProgressBarCmdZero tests progress bar at 0%.
func TestGenerateProgressBarCmdZero(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	store = nil

	rootCmd.SetArgs([]string{"generate-progress-bar", "--current", "0", "--total", "10"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate-progress-bar returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)
	inner, _ := result["result"].(map[string]interface{})

	bar, _ := inner["bar"].(string)
	pct, _ := inner["percentage"].(float64)

	if pct != 0 {
		t.Errorf("expected percentage 0, got %v", pct)
	}
	if !strings.Contains(bar, strings.Repeat("-", 30)) {
		t.Errorf("expected empty bar, got %q", bar)
	}
}

// TestGenerateCommandsRegistered verifies all generate commands are registered.
func TestGenerateCommandsRegistered(t *testing.T) {
	expectedCommands := []string{
		"generate-ant-name",
		"generate-commit-message",
		"generate-progress-bar",
		"generate-threshold-bar",
	}

	for _, name := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not registered in rootCmd", name)
		}
	}
}
