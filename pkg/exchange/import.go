package exchange

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ImportPheromones imports pheromone signals from XML.
func ImportPheromones(xmlData []byte) ([]colony.PheromoneSignal, error) {
	var phXML PheromoneXML
	if err := xml.Unmarshal(xmlData, &phXML); err != nil {
		return nil, fmt.Errorf("exchange: unmarshal pheromones XML: %w", err)
	}

	signals := make([]colony.PheromoneSignal, len(phXML.Signals))
	for i, s := range phXML.Signals {
		active := s.Active == "true"

		// Build content as json.RawMessage matching {"text": "..."} format
		content, err := json.Marshal(map[string]string{"text": s.Content})
		if err != nil {
			return nil, fmt.Errorf("exchange: marshal content for signal %s: %w", s.ID, err)
		}

		var expiresAt *string
		if s.ExpiresAt != "" {
			expiresAt = &s.ExpiresAt
		}

		signals[i] = colony.PheromoneSignal{
			ID:        s.ID,
			Type:      s.Type,
			Priority:  s.Priority,
			Source:    s.Source,
			CreatedAt: s.CreatedAt,
			ExpiresAt: expiresAt,
			Active:    active,
			Content:   json.RawMessage(content),
		}
	}
	return signals, nil
}

// ImportWisdom imports wisdom entries from XML.
func ImportWisdom(xmlData []byte) ([]WisdomEntry, error) {
	var wXML WisdomXML
	if err := xml.Unmarshal(xmlData, &wXML); err != nil {
		return nil, fmt.Errorf("exchange: unmarshal wisdom XML: %w", err)
	}

	var entries []WisdomEntry
	for _, p := range wXML.Philosophies {
		conf, _ := strconv.ParseFloat(p.Confidence, 64)
		entries = append(entries, WisdomEntry{
			ID: p.ID, Category: "philosophy", Confidence: conf,
			Domain: p.Domain, Source: p.Source, CreatedAt: p.CreatedAt, Content: p.Content,
		})
	}
	for _, p := range wXML.Patterns {
		conf, _ := strconv.ParseFloat(p.Confidence, 64)
		entries = append(entries, WisdomEntry{
			ID: p.ID, Category: "pattern", Confidence: conf,
			Domain: p.Domain, Source: p.Source, CreatedAt: p.CreatedAt, Content: p.Content,
		})
	}
	return entries, nil
}

// ImportRegistry imports colony entries from XML.
func ImportRegistry(xmlData []byte) ([]ColonyEntry, error) {
	var rXML RegistryXML
	if err := xml.Unmarshal(xmlData, &rXML); err != nil {
		return nil, fmt.Errorf("exchange: unmarshal registry XML: %w", err)
	}

	entries := make([]ColonyEntry, len(rXML.Colonies))
	for i, c := range rXML.Colonies {
		entry := ColonyEntry{
			ID: c.ID, Name: c.Name, Status: c.Status,
			CreatedAt: c.CreatedAt, ParentID: c.ParentID,
		}
		if c.Lineage != nil {
			entry.Ancestors = make([]Ancestor, len(c.Lineage.Ancestors))
			for j, a := range c.Lineage.Ancestors {
				entry.Ancestors[j] = Ancestor{ID: a.ID, Depth: a.Depth}
			}
		}
		entries[i] = entry
	}
	return entries, nil
}
