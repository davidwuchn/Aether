package agent

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func strPtr(s string) *string { return &s }

func TestBuildTaskGraph_Empty(t *testing.T) {
	g, err := BuildTaskGraph(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes()) != 0 {
		t.Fatalf("expected empty graph, got %d nodes", len(g.Nodes()))
	}
}

func TestBuildTaskGraph_SingleTask(t *testing.T) {
	tasks := []colony.Task{
		{ID: strPtr("A"), Goal: "implement the API"},
	}
	g, err := BuildTaskGraph(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Nodes()) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes()))
	}
	node := g.Node("A")
	if node == nil {
		t.Fatal("node A not found")
	}
	if node.Goal != "implement the API" {
		t.Errorf("expected goal 'implement the API', got %q", node.Goal)
	}
	if node.Caste != CasteBuilder {
		t.Errorf("expected caste builder, got %q", node.Caste)
	}
	if node.Status != TaskNodePending {
		t.Errorf("expected status pending, got %q", node.Status)
	}
}

func TestBuildTaskGraph_WithDeps(t *testing.T) {
	tasks := []colony.Task{
		{ID: strPtr("A"), Goal: "implement base"},
		{ID: strPtr("B"), Goal: "implement feature", DependsOn: []string{"A"}},
		{ID: strPtr("C"), Goal: "test feature", DependsOn: []string{"B"}},
	}
	g, err := BuildTaskGraph(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A should have in-degree 0, B should have 1, C should have 2
	if g.inDegree["A"] != 0 {
		t.Errorf("expected A in-degree 0, got %d", g.inDegree["A"])
	}
	if g.inDegree["B"] != 1 {
		t.Errorf("expected B in-degree 1, got %d", g.inDegree["B"])
	}
	if g.inDegree["C"] != 1 {
		t.Errorf("expected C in-degree 1, got %d", g.inDegree["C"])
	}
}

func TestBuildTaskGraph_CycleDetection(t *testing.T) {
	tasks := []colony.Task{
		{ID: strPtr("A"), Goal: "task A", DependsOn: []string{"B"}},
		{ID: strPtr("B"), Goal: "task B", DependsOn: []string{"A"}},
	}
	_, err := BuildTaskGraph(tasks)
	if err == nil {
		t.Fatal("expected error for circular dependencies, got nil")
	}
	if err.Error() != "task graph has circular dependencies: processed 0 of 2 tasks" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTaskGraph_Ready(t *testing.T) {
	tasks := []colony.Task{
		{ID: strPtr("A"), Goal: "implement base"},
		{ID: strPtr("B"), Goal: "implement feature", DependsOn: []string{"A"}},
	}
	g, err := BuildTaskGraph(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ready := g.Ready()
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready task, got %d", len(ready))
	}
	if ready[0].ID != "A" {
		t.Errorf("expected ready task A, got %q", ready[0].ID)
	}
}

func TestTaskGraph_Complete(t *testing.T) {
	tasks := []colony.Task{
		{ID: strPtr("A"), Goal: "implement base"},
		{ID: strPtr("B"), Goal: "implement feature", DependsOn: []string{"A"}},
	}
	g, err := BuildTaskGraph(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newlyReady, err := g.Complete("A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(newlyReady) != 1 {
		t.Fatalf("expected 1 newly ready task, got %d", len(newlyReady))
	}
	if newlyReady[0].ID != "B" {
		t.Errorf("expected newly ready task B, got %q", newlyReady[0].ID)
	}

	node := g.Node("A")
	if node.Status != TaskNodeCompleted {
		t.Errorf("expected A status completed, got %q", node.Status)
	}
}

func TestTaskGraph_CompleteChain(t *testing.T) {
	tasks := []colony.Task{
		{ID: strPtr("A"), Goal: "implement base"},
		{ID: strPtr("B"), Goal: "implement middle", DependsOn: []string{"A"}},
		{ID: strPtr("C"), Goal: "implement top", DependsOn: []string{"B"}},
	}
	g, err := BuildTaskGraph(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Complete A -> B becomes ready
	newlyReady, err := g.Complete("A")
	if err != nil {
		t.Fatalf("unexpected error completing A: %v", err)
	}
	if len(newlyReady) != 1 || newlyReady[0].ID != "B" {
		t.Fatalf("expected B ready after A, got %v", taskIDs(newlyReady))
	}

	// Complete B -> C becomes ready
	newlyReady, err = g.Complete("B")
	if err != nil {
		t.Fatalf("unexpected error completing B: %v", err)
	}
	if len(newlyReady) != 1 || newlyReady[0].ID != "C" {
		t.Fatalf("expected C ready after B, got %v", taskIDs(newlyReady))
	}

	// Complete C -> nothing else
	newlyReady, err = g.Complete("C")
	if err != nil {
		t.Fatalf("unexpected error completing C: %v", err)
	}
	if len(newlyReady) != 0 {
		t.Fatalf("expected no new tasks after C, got %v", taskIDs(newlyReady))
	}
}

func TestTaskGraph_TypeHintAssignment(t *testing.T) {
	tasks := []colony.Task{
		{ID: strPtr("A"), Goal: "[test] verify the login"},
		{ID: strPtr("B"), Goal: "[research] investigate caching"},
	}
	g, err := BuildTaskGraph(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Node("A").Caste != CasteWatcher {
		t.Errorf("expected watcher caste for [test] hint, got %q", g.Node("A").Caste)
	}
	if g.Node("B").Caste != CasteScout {
		t.Errorf("expected scout caste for [research] hint, got %q", g.Node("B").Caste)
	}
}

func taskIDs(nodes []*TaskNode) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}
