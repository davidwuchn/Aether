package graph

import (
	"reflect"
	"sort"
	"testing"
)

func TestDetectCyclesSimple(t *testing.T) {
	// A -> B -> A
	g := NewGraph()
	g.AddNode(Node{ID: "A", Type: NodeInstinct})
	g.AddNode(Node{ID: "B", Type: NodeInstinct})
	g.AddEdge("A", "B", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("B", "A", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Fatal("DetectCycles found no cycles, want at least 1")
	}

	// Verify the cycle contains A and B
	found := false
	for _, c := range cycles {
		cs := cycleSet(c)
		if cs["A"] && cs["B"] && len(c) == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no cycle contains [A, B] among %v", cycles)
	}
}

func TestDetectCyclesTriangle(t *testing.T) {
	// A -> B -> C -> A
	g := NewGraph()
	for _, id := range []string{"A", "B", "C"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("A", "B", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("B", "C", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("C", "A", EdgeRelated, 0.6, "2026-04-01T00:00:00Z")

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Fatal("DetectCycles found no cycles, want at least 1")
	}

	// Verify a cycle contains all three nodes
	found := false
	for _, c := range cycles {
		cs := cycleSet(c)
		if cs["A"] && cs["B"] && cs["C"] && len(c) == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no cycle contains [A, B, C] among %v", cycles)
	}
}

func TestDetectCyclesNone(t *testing.T) {
	// A -> B -> C (no back edges)
	g := NewGraph()
	for _, id := range []string{"A", "B", "C"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("A", "B", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("B", "C", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")

	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("DetectCycles on DAG = %v, want empty", cycles)
	}
}

func TestDetectCyclesSingleSelfLoop(t *testing.T) {
	// A -> A
	g := NewGraph()
	g.AddNode(Node{ID: "A", Type: NodeInstinct})
	g.AddEdge("A", "A", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Fatal("DetectCycles found no self-loop, want [A]")
	}

	found := false
	for _, c := range cycles {
		if reflect.DeepEqual(c, []string{"A"}) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no self-loop cycle [A] among %v", cycles)
	}
}

func TestDetectCyclesDisjoint(t *testing.T) {
	// Two independent cycles: A -> B -> A, C -> D -> C
	g := NewGraph()
	for _, id := range []string{"A", "B", "C", "D"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("A", "B", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("B", "A", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("C", "D", EdgeRelated, 0.6, "2026-04-01T00:00:00Z")
	g.AddEdge("D", "C", EdgeSupersedes, 0.5, "2026-04-01T00:00:00Z")

	cycles := g.DetectCycles()
	if len(cycles) < 2 {
		t.Fatalf("DetectCycles found %d cycles, want at least 2", len(cycles))
	}

	// Sort all cycles into normalized form for comparison
	var foundAB, foundCD bool
	for _, c := range cycles {
		sorted := make([]string, len(c))
		copy(sorted, c)
		sort.Strings(sorted)
		if reflect.DeepEqual(sorted, []string{"A", "B"}) {
			foundAB = true
		}
		if reflect.DeepEqual(sorted, []string{"C", "D"}) {
			foundCD = true
		}
	}
	if !foundAB {
		t.Error("cycle [A, B] not found")
	}
	if !foundCD {
		t.Error("cycle [C, D] not found")
	}
}

// cycleSet returns a set of node IDs in a cycle for easy membership testing.
func cycleSet(cycle []string) map[string]bool {
	s := make(map[string]bool, len(cycle))
	for _, id := range cycle {
		s[id] = true
	}
	return s
}
