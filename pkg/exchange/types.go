package exchange

import "encoding/xml"

// PheromoneXML is the root XML element for pheromone exchange.
// Matches the shell output format from pheromone-xml.sh.
type PheromoneXML struct {
	XMLName     xml.Name    `xml:"pheromones"`
	Version     string      `xml:"version,attr"`
	GeneratedAt string      `xml:"generated_at,attr"`
	Count       int         `xml:"count,attr"`
	Signals     []SignalXML `xml:"signal"`
}

// SignalXML represents a single pheromone signal in XML.
type SignalXML struct {
	XMLName   xml.Name `xml:"signal"`
	ID        string   `xml:"id,attr"`
	Type      string   `xml:"type,attr"`
	Priority  string   `xml:"priority,attr"`
	Source    string   `xml:"source,attr"`
	CreatedAt string   `xml:"created_at,attr"`
	ExpiresAt string   `xml:"expires_at,attr,omitempty"`
	Active    string   `xml:"active,attr"`
	Content   string   `xml:"content>text"`
}

// WisdomXML is the root XML element for queen wisdom exchange.
type WisdomXML struct {
	XMLName      xml.Name         `xml:"queen-wisdom"`
	Version      string           `xml:"version,attr"`
	ColonyID     string           `xml:"colony_id,attr"`
	GeneratedAt  string           `xml:"generated_at,attr"`
	Philosophies []PhilosophyXML  `xml:"philosophies>philosophy"`
	Patterns     []PatternXML     `xml:"patterns>pattern"`
}

// PhilosophyXML represents a queen philosophy entry.
type PhilosophyXML struct {
	XMLName    xml.Name `xml:"philosophy"`
	ID         string   `xml:"id,attr"`
	Confidence string   `xml:"confidence,attr"`
	Domain     string   `xml:"domain,attr"`
	Source     string   `xml:"source,attr"`
	CreatedAt  string   `xml:"created_at,attr"`
	Content    string   `xml:"content"`
}

// PatternXML represents a wisdom pattern entry.
type PatternXML struct {
	XMLName    xml.Name `xml:"pattern"`
	ID         string   `xml:"id,attr"`
	Confidence string   `xml:"confidence,attr"`
	Domain     string   `xml:"domain,attr"`
	Source     string   `xml:"source,attr"`
	CreatedAt  string   `xml:"created_at,attr"`
	Content    string   `xml:"content"`
}

// RegistryXML is the root XML element for colony registry exchange.
type RegistryXML struct {
	XMLName     xml.Name    `xml:"colony-registry"`
	Version     string      `xml:"version,attr"`
	GeneratedAt string      `xml:"generated_at,attr"`
	Colonies    []ColonyXML `xml:"colony"`
}

// ColonyXML represents a colony in registry XML.
type ColonyXML struct {
	XMLName  xml.Name    `xml:"colony"`
	ID       string      `xml:"id,attr"`
	Status   string      `xml:"status,attr"`
	Name     string      `xml:"name"`
	ParentID string      `xml:"parent_id,omitempty"`
	Lineage  *LineageXML `xml:"lineage,omitempty"`
	CreatedAt string     `xml:"created_at,attr"`
}

// LineageXML represents colony ancestry.
type LineageXML struct {
	Ancestors []AncestorXML `xml:"ancestor"`
}

// AncestorXML represents a single ancestor in the lineage.
type AncestorXML struct {
	XMLName xml.Name `xml:"ancestor"`
	ID      string   `xml:"id,attr"`
	Depth   int      `xml:"depth,attr"`
}

// ColonyArchiveXML is the composite archive format.
type ColonyArchiveXML struct {
	XMLName    xml.Name      `xml:"colony-archive"`
	ColonyID   string        `xml:"colony_id,attr"`
	SealedAt   string        `xml:"sealed_at,attr"`
	Version    string        `xml:"version,attr"`
	Pheromones *PheromoneXML `xml:"pheromones"`
	Wisdom     *WisdomXML    `xml:"queen-wisdom"`
	Registry   *RegistryXML  `xml:"colony-registry"`
}

// WisdomEntry represents a wisdom entry for export/import.
type WisdomEntry struct {
	ID         string
	Category   string // "philosophy" or "pattern"
	Confidence float64
	Domain     string
	Source     string
	CreatedAt  string
	Content    string
}

// ColonyEntry represents a colony for registry export/import.
type ColonyEntry struct {
	ID        string
	Name      string
	Status    string
	CreatedAt string
	ParentID  string
	Ancestors []Ancestor
}

// Ancestor represents a colony ancestor.
type Ancestor struct {
	ID    string
	Depth int
}
