package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// mockToolServer creates a mock Anthropic API server that returns different
// responses based on the call count.
func mockToolServer(t *testing.T, responses []map[string]interface{}) *httptest.Server {
	t.Helper()
	var callCount int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := int(atomic.AddInt32(&callCount, 1)) - 1
		if idx >= len(responses) {
			t.Fatalf("unexpected API call %d (have %d responses)", idx+1, len(responses))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses[idx])
	}))
}

// textResponse creates a mock API response with text content only.
func textResponse(text string) map[string]interface{} {
	return map[string]interface{}{
		"id":          "msg_text",
		"type":        "message",
		"role":        "assistant",
		"model":       "claude-sonnet-4-20250514",
		"stop_reason": "end_turn",
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"usage": map[string]interface{}{
			"input_tokens":  10,
			"output_tokens": 5,
		},
	}
}

// toolUseResponse creates a mock API response with tool_use content blocks.
func toolUseResponse(toolID, toolName string, input map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":          "msg_tooluse",
		"type":        "message",
		"role":        "assistant",
		"model":       "claude-sonnet-4-20250514",
		"stop_reason": "tool_use",
		"content": []map[string]interface{}{
			{
				"type":  "tool_use",
				"id":    toolID,
				"name":  toolName,
				"input": input,
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":  15,
			"output_tokens": 10,
		},
	}
}

// multiToolUseResponse creates a mock API response with multiple tool_use blocks.
func multiToolUseResponse(tools []struct {
	ID   string
	Name string
	In   map[string]interface{}
}) map[string]interface{} {
	content := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		content[i] = map[string]interface{}{
			"type":  "tool_use",
			"id":    tool.ID,
			"name":  tool.Name,
			"input": tool.In,
		}
	}
	return map[string]interface{}{
		"id":          "msg_multi_tooluse",
		"type":        "message",
		"role":        "assistant",
		"model":       "claude-sonnet-4-20250514",
		"stop_reason": "tool_use",
		"content":     content,
		"usage": map[string]interface{}{
			"input_tokens":  20,
			"output_tokens": 15,
		},
	}
}

func newTestToolRunner(serverURL string, funcs map[string]ToolFunc, opts ...ToolRunnerOption) *ToolRunner {
	sdkClient := anthropic.NewClient(
		option.WithAPIKey("test-key"),
		option.WithBaseURL(serverURL),
	)
	client := &Client{
		apiClient: &sdkClient,
		model:     "claude-sonnet-4-20250514",
		maxTokens: 4096,
	}
	return NewToolRunner(client, funcs, opts...)
}

func TestToolRunnerNoToolUse(t *testing.T) {
	server := mockToolServer(t, []map[string]interface{}{
		textResponse("No tools needed."),
	})
	defer server.Close()

	runner := newTestToolRunner(server.URL, map[string]ToolFunc{
		"test_tool": func(ctx context.Context, input map[string]interface{}) (string, error) {
			return "should not be called", nil
		},
	})

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")),
	}

	resp, err := runner.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("Content blocks = %d, want 1", len(resp.Content))
	}
	if resp.Content[0].Text != "No tools needed." {
		t.Errorf("Text = %q, want %q", resp.Content[0].Text, "No tools needed.")
	}
}

func TestToolRunnerSingleToolCall(t *testing.T) {
	var toolCalled int32
	var receivedInput map[string]interface{}

	server := mockToolServer(t, []map[string]interface{}{
		toolUseResponse("toolu_123", "read_file", map[string]interface{}{
			"path": "/tmp/test.txt",
		}),
		textResponse("The file contains: hello world"),
	})
	defer server.Close()

	runner := newTestToolRunner(server.URL, map[string]ToolFunc{
		"read_file": func(ctx context.Context, input map[string]interface{}) (string, error) {
			atomic.AddInt32(&toolCalled, 1)
			receivedInput = input
			return "file contents: hello world", nil
		},
	})

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("Read the file")),
	}

	resp, err := runner.Run(context.Background(), messages)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify tool was called
	if atomic.LoadInt32(&toolCalled) != 1 {
		t.Errorf("tool called %d times, want 1", toolCalled)
	}

	// Verify tool received correct input
	if receivedInput["path"] != "/tmp/test.txt" {
		t.Errorf("tool input path = %v, want /tmp/test.txt", receivedInput["path"])
	}

	// Verify final response
	if len(resp.Content) != 1 {
		t.Fatalf("Content blocks = %d, want 1", len(resp.Content))
	}
	if resp.Content[0].Text != "The file contains: hello world" {
		t.Errorf("Text = %q, want %q", resp.Content[0].Text, "The file contains: hello world")
	}
}

