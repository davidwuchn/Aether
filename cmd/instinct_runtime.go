package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func loadActiveInstinctEntriesFromStore(s *storage.Store) ([]colony.InstinctEntry, error) {
	if s == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	var file colony.InstinctsFile
	if err := s.LoadJSON("instincts.json", &file); err != nil {
		return nil, err
	}

	active := make([]colony.InstinctEntry, 0, len(file.Instincts))
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		active = append(active, inst)
	}
	return active, nil
}

func loadRuntimeInstincts(s *storage.Store, state *colony.ColonyState) []colony.Instinct {
	if entries, err := loadActiveInstinctEntriesFromStore(s); err == nil {
		instincts := make([]colony.Instinct, 0, len(entries))
		for _, entry := range entries {
			instincts = append(instincts, instinctEntryToLegacy(entry))
		}
		return instincts
	}

	if state == nil || state.Memory.Instincts == nil {
		return []colony.Instinct{}
	}

	instincts := make([]colony.Instinct, len(state.Memory.Instincts))
	copy(instincts, state.Memory.Instincts)
	return instincts
}

func instinctEntryToLegacy(entry colony.InstinctEntry) colony.Instinct {
	evidence := []string{}
	if entry.Provenance.Evidence != "" {
		evidence = []string{entry.Provenance.Evidence}
	}

	applications, successes, failures := instinctApplicationStats(entry)
	if applications < entry.Provenance.ApplicationCount {
		applications = entry.Provenance.ApplicationCount
	}

	status := "active"
	if entry.Archived {
		status = "archived"
	}

	return colony.Instinct{
		ID:           entry.ID,
		Trigger:      entry.Trigger,
		Action:       entry.Action,
		Confidence:   entry.Confidence,
		Status:       status,
		Domain:       entry.Domain,
		Source:       entry.Provenance.Source,
		Evidence:     evidence,
		Tested:       applications > 0,
		CreatedAt:    entry.Provenance.CreatedAt,
		LastApplied:  entry.Provenance.LastApplied,
		Applications: applications,
		Successes:    successes,
		Failures:     failures,
	}
}

func instinctApplicationStats(entry colony.InstinctEntry) (applications, successes, failures int) {
	for _, raw := range entry.ApplicationHistory {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		applications++
		success, ok := item["success"].(bool)
		if !ok {
			continue
		}
		if success {
			successes++
		} else {
			failures++
		}
	}
	return applications, successes, failures
}

func loadInstinctFileOrEmpty(s *storage.Store) colony.InstinctsFile {
	file := colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{}}
	if s == nil {
		return file
	}
	if err := s.LoadJSON("instincts.json", &file); err != nil {
		return file
	}
	if file.Version == "" {
		file.Version = "1.0"
	}
	if file.Instincts == nil {
		file.Instincts = []colony.InstinctEntry{}
	}
	return file
}

func activeInstinctCount(s *storage.Store, state *colony.ColonyState) int {
	return len(loadRuntimeInstincts(s, state))
}

func sortedActiveInstinctEntries(file colony.InstinctsFile) []colony.InstinctEntry {
	active := make([]colony.InstinctEntry, 0, len(file.Instincts))
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		active = append(active, inst)
	}

	sort.Slice(active, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, active[i].Provenance.CreatedAt)
		tj, _ := time.Parse(time.RFC3339, active[j].Provenance.CreatedAt)
		if ti.Equal(tj) {
			return active[i].ID < active[j].ID
		}
		return ti.Before(tj)
	})
	return active
}

func loadRecentRuntimeInstincts(s *storage.Store, state *colony.ColonyState, limit int) []colony.Instinct {
	if limit <= 0 {
		return []colony.Instinct{}
	}

	file := loadInstinctFileOrEmpty(s)
	if recent := recentInstinctEntries(file, limit); len(recent) > 0 {
		out := make([]colony.Instinct, 0, len(recent))
		for _, entry := range recent {
			out = append(out, instinctEntryToLegacy(entry))
		}
		return out
	}

	if state == nil || len(state.Memory.Instincts) == 0 {
		return []colony.Instinct{}
	}

	sorted := make([]colony.Instinct, len(state.Memory.Instincts))
	copy(sorted, state.Memory.Instincts)
	sort.Slice(sorted, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, sorted[i].CreatedAt)
		tj, _ := time.Parse(time.RFC3339, sorted[j].CreatedAt)
		if ti.Equal(tj) {
			return sorted[i].ID < sorted[j].ID
		}
		return ti.After(tj)
	})
	if limit > len(sorted) {
		limit = len(sorted)
	}
	return sorted[:limit]
}

func recentInstinctEntries(file colony.InstinctsFile, limit int) []colony.InstinctEntry {
	active := sortedActiveInstinctEntries(file)
	if len(active) == 0 {
		return []colony.InstinctEntry{}
	}
	if limit > len(active) {
		limit = len(active)
	}
	recent := make([]colony.InstinctEntry, 0, limit)
	for i := len(active) - 1; i >= 0 && len(recent) < limit; i-- {
		recent = append(recent, active[i])
	}
	return recent
}
