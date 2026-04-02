package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

func setupQueenTest(t *testing.T) (*QueenService, *storage.Store, *events.Bus, func()) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	bus := events.NewBus(store, events.Config{JSONLFile: "events.jsonl"})
	svc := NewQueenService(store, bus)
	return svc, store, bus, func() { bus.Close() }
}

// TestQueenInit verifies that when QUEEN.md does not exist, WriteEntry creates
// the V2 template with all 4 sections, Evolution Log table header, and METADATA
// HTML comment with total_entries=0 and a valid last_updated timestamp.
func TestQueenInit(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **security** (0.80): When sanitize inputs, then validate all inputs", "test-colony")
	if err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	content := string(data)

	// Verify all 4 sections exist
	for _, section := range []string{"## User Preferences", "## Codebase Patterns", "## Build Learnings", "## Instincts"} {
		if !strings.Contains(content, section) {
			t.Errorf("missing section %q in QUEEN.md", section)
		}
	}

	// Verify Evolution Log section and table header
	if !strings.Contains(content, "## Evolution Log") {
		t.Error("missing Evolution Log section")
	}
	if !strings.Contains(content, "| Date | Source | Type | Details |") {
		t.Error("missing Evolution Log table header")
	}
	if !strings.Contains(content, "|------|--------|------|---------|") {
		t.Error("missing Evolution Log separator")
	}

	// Verify METADATA HTML comment
	if !strings.Contains(content, "<!-- METADATA") {
		t.Error("missing METADATA HTML comment")
	}
}

