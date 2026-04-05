package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestWorktreeMerge_MissingBranch(t *testing.T) {
	var stderrBuf bytes.Buffer
	stderr = &stderrBuf
	defer func() { stderr = os.Stderr }()

	rootCmd.SetArgs([]string{"worktree-merge"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	got := strings.TrimSpace(stderrBuf.String())
	if !strings.Contains(got, `"ok":false`) {
		t.Errorf("expected ok:false in output, got: %q", got)
	}
	if !strings.Contains(got, "flag --branch is required") {
		t.Errorf("expected 'flag --branch is required' in output, got: %q", got)
	}

	// Verify valid JSON
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %q", err, got)
	}
}

func TestWorktreeMerge_DirtyWorktree(t *testing.T) {
	var stderrBuf bytes.Buffer
	stderr = &stderrBuf
	defer func() { stderr = os.Stderr }()

	// Create a temp git repo with a dirty working tree
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "checkout", "-b", "main")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Create initial commit on main
	writeFile(t, tmpDir+"/README.md", "hello")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Create a branch with a file
	runGit(t, tmpDir, "checkout", "-b", "test-dirty-branch")
	writeFile(t, tmpDir+"/feature.go", "package main")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "feature")

	// Add an uncommitted change (making it dirty)
	writeFile(t, tmpDir+"/dirty.go", "dirty content")
	runGit(t, tmpDir, "add", "dirty.go")

	// Switch back to main -- staged dirty.go carries over
	runGit(t, tmpDir, "checkout", "main")

	// Set AETHER_ROOT to our test repo
	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "test-dirty-branch"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	got := strings.TrimSpace(stderrBuf.String())
	if !strings.Contains(got, `"ok":false`) {
		t.Errorf("expected ok:false, got: %q", got)
	}
	if !strings.Contains(got, "dirty worktree") {
		t.Errorf("expected 'dirty worktree' in error, got: %q", got)
	}
}

func TestWorktreeMerge_ConflictDetection(t *testing.T) {
	var stderrBuf bytes.Buffer
	stderr = &stderrBuf
	defer func() { stderr = os.Stderr }()

	// Create a temp git repo with conflicting changes
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "checkout", "-b", "main")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Create initial commit with a shared file
	writeFile(t, tmpDir+"/shared.go", "package main\n\nfunc main() {}")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Create a branch and modify the shared file
	runGit(t, tmpDir, "checkout", "-b", "conflict-branch")
	writeFile(t, tmpDir+"/shared.go", "package main\n\nfunc main() { /* branch */ }")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "branch change")

	// Switch to main and modify the same file differently
	runGit(t, tmpDir, "checkout", "main")
	writeFile(t, tmpDir+"/shared.go", "package main\n\nfunc main() { /* main */ }")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "main change")

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "conflict-branch"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	got := strings.TrimSpace(stderrBuf.String())
	if !strings.Contains(got, `"ok":false`) {
		t.Errorf("expected ok:false, got: %q", got)
	}
	if !strings.Contains(got, "conflict") {
		t.Errorf("expected 'conflict' in error, got: %q", got)
	}
}

func TestWorktreeMerge_SuccessfulMerge(t *testing.T) {
	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf
	defer func() { stdout = os.Stdout }()

	// Create a temp git repo with a clean branch that has extra commits
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "checkout", "-b", "main")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Create initial commit on main
	writeFile(t, tmpDir+"/README.md", "hello")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Create a branch with a new file (no conflicts)
	runGit(t, tmpDir, "checkout", "-b", "feature-branch")
	writeFile(t, tmpDir+"/new_feature.go", "package main\n\nfunc NewFeature() {}")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add new feature")

	// Switch back to main
	runGit(t, tmpDir, "checkout", "main")

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "feature-branch"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	got := strings.TrimSpace(stdoutBuf.String())
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %q", err, got)
	}

	if m["ok"] != true {
		t.Errorf("expected ok:true, got: %v", m["ok"])
	}

	result, ok := m["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	if result["merged"] != true {
		t.Errorf("expected merged:true, got: %v", result["merged"])
	}
	if result["branch"] != "feature-branch" {
		t.Errorf("expected branch 'feature-branch', got: %v", result["branch"])
	}
	if result["target"] != "main" {
		t.Errorf("expected target 'main', got: %v", result["target"])
	}
	sha, ok := result["sha"].(string)
	if !ok || sha == "" {
		t.Errorf("expected non-empty sha string, got: %v", result["sha"])
	}
}

