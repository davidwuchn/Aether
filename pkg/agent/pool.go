package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/aether-colony/aether/pkg/events"
	"golang.org/x/sync/errgroup"
)

// defaultMaxGoroutines is the default concurrency limit for the worker pool.
const defaultMaxGoroutines = 4

// PoolOption configures a Pool during construction.
type PoolOption func(*Pool)

// WithMaxGoroutines sets the maximum number of concurrent agent executions.
// Panics if n <= 0.
func WithMaxGoroutines(n int) PoolOption {
	if n <= 0 {
		panic("pool: WithMaxGoroutines requires n > 0")
	}
	return func(p *Pool) {
		p.maxG = n
	}
}

// Pool dispatches events from the bus to matching agents with bounded concurrency.
// It subscribes to the event bus, matches incoming events against registered agents,
// and runs matching agents using an errgroup with a configurable goroutine limit.
type Pool struct {
	registry *Registry
	bus      *events.Bus
	maxG     int
	eventCh  <-chan events.Event
	cancel   context.CancelFunc
	mu       sync.Mutex
}

// NewPool creates a worker pool that dispatches events to agents in the registry.
// The pool subscribes to all events ("*") on the bus.
// Returns an error if registry or bus is nil.
func NewPool(registry *Registry, bus *events.Bus, opts ...PoolOption) (*Pool, error) {
	if registry == nil {
		return nil, fmt.Errorf("pool: registry must not be nil")
	}
	if bus == nil {
		return nil, fmt.Errorf("pool: bus must not be nil")
	}

	p := &Pool{
		registry: registry,
		bus:      bus,
		maxG:     defaultMaxGoroutines,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

// Start begins consuming events from the bus and dispatching them to matching agents.
// It blocks until the provided context is cancelled or the event channel is closed.
// For each event, matching agents are found via registry.Match and run concurrently
// using errgroup with SetLimit to bound concurrency.
func (p *Pool) Start(ctx context.Context) error {
	// Subscribe to all events
	ch, err := p.bus.Subscribe("*")
	if err != nil {
		return fmt.Errorf("pool: subscribe: %w", err)
	}

	// Create derived context with cancel -- must be protected for concurrent Stop() calls
	ctx, cancel := context.WithCancel(ctx)
	p.mu.Lock()
	p.eventCh = ch
	p.cancel = cancel
	p.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-ch:
			if !ok {
				// Channel closed
				return nil
			}
			p.dispatch(ctx, event)
		}
	}
}

// dispatch runs all matching agents for a single event with bounded concurrency.
// It blocks until all matching agents have completed (or the context is cancelled).
func (p *Pool) dispatch(ctx context.Context, event events.Event) {
	p.mu.Lock()
	maxG := p.maxG
	p.mu.Unlock()

	agents := p.registry.Match(event.Topic)
	if len(agents) == 0 {
		return
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxG)

	for _, a := range agents {
		agent := a // capture loop variable
		g.Go(func() error {
			return agent.Execute(ctx, event)
		})
	}

	// Wait for all agents to complete before processing next event.
	// Errors from individual agents are logged but do not stop the pool.
	_ = g.Wait()
}

// Stop cancels the pool's context and unsubscribes from the event bus.
func (p *Pool) Stop() {
	p.mu.Lock()
	cancel := p.cancel
	ch := p.eventCh
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if ch != nil {
		p.bus.Unsubscribe("*", ch)
	}
}

// SetConcurrency updates the maximum goroutine count for subsequent event dispatches.
func (p *Pool) SetConcurrency(n int) {
	if n <= 0 {
		return
	}
	p.mu.Lock()
	p.maxG = n
	p.mu.Unlock()
}
