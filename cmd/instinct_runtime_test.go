package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestLoadRuntimeInstincts_EmptyStandaloneFileDoesNotFallback(t *testing.T) {
	saveGlobals(t)

	dataDir := t.TempDir()
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	state := colony.ColonyState{
		Memory: colony.Memory{
			Instincts: []colony.Instinct{
				{ID: "legacy-1", Trigger: "legacy trigger", Action: "legacy action", Confidence: 0.8},
			},
		},
	}

	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{
		Version:   "1.0",
		Instincts: []colony.InstinctEntry{},
	}); err != nil {
		t.Fatalf("failed to save instincts.json: %v", err)
	}

	got := loadRuntimeInstincts(s, &state)
	if len(got) != 0 {
		t.Fatalf("expected no runtime instincts when standalone file is empty, got %+v", got)
	}
}
