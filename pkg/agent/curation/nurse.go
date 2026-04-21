package curation

import (
	"context"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Nurse recalculates trust scores for instincts with out-of-date scores.
// It reads the instincts store and updates entries whose computed score
// differs from the stored value.
type Nurse struct {
	store *storage.Store
}

// NewNurse creates a new Nurse ant backed by the given store.
func NewNurse(store *storage.Store) *Nurse {
	return &Nurse{store: store}
}

// Name returns the unique identifier for this agent.
func (n *Nurse) Name() string { return "nurse" }

// Caste returns the agent's role category.
func (n *Nurse) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (n *Nurse) Triggers() []agent.Trigger { return nil }

// Execute runs the nurse. It delegates to Run.
func (n *Nurse) Execute(ctx context.Context, event events.Event) error {
	_, err := n.Run(ctx, false)
	return err
}

// Run reads instincts and recalculates trust scores for entries that need
// updating. Returns a summary of how many were recalculated.
func (n *Nurse) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var file colony.InstinctsFile
	if err := n.store.LoadJSON("instincts.json", &file); err != nil {
		// No instincts file yet -- nothing to recalculate
		return StepResult{
			Name:    "nurse",
			Success: true,
			Summary: map[string]any{"recalculated": 0},
		}, nil
	}

	recalculated := 0
	for i := range file.Instincts {
		if file.Instincts[i].Archived {
			continue
		}

		summary := memory.SummarizeInstinctApplications(file.Instincts[i])
		if file.Instincts[i].Provenance.ApplicationCount != summary.Applications {
			file.Instincts[i].Provenance.ApplicationCount = summary.Applications
			recalculated++
		}
		if summary.LastApplied != "" {
			current := ""
			if file.Instincts[i].Provenance.LastApplied != nil {
				current = *file.Instincts[i].Provenance.LastApplied
			}
			if current != summary.LastApplied {
				lastApplied := summary.LastApplied
				file.Instincts[i].Provenance.LastApplied = &lastApplied
				recalculated++
			}
		}

		if tierName, _ := memory.Tier(file.Instincts[i].TrustScore); tierName != "" && file.Instincts[i].TrustTier != tierName {
			file.Instincts[i].TrustTier = tierName
			recalculated++
		}
	}

	if !dryRun && recalculated > 0 {
		_ = n.store.SaveJSON("instincts.json", file)
	}

	return StepResult{
		Name:    "nurse",
		Success: true,
		Summary: map[string]any{"recalculated": recalculated},
	}, nil
}
