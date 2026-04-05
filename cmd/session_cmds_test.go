package cmd

import (
	"bytes"
	"github.com/aether-colony/aether/pkg/storage"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// session-init tests
// ---------------------------------------------------------------------------

func TestSessionInit(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-init", "--goal", "test colony goal"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-init returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, "test colony goal") {
		t.Errorf("expected goal in output, got: %s", output)
	}

	// Verify session.json was written
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("session.json not written: %v", err)
	}
	if session.ColonyGoal != "test colony goal" {
		t.Errorf("session goal = %q, want %q", session.ColonyGoal, "test colony goal")
	}
	if session.SessionID == "" {
		t.Error("session_id should not be empty")
	}
	if session.StartedAt == "" {
		t.Error("started_at should not be empty")
	}
}

func TestSessionInitWithExplicitID(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-init", "--session-id", "custom-id-123", "--goal", "test"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-init returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom-id-123") {
		t.Errorf("expected custom session_id in output, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// session-read tests
// ---------------------------------------------------------------------------

func TestSessionRead(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	session := colony.SessionFile{
		SessionID:  "test-session-read",
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
		ColonyGoal: "read test goal",
	}
	if err := s.SaveJSON("session.json", session); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"session-read"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-read returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, "test-session-read") {
		t.Errorf("expected session_id in output, got: %s", output)
	}
}

func TestSessionReadMissing(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-read"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-read returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"exists":false`) {
		t.Errorf("expected exists:false for missing session, got: %s", output)
	}
}

func TestSessionReadStale(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	oldTime := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	session := colony.SessionFile{
		SessionID:     "stale-session",
		StartedAt:     oldTime,
		LastCommandAt: oldTime,
		ColonyGoal:    "stale test",
	}
	if err := s.SaveJSON("session.json", session); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"session-read"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-read returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"is_stale":true`) {
		t.Errorf("expected is_stale:true for old session, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// session-update tests
// ---------------------------------------------------------------------------

func TestSessionUpdate(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	session := colony.SessionFile{
		SessionID: "update-test",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.SaveJSON("session.json", session); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"session-update", "--command", "build", "--summary", "built phase 1"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-update returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}

	var updated colony.SessionFile
	if err := s.LoadJSON("session.json", &updated); err != nil {
		t.Fatal(err)
	}
	if updated.LastCommand != "build" {
		t.Errorf("last_command = %q, want %q", updated.LastCommand, "build")
	}
	if updated.Summary != "built phase 1" {
		t.Errorf("summary = %q, want %q", updated.Summary, "built phase 1")
	}
}

func TestSessionUpdateAutoInit(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-update", "--command", "status"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-update returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"auto_initialized":true`) {
		t.Errorf("expected auto_initialized:true, got: %s", output)
	}
}

func TestSessionUpdateRequiresCommand(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf
	defer func() { stdout = os.Stdout; stderr = os.Stderr }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-update"})
	defer rootCmd.SetArgs([]string{})

	rootCmd.Execute()

	combined := outBuf.String() + errBuf.String()
	if !strings.Contains(combined, "command") && !strings.Contains(combined, "required") {
		t.Errorf("expected error about missing --command, got stdout=%s stderr=%s", outBuf.String(), errBuf.String())
	}
}

// ---------------------------------------------------------------------------
// session-clear tests
// ---------------------------------------------------------------------------

func TestSessionClearDryRun(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	oracleDir := tmpDir + "/.aether/oracle"
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(oracleDir+"/state.json", []byte("{}"), 0644)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-clear", "--command", "oracle", "--dry-run"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-clear returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"dry_run":true`) {
		t.Errorf("expected dry_run:true, got: %s", output)
	}

	// Verify file was NOT deleted (dry run)
	if _, err := os.Stat(oracleDir + "/state.json"); err != nil {
		t.Error("state.json should not be deleted in dry-run mode")
	}
}

func TestSessionClearProtectedCommand(t *testing.T) {
	var buf bytes.Buffer
	stderr = &buf
	defer func() { stderr = os.Stderr }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-clear", "--command", "init"})
	defer rootCmd.SetArgs([]string{})

	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "protected") {
		t.Errorf("expected protected error for init, got: %s", output)
	}
}

