package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestOutputOK(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	outputOK("test")

	got := strings.TrimSpace(buf.String())
	expected := `{"ok":true,"result":"test"}`

	if got != expected {
		t.Errorf("outputOK(\"test\") = %q, want %q", got, expected)
	}

	// Verify it's valid JSON
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("outputOK produced invalid JSON: %v", err)
	}
	if m["ok"] != true {
		t.Errorf("ok = %v, want true", m["ok"])
	}
	if m["result"] != "test" {
		t.Errorf("result = %v, want \"test\"", m["result"])
	}
}

func TestOutputOKMap(t *testing.T) {
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	outputOK(map[string]string{"key": "value"})

	got := strings.TrimSpace(buf.String())

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("outputOK map produced invalid JSON: %v", err)
	}
	if m["ok"] != true {
		t.Errorf("ok = %v, want true", m["ok"])
	}
	result, ok := m["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if result["key"] != "value" {
		t.Errorf("result.key = %v, want \"value\"", result["key"])
	}
}

func TestOutputError(t *testing.T) {
	var buf bytes.Buffer
	stderr = &buf
	defer func() { stderr = os.Stderr }()

	outputError(1, "fail", nil)

	got := strings.TrimSpace(buf.String())
	expected := `{"ok":false,"error":"fail","code":1}`

	if got != expected {
		t.Errorf("outputError(1, \"fail\", nil) = %q, want %q", got, expected)
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("outputError produced invalid JSON: %v", err)
	}
	if m["ok"] != false {
		t.Errorf("ok = %v, want false", m["ok"])
	}
	if m["error"] != "fail" {
		t.Errorf("error = %v, want \"fail\"", m["error"])
	}
	if m["code"] != float64(1) {
		t.Errorf("code = %v, want 1", m["code"])
	}
}

func TestOutputErrorMessage(t *testing.T) {
	var buf bytes.Buffer
	stderr = &buf
	defer func() { stderr = os.Stderr }()

	outputErrorMessage("something went wrong")

	got := strings.TrimSpace(buf.String())

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("outputErrorMessage produced invalid JSON: %v", err)
	}
	if m["ok"] != false {
		t.Errorf("ok = %v, want false", m["ok"])
	}
	if m["error"] != "something went wrong" {
		t.Errorf("error = %v, want \"something went wrong\"", m["error"])
	}
	if m["code"] != float64(1) {
		t.Errorf("code = %v, want 1 (default code)", m["code"])
	}
}

func TestEnvelopeJSONMatch(t *testing.T) {
	// Verify outputOK produces byte-for-byte match with shell json_ok format:
	// json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }
	// When called with a quoted string: json_ok '"hello"'
	// Expected: {"ok":true,"result":"hello"}
	var buf bytes.Buffer
	stdout = &buf
	defer func() { stdout = os.Stdout }()

	outputOK("hello")

	got := strings.TrimSpace(buf.String())
	expected := `{"ok":true,"result":"hello"}`

	if got != expected {
		t.Errorf("envelope mismatch:\ngot:      %q\nexpected: %q", got, expected)
	}
}