func TestToolRunnerMaxIterations(t *testing.T) {
	// Server always returns tool_use, causing infinite loop
	server := mockToolServer(t, []map[string]interface{}{
		toolUseResponse("toolu_1", "loop_tool", map[string]interface{}{"n": 1}),
		toolUseResponse("toolu_2", "loop_tool", map[string]interface{}{"n": 2}),
		toolUseResponse("toolu_3", "loop_tool", map[string]interface{}{"n": 3}),
	})
	defer server.Close()

	runner := newTestToolRunner(server.URL, map[string]ToolFunc{
		"loop_tool": func(ctx context.Context, input map[string]interface{}) (string, error) {
			return "looping", nil
		},
	}, WithToolRunnerMaxIterations(3))

	_, err := runner.Run(context.Background(), []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("loop")),
	})
	if err == nil {
		t.Fatal("expected error when max iterations exceeded")
	}
	if !strings.Contains(err.Error(), "max iterations") {
		t.Errorf("error = %q, want containing 'max iterations'", err.Error())
	}
}

func TestToolRunnerUnknownTool(t *testing.T) {
	server := mockToolServer(t, []map[string]interface{}{
		toolUseResponse("toolu_unk", "nonexistent_tool", map[string]interface{}{}),
	})
	defer server.Close()

	runner := newTestToolRunner(server.URL, map[string]ToolFunc{
		"known_tool": func(ctx context.Context, input map[string]interface{}) (string, error) {
			return "ok", nil
		},
	})

	_, err := runner.Run(context.Background(), []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("use unknown tool")),
	})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "tool not found") {
		t.Errorf("error = %q, want containing 'tool not found'", err.Error())
	}
	if !strings.Contains(err.Error(), "nonexistent_tool") {
		t.Errorf("error = %q, want containing 'nonexistent_tool'", err.Error())
	}
}

func TestToolRunnerMultipleTools(t *testing.T) {
	var callCount int32

	server := mockToolServer(t, []map[string]interface{}{
		multiToolUseResponse([]struct {
			ID   string
			Name string
			In   map[string]interface{}
		}{
			{ID: "toolu_a", Name: "tool_a", In: map[string]interface{}{"x": 1}},
			{ID: "toolu_b", Name: "tool_b", In: map[string]interface{}{"y": 2}},
		}),
		textResponse("Both tools completed."),
	})
	defer server.Close()

	runner := newTestToolRunner(server.URL, map[string]ToolFunc{
		"tool_a": func(ctx context.Context, input map[string]interface{}) (string, error) {
			atomic.AddInt32(&callCount, 1)
			return "result_a", nil
		},
		"tool_b": func(ctx context.Context, input map[string]interface{}) (string, error) {
			atomic.AddInt32(&callCount, 1)
			return "result_b", nil
		},
	})

	resp, err := runner.Run(context.Background(), []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("run both")),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Both tools should have been called
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("tools called %d times, want 2", callCount)
	}

	// Verify final response
	if len(resp.Content) != 1 {
		t.Fatalf("Content blocks = %d, want 1", len(resp.Content))
	}
	if resp.Content[0].Text != "Both tools completed." {
		t.Errorf("Text = %q, want %q", resp.Content[0].Text, "Both tools completed.")
	}
}

func TestRegisterTool(t *testing.T) {
	sdkClient := anthropic.NewClient(option.WithAPIKey("test-key"))
	client := &Client{
		apiClient: &sdkClient,
		model:     "claude-sonnet-4-20250514",
		maxTokens: 4096,
	}
	runner := NewToolRunner(client, map[string]ToolFunc{})

	err := runner.RegisterTool("my_tool", struct{}{}, func(ctx context.Context, input map[string]interface{}) (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}
	if _, ok := runner.tools["my_tool"]; !ok {
		t.Error("tool not registered in tools map")
	}
}

func TestRegisterToolDuplicate(t *testing.T) {
	sdkClient := anthropic.NewClient(option.WithAPIKey("test-key"))
	client := &Client{
		apiClient: &sdkClient,
		model:     "claude-sonnet-4-20250514",
		maxTokens: 4096,
	}
	runner := NewToolRunner(client, map[string]ToolFunc{
		"dup": func(ctx context.Context, input map[string]interface{}) (string, error) {
			return "first", nil
		},
	})

	err := runner.RegisterTool("dup", struct{}{}, func(ctx context.Context, input map[string]interface{}) (string, error) {
		return "second", nil
	})
	if err == nil {
		t.Fatal("expected error for duplicate tool registration")
	}
}

// Ensure fmt is used
var _ = fmt.Sprintf
