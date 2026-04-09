package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// colonyPrimeTestEnv creates a minimal colony state + optional pheromones
// and returns the store, temp dir, and a cleanup function.
func colonyPrimeTestEnv(t *testing.T, pheromones *colony.PheromoneFile) (*colony.ColonyState, *colony.PheromoneFile) {
	t.Helper()

	goal := "test pheromone injection"
	state := &colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress", Tasks: []colony.Task{{Status: "in_progress", Goal: "Build feature"}}},
			},
		},
	}

	if pheromones == nil {
		pheromones = &colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
	}

	return state, pheromones
}

// runColonyPrime executes colony-prime with the given flags and returns parsed output.
func runColonyPrime(t *testing.T, s interface{ SaveJSON(string, interface{}) error }, flags []string) map[string]interface{} {
	t.Helper()
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	args := append([]string{"colony-prime"}, flags...)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v (stderr: %s)", err, errBuf.String())
	}

	return parseEnvelopeCmd(t, buf.String())
}

// --- Test: colony-prime includes pheromone signals in assembled context ---

func TestColonyPrime_IncludesPheromoneSignals(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, pheromones := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pheromones.Signals = []colony.PheromoneSignal{
		{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Focus on error handling"}`)},
		{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Avoid global state"}`)},
		{ID: "s3", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Tests looking good"}`)},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Verify the pheromone section header appears
	if !strings.Contains(contextStr, "## Pheromone Signals") {
		t.Error("colony-prime context missing '## Pheromone Signals' section header")
	}

	// Verify all three signal type markers appear in the output
	for _, sigType := range []string{"FOCUS", "REDIRECT", "FEEDBACK"} {
		if !strings.Contains(contextStr, sigType) {
			t.Errorf("colony-prime context missing signal type marker '%s'", sigType)
		}
	}

	// Verify the actual signal content text appears
	if !strings.Contains(contextStr, "Focus on error handling") {
		t.Error("colony-prime context missing FOCUS signal content 'Focus on error handling'")
	}
	if !strings.Contains(contextStr, "Avoid global state") {
		t.Error("colony-prime context missing REDIRECT signal content 'Avoid global state'")
	}
	if !strings.Contains(contextStr, "Tests looking good") {
		t.Error("colony-prime context missing FEEDBACK signal content 'Tests looking good'")
	}

	// Verify the format: each signal should be formatted as [TYPE] content
	if !strings.Contains(contextStr, "[FOCUS] Focus on error handling") {
		t.Error("colony-prime context missing formatted FOCUS signal '[FOCUS] Focus on error handling'")
	}
	if !strings.Contains(contextStr, "[REDIRECT] Avoid global state") {
		t.Error("colony-prime context missing formatted REDIRECT signal '[REDIRECT] Avoid global state'")
	}
	if !strings.Contains(contextStr, "[FEEDBACK] Tests looking good") {
		t.Error("colony-prime context missing formatted FEEDBACK signal '[FEEDBACK] Tests looking good'")
	}

	// Verify sections count reflects pheromone section being present
	sections := result["sections"].(float64)
	if sections < 2 {
		t.Errorf("expected at least 2 sections (state + pheromones), got %f", sections)
	}
}

// --- Test: colony-prime with no pheromones shows no pheromone section ---

func TestColonyPrime_NoPheromones_NoPheromoneSection(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Context should contain the goal but no pheromone section
	if !strings.Contains(contextStr, "test pheromone injection") {
		t.Error("colony-prime context missing colony goal text")
	}
	if strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("colony-prime context should NOT contain 'Pheromone Signals' when no pheromones exist")
	}
}

// --- Test: colony-prime with only inactive signals shows no pheromone section ---

func TestColonyPrime_OnlyInactiveSignals_NoPheromoneSection(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: false, Strength: &s0_9, Content: json.RawMessage(`{"text": "Inactive focus"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: false, Strength: &s0_9, Content: json.RawMessage(`{"text": "Inactive redirect"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Inactive signals should not appear in context
	if strings.Contains(contextStr, "Inactive focus") {
		t.Error("colony-prime context should not contain inactive signal content 'Inactive focus'")
	}
	if strings.Contains(contextStr, "Inactive redirect") {
		t.Error("colony-prime context should not contain inactive signal content 'Inactive redirect'")
	}
	if strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("colony-prime context should not contain 'Pheromone Signals' section when all signals are inactive")
	}
}

// --- Test: colony-prime with mixed active and inactive signals includes only active ---

func TestColonyPrime_MixedActiveInactive_OnlyActiveIncluded(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Active focus signal"}`)},
			{ID: "s2", Type: "REDIRECT", Priority: "high", Source: "user", CreatedAt: now, Active: false, Strength: &s0_9, Content: json.RawMessage(`{"text": "Hidden redirect"}`)},
			{ID: "s3", Type: "FEEDBACK", Priority: "low", Source: "auto", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Active feedback signal"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// Active signals should be present
	if !strings.Contains(contextStr, "Active focus signal") {
		t.Error("colony-prime context missing active FOCUS signal content")
	}
	if !strings.Contains(contextStr, "Active feedback signal") {
		t.Error("colony-prime context missing active FEEDBACK signal content")
	}

	// Inactive signal should NOT be present
	if strings.Contains(contextStr, "Hidden redirect") {
		t.Error("colony-prime context should not contain inactive REDIRECT signal 'Hidden redirect'")
	}
}

// --- Test: colony-prime --compact flag works with pheromones ---

func TestColonyPrime_CompactFlag_IncludesPheromones(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Compact mode focus"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, []string{"--compact"})
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)
	budget := result["budget"].(float64)

	// Verify compact uses 4000 char budget
	if budget != 4000 {
		t.Errorf("expected budget=4000 with --compact, got %f", budget)
	}

	// Verify pheromones still appear in compact mode
	if !strings.Contains(contextStr, "Pheromone Signals") {
		t.Error("colony-prime --compact context missing 'Pheromone Signals' section")
	}
	if !strings.Contains(contextStr, "Compact mode focus") {
		t.Error("colony-prime --compact context missing signal content 'Compact mode focus'")
	}
}

// --- Test: colony-prime with expired pheromones (strength decayed to near zero) ---

func TestColonyPrime_OldSignals_StillAppear(t *testing.T) {
	// colony-prime does NOT apply strength decay -- it only checks sig.Active.
	// Signals created long ago but still marked Active should still appear.
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	// Create a signal from 60 days ago
	oldTime := time.Now().Add(-60 * 24 * time.Hour).Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: oldTime, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Old but active focus"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)

	// colony-prime only checks Active flag, not effective strength decay
	if !strings.Contains(contextStr, "Old but active focus") {
		t.Error("colony-prime should include old-but-still-Active signals in context")
	}
}

// --- Test: colony-prime default budget is 8000 ---

func TestColonyPrime_DefaultBudget(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)
	result := envelope["result"].(map[string]interface{})
	budget := result["budget"].(float64)

	if budget != 8000 {
		t.Errorf("expected default budget=8000, got %f", budget)
	}
}

// --- Test: colony-prime output structure is valid ---

func TestColonyPrime_OutputStructure(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	state, _ := colonyPrimeTestEnv(t, nil)
	now := time.Now().Format(time.RFC3339)
	s0_9 := 0.9

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "s1", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &s0_9, Content: json.RawMessage(`{"text": "Structure test"}`)},
		},
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	envelope := runColonyPrime(t, s, nil)

	// Envelope must have ok:true
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got %v", envelope["ok"])
	}

	result := envelope["result"].(map[string]interface{})

	// Required fields
	requiredFields := []string{"context", "budget", "used", "sections", "trimmed"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("colony-prime result missing required field '%s'", field)
		}
	}

	// context must be a non-empty string
	contextStr, ok := result["context"].(string)
	if !ok || contextStr == "" {
		t.Error("colony-prime result.context must be a non-empty string")
	}

	// trimmed should be an array (may be empty)
	_, ok = result["trimmed"].([]interface{})
	if !ok {
		t.Error("colony-prime result.trimmed must be an array")
	}
}
