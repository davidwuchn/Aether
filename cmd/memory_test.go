package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestMemoryMetrics(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf
	

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"memory-metrics"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("memory-metrics returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("memory-metrics produced invalid JSON: %v", err)
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}

	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Should have wisdom, pending, recent_failures, last_activity keys
	for _, key := range []string{"wisdom", "pending", "recent_failures", "last_activity"} {
		if _, exists := result[key]; !exists {
			t.Errorf("result missing key %q", key)
		}
	}
}

func TestMemoryMetricsEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf
	

	// Create empty store
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"memory-metrics"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("memory-metrics returned error on empty store: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
}

func TestColonyVitalSigns(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer

	stdout = &buf
	

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s
	rootCmd.SetArgs([]string{"colony-vital-signs"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("colony-vital-signs returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("colony-vital-signs produced invalid JSON: %v", err)
	}

	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Should have health metrics
	for _, key := range []string{"build_velocity", "error_rate", "signal_health", "memory_pressure", "overall_health", "health_label"} {
		if _, exists := result[key]; !exists {
			t.Errorf("result missing key %q", key)
		}
	}

	// Health label should be a known value
	label := result["health_label"].(string)
	validLabels := map[string]bool{"Thriving": true, "Healthy": true, "Stable": true, "Struggling": true, "Critical": true}
	if !validLabels[label] {
		t.Errorf("health_label = %q, want one of Thriving/Healthy/Stable/Struggling/Critical", label)
	}
}
