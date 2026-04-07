package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/tidwall/gjson"
)

func TestSetNestedFieldJSON_NumericValue(t *testing.T) {
	// Simulate a COLONY_STATE.json with a plan that has a confidence field.
	state := map[string]interface{}{
		"version": "1.0.0",
		"plan": map[string]interface{}{
			"confidence": 50.0,
			"phases": []interface{}{
				map[string]interface{}{
					"id":     1,
					"status": "pending",
				},
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	// Set plan.confidence to "85" via the --field path.
	// The current buggy code uses sjson.SetBytes which JSON-encodes the string,
	// producing "85" (a quoted string) instead of 85 (a number).
	result, err := setNestedFieldJSON(data, "plan.confidence", "85")
	if err != nil {
		t.Fatalf("setNestedFieldJSON returned error: %v", err)
	}

	// Assert that plan.confidence is numeric 85, not a quoted string.
	confidenceResult := gjson.GetBytes(result, "plan.confidence")
	if !confidenceResult.Exists() {
		t.Fatal("plan.confidence does not exist in result")
	}
	if confidenceResult.Type != gjson.Number {
		t.Errorf("plan.confidence should be a number, got type %v with raw value %q", confidenceResult.Type, confidenceResult.Raw)
	}
	if confidenceResult.Int() != 85 {
		t.Errorf("plan.confidence = %v, want 85", confidenceResult.Int())
	}
}

func TestSetNestedFieldJSON_DeepNestedPath(t *testing.T) {
	// Test setting a deeply nested field like plan.phases.0.status.
	state := map[string]interface{}{
		"version": "1.0.0",
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"id":     1,
					"status": "pending",
				},
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	result, err := setNestedFieldJSON(data, "plan.phases.0.status", "in-progress")
	if err != nil {
		t.Fatalf("setNestedFieldJSON returned error: %v", err)
	}

	statusResult := gjson.GetBytes(result, "plan.phases.0.status")
	if !statusResult.Exists() {
		t.Fatal("plan.phases.0.status does not exist in result")
	}
	if statusResult.Str != "in-progress" {
		t.Errorf("plan.phases.0.status = %q, want %q", statusResult.Str, "in-progress")
	}
	// Also verify it's a string, not a double-quoted string.
	if statusResult.Raw != `"in-progress"` {
		t.Errorf("plan.phases.0.status raw = %q, want %q", statusResult.Raw, `"in-progress"`)
	}
}

func TestSetNestedFieldJSON_NumericArrayElement(t *testing.T) {
	// Test setting an array element to a numeric value.
	state := map[string]interface{}{
		"version": "1.0.0",
		"plan": map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"id":  1,
					"seq": 0,
				},
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	result, err := setNestedFieldJSON(data, "plan.phases.0.seq", "42")
	if err != nil {
		t.Fatalf("setNestedFieldJSON returned error: %v", err)
	}

	seqResult := gjson.GetBytes(result, "plan.phases.0.seq")
	if !seqResult.Exists() {
		t.Fatal("plan.phases.0.seq does not exist in result")
	}
	if seqResult.Type != gjson.Number {
		t.Errorf("plan.phases.0.seq should be a number, got type %v with raw value %q", seqResult.Type, seqResult.Raw)
	}
	if seqResult.Int() != 42 {
		t.Errorf("plan.phases.0.seq = %v, want 42", seqResult.Int())
	}
}

func TestStateMutateBracket(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "build the thing"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: "pending"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// Bracket notation should set plan.phases[0].status to "completed".
	// Currently fails because reFieldSet regex only accepts [\w.]+
	// which does not include [ or ].
	rootCmd.SetArgs([]string{"state-mutate", `.plan.phases[0].status = "completed"`})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected state-mutate to succeed with bracket notation, got: %v", env)
	}

	// Verify the phase status was actually updated in the file.
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Plan.Phases) == 0 {
		t.Fatal("expected at least 1 phase in state")
	}
	if updated.Plan.Phases[0].Status != "completed" {
		t.Errorf("phases[0].status = %q, want %q", updated.Plan.Phases[0].Status, "completed")
	}
}

// --- audit tests for state-mutate ---

// readAuditChangelog reads the state-changelog.jsonl and returns parsed entries.
func readAuditChangelog(t *testing.T, s *storage.Store) []storage.AuditEntry {
	t.Helper()
	entries, err := storage.NewAuditLogger(s).ReadHistory(0)
	if err != nil {
		t.Fatalf("failed to read audit changelog: %v", err)
	}
	return entries
}

func TestStateMutateFieldProducesAuditEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "build the thing"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: "pending"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", "--field", "goal", "--value", "new goal"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	// Verify audit entry was created
	entries := readAuditChangelog(t, s)
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Command != "state-mutate" {
		t.Errorf("audit command = %q, want %q", entries[0].Command, "state-mutate")
	}
	if !strings.Contains(entries[0].Summary, "goal") {
		t.Errorf("audit summary = %q, want it to contain 'goal'", entries[0].Summary)
	}
	if entries[0].Checksum == "" {
		t.Error("audit entry should have a non-empty checksum")
	}
}

func TestStateMutateExpressionProducesAuditEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "build the thing"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: "pending"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", `.plan.phases[0].status = "completed"`})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	entries := readAuditChangelog(t, s)
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Command != "state-mutate" {
		t.Errorf("audit command = %q, want %q", entries[0].Command, "state-mutate")
	}
}

func TestStateMutateCorruptionProducesError(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "build the thing"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Events:  []string{`.goal = "injected"`},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", "--field", "milestone", "--value", "test"})

	rootCmd.Execute()

	// Should produce an error (corruption detected)
	errOutput := buf.String()
	if !strings.Contains(errOutput, "corruption") {
		t.Errorf("expected corruption error, got: %s", errOutput)
	}

	// No audit entry should be created for rejected mutations
	entries := readAuditChangelog(t, s)
	if len(entries) != 0 {
		t.Errorf("expected 0 audit entries for rejected mutation, got %d", len(entries))
	}
}

func TestStateMutatePhaseAdvanceCreatesCheckpoint(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "build the thing"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "phase one", Status: "in_progress"},
				{ID: 2, Name: "phase two", Status: "pending"},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", "--field", "current_phase", "--value", "2"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	// Verify audit entry with destructive=true
	entries := readAuditChangelog(t, s)
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if !entries[0].Destructive {
		t.Error("phase advance should produce destructive=true audit entry")
	}

	// Verify auto-checkpoint was created
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	checkpointDir := filepath.Join(dataDir, "checkpoints")
	entries2, err := os.ReadDir(checkpointDir)
	if err != nil {
		t.Fatalf("checkpoints directory not found: %v", err)
	}
	foundCheckpoint := false
	for _, e := range entries2 {
		if strings.HasPrefix(e.Name(), "auto-") {
			foundCheckpoint = true
			break
		}
	}
	if !foundCheckpoint {
		t.Error("expected auto-checkpoint file in checkpoints/ directory")
	}
}
