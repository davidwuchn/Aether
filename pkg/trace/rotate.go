package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// RotateTraceFile checks the size of trace.jsonl and rotates it if it exceeds maxSizeMB.
// If rotated, the old file is renamed to trace.YYYY-MM-DD-HHMMSS.jsonl and a new empty
// trace.jsonl is created atomically. Returns true if rotation occurred.
func RotateTraceFile(store *storage.Store, maxSizeMB int) (rotated bool, err error) {
	if store == nil {
		return false, fmt.Errorf("trace: no store available")
	}
	if maxSizeMB <= 0 {
		maxSizeMB = 50
	}

	tracePath := filepath.Join(store.BasePath(), "trace.jsonl")
	info, err := os.Stat(tracePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("trace: stat trace.jsonl: %w", err)
	}

	maxBytes := int64(maxSizeMB) * 1024 * 1024
	if info.Size() <= maxBytes {
		return false, nil
	}

	suffix := time.Now().UTC().Format("2006-01-02-150405")
	rotatedPath := filepath.Join(store.BasePath(), fmt.Sprintf("trace.%s.jsonl", suffix))

	if err := os.Rename(tracePath, rotatedPath); err != nil {
		return false, fmt.Errorf("trace: rotate rename failed: %w", err)
	}

	f, err := os.OpenFile(tracePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("trace: create new trace.jsonl failed: %w", err)
	}
	_ = f.Close()

	return true, nil
}
