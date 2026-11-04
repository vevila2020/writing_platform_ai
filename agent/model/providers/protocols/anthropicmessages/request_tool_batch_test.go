package anthropicmessages

import (
	"encoding/json"
	"testing"

	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func TestRequestMessagesGroupsParallelToolResultsIntoOneUserTurn(t *testing.T) {
	calls := []agentschema.ToolCall{
		{ID: "toolu_a", Type: "function", Function: agentschema.FunctionCall{Name: "read", Arguments: `{"path":"a.md"}`}},
		{ID: "toolu_b", Type: "function", Function: agentschema.FunctionCall{Name: "read", Arguments: `{"path":"b.md"}`}},
		{ID: "toolu_c", Type: "function", Function: agentschema.FunctionCall{Name: "glob", Arguments: `{"paths":["*.md"]}`}},
	}
	input := []*agentschema.Message{
		agentschema.UserMessage("inspect the book"),
		agentschema.AssistantMessage("", calls),
		// Completion order must not leak into the provider turn: Anthropic pairs
		// results with the assistant tool_use blocks in call order.
		agentschema.ToolMessage(agentschema.TextToolResult("c"), "toolu_c"),
		agentschema.ToolMessage(agentschema.TextToolResult("a"), "toolu_a"),
		agentschema.ToolMessage(agentschema.TextToolResult("b"), "toolu_b"),
	}
	_, messages, err := requestMessages(input, providers.ModelConfig{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(messages)
	if err != nil {
		t.Fatal(err)
	}
	var actual []struct {
		Role    string `json:"role"`
		Content []struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
		} `json:"content"`
	}
	if err := json.Unmarshal(encoded, &actual); err != nil {
		t.Fatal(err)
	}
	if len(actual) != 3 {
		t.Fatalf("messages = %s, want user, assistant, and one combined tool-result user turn", encoded)
	}
	if actual[0].Role != "user" || actual[1].Role != "assistant" || actual[2].Role != "user" {
		t.Fatalf("roles = %s", encoded)
	}
	results := actual[2].Content
	if len(results) != len(calls) {
		t.Fatalf("tool results = %s, want %d blocks in one user message", encoded, len(calls))
	}
	for index, call := range calls {
		if results[index].Type != "tool_result" || results[index].ToolUseID != call.ID {
			t.Fatalf("tool result %d = %s, want tool_result for %s", index, encoded, call.ID)
		}
	}
}

func TestRequestMessagesKeepsStandaloneToolResultOrder(t *testing.T) {
	input := []*agentschema.Message{
		agentschema.ToolMessage(agentschema.TextToolResult("first"), "toolu_only"),
	}
	_, messages, err := requestMessages(input, providers.ModelConfig{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(messages)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || len(messages[0].Content) != 1 {
		t.Fatalf("standalone tool result was not preserved: %s", encoded)
	}
}
