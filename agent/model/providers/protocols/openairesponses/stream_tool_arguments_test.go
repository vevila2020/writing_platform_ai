package openairesponses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

type parallelCallFixture struct {
	itemID string
	callID string
	name   string
}

var parallelCalls = []parallelCallFixture{
	{itemID: "fc_a", callID: "call_a", name: "read"},
	{itemID: "fc_b", callID: "call_b", name: "glob"},
}

func functionCallItem(call parallelCallFixture, arguments, status string) map[string]any {
	return map[string]any{
		"id": call.itemID, "type": "function_call", "call_id": call.callID,
		"name": call.name, "arguments": arguments, "status": status,
	}
}

func TestStreamRecoversParallelToolArgumentsWithoutDeltas(t *testing.T) {
	compact := []string{`{"path":"CREATOR.md"}`, `{"pattern":"chapters/*.md"}`}
	spaced := []string{`{"path": "CREATOR.md"}`, `{"pattern": "chapters/*.md"}`}
	tests := []struct {
		name string
		// events builds the per-call events between output_item.added and response.completed.
		addedArguments     func(index int) string
		events             func(index int, call parallelCallFixture) []map[string]any
		completedArguments func(index int) string
		want               []string
	}{
		{
			name:               "arguments only on output_item.added",
			addedArguments:     func(index int) string { return spaced[index] },
			completedArguments: func(int) string { return "" },
			want:               spaced,
		},
		{
			name: "arguments only on function_call_arguments.done",
			events: func(index int, call parallelCallFixture) []map[string]any {
				return []map[string]any{{
					"type": "response.function_call_arguments.done", "item_id": call.itemID,
					"output_index": index, "name": call.name, "arguments": spaced[index],
				}}
			},
			completedArguments: func(int) string { return "" },
			want:               spaced,
		},
		{
			name: "arguments only on output_item.done",
			events: func(index int, call parallelCallFixture) []map[string]any {
				return []map[string]any{{
					"type": "response.output_item.done", "output_index": index,
					"item": functionCallItem(call, spaced[index], "completed"),
				}}
			},
			completedArguments: func(int) string { return "" },
			want:               spaced,
		},
		{
			name:               "arguments only on response.completed",
			completedArguments: func(index int) string { return spaced[index] },
			want:               spaced,
		},
		{
			name: "partial delta completed by done event",
			events: func(index int, call parallelCallFixture) []map[string]any {
				return []map[string]any{
					{"type": "response.function_call_arguments.delta", "item_id": call.itemID, "output_index": index, "delta": compact[index][:5]},
					{"type": "response.function_call_arguments.done", "item_id": call.itemID, "output_index": index, "name": call.name, "arguments": compact[index]},
				}
			},
			completedArguments: func(index int) string { return compact[index] },
			want:               compact,
		},
		{
			name: "full delta is not duplicated by reformatted final arguments",
			events: func(index int, call parallelCallFixture) []map[string]any {
				return []map[string]any{
					{"type": "response.function_call_arguments.delta", "item_id": call.itemID, "output_index": index, "delta": compact[index]},
					{"type": "response.function_call_arguments.done", "item_id": call.itemID, "output_index": index, "name": call.name, "arguments": spaced[index]},
				}
			},
			completedArguments: func(index int) string { return spaced[index] },
			want:               compact,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			frames := []map[string]any{{
				"type":     "response.created",
				"response": map[string]any{"id": "resp_parallel", "object": "response", "created_at": 1, "status": "in_progress", "model": "m", "output": []any{}},
			}}
			for index, call := range parallelCalls {
				added := ""
				if test.addedArguments != nil {
					added = test.addedArguments(index)
				}
				frames = append(frames, map[string]any{
					"type": "response.output_item.added", "output_index": index,
					"item": functionCallItem(call, added, "in_progress"),
				})
			}
			if test.events != nil {
				for index, call := range parallelCalls {
					frames = append(frames, test.events(index, call)...)
				}
			}
			output := make([]any, 0, len(parallelCalls))
			for index, call := range parallelCalls {
				output = append(output, functionCallItem(call, test.completedArguments(index), "completed"))
			}
			frames = append(frames, map[string]any{
				"type":     "response.completed",
				"response": map[string]any{"id": "resp_parallel", "object": "response", "created_at": 1, "status": "completed", "model": "m", "output": output},
			})

			merged, err := streamFrames(t, frames)
			if err != nil {
				t.Fatalf("stream: %v", err)
			}
			if len(merged.ToolCalls) != len(parallelCalls) {
				t.Fatalf("tool calls = %#v", merged.ToolCalls)
			}
			for index, call := range parallelCalls {
				got := merged.ToolCalls[index]
				if got.ID != call.callID || got.Function.Name != call.name || got.Function.Arguments != test.want[index] {
					t.Fatalf("tool call %d = %#v, want id=%s name=%s arguments=%s", index, got, call.callID, call.name, test.want[index])
				}
			}
		})
	}
}

