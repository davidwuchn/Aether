package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// recordingAgent is a test double that records Execute calls.
type recordingAgent struct {
	name     string
	caste    Caste
	triggers []Trigger
	mu       sync.Mutex
	calls    []events.Event
	execErr  error
	// optional callback for simulating work
	onExecute func()
}

func (r *recordingAgent) Name() string        { return r.name }
func (r *recordingAgent) Caste() Caste        { return r.caste }
func (r *recordingAgent) Triggers() []Trigger { return r.triggers }
func (r *recordingAgent) Execute(_ context.Context, event events.Event) error {
	r.mu.Lock()
	r.calls = append(r.calls, event)
	r.mu.Unlock()
	if r.onExecute != nil {
		r.onExecute()
	}
	return r.execErr
}

func (r *recordingAgent) CallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *recordingAgent) Calls() []events.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]events.Event(nil), r.calls...)
}

// helper: create a test bus backed by a temp store
func newTestBus(t *testing.T) (*events.Bus, *storage.Store) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	bus := events.NewBus(store, events.DefaultConfig())
	return bus, store
}

func TestPoolNew(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	p, err := NewPool(reg, bus)
	if err != nil {
		t.Fatalf("NewPool() returned unexpected error: %v", err)
	}
	if p.maxG != 4 {
		t.Errorf("default maxG = %d, want 4", p.maxG)
	}
}

func TestPoolNewWithConcurrency(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	p, err := NewPool(reg, bus, WithMaxGoroutines(2))
	if err != nil {
		t.Fatalf("NewPool() returned unexpected error: %v", err)
	}
	if p.maxG != 2 {
		t.Errorf("maxG = %d, want 2", p.maxG)
	}
}

func TestPoolNewNilRegistry(t *testing.T) {
	bus, _ := newTestBus(t)

	_, err := NewPool(nil, bus)
	if err == nil {
		t.Fatal("NewPool(nil, bus) should return error")
	}
}

func TestPoolNewNilBus(t *testing.T) {
	reg := NewRegistry()

	_, err := NewPool(reg, nil)
	if err == nil {
		t.Fatal("NewPool(reg, nil) should return error")
	}
}

func TestPoolDispatch(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	agent := &recordingAgent{
		name:     "test-agent",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "test.*"}},
	}
	reg.Register(agent)

	p, err := NewPool(reg, bus)
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Publish an event
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	evt, err := bus.Publish(context.Background(), "test.run", payload, "test")
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	// Wait for agent to be called
	deadline := time.After(2 * time.Second)
	for {
		if agent.CallCount() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for agent Execute to be called")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Verify the event was dispatched correctly
	calls := agent.Calls()
	if len(calls) != 1 {
		t.Fatalf("Execute called %d times, want 1", len(calls))
	}
	if calls[0].Topic != "test.run" {
		t.Errorf("event topic = %q, want %q", calls[0].Topic, "test.run")
	}
	if calls[0].ID != evt.ID {
		t.Errorf("event ID = %q, want %q", calls[0].ID, evt.ID)
	}

	// Stop pool
	p.Stop()
	<-done
}

func TestPoolMultipleAgents(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	var agents []*recordingAgent
	for i := 0; i < 3; i++ {
		a := &recordingAgent{
			name:     fmt.Sprintf("agent-%d", i),
			caste:    CasteBuilder,
			triggers: []Trigger{{Topic: "task.*"}},
		}
		reg.Register(a)
		agents = append(agents, a)
	}

	p, err := NewPool(reg, bus)
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	payload, _ := json.Marshal(map[string]string{"task": "work"})
	bus.Publish(context.Background(), "task.work", payload, "test")

	// Wait for all agents
	deadline := time.After(2 * time.Second)
	for {
		allDone := true
		for _, a := range agents {
			if a.CallCount() < 1 {
				allDone = false
			}
		}
		if allDone {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for all agents to be called")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	for i, a := range agents {
		if a.CallCount() != 1 {
			t.Errorf("agent[%d] Execute called %d times, want 1", i, a.CallCount())
		}
	}

	p.Stop()
	<-done
}

func TestPoolBoundedConcurrency(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	var currentRunning atomic.Int32
	var maxRunning atomic.Int32

	// Create 5 agents that all match "work.*"
	for i := 0; i < 5; i++ {
		a := &recordingAgent{
			name:     fmt.Sprintf("bounded-%d", i),
			caste:    CasteBuilder,
			triggers: []Trigger{{Topic: "work.*"}},
			onExecute: func() {
				// Track concurrent executions
				cur := currentRunning.Add(1)
				for {
					old := maxRunning.Load()
					if cur <= old || maxRunning.CompareAndSwap(old, cur) {
						break
					}
				}
				// Simulate work
				time.Sleep(50 * time.Millisecond)
				currentRunning.Add(-1)
			},
		}
		reg.Register(a)
	}

	p, err := NewPool(reg, bus, WithMaxGoroutines(2))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	payload, _ := json.Marshal(map[string]string{"work": "bounded"})
	bus.Publish(context.Background(), "work.exec", payload, "test")

	// Wait for all 5 agents to complete
	deadline := time.After(5 * time.Second)
	for {
		// Check if all agents have been called
		allDone := true
		for i := 0; i < 5; i++ {
			agent, _ := reg.Get(fmt.Sprintf("bounded-%d", i))
			ra := agent.(*recordingAgent)
			if ra.CallCount() < 1 {
				allDone = false
			}
		}
		if allDone {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for bounded agents")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Verify max concurrent never exceeded limit of 2
	max := maxRunning.Load()
	if max > 2 {
		t.Errorf("max concurrent executions = %d, want <= 2", max)
	}

	// Verify all 5 completed (none dropped)
	// Wait for currentRunning to go back to 0
	for i := 0; i < 100; i++ {
		if currentRunning.Load() == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	p.Stop()
	<-done
}

func TestPoolStop(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	p, err := NewPool(reg, bus)
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	p.Stop()

	select {
	case <-done:
		// Success: Start returned after Stop
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Stop")
	}
}

func TestPoolNoMatchingAgents(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	// Register an agent that only matches "specific.*"
	agent := &recordingAgent{
		name:     "specific-agent",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "specific.*"}},
	}
	reg.Register(agent)

	p, err := NewPool(reg, bus)
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Publish event that does NOT match any agent
	payload, _ := json.Marshal(map[string]string{"action": "other"})
	bus.Publish(context.Background(), "other.event", payload, "test")

	time.Sleep(100 * time.Millisecond)

	// Verify no Execute calls
	if agent.CallCount() != 0 {
		t.Errorf("Execute called %d times, want 0 for non-matching event", agent.CallCount())
	}

	p.Stop()
	<-done
}
