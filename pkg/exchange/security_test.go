package exchange

import (
	"strings"
	"testing"
)

// TestXXEBlocked verifies that XML external entity attacks are rejected.
// Go's encoding/xml does not resolve external entities by default,
// so this should return a parse error or ignore the entity.
func TestXXEBlocked(t *testing.T) {
	xxeXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE pheromones [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<pheromones version="1.0" count="1">
  <signal id="sig_001" type="FOCUS" priority="normal" source="test" created_at="2026-01-01T00:00:00Z" active="true">
    <content><text>&xxe;</text></content>
  </signal>
</pheromones>`

	signals, err := ImportPheromones([]byte(xxeXML))
	// encoding/xml does not expand external entities — the entity reference
	// is left as-is or causes an error. Either way, the content should NOT
	// contain /etc/passwd data.
	if err != nil {
		t.Logf("XXE rejected with error (good): %v", err)
		return
	}
	for _, s := range signals {
		if strings.Contains(s.ID, "passwd") || strings.Contains(string(s.Content), "root:") {
			t.Error("XXE attack succeeded — file contents leaked into signal data")
		}
	}
}

// TestBillionLaughsBlocked verifies that exponential entity expansion (billion laughs)
// does not consume excessive memory.
func TestBillionLaughsBlocked(t *testing.T) {
	billionLaughs := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
]>
<pheromones version="1.0" count="1">
  <signal id="sig_001" type="FOCUS" priority="normal" source="test" created_at="2026-01-01T00:00:00Z" active="true">
    <content><text>&lol3;</text></content>
  </signal>
</pheromones>`

	// encoding/xml does not expand entities — this should parse without
	// blowing up memory, and the entity references should remain unresolved.
	signals, err := ImportPheromones([]byte(billionLaughs))
	if err != nil {
		t.Logf("Billion laughs rejected with error (acceptable): %v", err)
		return
	}
	// If it parses, content should not be gigabytes of "lol"
	for _, s := range signals {
		if len(string(s.Content)) > 1000 {
			t.Error("Billion laughs attack expanded — content is suspiciously large")
		}
	}
}

// TestDeepNesting verifies that deeply nested XML is handled gracefully.
func TestDeepNesting(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><pheromones version="1.0" count="1">`)
	b.WriteString(`<signal id="sig_001" type="FOCUS" priority="normal" source="test" created_at="2026-01-01T00:00:00Z" active="true">`)
	// 100 levels of nesting inside content
	for i := 0; i < 100; i++ {
		b.WriteString("<content>")
	}
	b.WriteString("deep text")
	for i := 0; i < 100; i++ {
		b.WriteString("</content>")
	}
	b.WriteString(`</signal></pheromones>`)

	// This should either parse correctly (finding the text) or error gracefully
	signals, err := ImportPheromones([]byte(b.String()))
	if err != nil {
		t.Logf("Deep nesting rejected: %v (acceptable)", err)
		return
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
}

// TestMalformedXML verifies that broken XML returns a clear error.
func TestMalformedXML(t *testing.T) {
	cases := []struct {
		name string
		xml  string
	}{
		{"not xml", "this is not xml at all"},
		{"unclosed tag", `<pheromones version="1.0"><signal id="s1"</pheromones>`},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ImportPheromones([]byte(tc.xml))
			if err == nil {
				t.Error("expected error for malformed XML, got nil")
			}
		})
	}
}
