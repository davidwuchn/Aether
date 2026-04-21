package curation

import (
	"context"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Archivist archives low-trust instincts. Instincts with confidence < 0.30
// are marked as "archived" to indicate they should not influence colony
// behavior. The instinct data is preserved but flagged for future pruning.
type Archivist struct {
	store     *storage.Store
	threshold float64
}

// NewArchivist creates a new Archivist ant backed by the given store.
func NewArchivist(store *storage.Store) *Archivist {
	return NewArchivistWithThreshold(store, 0.30)
}

// NewArchivistWithThreshold creates an Archivist using a custom archive threshold.
func NewArchivistWithThreshold(store *storage.Store, threshold float64) *Archivist {
	if threshold <= 0 {
		threshold = 0.30
	}
	return &Archivist{store: store, threshold: threshold}
}

// Name returns the unique identifier for this agent.
func (a *Archivist) Name() string { return "archivist" }

// Caste returns the agent's role category.
func (a *Archivist) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (a *Archivist) Triggers() []agent.Trigger { return nil }

// Execute runs the archivist. It delegates to Run.
func (a *Archivist) Execute(ctx context.Context, event events.Event) error {
	_, err := a.Run(ctx, false)
	return err
}

// Run reads instincts and marks those with confidence < 0.30 as archived.
// Returns the count of archived instincts.
func (a *Archivist) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var file colony.InstinctsFile
	if err := a.store.LoadJSON("instincts.json", &file); err != nil {
		return StepResult{
			Name:    "archivist",
			Success: true,
			Summary: map[string]any{"archived": 0, "threshold": a.threshold},
		}, nil
	}

	archived := 0
	for i := range file.Instincts {
		if file.Instincts[i].Archived {
			continue
		}
		if file.Instincts[i].TrustScore < a.threshold {
			file.Instincts[i].Archived = true
			archived++
		}
	}

	if !dryRun && archived > 0 {
		_ = a.store.SaveJSON("instincts.json", file)
	}

	return StepResult{
		Name:    "archivist",
		Success: true,
		Summary: map[string]any{"archived": archived, "threshold": a.threshold},
	}, nil
}
