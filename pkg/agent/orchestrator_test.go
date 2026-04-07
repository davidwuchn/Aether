package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// mockOrchestratorAgent is a test double that records received events.
type mockOrchestratorAgent struct {
	name      string
	caste     Caste
	triggers  []Trigger
	execErr   error
	delay     time.Duration
	received  []events.Event
	mu        sync.Mutex
	execCount int64
}

func (m *mockOrchestratorAgent) Name() string        { return m.name }
func (m *mockOrchestratorAgent) Caste() Caste        { return m.caste }
func (m *mockOrchestratorAgent) Triggers() []Trigger { return m.triggers }

func (m *mockOrchestratorAgent) Execute(ctx context.Context, event events.Event) error {
	atomic.AddInt64(&m.execCount, 1)
	m.mu.Lock()
	m.received = append(m.received, event)
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.execErr
}

func (m *mockOrchestratorAgent) getReceived() []events.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.received
}

func makeOrchPhase(tasks []colony.Task) colony.Phase {
	return colony.Phase{
		ID:     1,
		Name:   "test-phase",
		Tasks:  tasks,
		Status: "pending",
	}
}

func TestPhaseOrchestrator_BuildGraph(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	builderAgent := &mockOrchestratorAgent{
		name:  "builder-1",
		caste: CasteBuilder,
		triggers: []Trigger{{Topic: "task.builder"}},
	}
	reg.Register(builderAgent)

	o := NewPhaseOrchestrator(reg, bus, nil)

	tasks := []colony.Task{
		{ID: strPtr("t1"), Goal: "implement feature", Status: "pending"},
	}
	phase := makeOrchPhase(tasks)

	result, err := o.Run(context.Background(), phase)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if result.PhaseID != 1 {
		t.Errorf("PhaseID = %d, want 1", result.PhaseID)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("Tasks count = %d, want 1", len(result.Tasks))
	}
	if !result.Tasks[0].Success {
		t.Errorf("Task[0].Success = false, want true")
	}
}

func TestPhaseOrchestrator_DispatchOrder(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	builderAgent := &mockOrchestratorAgent{
		name:  "builder-1",
		caste: CasteBuilder,
		triggers: []Trigger{{Topic: "task.builder"}},
	}
	watcherAgent := &mockOrchestratorAgent{
		name:  "watcher-1",
		caste: CasteWatcher,
		triggers: []Trigger{{Topic: "task.watcher"}},
	}
	reg.Register(builderAgent)
	reg.Register(watcherAgent)

	o := NewPhaseOrchestrator(reg, bus, nil)

	tasks := []colony.Task{
		{ID: strPtr("t1"), Goal: "implement auth", Status: "pending"},
		{ID: strPtr("t2"), Goal: "test auth", Status: "pending", DependsOn: []string{"t1"}},
	}
	phase := makeOrchPhase(tasks)

	result, err := o.Run(context.Background(), phase)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if result.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", result.Succeeded)
	}

	// Verify that both agents were dispatched (builder got t1, watcher got t2)
	builderReceived := builderAgent.getReceived()
	watcherReceived := watcherAgent.getReceived()

	if len(builderReceived) != 1 {
		t.Fatalf("builder received %d events, want 1", len(builderReceived))
	}
	if len(watcherReceived) != 1 {
		t.Fatalf("watcher received %d events, want 1", len(watcherReceived))
	}
}

func TestPhaseOrchestrator_ConcurrentDispatch(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	// Two agents with different castes so tasks dispatch concurrently
	builderAgent := &mockOrchestratorAgent{
		name:  "builder-1",
		caste: CasteBuilder,
		triggers: []Trigger{{Topic: "task.builder"}},
		delay: 50 * time.Millisecond,
	}
	watcherAgent := &mockOrchestratorAgent{
		name:  "watcher-1",
		caste: CasteWatcher,
		triggers: []Trigger{{Topic: "task.watcher"}},
		delay: 50 * time.Millisecond,
	}
	reg.Register(builderAgent)
	reg.Register(watcherAgent)

	o := NewPhaseOrchestrator(reg, bus, nil)

	// Two independent tasks (no dependencies) should run concurrently
	tasks := []colony.Task{
		{ID: strPtr("t1"), Goal: "implement feature", Status: "pending"},
		{ID: strPtr("t2"), Goal: "verify feature", Status: "pending"},
	}
	phase := makeOrchPhase(tasks)

	start := time.Now()
	result, err := o.Run(context.Background(), phase)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if result.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", result.Succeeded)
	}

	// If tasks ran concurrently, total time should be ~50ms, not ~100ms
	if elapsed > 90*time.Millisecond {
		t.Errorf("elapsed = %v, expected < 90ms (concurrent dispatch)", elapsed)
	}
}

