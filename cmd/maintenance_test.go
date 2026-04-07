package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- isTestArtifact Tests (INTG-04) ---

func TestIsTestArtifact(t *testing.T) {
	tests := []struct {
		name   string
		signal map[string]interface{}
		want   bool
	}{
		{
			name: "user source with test content is never flagged",
			signal: map[string]interface{}{
				"source": "user",
				"content": "test signal for validation",
				"id":      "user_signal_abc",
			},
			want: false,
		},
		{
			name: "cli source with test_ id prefix is never flagged",
			signal: map[string]interface{}{
				"source":  "cli",
				"id":      "test_my_signal",
				"content": "some content",
			},
			want: false,
		},
		{
			name: "auto source with test content is flagged",
			signal: map[string]interface{}{
				"source":  "auto",
				"content": "test signal",
				"id":      "auto_signal_1",
			},
			want: true,
		},
		{
			name: "auto source with test_ id prefix is flagged",
			signal: map[string]interface{}{
				"source":  "auto",
				"id":      "test_auto_generated",
				"content": "normal content",
			},
			want: true,
		},
		{
			name: "user source with demo content is never flagged",
			signal: map[string]interface{}{
				"source":  "user",
				"content": "demo pattern",
				"id":      "user_demo_signal",
			},
			want: false,
		},
		{
			name: "promotion source with test_ prefix is flagged",
			signal: map[string]interface{}{
				"source":  "promotion",
				"id":      "test_promoted",
				"content": "promoted content",
			},
			want: true,
		},
		{
			name: "missing source falls back to content checks",
			signal: map[string]interface{}{
				"id":      "test_no_source",
				"content": "test signal",
			},
			want: true,
		},
		{
			name: "missing source with normal content is not flagged",
			signal: map[string]interface{}{
				"id":      "normal_no_source",
				"content": "perfectly normal content",
			},
			want: false,
		},
		{
			name: "user source with content as map containing test text",
			signal: map[string]interface{}{
				"source": "user",
				"content": map[string]interface{}{
					"text": "this is a test signal from user",
				},
				"id": "user_map_signal",
			},
			want: false,
		},
		{
			name: "cli source with demo_ id prefix is never flagged",
			signal: map[string]interface{}{
				"source":  "cli",
				"id":      "demo_cli_signal",
				"content": "some content",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTestArtifact(tt.signal)
			if got != tt.want {
				t.Errorf("isTestArtifact() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- backup-prune-global Confirmation Gate Tests (INTG-05) ---

func TestBackupPruneGlobalConfirmGate(t *testing.T) {
	t.Run("without --confirm outputs dry_run and deletes nothing", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		// Create backup dir with 5 files
		backupDir := filepath.Join(s.BasePath(), "backups")
		os.MkdirAll(backupDir, 0755)
		for i := 0; i < 5; i++ {
			os.WriteFile(filepath.Join(backupDir, "backup-prune-test-0000"+string(rune('0'+i))+".json"), []byte("{}"), 0644)
		}

		rootCmd.SetArgs([]string{"backup-prune-global", "--cap", "3"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("expected ok:true, got: %v", env["ok"])
		}
		result := env["result"].(map[string]interface{})
		if result["dry_run"] != true {
			t.Errorf("dry_run = %v, want true without --confirm", result["dry_run"])
		}
		if result["pruned"] != float64(0) {
			t.Errorf("pruned = %v, want 0 in dry-run", result["pruned"])
		}
		if result["would_prune"] != float64(2) {
			t.Errorf("would_prune = %v, want 2", result["would_prune"])
		}

		// Verify all 5 files still exist
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			t.Fatalf("failed to read backup dir: %v", err)
		}
		fileCount := 0
		for _, e := range entries {
			if !e.IsDir() {
				fileCount++
			}
		}
		if fileCount != 5 {
			t.Errorf("file count = %d, want 5 (nothing should be deleted in dry-run)", fileCount)
		}
	})

	t.Run("with --confirm actually prunes files beyond cap", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		// Create backup dir with 5 files
		backupDir := filepath.Join(s.BasePath(), "backups")
		os.MkdirAll(backupDir, 0755)
		for i := 0; i < 5; i++ {
			os.WriteFile(filepath.Join(backupDir, "backup-confirm-test-0000"+string(rune('0'+i))+".json"), []byte("{}"), 0644)
		}

		rootCmd.SetArgs([]string{"backup-prune-global", "--cap", "3", "--confirm"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["pruned"] != float64(2) {
			t.Errorf("pruned = %v, want 2", result["pruned"])
		}
		if result["kept"] != float64(3) {
			t.Errorf("kept = %v, want 3", result["kept"])
		}

		// Verify only 3 files remain
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			t.Fatalf("failed to read backup dir: %v", err)
		}
		fileCount := 0
		for _, e := range entries {
			if !e.IsDir() {
				fileCount++
			}
		}
		if fileCount != 3 {
			t.Errorf("file count = %d, want 3 after pruning", fileCount)
		}
	})
}

// --- temp-clean Confirmation Gate Tests (INTG-05) ---

func TestTempCleanConfirmGate(t *testing.T) {
	t.Run("without --confirm outputs dry_run and deletes nothing", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStoreWithRoot(t)
		defer os.RemoveAll(tmpDir)
		store = s

		// Create temp dir with old and recent files
		tempDir := filepath.Join(tmpDir, ".aether", "temp")
		os.MkdirAll(tempDir, 0755)

		oldTime := time.Now().Add(-8 * 24 * time.Hour)
		os.WriteFile(filepath.Join(tempDir, "old-dryrun.txt"), []byte("old"), 0644)
		os.Chtimes(filepath.Join(tempDir, "old-dryrun.txt"), oldTime, oldTime)

		os.WriteFile(filepath.Join(tempDir, "recent-dryrun.txt"), []byte("recent"), 0644)

		rootCmd.SetArgs([]string{"temp-clean"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("expected ok:true, got: %v", env["ok"])
		}
		result := env["result"].(map[string]interface{})
		if result["dry_run"] != true {
			t.Errorf("dry_run = %v, want true without --confirm", result["dry_run"])
		}
		if result["cleaned"] != float64(0) {
			t.Errorf("cleaned = %v, want 0 in dry-run", result["cleaned"])
		}
		if result["would_clean"] != float64(1) {
			t.Errorf("would_clean = %v, want 1", result["would_clean"])
		}

		// Verify old file still exists
		if _, err := os.Stat(filepath.Join(tempDir, "old-dryrun.txt")); err != nil {
			t.Error("old file should still exist in dry-run mode")
		}
	})

	t.Run("with --confirm removes old temp files", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStoreWithRoot(t)
		defer os.RemoveAll(tmpDir)
		store = s

		// Create temp dir with old and recent files
		tempDir := filepath.Join(tmpDir, ".aether", "temp")
		os.MkdirAll(tempDir, 0755)

		oldTime8 := time.Now().Add(-8 * 24 * time.Hour)
		os.WriteFile(filepath.Join(tempDir, "old8.txt"), []byte("old8"), 0644)
		os.Chtimes(filepath.Join(tempDir, "old8.txt"), oldTime8, oldTime8)

		oldTime10 := time.Now().Add(-10 * 24 * time.Hour)
		os.WriteFile(filepath.Join(tempDir, "old10.txt"), []byte("old10"), 0644)
		os.Chtimes(filepath.Join(tempDir, "old10.txt"), oldTime10, oldTime10)

		os.WriteFile(filepath.Join(tempDir, "recent-confirm.txt"), []byte("recent"), 0644)

		rootCmd.SetArgs([]string{"temp-clean", "--confirm"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["cleaned"] != float64(2) {
			t.Errorf("cleaned = %v, want 2", result["cleaned"])
		}

		// Verify old files are gone
		if _, err := os.Stat(filepath.Join(tempDir, "old8.txt")); err == nil {
			t.Error("old8.txt should be deleted with --confirm")
		}
		if _, err := os.Stat(filepath.Join(tempDir, "old10.txt")); err == nil {
			t.Error("old10.txt should be deleted with --confirm")
		}

		// Verify recent file still exists
		if _, err := os.Stat(filepath.Join(tempDir, "recent-confirm.txt")); err != nil {
			t.Error("recent file should still exist after temp-clean --confirm")
		}
	})
}
