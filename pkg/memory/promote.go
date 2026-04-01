package memory

import (
	"context"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

const triggerPrefixLen = 50
const instinctCap = 50

// PromotionResult holds the outcome of an instinct promotion.
type PromotionResult struct {
	Instinct   colony.InstinctEntry
	IsNew      bool
	WasDeduped bool
}

// PromoteService promotes observations to instincts.
type PromoteService struct {
	store *storage.Store
	bus   *events.Bus
}

// NewPromoteService creates a new promotion service.
func NewPromoteService(store *storage.Store, bus *events.Bus) *PromoteService {
	return &PromoteService{store: store, bus: bus}
}

// Promote promotes an observation to an instinct.
// RED phase stub: always returns error.
func (s *PromoteService) Promote(ctx context.Context, obs colony.Observation, colonyName string) (*PromotionResult, error) {
	return nil, nil
}
