package engine

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
	agenttool "github.com/vevila2020/writing_platform_ai/agent/tool"
)

func TestNormalizeToolCallBatchFoldsEquivalentDuplicateIDs(t *testing.T) {
	firstIndex, secondIndex := 0, 1
	calls := []agentschema.ToolCall{
		{Index: &firstIndex, ID: "call-1", Type: "function", Function: agentschema.FunctionCall{Name: "read", Arguments: `{"path":"a.md"}`}},
		{Index: &secondIndex, ID: "call-1", Type: "function", Function: agentschema.FunctionCall{Name: "read", Arguments: `{ "path": "a.md" }`}},
		{ID: "call-2", Type: "function", Function: agentschema.FunctionCall{Name: "read", Arguments: `{"path":"b.md"}`}},
	}
	normalized, err := normalizeToolCallBatch(calls)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized) != 2 || normalized[0].ID != "call-1" || normalized[1].ID != "call-2" {
		t.Fatalf("normalized calls = %#v", normalized)
	}
	if len(calls) != 3 || calls[0].Function.Arguments != `{"path":"a.md"}` {
		t.Fatal("normalization mutated the model response")
	}
}

func TestToolInputProjectionFoldsEquivalentDuplicateCalls(t *testing.T) {
	firstIndex, secondIndex := 0, 1
	call := agentschema.ToolCall{
		Index: &firstIndex, ID: "call-1", Type: "function",
		Function: agentschema.FunctionCall{Name: "read", Arguments: ""},
	}
	duplicate := call
	duplicate.Index = &secondIndex
	duplicate.Function.Arguments = `{"path":"chapter.md"}`
	variant := &loopMessage{Role: agentschema.Assistant, ModelResponseOrdinal: 1, ToolExecutionNamespace: "test"}
	projector := newToolInputProjector(variant, EventSource{})
	started := make([]ToolInputStarted, 0, 2)
	deltas := make([]ToolInputDelta, 0, 2)
	emit := func(event Event) error {
		switch value := event.(type) {
		case ToolInputStarted:
			started = append(started, value)
		case ToolInputDelta:
			deltas = append(deltas, value)
		}
		return nil
	}
	for _, calls := range [][]agentschema.ToolCall{
		{call},
		{call, duplicate},
		{duplicate},
	} {
		if err := projector.observe(&agentschema.Message{ToolCalls: calls}, emit); err != nil {
			t.Fatal(err)
		}
	}
	if len(started) != 1 || len(deltas) != 1 || deltas[0].Delta != duplicate.Function.Arguments {
		t.Fatalf("started=%#v deltas=%#v", started, deltas)
	}
}

func TestToolInputProjectionKeepsDistinctParallelCalls(t *testing.T) {
	firstIndex, secondIndex := 0, 1
	call := agentschema.FunctionCall{Name: "read", Arguments: `{"path":"chapter.md"}`}
	variant := &loopMessage{Role: agentschema.Assistant, ModelResponseOrdinal: 1, ToolExecutionNamespace: "test"}
	projector := newToolInputProjector(variant, EventSource{})
	started := make([]ToolInputStarted, 0, 2)
	emit := func(event Event) error {
		if value, ok := event.(ToolInputStarted); ok {
			started = append(started, value)
		}
		return nil
	}
	// Parallel streams can announce the second call before the first. The
	// provider index must keep display identities aligned with the final batch.
	for _, calls := range [][]agentschema.ToolCall{
		{{Index: &secondIndex, ID: "call-2", Type: "function", Function: call}},
		{
			{Index: &firstIndex, ID: "call-1", Type: "function", Function: call},
			{Index: &secondIndex, ID: "call-2", Type: "function", Function: call},
		},
	} {
		if err := projector.observe(&agentschema.Message{ToolCalls: calls}, emit); err != nil {
			t.Fatal(err)
		}
	}
	if len(started) != 2 || started[0].ProviderCallID != "call-2" || started[0].Index != secondIndex ||
		started[1].ProviderCallID != "call-1" || started[1].Index != firstIndex {
		t.Fatalf("started=%#v", started)
	}
}

func TestNormalizeToolCallBatchRejectsConflictingDuplicateIDs(t *testing.T) {
	_, err := normalizeToolCallBatch([]agentschema.ToolCall{
		{ID: "call-1", Type: "function", Function: agentschema.FunctionCall{Name: "read", Arguments: `{"path":"a.md"}`}},
		{ID: "call-1", Type: "function", Function: agentschema.FunctionCall{Name: "read", Arguments: `{"path":"b.md"}`}},
	})
	if err == nil || !strings.Contains(err.Error(), `call ID "call-1"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestCanonicalToolBatchFoldsEquivalentDuplicateCalls(t *testing.T) {
	var calls atomic.Int32
	tool := &functionTool{name: "read", run: func(context.Context, string) (string, error) {
		calls.Add(1)
		return "chapter", nil
	}}
	duplicate := agentschema.ToolCall{
		ID: "call-1", Type: "function",
		Function: agentschema.FunctionCall{Name: "read", Arguments: `{"path":"chapter.md"}`},
	}
	model := &scriptedModel{responses: []scriptedModelResponse{
		{message: agentschema.AssistantMessage("reading", []agentschema.ToolCall{duplicate, duplicate})},
		{message: agentschema.AssistantMessage("done", nil)},
	}}
	native, err := newModelToolLoop(context.Background(), loopConfig{
		Name: "duplicate-tool", Model: model, Tools: []agenttool.ToolDefinition{testToolDefinition(tool)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextWithToolStartReceipt(context.Background())
	events := newLoopRunner(loopRunnerConfig{Agent: native}).Query(ctx, "go")
	var preparedCalls, completedResults int
	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			t.Fatal(event.Err)
		}
		if event.Output == nil {
			continue
		}
		if attempt := event.Output.ModelAttempt; attempt != nil {
			attempt.Receipt <- nil
		}
		if boundary := event.Output.ToolBatch; boundary != nil {
			phase, messages := boundary.snapshot()
			if phase == toolBatchPrepared {
				preparedCalls = len(messages[0].ToolCalls)
			}
			if phase == toolBatchCompleted {
				completedResults = len(messages) - 1
			}
			boundary.acknowledge(nil)
		}
		if execution := event.Output.ToolExecution; execution != nil && execution.Phase == toolExecutionStarted {
			execution.acknowledgeStart(nil)
		}
		if execution := event.Output.ToolExecution; execution != nil && execution.finishReceipt != nil {
			execution.finishReceipt <- nil
		}
	}
	if calls.Load() != 1 || preparedCalls != 1 || completedResults != 1 {
		t.Fatalf("executions=%d prepared=%d results=%d", calls.Load(), preparedCalls, completedResults)
	}
}