func TestSessionClearUnknownCommand(t *testing.T) {
	var buf bytes.Buffer
	stderr = &buf
	defer func() { stderr = os.Stderr }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-clear", "--command", "nonexistent"})
	defer rootCmd.SetArgs([]string{})

	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "Unknown command") {
		t.Errorf("expected unknown command error, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// session-verify-fresh tests
// ---------------------------------------------------------------------------

func TestSessionVerifyFresh(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	oracleDir := tmpDir + "/.aether/oracle"
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(oracleDir+"/state.json", []byte("{}"), 0644)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	sessionStart := time.Now().Unix()

	rootCmd.SetArgs([]string{"session-verify-fresh", "--command", "oracle", fmt.Sprintf("%d", sessionStart)})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-verify-fresh returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true for fresh files, got: %s", output)
	}
}

func TestSessionVerifyFreshForce(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	oracleDir := tmpDir + "/.aether/oracle"
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(oracleDir+"/state.json", []byte("{}"), 0644)

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	sessionStart := time.Now().Add(1 * time.Hour).Unix()

	rootCmd.SetArgs([]string{"session-verify-fresh", "--command", "oracle", "--force", fmt.Sprintf("%d", sessionStart)})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-verify-fresh returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true with force, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// session-mark-resumed tests
// ---------------------------------------------------------------------------

func TestSessionMarkResumed(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	session := colony.SessionFile{
		SessionID:      "resume-test",
		StartedAt:      time.Now().UTC().Format(time.RFC3339),
		ContextCleared: true,
	}
	if err := s.SaveJSON("session.json", session); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"session-mark-resumed"})
	defer rootCmd.SetArgs([]string{})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("session-mark-resumed returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) {
		t.Errorf("expected ok:true, got: %s", output)
	}
	if !strings.Contains(output, `"resumed":true`) {
		t.Errorf("expected resumed:true, got: %s", output)
	}

	var updated colony.SessionFile
	if err := s.LoadJSON("session.json", &updated); err != nil {
		t.Fatal(err)
	}
	if updated.ResumedAt == nil {
		t.Error("resumed_at should be set")
	}
	if updated.ContextCleared {
		t.Error("context_cleared should be false after resume")
	}
}

func TestSessionMarkResumedNoSession(t *testing.T) {
	var buf bytes.Buffer
	stderr = &buf
	defer func() { stderr = os.Stderr }()

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", origRoot)

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store = s

	rootCmd.SetArgs([]string{"session-mark-resumed"})
	defer rootCmd.SetArgs([]string{})

	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "No active session") {
		t.Errorf("expected error about no active session, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// helper function tests
// ---------------------------------------------------------------------------

func TestRandomHex(t *testing.T) {
	val := randomHex(4)
	if len(val) != 8 {
		t.Errorf("randomHex(4) = %q, want 8 chars", val)
	}

	val2 := randomHex(4)
	if val == val2 {
		t.Errorf("randomHex produced same value twice: %q", val)
	}
}

func TestCountLines(t *testing.T) {
	tmpFile := t.TempDir() + "/test.txt"
	content := "line1\nline2\nline3\n"
	os.WriteFile(tmpFile, []byte(content), 0644)

	count, err := countLines(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("countLines = %d, want 3", count)
	}
}

func TestCountLinesEmpty(t *testing.T) {
	tmpFile := t.TempDir() + "/empty.txt"
	os.WriteFile(tmpFile, []byte(""), 0644)

	count, err := countLines(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("countLines(empty) = %d, want 0", count)
	}
}

// ---------------------------------------------------------------------------
// commandFileMap coverage
// ---------------------------------------------------------------------------

func TestCommandFileMapKnownCommands(t *testing.T) {
	cfm := commandFileMap()

	expected := []string{"survey", "oracle", "watch", "swarm", "init", "seal", "entomb"}
	for _, cmd := range expected {
		if _, ok := cfm[cmd]; !ok {
			t.Errorf("commandFileMap missing entry for %q", cmd)
		}
	}
}

func TestProtectedCommands(t *testing.T) {
	expected := []string{"init", "seal", "entomb"}
	for _, cmd := range expected {
		if !protectedCommands[cmd] {
			t.Errorf("%q should be in protectedCommands", cmd)
		}
	}

	if protectedCommands["oracle"] {
		t.Error("oracle should not be protected")
	}
}

// ---------------------------------------------------------------------------
// integration: command registration
// ---------------------------------------------------------------------------

func TestSessionCommandsRegistered(t *testing.T) {
	commands := []string{"session-init", "session-read", "session-update", "session-clear", "session-verify-fresh", "session-mark-resumed"}
	for _, name := range commands {
		cmd, _, err := rootCmd.Find([]string{name})
		if err != nil {
			t.Errorf("command %q not registered: %v", name, err)
			continue
		}
		if !strings.HasPrefix(cmd.Use, name) {
			t.Errorf("found command Use = %q, want prefix %q", cmd.Use, name)
		}
	}
}

// ---------------------------------------------------------------------------
// test helper
// ---------------------------------------------------------------------------

func createTestStore(dataDir string) (*storage.Store, error) {
	return storage.NewStore(dataDir)
}
