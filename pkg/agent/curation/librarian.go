package curation

import (
	"context"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Librarian produces inventory statistics across colony data stores. It
// counts entries in observations, instincts, events, and pheromones to
// give an overview of colony memory health.
type Librarian struct {
	store *storage.Store
	bus   *events.Bus
}

// NewLibrarian creates a new Librarian ant with access to the store and bus.
func NewLibrarian(store *storage.Store, bus *events.Bus) *Librarian {
	return &Librarian{store: store, bus: bus}
}

// Name returns the unique identifier for this agent.
func (l *Librarian) Name() string { return "librarian" }

// Caste returns the agent's role category.
func (l *Librarian) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (l *Librarian) Triggers() []agent.Trigger { return nil }

// Execute runs the librarian. It delegates to Run.
func (l *Librarian) Execute(ctx context.Context, event events.Event) error {
	_, err := l.Run(ctx, false)
	return err
}

// Run counts entries across data stores and returns inventory statistics.
func (l *Librarian) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	observations := 0
	promotionCandidates := 0
	var obsFile colony.LearningFile
	if err := l.store.LoadJSON("learning-observations.json", &obsFile); err == nil {
		observations = len(obsFile.Observations)
		for _, obs := range obsFile.Observations {
			eligible, _ := memory.CheckPromotion(obs)
			if eligible {
				promotionCandidates++
			}
		}
	}

	instinctsActive := 0
	instinctsArchived := 0
	instinctsApplied := 0
	var instinctFile colony.InstinctsFile
	if err := l.store.LoadJSON("instincts.json", &instinctFile); err == nil {
		for _, inst := range instinctFile.Instincts {
			if inst.Archived {
				instinctsArchived++
				continue
			}
			instinctsActive++
			if memory.SummarizeInstinctApplications(inst).Applications > 0 {
				instinctsApplied++
			}
		}
	}

	pheromones := 0
	var pheromoneFile colony.PheromoneFile
	if err := l.store.LoadJSON("pheromones.json", &pheromoneFile); err == nil {
		for _, sig := range pheromoneFile.Signals {
			if sig.Active {
				pheromones++
			}
		}
	}

	// Count events from JSONL
	events := 0
	if lines, err := l.store.ReadJSONL("event-bus.jsonl"); err == nil {
		events = len(lines)
	}

	return StepResult{
		Name:    "librarian",
		Success: true,
		Summary: map[string]any{
			"observations":         observations,
			"promotion_candidates": promotionCandidates,
			"instincts_active":     instinctsActive,
			"instincts_archived":   instinctsArchived,
			"instincts_applied":    instinctsApplied,
			"events":               events,
			"pheromones":           pheromones,
		},
	}, nil
}
