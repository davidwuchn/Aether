package curation

import (
	"context"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Herald promotes high-confidence instincts to QUEEN.md. It reads instincts
// with confidence >= 0.80 and writes their trigger patterns to the Patterns
// section of QUEEN.md, following the queen.PromoteInstinct pattern.
type Herald struct {
	store *storage.Store
}

// NewHerald creates a new Herald ant backed by the given store.
func NewHerald(store *storage.Store) *Herald {
	return &Herald{store: store}
}

// Name returns the unique identifier for this agent.
func (h *Herald) Name() string { return "herald" }

// Caste returns the agent's role category.
func (h *Herald) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (h *Herald) Triggers() []agent.Trigger { return nil }

// Execute runs the herald. It delegates to Run.
func (h *Herald) Execute(ctx context.Context, event events.Event) error {
	_, err := h.Run(ctx, false)
	return err
}

// Run promotes instincts with confidence >= 0.80 to QUEEN.md.
func (h *Herald) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var file colony.InstinctsFile
	if err := h.store.LoadJSON("instincts.json", &file); err != nil {
		return StepResult{
			Name:    "herald",
			Success: true,
			Summary: map[string]any{"eligible": 0, "promoted": 0},
		}, nil
	}

	eligible := 0
	promoted := 0
	var queen *memory.QueenService
	if !dryRun {
		queen = memory.NewQueenService(h.store, events.NewBus(h.store, events.DefaultConfig()))
	}

	for _, inst := range file.Instincts {
		if inst.Archived || inst.Confidence < 0.75 {
			continue
		}
		summary := memory.SummarizeInstinctApplications(inst)
		if summary.Applications < 3 {
			continue
		}
		eligible++
		if dryRun {
			promoted++
			continue
		}
		result, err := queen.PromoteInstinct(ctx, "QUEEN.md", inst, "curation-herald")
		if err != nil {
			return StepResult{
				Name:    "herald",
				Success: false,
				Summary: map[string]any{"eligible": eligible, "promoted": promoted},
			}, err
		}
		if result.EntriesAdded > 0 {
			promoted++
		}
	}

	return StepResult{
		Name:    "herald",
		Success: true,
		Summary: map[string]any{"eligible": eligible, "promoted": promoted},
	}, nil
}
