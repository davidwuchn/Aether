package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

// TestSaveLoadRoundTrip verifies that saving a graph and loading it back
// produces an identical graph (same nodes and edges).
func TestSaveLoadRoundTrip(t *testing.T) {
	store, dir := newTestStore(t)

	// Build a graph with nodes and edges
	g := NewGraph()
	g.AddNode(Node{ID: "inst_100_alpha", Type: NodeInstinct})
	g.AddNode(Node{ID: "inst_200_beta", Type: NodeInstinct})
	g.AddNode(Node{ID: "obs_hash123", Type: NodeLearning})
	_, _, err := g.AddEdge("inst_100_alpha", "inst_200_beta", EdgeReinforces, 0.8, "2026-04-01T12:00:00Z")
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	_, _, err = g.AddEdge("obs_hash123", "inst_100_alpha", EdgePromotedFrom, 0.7, "2026-04-01T12:02:00Z")
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}

	// Save
	if err := g.SaveToStore(store, "test-graph.json"); err != nil {
		t.Fatalf("SaveToStore: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "test-graph.json")); err != nil {
		t.Fatalf("file not found: %v", err)
	}

	// Load into new graph
	g2 := NewGraph()
	if err := g2.LoadFromStore(store, "test-graph.json"); err != nil {
		t.Fatalf("LoadFromStore: %v", err)
	}

	// Verify node count
	if g2.NodeCount() != g.NodeCount() {
		t.Errorf("NodeCount = %d, want %d", g2.NodeCount(), g.NodeCount())
	}

	// Verify edge count
	if g2.EdgeCount() != g.EdgeCount() {
		t.Errorf("EdgeCount = %d, want %d", g2.EdgeCount(), g.EdgeCount())
	}

	// Verify specific nodes exist
	for _, id := range []string{"inst_100_alpha", "inst_200_beta", "obs_hash123"} {
		n := g2.GetNode(id)
		if n == nil {
			t.Errorf("node %q not found after load", id)
		}
	}
}

// TestLoadShellFormat verifies that loading a shell-format JSON file
// (edges only, no nodes section) correctly loads edges and auto-creates nodes.
func TestLoadShellFormat(t *testing.T) {
	store, dir := newTestStore(t)

	// Copy testdata file into store directory
	testData, err := os.ReadFile("testdata/shell-edges.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "shell-edges.json"), testData, 0644); err != nil {
		t.Fatalf("write testdata: %v", err)
	}

	g := NewGraph()
	if err := g.LoadFromStore(store, "shell-edges.json"); err != nil {
		t.Fatalf("LoadFromStore: %v", err)
	}

	// Should have 3 edges
	if g.EdgeCount() != 3 {
		t.Errorf("EdgeCount = %d, want 3", g.EdgeCount())
	}

	// Should have auto-created 4 nodes: inst_100_alpha, inst_200_beta, inst_300_gamma, obs_hash123
	if g.NodeCount() != 4 {
		t.Errorf("NodeCount = %d, want 4", g.NodeCount())
	}
}

// TestLoadShellFormatAutoCreatesNodes verifies that nodes are created with
// correct inferred types when loading shell-format JSON (no nodes section).
func TestLoadShellFormatAutoCreatesNodes(t *testing.T) {
	store, dir := newTestStore(t)

	testData, err := os.ReadFile("testdata/shell-edges.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "shell-edges.json"), testData, 0644); err != nil {
		t.Fatalf("write testdata: %v", err)
	}

	g := NewGraph()
	if err := g.LoadFromStore(store, "shell-edges.json"); err != nil {
		t.Fatalf("LoadFromStore: %v", err)
	}

	// inst_ prefix -> NodeInstinct
	n := g.GetNode("inst_100_alpha")
	if n == nil {
		t.Fatal("node inst_100_alpha not found")
	}
	if n.Type != NodeInstinct {
		t.Errorf("inst_100_alpha type = %q, want %q", n.Type, NodeInstinct)
	}

	// obs_ prefix -> NodeLearning
	n2 := g.GetNode("obs_hash123")
	if n2 == nil {
		t.Fatal("node obs_hash123 not found")
	}
	if n2.Type != NodeLearning {
		t.Errorf("obs_hash123 type = %q, want %q", n2.Type, NodeLearning)
	}
}

// TestSaveIncludesNodes verifies that saved JSON has both "nodes" and "edges" arrays.
func TestSaveIncludesNodes(t *testing.T) {
	store, dir := newTestStore(t)

	g := NewGraph()
	g.AddNode(Node{ID: "inst_1", Type: NodeInstinct})
	_, _, err := g.AddEdge("inst_1", "inst_2", EdgeReinforces, 0.5, "2026-04-01T00:00:00Z")
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}

	if err := g.SaveToStore(store, "graph.json"); err != nil {
		t.Fatalf("SaveToStore: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "graph.json"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var jf JSONFile
	if err := json.Unmarshal(data, &jf); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if len(jf.Nodes) == 0 {
		t.Error("saved JSON should have nodes")
	}
	if len(jf.Edges) == 0 {
		t.Error("saved JSON should have edges")
	}
}

// TestLoadEmptyEdges verifies that loading a JSON with empty edge array
// produces a valid empty graph.
func TestLoadEmptyEdges(t *testing.T) {
	store, dir := newTestStore(t)

	emptyData := `{"version": "1.0", "nodes": [], "edges": []}`
	if err := os.WriteFile(filepath.Join(dir, "empty.json"), []byte(emptyData), 0644); err != nil {
		t.Fatalf("write empty: %v", err)
	}

	g := NewGraph()
	if err := g.LoadFromStore(store, "empty.json"); err != nil {
		t.Fatalf("LoadFromStore: %v", err)
	}

	if g.NodeCount() != 0 {
		t.Errorf("NodeCount = %d, want 0", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount = %d, want 0", g.EdgeCount())
	}
}

// TestSaveToStore verifies that SaveToStore uses storage.Store.SaveJSON
// by checking the file was written and is valid JSON.
func TestSaveToStore(t *testing.T) {
	store, dir := newTestStore(t)

	g := NewGraph()
	g.AddNode(Node{ID: "inst_1", Type: NodeInstinct})

	if err := g.SaveToStore(store, "persist-test.json"); err != nil {
		t.Fatalf("SaveToStore: %v", err)
	}

	// Verify file is valid JSON
	data, err := os.ReadFile(filepath.Join(dir, "persist-test.json"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !json.Valid(data) {
		t.Error("saved file is not valid JSON")
	}

	// Verify version field
	var jf JSONFile
	if err := json.Unmarshal(data, &jf); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if jf.Version != "1.0" {
		t.Errorf("version = %q, want %q", jf.Version, "1.0")
	}
}

// TestLoadFromStoreMissingFile verifies that loading a missing file returns an error.
func TestLoadFromStoreMissingFile(t *testing.T) {
	store, _ := newTestStore(t)

	g := NewGraph()
	err := g.LoadFromStore(store, "nonexistent.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// newTestStore creates a temp directory and returns a Store for testing.
func newTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return store, dir
}
