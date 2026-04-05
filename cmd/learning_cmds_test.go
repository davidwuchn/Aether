package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLearningInject(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Write an empty observations file to store path
	writeTestJSON(t, store.BasePath(), "learning-observations.json",
		map[string]interface{}{"observations": []interface{}{}})

	rootCmd.SetArgs([]string{"learning-inject",
		"--category", "testing",
		"--content", "test observation content",
		"--trust-score", "0.8",
		"--source", "unit-test",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("learning-inject failed: %v", err)
	}

	// Verify the observation was written
	data, err := os.ReadFile(filepath.Join(store.BasePath(), "learning-observations.json"))
	if err != nil {
		t.Fatalf("reading observations file: %v", err)
	}
	var file map[string]interface{}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parsing observations file: %v", err)
	}
	obs := file["observations"].([]interface{})
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}
	obsMap := obs[0].(map[string]interface{})
	if obsMap["content"] != "test observation content" {
		t.Errorf("expected content 'test observation content', got %v", obsMap["content"])
	}
}

func TestLearningDisplayProposals(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	// Create observations file with a proposed observation
	obsData := map[string]interface{}{
		"observations": []interface{}{
			map[string]interface{}{
				"content_hash": "obs_123",
				"content":      "proposed observation",
				"wisdom_type":  "testing",
				"source_type":  "proposed",
			},
			map[string]interface{}{
				"content_hash": "obs_456",
				"content":      "approved observation",
				"wisdom_type":  "testing",
				"source_type":  "approved",
			},
		},
	}
	writeTestJSON(t, store.BasePath(), "learning-observations.json", obsData)

	rootCmd.SetArgs([]string{"learning-display-proposals"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("learning-display-proposals failed: %v", err)
	}

	// Parse output to verify it only returns proposed observations
	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	proposals := result["proposals"].([]interface{})
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}
	p := proposals[0].(map[string]interface{})
	if p["content_hash"] != "obs_123" {
		t.Errorf("expected content_hash 'obs_123', got %v", p["content_hash"])
	}
}

func TestLearningPromote(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	// Create observations file
	obsData := map[string]interface{}{
		"observations": []interface{}{
			map[string]interface{}{
				"content_hash": "obs_999",
				"content":      "test observation",
				"wisdom_type":  "testing",
				"source_type":  "observation",
			},
		},
	}
	writeTestJSON(t, store.BasePath(), "learning-observations.json", obsData)

	// Create a minimal COLONY_STATE.json
	colonyData := map[string]interface{}{
		"version":        "2.0",
		"state":          "READY",
		"current_phase":  1,
		"plan":           map[string]interface{}{"phases": []interface{}{}},
		"memory":         map[string]interface{}{"phase_learnings": []interface{}{}, "decisions": []interface{}{}, "instincts": []interface{}{}},
		"errors":         map[string]interface{}{"records": []interface{}{}, "flagged_patterns": []interface{}{}},
		"signals":        []interface{}{},
		"graveyards":     []interface{}{},
		"events":         []interface{}{},
	}
	writeTestJSON(t, store.BasePath(), "COLONY_STATE.json", colonyData)

	rootCmd.SetArgs([]string{"learning-promote", "obs_999"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("learning-promote failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["promoted"] != true {
		t.Errorf("expected promoted true, got %v", result["promoted"])
	}
}

func TestLearningApproveProposals(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	obsData := map[string]interface{}{
		"observations": []interface{}{
			map[string]interface{}{
				"content_hash": "obs_a1",
				"content":      "proposed 1",
				"wisdom_type":  "testing",
				"source_type":  "proposed",
			},
			map[string]interface{}{
				"content_hash": "obs_a2",
				"content":      "proposed 2",
				"wisdom_type":  "testing",
				"source_type":  "proposed",
			},
		},
	}
	writeTestJSON(t, store.BasePath(), "learning-observations.json", obsData)

	rootCmd.SetArgs([]string{"learning-approve-proposals", "--all"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("learning-approve-proposals failed: %v", err)
	}

	// Verify both are now approved
	data, _ := os.ReadFile(filepath.Join(store.BasePath(), "learning-observations.json"))
	var file map[string]interface{}
	json.Unmarshal(data, &file)
	obs := file["observations"].([]interface{})
	for _, o := range obs {
		om := o.(map[string]interface{})
		if om["source_type"] != "approved" {
			t.Errorf("expected source_type 'approved', got %v", om["source_type"])
		}
	}
}

// writeTestJSON writes a JSON file to the specified directory.
func writeTestJSON(t *testing.T, dir, name string, data interface{}) {
	t.Helper()
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("marshaling %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), append(encoded, '\n'), 0644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}