func TestPhaseOrchestrator_ResultCollection(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	builderAgent := &mockOrchestratorAgent{
		name:  "builder-1",
		caste: CasteBuilder,
		triggers: []Trigger{{Topic: "task.builder"}},
	}
	reg.Register(builderAgent)

	o := NewPhaseOrchestrator(reg, bus, nil)

	tasks := []colony.Task{
		{ID: strPtr("t1"), Goal: "implement feature A", Status: "pending"},
		{ID: strPtr("t2"), Goal: "implement feature B", Status: "pending"},
	}
	phase := makeOrchPhase(tasks)

	result, err := o.Run(context.Background(), phase)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(result.Tasks) != 2 {
		t.Fatalf("Tasks count = %d, want 2", len(result.Tasks))
	}

	taskIDs := map[string]bool{}
	for _, tr := range result.Tasks {
		taskIDs[tr.TaskID] = tr.Success
	}

	if !taskIDs["t1"] || !taskIDs["t2"] {
		t.Errorf("expected both tasks to succeed, got: %v", taskIDs)
	}
	if result.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", result.Succeeded)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
}

func TestPhaseOrchestrator_AgentIsolation(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	builderAgent := &mockOrchestratorAgent{
		name:  "builder-1",
		caste: CasteBuilder,
		triggers: []Trigger{{Topic: "task.builder"}},
	}
	reg.Register(builderAgent)

	o := NewPhaseOrchestrator(reg, bus, nil)

	tasks := []colony.Task{
		{ID: strPtr("t1"), Goal: "implement feature A", Status: "pending", SuccessCriteria: []string{"code exists"}},
		{ID: strPtr("t2"), Goal: "implement feature B", Status: "pending", SuccessCriteria: []string{"tests pass"}},
	}
	phase := makeOrchPhase(tasks)

	_, err := o.Run(context.Background(), phase)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	received := builderAgent.getReceived()
	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}

	// Each event should contain ONLY the assigned task's data, no sibling tasks
	for _, evt := range received {
		var payload map[string]interface{}
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}

		// Must have task_id, goal, criteria, type_hint
		if _, ok := payload["task_id"]; !ok {
			t.Error("payload missing task_id")
		}
		if _, ok := payload["goal"]; !ok {
			t.Error("payload missing goal")
		}
		if _, ok := payload["criteria"]; !ok {
			t.Error("payload missing criteria")
		}

		// Must NOT have sibling task data or full phase plan
		if _, ok := payload["sibling_tasks"]; ok {
			t.Error("payload should not contain sibling_tasks")
		}
		if _, ok := payload["phase_plan"]; ok {
			t.Error("payload should not contain phase_plan")
		}
		if _, ok := payload["all_tasks"]; ok {
			t.Error("payload should not contain all_tasks")
		}
	}
}

func TestPhaseOrchestrator_TaskFailure(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	builderAgent := &mockOrchestratorAgent{
		name:  "builder-1",
		caste: CasteBuilder,
		triggers: []Trigger{{Topic: "task.builder"}},
		execErr: fmt.Errorf("build failed"),
	}
	watcherAgent := &mockOrchestratorAgent{
		name:  "watcher-1",
		caste: CasteWatcher,
		triggers: []Trigger{{Topic: "task.watcher"}},
	}
	reg.Register(builderAgent)
	reg.Register(watcherAgent)

	o := NewPhaseOrchestrator(reg, bus, nil)

	tasks := []colony.Task{
		{ID: strPtr("t1"), Goal: "implement feature", Status: "pending"},
		{ID: strPtr("t2"), Goal: "verify feature", Status: "pending"},
	}
	phase := makeOrchPhase(tasks)

	result, err := o.Run(context.Background(), phase)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	// One task should fail, the other should succeed
	if result.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", result.Succeeded)
	}
	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}

	// Verify which failed
	for _, tr := range result.Tasks {
		if tr.TaskID == "t1" && tr.Success {
			t.Error("t1 (builder with error) should have failed")
		}
		if tr.TaskID == "t2" && !tr.Success {
			t.Error("t2 (watcher) should have succeeded despite t1 failure")
		}
	}
}

func TestPhaseOrchestrator_CycleDetection(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	builderAgent := &mockOrchestratorAgent{
		name:  "builder-1",
		caste: CasteBuilder,
		triggers: []Trigger{{Topic: "task.builder"}},
	}
	reg.Register(builderAgent)

	o := NewPhaseOrchestrator(reg, bus, nil)

	// Circular dependencies: t1 -> t2 -> t1
	tasks := []colony.Task{
		{ID: strPtr("t1"), Goal: "task 1", Status: "pending", DependsOn: []string{"t2"}},
		{ID: strPtr("t2"), Goal: "task 2", Status: "pending", DependsOn: []string{"t1"}},
	}
	phase := makeOrchPhase(tasks)

	_, err := o.Run(context.Background(), phase)
	if err == nil {
		t.Fatal("Run() should return error for circular dependencies")
	}
}
