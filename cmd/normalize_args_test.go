package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestNormalizeArgsPositional(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"normalize-args", "hello", "world"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("normalize-args returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	if envelope["result"] != "hello world" {
		t.Errorf("expected result='hello world', got: %v", envelope["result"])
	}
}

func TestNormalizeArgsEnvVar(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	os.Setenv("ARGUMENTS", "test input")
	defer os.Unsetenv("ARGUMENTS")

	rootCmd.SetArgs([]string{"normalize-args"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("normalize-args returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["result"] != "test input" {
		t.Errorf("expected result='test input', got: %v", envelope["result"])
	}
}

func TestNormalizeArgsEnvVarPrecedence(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	os.Setenv("ARGUMENTS", "from env")
	defer os.Unsetenv("ARGUMENTS")

	rootCmd.SetArgs([]string{"normalize-args", "from", "args"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("normalize-args returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	// ARGUMENTS env var takes precedence over positional args
	if envelope["result"] != "from env" {
		t.Errorf("expected result='from env' (env precedence), got: %v", envelope["result"])
	}
}

func TestNormalizeArgsEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	os.Unsetenv("ARGUMENTS")

	rootCmd.SetArgs([]string{"normalize-args"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("normalize-args returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["result"] != "" {
		t.Errorf("expected result='' for empty input, got: %v", envelope["result"])
	}
}

func TestNormalizeArgsWhitespace(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	os.Setenv("ARGUMENTS", "  too   much  \t  whitespace  ")
	defer os.Unsetenv("ARGUMENTS")

	rootCmd.SetArgs([]string{"normalize-args"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("normalize-args returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["result"] != "too much whitespace" {
		t.Errorf("expected collapsed whitespace 'too much whitespace', got: %v", envelope["result"])
	}
}
