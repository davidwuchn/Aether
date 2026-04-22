package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// writeMiddenEntry records a failure in the colony midden.
// It returns the generated entry ID or an error if the store is unavailable.
func writeMiddenEntry(category, source, message string) (string, error) {
	if store == nil {
		return "", fmt.Errorf("no store initialized")
	}
	if message == "" {
		return "", fmt.Errorf("message is required")
	}

	ts := time.Now().UTC()
	entryID := fmt.Sprintf("midden_%d_%d", ts.Unix(), os.Getpid())

	var mf colony.MiddenFile
	if err := store.LoadJSON("midden.json", &mf); err != nil {
		mf = colony.MiddenFile{
			Version: "1.0.0",
			Entries: []colony.MiddenEntry{},
		}
	}
	if mf.Entries == nil {
		mf.Entries = []colony.MiddenEntry{}
	}

	entry := colony.MiddenEntry{
		ID:        entryID,
		Timestamp: ts.Format(time.RFC3339),
		Category:  category,
		Source:    source,
		Message:   message,
		Reviewed:  false,
		Tags:      []string{},
	}

	mf.Entries = append(mf.Entries, entry)

	if err := store.SaveJSON("midden.json", mf); err != nil {
		return "", fmt.Errorf("failed to save midden: %w", err)
	}

	return entryID, nil
}
