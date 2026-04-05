package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitResearchGo(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "build CLI", "--target", projectRoot})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["detected_type"] != "go" {
		t.Errorf("detected_type = %v, want go", result["detected_type"])
	}
	if result["goal"] != "build CLI" {
		t.Errorf("goal = %v, want 'build CLI'", result["goal"])
	}
}

func TestInitResearchNode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "package.json"), []byte(`{"name":"test"}`), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "web app", "--target", projectRoot})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["detected_type"] != "node" {
		t.Errorf("detected_type = %v, want node", result["detected_type"])
	}
}

func TestInitResearchUnknown(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Use a clean empty temp dir as target
	emptyDir := t.TempDir()

	rootCmd.SetArgs([]string{"init-research", "--goal", "new project", "--target", emptyDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["detected_type"] != "unknown" {
		t.Errorf("detected_type = %v, want unknown", result["detected_type"])
	}
	langs := result["languages"].([]interface{})
	if len(langs) != 0 {
		t.Errorf("languages = %v, want empty", langs)
	}
}

func TestInitResearchMissingGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf
	

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"init-research", "--goal", ""})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing goal, got: %v", env["ok"])
	}
}
