package graph

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/storage"
)

// LoadFromStore loads the graph from a JSON file via storage.Store.
// Supports both Go format (with nodes) and shell format (edges only).
// Shell format auto-creates nodes from edge source/target with inferred types.
func (g *Graph) LoadFromStore(store *storage.Store, path string) error {
	var jf JSONFile
	if err := store.LoadJSON(path, &jf); err != nil {
		return fmt.Errorf("graph: load from store %q: %w", path, err)
	}
	if err := g.Import(&jf); err != nil {
		return fmt.Errorf("graph: load from store %q: %w", path, err)
	}

	// Auto-create nodes for any edge source/target that wasn't in the nodes section.
	// This handles shell-format JSON which has edges but no nodes.
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, e := range jf.Edges {
		if _, ok := g.nodes[e.Source]; !ok {
			g.nodes[e.Source] = &Node{ID: e.Source, Type: inferType(e.Source)}
		}
		if _, ok := g.nodes[e.Target]; !ok {
			g.nodes[e.Target] = &Node{ID: e.Target, Type: inferType(e.Target)}
		}
	}

	return nil
}

// SaveToStore saves the graph to a JSON file via storage.Store.
// Uses atomic write via store.SaveJSON for crash safety.
func (g *Graph) SaveToStore(store *storage.Store, path string) error {
	jf, err := g.Export()
	if err != nil {
		return fmt.Errorf("graph: save to store %q: %w", path, err)
	}
	return store.SaveJSON(path, jf)
}