// TestQueenWriteInstinct verifies that PromoteInstinct with a real InstinctEntry
// writes the correctly formatted entry to the Instincts section, replacing the placeholder.
func TestQueenWriteInstinct(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()
	instinct := colony.InstinctEntry{
		Domain:     "security",
		Trigger:    "sanitize inputs",
		Action:     "validate all inputs",
		Confidence: 0.80,
	}

	result, err := svc.PromoteInstinct(ctx, "QUEEN.md", instinct, "test-colony")
	if err != nil {
		t.Fatalf("PromoteInstinct: %v", err)
	}
	if result.Section != "Instincts" {
		t.Errorf("Section = %q, want %q", result.Section, "Instincts")
	}
	if result.EntriesAdded != 1 {
		t.Errorf("EntriesAdded = %d, want 1", result.EntriesAdded)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	content := string(data)

	expected := "- [instinct] **security** (0.80): When sanitize inputs, then validate all inputs"
	if !strings.Contains(content, expected) {
		t.Errorf("missing instinct entry %q in QUEEN.md:\n%s", expected, content)
	}

	// Placeholder should be gone
	if strings.Contains(content, "_No instincts promoted yet._") {
		t.Error("placeholder should be replaced after writing instinct")
	}
}

// TestQueenWritePattern verifies PromotePattern writes the correctly formatted entry.
func TestQueenWritePattern(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()
	result, err := svc.PromotePattern(ctx, "QUEEN.md", "always use structured error types", "test-colony")
	if err != nil {
		t.Fatalf("PromotePattern: %v", err)
	}
	if result.Section != "Codebase Patterns" {
		t.Errorf("Section = %q, want %q", result.Section, "Codebase Patterns")
	}
	if result.EntriesAdded != 1 {
		t.Errorf("EntriesAdded = %d, want 1", result.EntriesAdded)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "- [pattern] **test-colony**") {
		t.Error("missing pattern entry prefix")
	}
	if !strings.Contains(content, "always use structured error types") {
		t.Error("missing pattern content")
	}

	// Placeholder should be gone
	if strings.Contains(content, "_No codebase patterns recorded yet._") {
		t.Error("Codebase Patterns placeholder should be replaced")
	}
}

// TestQueenWriteBuildLearning verifies PromoteBuildLearning writes the correctly formatted entry.
func TestQueenWriteBuildLearning(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()
	result, err := svc.PromoteBuildLearning(ctx, "QUEEN.md", "testing", "always write tests first", "47", "memory-pipeline", "test-colony")
	if err != nil {
		t.Fatalf("PromoteBuildLearning: %v", err)
	}
	if result.Section != "Build Learnings" {
		t.Errorf("Section = %q, want %q", result.Section, "Build Learnings")
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "- [testing] always write tests first") {
		t.Error("missing build learning entry")
	}
	if !strings.Contains(content, "Phase 47 (memory-pipeline)") {
		t.Error("missing phase reference in build learning")
	}

	// Placeholder should be gone
	if strings.Contains(content, "_No build learnings recorded yet._") {
		t.Error("Build Learnings placeholder should be replaced")
	}
}

// TestQueenWritePreference verifies PromotePreference writes the correctly formatted entry.
func TestQueenWritePreference(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()
	result, err := svc.PromotePreference(ctx, "QUEEN.md", "prefer concise commit messages", "test-colony")
	if err != nil {
		t.Fatalf("PromotePreference: %v", err)
	}
	if result.Section != "User Preferences" {
		t.Errorf("Section = %q, want %q", result.Section, "User Preferences")
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "- **test-colony**") {
		t.Error("missing preference entry prefix")
	}
	if !strings.Contains(content, "prefer concise commit messages") {
		t.Error("missing preference content")
	}

	// Placeholder should be gone
	if strings.Contains(content, "_No user preferences recorded yet._") {
		t.Error("User Preferences placeholder should be replaced")
	}
}

// TestQueenDedupEntry verifies that writing the same entry content to the same
// section twice results in only one entry -- the second call returns EntriesAdded=0.
func TestQueenDedupEntry(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()
	entry := "- [instinct] **security** (0.80): When sanitize inputs, then validate all inputs"

	result1, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", entry, "test-colony")
	if err != nil {
		t.Fatalf("first WriteEntry: %v", err)
	}
	if result1.EntriesAdded != 1 {
		t.Errorf("first WriteEntry EntriesAdded = %d, want 1", result1.EntriesAdded)
	}

	result2, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", entry, "test-colony")
	if err != nil {
		t.Fatalf("second WriteEntry: %v", err)
	}
	if result2.EntriesAdded != 0 {
		t.Errorf("dedup WriteEntry EntriesAdded = %d, want 0", result2.EntriesAdded)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	// Count occurrences of the entry
	count := strings.Count(content, "sanitize inputs, then validate all inputs")
	if count != 1 {
		t.Errorf("entry appears %d times, want 1", count)
	}
}

// TestQueenEmptyGuard verifies that if the assembled content is empty (zero bytes),
// the write is aborted and an error containing "refusing to overwrite with empty content"
// is returned -- the file on disk is NOT modified.
func TestQueenEmptyGuard(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()

	// First, write a valid entry to create the file
	_, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **test** (0.75): When x, then y", "test-colony")
	if err != nil {
		t.Fatalf("setup write: %v", err)
	}

	// Read the file to know its content before the empty guard test
	dataBefore, _ := store.ReadFile("QUEEN.md")

	// Now try to write an empty string entry
	_, err = svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "", "test-colony")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite with empty content") {
		t.Errorf("error = %q, want containing %q", err.Error(), "refusing to overwrite with empty content")
	}

	// File should be unchanged
	dataAfter, _ := store.ReadFile("QUEEN.md")
	if string(dataBefore) != string(dataAfter) {
		t.Error("QUEEN.md was modified despite empty guard")
	}
}

// TestQueenMetadataUpdate verifies that after writing an entry, the METADATA HTML
// comment contains the updated total_entries count (incremented) and a new last_updated timestamp.
func TestQueenMetadataUpdate(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()

	// Write first entry
	result, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **test** (0.75): When x, then y", "test-colony")
	if err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}
	if result.TotalEntries != 1 {
		t.Errorf("TotalEntries = %d, want 1 after first write", result.TotalEntries)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// Parse metadata from content
	meta := parseMetadataHelper(t, string(data))
	entriesFloat, ok := meta["total_entries"].(float64)
	if !ok {
		t.Fatalf("total_entries not a number in metadata: %v", meta["total_entries"])
	}
	if int(entriesFloat) != 1 {
		t.Errorf("METADATA total_entries = %d, want 1", int(entriesFloat))
	}

	lastUpdated, ok := meta["last_updated"].(string)
	if !ok || lastUpdated == "" {
		t.Error("METADATA last_updated is missing or not a string")
	}
}

