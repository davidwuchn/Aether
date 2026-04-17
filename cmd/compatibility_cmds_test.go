package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestQueenPromoteLegacyPositionalSanitizesContent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"queen-promote", "pattern", "Line one\n## injected"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env["ok"])
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "QUEEN.md"))
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "## Patterns") {
		t.Fatalf("QUEEN.md missing Patterns section:\n%s", text)
	}
	if strings.Contains(text, "\n## injected") {
		t.Fatalf("unsanitized injected header present:\n%s", text)
	}
	if !strings.Contains(text, "Line one ## injected") {
		t.Fatalf("sanitized content missing:\n%s", text)
	}
}

func TestCharterWriteLegacyFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{
		"charter-write",
		"--intent", "Ship\ncleanly",
		"--vision", "Great colony system",
		"--governance", "Go is source of truth",
		"--goals", "parity, resilience",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "QUEEN.md"))
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"## Colony Charter",
		"- **Intent:** Ship cleanly",
		"- **Vision:** Great colony system",
		"- **Governance:** Go is source of truth",
		"- **Goals:** parity, resilience",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("QUEEN.md missing %q:\n%s", want, text)
		}
	}
}

func TestMemoryCaptureSupportsPositionalContent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"memory-capture", "use positional content"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env["ok"])
	}
}

func TestHivePromoteDefaultsDomainForLegacyPlaybooks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"hive-promote", "--text", "Prefer focused fixes", "--source-repo", "aether", "--confidence", "0.9"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "hive", "wisdom.json"))
	if err != nil {
		t.Fatalf("read wisdom.json: %v", err)
	}
	if !strings.Contains(string(data), `"domain": "general"`) {
		t.Fatalf("wisdom.json missing default general domain:\n%s", string(data))
	}
}

func TestResumeAliasRestoresResumeColonyFlow(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Resume via alias"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
				{ID: 2, Name: "Verification", Status: colony.PhaseReady},
			},
		},
	})

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID:      "resume-alias",
		StartedAt:      "2026-04-17T10:00:00Z",
		ColonyGoal:     goal,
		CurrentPhase:   1,
		SuggestedNext:  "aether continue",
		ContextCleared: true,
		Summary:        "Paused with a handoff",
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	handoffPath := filepath.Join(os.Getenv("AETHER_ROOT"), ".aether", "HANDOFF.md")
	if err := os.MkdirAll(filepath.Dir(handoffPath), 0755); err != nil {
		t.Fatalf("mkdir handoff dir: %v", err)
	}
	if err := os.WriteFile(handoffPath, []byte("# Colony Session — Paused Colony\n\nResume from here.\n"), 0644); err != nil {
		t.Fatalf("seed handoff: %v", err)
	}

	rootCmd.SetArgs([]string{"resume"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["resumed"] != true {
		t.Fatalf("expected resumed:true, got %v", result)
	}
	if result["handoff_found"] != true {
		t.Fatalf("expected handoff_found:true, got %v", result["handoff_found"])
	}
}

func TestSwarmCompatibilityWatchReportsActiveWorkers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Watch worker activity"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
			},
		},
	})

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-1", "Fix the remaining issue", 1); err != nil {
		t.Fatalf("record spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Hammer-1", "active", "Awaiting continue verification"); err != nil {
		t.Fatalf("mark active: %v", err)
	}

	rootCmd.SetArgs([]string{"swarm", "--watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm --watch returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["mode"] != "watch" {
		t.Fatalf("mode = %v, want watch", result["mode"])
	}
	if result["active_count"] != float64(1) {
		t.Fatalf("active_count = %v, want 1", result["active_count"])
	}
}

