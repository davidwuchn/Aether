package curation

import (
	"context"
	"strings"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Critic detects contradictions in the instinct store. Two instincts are
// considered contradictory if they share the same topic but express opposing
// conclusions (e.g., one says "prefer X" and another says "avoid X").
type Critic struct {
	store *storage.Store
}

// NewCritic creates a new Critic ant backed by the given store.
func NewCritic(store *storage.Store) *Critic {
	return &Critic{store: store}
}

// Name returns the unique identifier for this agent.
func (c *Critic) Name() string { return "critic" }

// Caste returns the agent's role category.
func (c *Critic) Caste() agent.Caste { return agent.CasteCurator }

// Triggers returns nil because the orchestrator handles triggering.
func (c *Critic) Triggers() []agent.Trigger { return nil }

// Execute runs the critic. It delegates to Run.
func (c *Critic) Execute(ctx context.Context, event events.Event) error {
	_, err := c.Run(ctx, false)
	return err
}

// Run scans instincts for contradictions. Two instincts sharing the same
// topic with different conclusion fields are flagged.
func (c *Critic) Run(ctx context.Context, dryRun bool) (StepResult, error) {
	var file colony.InstinctsFile
	if err := c.store.LoadJSON("instincts.json", &file); err != nil {
		return StepResult{
			Name:    "critic",
			Success: true,
			Summary: map[string]any{"contradictions": 0},
		}, nil
	}

	// Group by normalized trigger to find incompatible actions on the same cue.
	triggerActions := make(map[string][]string)
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		trigger := strings.ToLower(strings.TrimSpace(inst.Trigger))
		action := strings.ToLower(strings.TrimSpace(inst.Action))
		if trigger != "" && action != "" {
			triggerActions[trigger] = append(triggerActions[trigger], action)
		}
	}

	contradictions := 0
	for _, conclusions := range triggerActions {
		seen := make(map[string]bool)
		for _, conclusion := range conclusions {
			if seen[conclusion] {
				continue
			}
			seen[conclusion] = true
		}
		if len(seen) > 1 {
			contradictions++
		}
	}

	return StepResult{
		Name:    "critic",
		Success: true,
		Summary: map[string]any{"contradictions": contradictions},
	}, nil
}
