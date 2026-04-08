package agent

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Task node status constants.
const (
	TaskNodePending    = "pending"
	TaskNodeInProgress = "in_progress"
	TaskNodeCompleted  = "completed"
	TaskNodeFailed     = "failed"
)

// TaskNode represents a single task in the dependency graph.
type TaskNode struct {
	ID        string
	Goal      string
	Caste     Caste
	Status    string
	DependsOn []string
	Criteria  []string
	TypeHint  string
}

// TaskResult holds the outcome of executing a single task.
type TaskResult struct {
	TaskID    string `json:"task_id"`
	AgentName string `json:"agent_name"`
	Caste     Caste  `json:"caste"`
	Success   bool   `json:"success"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	Duration  int64  `json:"duration_ms"`
}

// TaskContract defines an explicit, versioned agent-role contract.
type TaskContract struct {
	Version       int      `json:"version"`
	TaskType      string   `json:"task_type"`
	RequiredCaste Caste    `json:"required_caste"`
	Scope         []string `json:"scope"`
	Criteria      []string `json:"criteria"`
}

// TaskGraph is a dependency-aware task scheduler using a map-based adjacency list.
type TaskGraph struct {
	tasks    map[string]*TaskNode
	edges    map[string][]string // task ID -> list of dependent task IDs
	inDegree map[string]int      // task ID -> unresolved dependency count
}

// BuildTaskGraph constructs a TaskGraph from a slice of colony tasks.
// It assigns castes via RouteTask, detects cycles via Kahn's algorithm,
// and returns an error if circular dependencies are found.
func BuildTaskGraph(tasks []colony.Task) (*TaskGraph, error) {
	g := &TaskGraph{
		tasks:    make(map[string]*TaskNode, len(tasks)),
		edges:    make(map[string][]string),
		inDegree: make(map[string]int, len(tasks)),
	}

	// Phase 1: create nodes and set initial in-degree
	for _, t := range tasks {
		id := taskID(t)
		hint := ParseTypeHint(t.Goal)
		node := &TaskNode{
			ID:        id,
			Goal:      t.Goal,
			Caste:     RouteTask(t.Goal),
			Status:    TaskNodePending,
			DependsOn: t.DependsOn,
			Criteria:  t.SuccessCriteria,
			TypeHint:  hint,
		}
		g.tasks[id] = node
		g.inDegree[id] = len(t.DependsOn)
	}

	// Phase 2: build reverse edges (dependency -> dependents)
	for _, t := range tasks {
		id := taskID(t)
		for _, dep := range t.DependsOn {
			g.edges[dep] = append(g.edges[dep], id)
		}
	}

	// Phase 3: cycle detection via Kahn's algorithm
	processed := 0
	queue := make([]string, 0)
	for id, deg := range g.inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		processed++
		for _, dependent := range g.edges[curr] {
			g.inDegree[dependent]--
			if g.inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if processed < len(g.tasks) {
		return nil, fmt.Errorf("task graph has circular dependencies: processed %d of %d tasks", processed, len(g.tasks))
	}

	// Reset in-degree for runtime scheduling (Kahn's modified it)
	for _, t := range tasks {
		id := taskID(t)
		g.inDegree[id] = len(t.DependsOn)
	}

	return g, nil
}

// Ready returns all task nodes with in-degree 0 and status "pending".
func (g *TaskGraph) Ready() []*TaskNode {
	var ready []*TaskNode
	for id, deg := range g.inDegree {
		if deg == 0 && g.tasks[id].Status == TaskNodePending {
			ready = append(ready, g.tasks[id])
		}
	}
	return ready
}

// Complete marks a task as completed and decrements the in-degree of its
// dependents. It returns any newly-ready task nodes.
func (g *TaskGraph) Complete(id string) ([]*TaskNode, error) {
	node, ok := g.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	node.Status = TaskNodeCompleted

	var newlyReady []*TaskNode
	for _, dependent := range g.edges[id] {
		g.inDegree[dependent]--
		if g.inDegree[dependent] == 0 && g.tasks[dependent].Status == TaskNodePending {
			newlyReady = append(newlyReady, g.tasks[dependent])
		}
	}
	return newlyReady, nil
}

// Nodes returns all task nodes in the graph.
func (g *TaskGraph) Nodes() []*TaskNode {
	nodes := make([]*TaskNode, 0, len(g.tasks))
	for _, n := range g.tasks {
		nodes = append(nodes, n)
	}
	return nodes
}

// Node returns a single task node by ID, or nil if not found.
func (g *TaskGraph) Node(id string) *TaskNode {
	return g.tasks[id]
}

// taskID extracts a stable ID from a colony task. Falls back to Goal if no ID is set.
func taskID(t colony.Task) string {
	if t.ID != nil && *t.ID != "" {
		return *t.ID
	}
	return t.Goal
}