func TestStreamIgnoresProviderSideWebSearchItems(t *testing.T) {
	frames := []map[string]any{
		{"type": "response.created", "response": map[string]any{"id": "resp_web", "object": "response", "created_at": 1, "status": "in_progress", "model": "m", "output": []any{}}},
		{"type": "response.output_item.added", "output_index": 0, "item": map[string]any{
			"id": "ws_1", "type": "web_search_call", "status": "in_progress",
		}},
		{"type": "response.web_search_call.in_progress", "item_id": "ws_1", "output_index": 0},
		{"type": "response.web_search_call.searching", "item_id": "ws_1", "output_index": 0},
		{"type": "response.web_search_call.completed", "item_id": "ws_1", "output_index": 0},
		{"type": "response.output_item.done", "output_index": 0, "item": map[string]any{
			"id": "ws_1", "type": "web_search_call", "status": "completed",
		}},
		{"type": "response.output_text.annotation.added", "item_id": "msg_1", "output_index": 1, "content_index": 0, "annotation_index": 0, "annotation": map[string]any{
			"type": "url_citation", "url": "https://example.com", "title": "Example",
		}},
		{"type": "response.output_text.delta", "item_id": "msg_1", "output_index": 1, "content_index": 0, "delta": "研究结果"},
		{"type": "response.completed", "response": map[string]any{
			"id": "resp_web", "object": "response", "created_at": 1, "status": "completed", "model": "m",
			"output": []any{
				map[string]any{"id": "ws_1", "type": "web_search_call", "status": "completed"},
				map[string]any{
					"id": "msg_1", "type": "message", "role": "assistant", "status": "completed",
					"content": []any{map[string]any{"type": "output_text", "text": "研究结果", "annotations": []any{}}},
				},
			},
		}},
	}
	merged, err := streamFrames(t, frames)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if merged.Content != "研究结果" {
		t.Fatalf("content = %q, want 研究结果", merged.Content)
	}
	if len(merged.ToolCalls) != 0 {
		t.Fatalf("provider-side web search must not surface a client tool call: %#v", merged.ToolCalls)
	}
	if _, ok := merged.Extra[providers.ExtraKeyContinuation]; !ok {
		t.Fatal("continuation payload should retain the raw web search item")
	}
}

func TestStreamRejectsToolArgumentsThatContradictStreamedDeltas(t *testing.T) {
	call := parallelCalls[0]
	frames := []map[string]any{
		{"type": "response.created", "response": map[string]any{"id": "resp_conflict", "object": "response", "created_at": 1, "status": "in_progress", "model": "m", "output": []any{}}},
		{"type": "response.output_item.added", "output_index": 0, "item": functionCallItem(call, "", "in_progress")},
		{"type": "response.function_call_arguments.delta", "item_id": call.itemID, "output_index": 0, "delta": `{"path":"a.md"}`},
		{"type": "response.function_call_arguments.done", "item_id": call.itemID, "output_index": 0, "name": call.name, "arguments": `{"path":"b.md"}`},
	}
	_, err := streamFrames(t, frames)
	if err == nil || !strings.Contains(err.Error(), "fc_a") {
		t.Fatalf("stream error = %v, want contradiction error naming the item", err)
	}
}

func streamFrames(t *testing.T, frames []map[string]any) (*agentschema.Message, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		flusher := writer.(http.Flusher)
		for sequence, frame := range frames {
			frame["sequence_number"] = sequence
			encoded, err := json.Marshal(frame)
			if err != nil {
				panic(err)
			}
			_, _ = fmt.Fprintf(writer, "data: %s\n\n", encoded)
			flusher.Flush()
		}
	}))
	defer server.Close()

	model, err := NewAdapter().New(context.Background(), providers.ModelConfig{
		Provider:   providers.ProviderOpenAI,
		Protocol:   providers.ProtocolOpenAIResponses,
		Model:      "m",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := model.Stream(context.Background(), []*agentschema.Message{agentschema.UserMessage("hello")})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var chunks []*agentschema.Message
	for {
		chunk, receiveErr := stream.Recv()
		if errors.Is(receiveErr, io.EOF) {
			break
		}
		if receiveErr != nil {
			return nil, receiveErr
		}
		chunks = append(chunks, chunk)
	}
	return agentschema.ConcatMessages(chunks)
}
