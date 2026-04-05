package graph

import (
	"reflect"
	"testing"
)

func TestShortestPathDirect(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")

	path, err := g.ShortestPath("a", "b")
	if err != nil {
		t.Fatalf("ShortestPath error: %v", err)
	}
	want := []string{"a", "b"}
	if !reflect.DeepEqual(path, want) {
		t.Errorf("ShortestPath = %v, want %v", path, want)
	}
}

func TestShortestPathTwoHops(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")

	path, err := g.ShortestPath("a", "c")
	if err != nil {
		t.Fatalf("ShortestPath error: %v", err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(path, want) {
		t.Errorf("ShortestPath = %v, want %v", path, want)
	}
}

func TestShortestPathNotFound(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})

	path, err := g.ShortestPath("a", "b")
	if err != nil {
		t.Fatalf("ShortestPath error: %v", err)
	}
	if path != nil {
		t.Errorf("ShortestPath = %v, want nil", path)
	}
}

func TestShortestPathSameNode(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})

	path, err := g.ShortestPath("a", "a")
	if err != nil {
		t.Fatalf("ShortestPath error: %v", err)
	}
	want := []string{"a"}
	if !reflect.DeepEqual(path, want) {
		t.Errorf("ShortestPath = %v, want %v", path, want)
	}
}

func TestReach2Hops(t *testing.T) {
	// a -> b, a -> c, b -> d
	// From "a" with maxHops=2: b(1), c(1), d(2)
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "d", EdgeRelated, 0.6, "2026-04-01T00:00:00Z")

	result, err := g.Reach("a", 2, 0)
	if err != nil {
		t.Fatalf("Reach error: %v", err)
	}
	if result.Count != 3 {
		t.Fatalf("Reach count = %d, want 3", result.Count)
	}

	hopMap := map[string]int{}
	for _, rn := range result.Reachable {
		hopMap[rn.ID] = rn.Hop
	}
	if hopMap["b"] != 1 {
		t.Errorf("b hop = %d, want 1", hopMap["b"])
	}
	if hopMap["c"] != 1 {
		t.Errorf("c hop = %d, want 1", hopMap["c"])
	}
	if hopMap["d"] != 2 {
		t.Errorf("d hop = %d, want 2", hopMap["d"])
	}
}

func TestReach3Hops(t *testing.T) {
	// a -> b -> c -> d
	// From "a" with maxHops=3: b(1), c(2), d(3)
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("c", "d", EdgeRelated, 0.6, "2026-04-01T00:00:00Z")

	result, err := g.Reach("a", 3, 0)
	if err != nil {
		t.Fatalf("Reach error: %v", err)
	}
	if result.Count != 3 {
		t.Fatalf("Reach count = %d, want 3", result.Count)
	}

	hopMap := map[string]int{}
	for _, rn := range result.Reachable {
		hopMap[rn.ID] = rn.Hop
	}
	if hopMap["b"] != 1 || hopMap["c"] != 2 || hopMap["d"] != 3 {
		t.Errorf("hop map = %v, want {b:1, c:2, d:3}", hopMap)
	}
}

func TestReachMaxHopsClamped(t *testing.T) {
	// a -> b -> c -> d -> e (5 nodes, 4 edges)
	// Request maxHops=10, should be clamped to 3
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("c", "d", EdgeRelated, 0.6, "2026-04-01T00:00:00Z")
	g.AddEdge("d", "e", EdgeSupersedes, 0.5, "2026-04-01T00:00:00Z")

	result, err := g.Reach("a", 10, 0)
	if err != nil {
		t.Fatalf("Reach error: %v", err)
	}
	// With maxHops clamped to 3: b(1), c(2), d(3). e is at hop 4, so excluded.
	if result.Count != 3 {
		t.Fatalf("Reach(count with clamped hops) = %d, want 3", result.Count)
	}
}

func TestReachMinWeight(t *testing.T) {
	// a -> b (w=0.8), a -> c (w=0.2)
	// minWeight=0.5 should exclude c
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeExtends, 0.2, "2026-04-01T00:00:00Z")

	result, err := g.Reach("a", 1, 0.5)
	if err != nil {
		t.Fatalf("Reach error: %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("Reach(minWeight) count = %d, want 1", result.Count)
	}
	if result.Reachable[0].ID != "b" {
		t.Errorf("Reach(minWeight)[0].ID = %q, want %q", result.Reachable[0].ID, "b")
	}
}

func TestReachEmpty(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "lonely", Type: NodeInstinct})

	result, err := g.Reach("lonely", 1, 0)
	if err != nil {
		t.Fatalf("Reach error: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Reach(empty) count = %d, want 0", result.Count)
	}
	if result.HopsSearched != 0 {
		t.Errorf("Reach(empty) hopsSearched = %d, want 0", result.HopsSearched)
	}
}

func TestReachPathTracking(t *testing.T) {
	// a -> b -> c
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")

	result, err := g.Reach("a", 2, 0)
	if err != nil {
		t.Fatalf("Reach error: %v", err)
	}

	for _, rn := range result.Reachable {
		// Every path must start with "a"
		if len(rn.Path) == 0 || rn.Path[0] != "a" {
			t.Errorf("ReachNode %q path does not start with %q: %v", rn.ID, "a", rn.Path)
		}
		// Every path must end with the node's own ID
		if rn.Path[len(rn.Path)-1] != rn.ID {
			t.Errorf("ReachNode %q path does not end with its own ID: %v", rn.ID, rn.Path)
		}
	}

	// Verify specific paths
	pathMap := map[string][]string{}
	for _, rn := range result.Reachable {
		pathMap[rn.ID] = rn.Path
	}
	if !reflect.DeepEqual(pathMap["b"], []string{"a", "b"}) {
		t.Errorf("path to b = %v, want [a b]", pathMap["b"])
	}
	if !reflect.DeepEqual(pathMap["c"], []string{"a", "b", "c"}) {
		t.Errorf("path to c = %v, want [a b c]", pathMap["c"])
	}
}

func TestReachStartNotInResults(t *testing.T) {
	// a -> b -> a (cycle back to start)
	g := NewGraph()
	for _, id := range []string{"a", "b"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "a", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")

	result, err := g.Reach("a", 2, 0)
	if err != nil {
		t.Fatalf("Reach error: %v", err)
	}
	for _, rn := range result.Reachable {
		if rn.ID == "a" {
			t.Error("start node should not appear in Reach results")
		}
	}
}

func TestReachNotFound(t *testing.T) {
	g := NewGraph()
	_, err := g.Reach("nonexistent", 1, 0)
	if err == nil {
		t.Error("Reach on nonexistent node should return error")
	}
}
