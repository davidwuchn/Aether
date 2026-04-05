package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/aether-colony/aether/pkg/storage"
)

func TestFlagsListJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"flag-list", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --json returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	flags, ok := result["flags"].([]interface{})
	if !ok {
		t.Fatalf("result.flags is not an array, got: %T", result["flags"])
	}
	if len(flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(flags))
	}
}

func TestFlagsListJSONEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	// Create store with no flags file
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	os.WriteFile(dataDir+"/COLONY_STATE.json", []byte(state), 0644)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	s, _ := storage.NewStore(dataDir)
	store = s

	rootCmd.SetArgs([]string{"flag-list", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --json with no flags returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result := envelope["result"].(map[string]interface{})
	flags := result["flags"].([]interface{})
	if len(flags) != 0 {
		t.Errorf("expected 0 flags for empty case, got %d", len(flags))
	}
}

func TestFlagsList(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"flag-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list returned error: %v", err)
	}

	output := buf.String()
	// Testdata has 3 flags: blocker, issue, note
	if !strings.Contains(output, "flag_001") {
		t.Errorf("expected flag_001, got: %s", output)
	}
	if !strings.Contains(output, "flag_002") {
		t.Errorf("expected flag_002, got: %s", output)
	}
	if !strings.Contains(output, "Critical dependency missing") {
		t.Errorf("expected flag description, got: %s", output)
	}
}

func TestFlagsAlias(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"flags"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flags alias returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "flag_001") {
		t.Errorf("expected flag_001 via alias, got: %s", output)
	}
}

func TestFlagsFilterByType(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"flag-list", "--type", "blocker"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --type blocker returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "flag_001") {
		t.Errorf("expected blocker flag_001, got: %s", output)
	}
	if strings.Contains(output, "flag_002") {
		t.Errorf("did not expect issue flag_002 when filtering by blocker, got: %s", output)
	}
	if strings.Contains(output, "flag_003") {
		t.Errorf("did not expect note flag_003 when filtering by blocker, got: %s", output)
	}
}

func TestFlagsFilterByStatus(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := setupTestStore(t)
	defer os.RemoveAll(tmpDir)

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	store = s

	rootCmd.SetArgs([]string{"flag-list", "--status", "resolved"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --status resolved returned error: %v", err)
	}

	output := buf.String()
	// Only flag_003 is resolved
	if !strings.Contains(output, "flag_003") {
		t.Errorf("expected resolved flag_003, got: %s", output)
	}
	if strings.Contains(output, "flag_001") {
		t.Errorf("did not expect active flag_001 when filtering by resolved, got: %s", output)
	}
}