// TestQueenEvolutionLog verifies that after writing an entry, the Evolution Log
// table has a new row with the date, colonyName, section name, and a summary of the entry.
func TestQueenEvolutionLog(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **security** (0.80): When sanitize, then validate", "my-colony")
	if err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "| my-colony | Instincts |") {
		t.Errorf("Evolution Log missing row with colony and section:\n%s", content)
	}
}

// TestQueenEventPublished verifies that after a successful write, a "queen.write"
// event is published to the event bus with a payload containing the section and queen_path.
func TestQueenEventPublished(t *testing.T) {
	svc, _, bus, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()

	// Subscribe to queen.write events
	ch, err := bus.Subscribe("queen.write")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	_, err = svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **test** (0.75): When x, then y", "test-colony")
	if err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	// Wait for event with timeout
	select {
	case evt := <-ch:
		var payload map[string]interface{}
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload["section"] != "Instincts" {
			t.Errorf("payload section = %v, want Instincts", payload["section"])
		}
		if payload["queen_path"] != "QUEEN.md" {
			t.Errorf("payload queen_path = %v, want QUEEN.md", payload["queen_path"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for queen.write event")
	}

	// Also verify via JSONL persistence
	evts, err := bus.Query(ctx, "queen.write", time.Time{}, 10)
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	if len(evts) == 0 {
		t.Error("no queen.write events in JSONL")
	}
}

// TestQueenMultipleSections verifies that writing to different sections
// (e.g., Instincts then Codebase Patterns) results in both sections containing entries.
func TestQueenMultipleSections(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **test** (0.75): When x, then y", "colony")
	if err != nil {
		t.Fatalf("write instinct: %v", err)
	}

	_, err = svc.WriteEntry(ctx, "QUEEN.md", "Codebase Patterns", "- [pattern] **colony** (2026-01-01T00:00:00Z): some pattern", "colony")
	if err != nil {
		t.Fatalf("write pattern: %v", err)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "When x, then y") {
		t.Error("Instincts entry missing after writing to Codebase Patterns")
	}
	if !strings.Contains(content, "some pattern") {
		t.Error("Codebase Patterns entry missing")
	}
}

// TestQueenPreservesExisting verifies that writing a new entry to a QUEEN.md
// that already has entries preserves all existing entries and appends the new one.
func TestQueenPreservesExisting(t *testing.T) {
	svc, store, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()

	// Write first instinct
	_, err := svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **security** (0.80): When sanitize, then validate", "colony")
	if err != nil {
		t.Fatalf("first write: %v", err)
	}

	// Write second instinct
	_, err = svc.WriteEntry(ctx, "QUEEN.md", "Instincts", "- [instinct] **testing** (0.75): When build, then test", "colony")
	if err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := store.ReadFile("QUEEN.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "When sanitize, then validate") {
		t.Error("first instinct entry was lost")
	}
	if !strings.Contains(content, "When build, then test") {
		t.Error("second instinct entry missing")
	}
}

// TestQueenInvalidSection verifies that an invalid section name returns an error
// containing "invalid section".
func TestQueenInvalidSection(t *testing.T) {
	svc, _, _, cleanup := setupQueenTest(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.WriteEntry(ctx, "QUEEN.md", "InvalidSection", "some entry", "colony")
	if err == nil {
		t.Fatal("expected error for invalid section, got nil")
	}
	if !strings.Contains(err.Error(), "invalid section") {
		t.Errorf("error = %q, want containing %q", err.Error(), "invalid section")
	}
}

// parseMetadataHelper extracts the METADATA JSON from a QUEEN.md content string.
func parseMetadataHelper(t *testing.T, content string) map[string]interface{} {
	t.Helper()
	start := strings.Index(content, "<!-- METADATA ")
	if start == -1 {
		t.Fatal("no METADATA HTML comment found")
	}
	start += len("<!-- METADATA ")
	end := strings.Index(content[start:], " -->")
	if end == -1 {
		t.Fatal("METADATA comment not closed")
	}
	jsonStr := content[start : start+end]
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
		t.Fatalf("parse METADATA JSON: %v", err)
	}
	return meta
}