func TestAutopilotSuccessStatusCountsAsCompleted(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	_, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-init returned error: %v", err)
	}

	buf.Reset()
	rootCmd.SetArgs([]string{"autopilot-update", "--phase", "1", "--status", "success"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-update returned error: %v", err)
	}

	buf.Reset()
	rootCmd.SetArgs([]string{"autopilot-check-replan", "--interval", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-check-replan returned error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["replan"] != true {
		t.Fatalf("expected replan:true after success status normalization, got %v", result)
	}
}

func TestWatchCompatibilityWritesArtifacts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Watch live workers"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Execution", Status: colony.PhaseInProgress}},
		},
	})

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-9", "Investigate release issue", 1); err != nil {
		t.Fatalf("record spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Hammer-9", "active", "Running"); err != nil {
		t.Fatalf("mark active: %v", err)
	}

	rootCmd.SetArgs([]string{"watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("watch returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["mode"] != "watch" {
		t.Fatalf("mode = %v, want watch", result["mode"])
	}
	if result["active_count"] != float64(1) {
		t.Fatalf("active_count = %v, want 1", result["active_count"])
	}

	statusPath := filepath.Join(dataDir, "watch-status.txt")
	progressPath := filepath.Join(dataDir, "watch-progress.txt")
	if _, err := os.Stat(statusPath); err != nil {
		t.Fatalf("watch-status.txt missing: %v", err)
	}
	if _, err := os.Stat(progressPath); err != nil {
		t.Fatalf("watch-progress.txt missing: %v", err)
	}
}

func TestOracleCompatibilityCreatesAndStopsWorkspace(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	rootCmd.SetArgs([]string{"oracle", "release parity"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle start returned error: %v", err)
	}

	startEnv := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	startResult := startEnv["result"].(map[string]interface{})
	if startResult["started"] != true {
		t.Fatalf("expected started:true, got %v", startResult)
	}
	if startResult["detected_type"] != "go" {
		t.Fatalf("detected_type = %v, want go", startResult["detected_type"])
	}

	stdout.(*bytes.Buffer).Reset()
	rootCmd.SetArgs([]string{"oracle", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle status returned error: %v", err)
	}

	statusEnv := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	statusResult := statusEnv["result"].(map[string]interface{})
	if statusResult["has_state"] != true || statusResult["has_plan"] != true {
		t.Fatalf("expected oracle workspace files to exist, got %v", statusResult)
	}

	stdout.(*bytes.Buffer).Reset()
	rootCmd.SetArgs([]string{"oracle", "stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle stop returned error: %v", err)
	}

	stopEnv := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	stopResult := stopEnv["result"].(map[string]interface{})
	if stopResult["stopped"] != true {
		t.Fatalf("expected stopped:true, got %v", stopResult)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "oracle", ".stop")); err != nil {
		t.Fatalf("expected .stop marker, got %v", err)
	}
}

func TestRunCompatibilityDryRunPlansLifecycle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Ship the release"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: ptrString("1.1"), Goal: "Do work", Status: colony.TaskPending}}},
				{ID: 2, Name: "Phase 2", Status: colony.PhasePending, Tasks: []colony.Task{{ID: ptrString("2.1"), Goal: "More work", Status: colony.TaskPending}}},
			},
		},
	})

	rootCmd.SetArgs([]string{"run", "--dry-run", "--max-phases", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run --dry-run returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["dry_run"] != true {
		t.Fatalf("expected dry_run:true, got %v", result)
	}
	if result["stopped_reason"] != "max_phases_reached" {
		t.Fatalf("stopped_reason = %v, want max_phases_reached", result["stopped_reason"])
	}
	steps := result["steps"].([]interface{})
	if len(steps) != 2 {
		t.Fatalf("expected 2 dry-run steps, got %d", len(steps))
	}
}

func TestRunCompatibilityExecutesSinglePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	for _, agentName := range []string{"aether-builder.toml", "aether-watcher.toml"} {
		role := strings.TrimSuffix(strings.TrimPrefix(agentName, "aether-"), ".toml")
		if err := os.WriteFile(filepath.Join(agentsDir, agentName), validCodexAgentTOML(strings.TrimSuffix(agentName, ".toml"), role), 0644); err != nil {
			t.Fatalf("write %s: %v", agentName, err)
		}
	}

	goal := "Run one autopilot phase"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		CurrentPhase:  1,
		InitializedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: ptrString("1.1"), Goal: "Implement it", Status: colony.TaskPending}}},
			},
		},
	})

	rootCmd.SetArgs([]string{"run", "--max-phases", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["completed"] != true {
		t.Fatalf("expected completed:true, got %v", result)
	}
	if result["next"] != "aether seal" {
		t.Fatalf("next = %v, want aether seal", result["next"])
	}
	if result["phases_completed"] != float64(1) {
		t.Fatalf("phases_completed = %v, want 1", result["phases_completed"])
	}

	var autopilot autopilotState
	if err := store.LoadJSON(autopilotStatePath, &autopilot); err != nil {
		t.Fatalf("load autopilot state: %v", err)
	}
	if autopilot.Status != "completed" {
		t.Fatalf("autopilot status = %q, want completed", autopilot.Status)
	}
}
