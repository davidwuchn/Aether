package colony

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptIntegrityClassifyPromptSource(t *testing.T) {
	cases := []struct {
		source string
		want   PromptTrustClass
	}{
		{source: "/tmp/repo/.aether/data/COLONY_STATE.json", want: PromptTrustAuthorized},
		{source: "/tmp/home/.aether/QUEEN.md", want: PromptTrustTrusted},
		{source: "/tmp/home/.aether/hive/wisdom.json", want: PromptTrustAuthorized},
		{source: "/tmp/repo/README.md", want: PromptTrustUnknown},
		{source: "inline:task-brief", want: PromptTrustUnknown},
	}

	for _, tc := range cases {
		if got := ClassifyPromptSource(tc.source); got != tc.want {
			t.Errorf("ClassifyPromptSource(%q) = %q, want %q", tc.source, got, tc.want)
		}
	}
}

func TestPromptIntegrityDetectsRepoFixtureInstruction(t *testing.T) {
	fixture := filepath.Join("..", "..", "cmd", "testdata", "prompt-integrity-fixtures", "repo-instruction", "README.md")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	assessment := AssessPromptSource(filepath.ToSlash(filepath.Clean(fixture)), string(data))
	if assessment.BaseTrustClass != PromptTrustUnknown {
		t.Fatalf("base trust = %q, want %q", assessment.BaseTrustClass, PromptTrustUnknown)
	}
	if assessment.TrustClass != PromptTrustSuspicious {
		t.Fatalf("trust class = %q, want %q", assessment.TrustClass, PromptTrustSuspicious)
	}
	if assessment.Action != PromptIntegrityActionBlock {
		t.Fatalf("action = %q, want %q", assessment.Action, PromptIntegrityActionBlock)
	}
	if len(assessment.Findings) == 0 {
		t.Fatal("expected prompt integrity findings for suspicious fixture")
	}
	if !strings.Contains(strings.ToLower(assessment.Findings[0].Message), "prompt injection") {
		t.Fatalf("unexpected finding message: %q", assessment.Findings[0].Message)
	}
}
