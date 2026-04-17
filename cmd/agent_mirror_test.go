package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackagedAgentMirrorsMatchCanonicalSources(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	testCases := []struct {
		name string
		src  string
		dst  string
		ext  string
	}{
		{
			name: "claude mirror",
			src:  filepath.Join(repoRoot, ".claude", "agents", "ant"),
			dst:  filepath.Join(repoRoot, ".aether", "agents-claude"),
			ext:  ".md",
		},
		{
			name: "codex mirror",
			src:  filepath.Join(repoRoot, ".codex", "agents"),
			dst:  filepath.Join(repoRoot, ".aether", "agents-codex"),
			ext:  ".toml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srcEntries, err := os.ReadDir(tc.src)
			if err != nil {
				t.Fatalf("read %s: %v", tc.src, err)
			}

			for _, entry := range srcEntries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != tc.ext {
					continue
				}
				srcPath := filepath.Join(tc.src, entry.Name())
				dstPath := filepath.Join(tc.dst, entry.Name())

				srcData, err := os.ReadFile(srcPath)
				if err != nil {
					t.Fatalf("read %s: %v", srcPath, err)
				}
				dstData, err := os.ReadFile(dstPath)
				if err != nil {
					t.Fatalf("read %s: %v", dstPath, err)
				}
				if string(srcData) != string(dstData) {
					t.Fatalf("mirror drift detected for %s", entry.Name())
				}
			}
		})
	}
}