func TestDataConflictPreference(t *testing.T) {
	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf
	defer func() { stdout = os.Stdout }()

	// Create a temp git repo. On main: commit .aether/data/test.json with {"main":true}.
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "checkout", "-b", "main")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Initial commit with a README
	writeFile(t, tmpDir+"/README.md", "hello")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Add .aether/data/test.json on main with {"main":true}
	if err := os.MkdirAll(tmpDir+"/.aether/data", 0755); err != nil {
		t.Fatalf("failed to create .aether/data: %v", err)
	}
	writeFile(t, tmpDir+"/.aether/data/test.json", `{"main":true}`)
	runGit(t, tmpDir, "add", ".aether/data/test.json")
	runGit(t, tmpDir, "commit", "-m", "add data on main")

	// Create branch and modify .aether/data/test.json to {"branch":true}
	runGit(t, tmpDir, "checkout", "-b", "data-conflict-branch")
	writeFile(t, tmpDir+"/.aether/data/test.json", `{"branch":true}`)
	runGit(t, tmpDir, "add", ".aether/data/test.json")
	runGit(t, tmpDir, "commit", "-m", "modify data on branch")

	// Switch back to main
	runGit(t, tmpDir, "checkout", "main")

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "data-conflict-branch"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	// Verify merge succeeded
	got := strings.TrimSpace(stdoutBuf.String())
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %q", err, got)
	}
	if m["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", m["ok"])
	}

	// After merge, .aether/data/test.json should contain {"main":true} (target wins)
	content, err := os.ReadFile(tmpDir + "/.aether/data/test.json")
	if err != nil {
		t.Fatalf("failed to read .aether/data/test.json: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, `"main":true`) {
		t.Errorf("expected .aether/data/test.json to contain main version, got: %s", contentStr)
	}
	if strings.Contains(contentStr, `"branch":true`) {
		t.Errorf("expected .aether/data/test.json NOT to contain branch version, but it does: %s", contentStr)
	}
}

func TestDataConflictMergeSuccess(t *testing.T) {
	var stdoutBuf bytes.Buffer
	stdout = &stdoutBuf
	defer func() { stdout = os.Stdout }()

	// Create a temp git repo where only the branch has .aether/data/new.json.
	tmpDir := t.TempDir()
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "checkout", "-b", "main")
	runGit(t, tmpDir, "config", "user.email", "test@test.com")
	runGit(t, tmpDir, "config", "user.name", "Test")

	// Initial commit on main (no .aether/data/)
	writeFile(t, tmpDir+"/README.md", "hello")
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Create branch and add .aether/data/new.json
	runGit(t, tmpDir, "checkout", "-b", "data-new-branch")
	if err := os.MkdirAll(tmpDir+"/.aether/data", 0755); err != nil {
		t.Fatalf("failed to create .aether/data: %v", err)
	}
	writeFile(t, tmpDir+"/.aether/data/new.json", `{"new":true}`)
	runGit(t, tmpDir, "add", ".aether/data/new.json")
	runGit(t, tmpDir, "commit", "-m", "add new data on branch")

	// Switch back to main
	runGit(t, tmpDir, "checkout", "main")

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	rootCmd.SetArgs([]string{"worktree-merge", "--branch", "data-new-branch"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	// Verify merge succeeded
	got := strings.TrimSpace(stdoutBuf.String())
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %q", err, got)
	}
	if m["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", m["ok"])
	}

	// .aether/data/new.json should exist with correct content (branch-only file preserved)
	content, err := os.ReadFile(tmpDir + "/.aether/data/new.json")
	if err != nil {
		t.Fatalf("expected .aether/data/new.json to exist after merge, but got error: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, `"new":true`) {
		t.Errorf("expected .aether/data/new.json to contain {\"new\":true}, got: %s", contentStr)
	}
}

// --- Test helpers ---

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\noutput: %s", args, err, output)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
