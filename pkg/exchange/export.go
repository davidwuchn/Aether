package exchange

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
)

// extractTextContent extracts the "text" field from a json.RawMessage content.
// Returns empty string if content is not in {"text": "..."} format.
func extractTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return m["text"]
}

// ExportPheromones exports pheromone signals to XML matching shell format.
func ExportPheromones(signals []colony.PheromoneSignal) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	xmlSignals := make([]SignalXML, len(signals))
	for i, s := range signals {
		activeStr := "false"
		if s.Active {
			activeStr = "true"
		}
		expiresAt := ""
		if s.ExpiresAt != nil {
			expiresAt = *s.ExpiresAt
		}
		xmlSignals[i] = SignalXML{
			ID:        s.ID,
			Type:      s.Type,
			Priority:  s.Priority,
			Source:    s.Source,
			CreatedAt: s.CreatedAt,
			ExpiresAt: expiresAt,
			Active:    activeStr,
			Content:   extractTextContent(s.Content),
		}
	}

	phXML := PheromoneXML{
		Version:     "1.0",
		GeneratedAt: now,
		Count:       len(xmlSignals),
		Signals:     xmlSignals,
	}

	data, err := xml.MarshalIndent(phXML, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("exchange: marshal pheromones XML: %w", err)
	}

	return append([]byte(xml.Header), data...), nil
}

// ExportWisdom exports wisdom entries to XML, filtering by confidence threshold.
func ExportWisdom(entries []WisdomEntry, minConfidence float64, colonyID string) ([]byte, error) {
	var philosophies []PhilosophyXML
	var patterns []PatternXML

	for _, e := range entries {
		if e.Confidence < minConfidence {
			continue
		}
		confStr := fmt.Sprintf("%.2f", e.Confidence)
		if e.Category == "philosophy" {
			philosophies = append(philosophies, PhilosophyXML{
				ID: e.ID, Confidence: confStr, Domain: e.Domain,
				Source: e.Source, CreatedAt: e.CreatedAt, Content: e.Content,
			})
		} else {
			patterns = append(patterns, PatternXML{
				ID: e.ID, Confidence: confStr, Domain: e.Domain,
				Source: e.Source, CreatedAt: e.CreatedAt, Content: e.Content,
			})
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	wisdomXML := WisdomXML{
		Version: "1.0", ColonyID: colonyID, GeneratedAt: now,
		Philosophies: philosophies, Patterns: patterns,
	}

	data, err := xml.MarshalIndent(wisdomXML, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("exchange: marshal wisdom XML: %w", err)
	}
	return append([]byte(xml.Header), data...), nil
}

// ExportRegistry exports colony entries to XML.
func ExportRegistry(entries []ColonyEntry) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	colonies := make([]ColonyXML, len(entries))
	for i, c := range entries {
		var lineage *LineageXML
		if len(c.Ancestors) > 0 {
			ancestors := make([]AncestorXML, len(c.Ancestors))
			for j, a := range c.Ancestors {
				ancestors[j] = AncestorXML{ID: a.ID, Depth: a.Depth}
			}
			const maxAncestors = 10
			if len(ancestors) > maxAncestors {
				ancestors = ancestors[:maxAncestors]
			}
			lineage = &LineageXML{Ancestors: ancestors}
		}
		colonies[i] = ColonyXML{
			ID: c.ID, Status: c.Status, CreatedAt: c.CreatedAt,
			Name: c.Name, ParentID: c.ParentID, Lineage: lineage,
		}
	}

	regXML := RegistryXML{
		Version: "1.0", GeneratedAt: now, Colonies: colonies,
	}

	data, err := xml.MarshalIndent(regXML, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("exchange: marshal registry XML: %w", err)
	}
	return append([]byte(xml.Header), data...), nil
}
