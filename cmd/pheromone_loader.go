package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// loadPheromonesOnce loads pheromones.json using the session cache if available,
// falling back to a direct store.LoadJSON read. On a cache miss (or nil cache),
// the result is stored in the cache for subsequent calls within the same session.
func loadPheromonesOnce(s *storage.Store, c *cache.SessionCache) (colony.PheromoneFile, error) {
	var pf colony.PheromoneFile

	if c != nil {
		fullPath := filepath.Join(s.BasePath(), "pheromones.json")
		if cached, ok := c.Get(fullPath); ok {
			// Re-marshal the interface{} from cache and unmarshal into typed struct.
			raw, err := json.Marshal(cached)
			if err == nil {
				if err := json.Unmarshal(raw, &pf); err == nil {
					return pf, nil
				}
			}
			// Cache data corrupted -- fall through to disk load
		}
	}

	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return colony.PheromoneFile{}, err
	}

	if c != nil {
		fullPath := filepath.Join(s.BasePath(), "pheromones.json")
		_ = c.Set(fullPath, pf) // non-fatal: cache write failure doesn't affect result
	}

	return pf, nil
}

// loadPheromones loads pheromones.json using the global store, with no session cache.
// Returns nil if the file is missing or unreadable.
func loadPheromones() *colony.PheromoneFile {
	if store == nil {
		return nil
	}
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}
	return &pf
}

func signalActiveForPrompt(sig colony.PheromoneSignal, now time.Time) bool {
	if !sig.Active {
		return false
	}
	if signalExpiredByTime(sig, now) {
		return false
	}
	return computeEffectiveStrength(sig, now) >= 0.1
}

func filterSignalsForPrompt(signals []colony.PheromoneSignal, now time.Time) []colony.PheromoneSignal {
	filtered := make([]colony.PheromoneSignal, 0, len(signals))
	for _, sig := range signals {
		if signalActiveForPrompt(sig, now) {
			filtered = append(filtered, sig)
		}
	}
	return filtered
}

// extractSignalTextsFrom computes effective strengths, sorts, and returns formatted
// signal texts from a pre-loaded PheromoneFile. This avoids a redundant disk read
// when pheromones have already been loaded by the caller.
func extractSignalTextsFrom(pf *colony.PheromoneFile, maxSignals int) []string {
	if pf == nil || len(pf.Signals) == 0 {
		return nil
	}

	now := time.Now()
	signals := filterSignalsForPrompt(pf.Signals, now)
	if len(signals) == 0 {
		return nil
	}

	type scoredSignal struct {
		priority          int
		effectiveStrength float64
		text              string
	}

	var scored []scoredSignal
	for _, sig := range signals {
		eff := computeEffectiveStrength(sig, now)
		text := extractSignalText(sig.Content)
		if text == "" {
			continue
		}
		scored = append(scored, scoredSignal{
			priority:          signalPriority(sig.Type),
			effectiveStrength: eff,
			text:              fmt.Sprintf("%s: %s", sig.Type, text),
		})
	}

	// Sort by priority (ascending), then by effective strength (descending)
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].priority != scored[j].priority {
			return scored[i].priority < scored[j].priority
		}
		return scored[i].effectiveStrength > scored[j].effectiveStrength
	})

	// Take top N
	if len(scored) > maxSignals {
		scored = scored[:maxSignals]
	}

	result := make([]string, len(scored))
	for i, s := range scored {
		result[i] = s.text
	}
	return result
}
