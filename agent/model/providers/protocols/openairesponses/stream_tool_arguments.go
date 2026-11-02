package openairesponses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	agentschema "github.com/alfredxw/denova/agent/schema"
	"github.com/openai/openai-go/v3/responses"
)

// streamToolCalls tracks the function_call items of one Responses stream so tool
// arguments survive every delivery shape the protocol allows. OpenAI streams
// arguments as function_call_arguments.delta events, but compatible gateways may
// skip deltas (observed for parallel tool calls) and only carry the complete
// arguments on output_item.added, function_call_arguments.done,
// output_item.done, or response.completed.
//
// Streamed chunks cannot be retracted once emitted, so every later source is
// reconciled against what was already streamed: missing text is emitted as an
// additional chunk with the same index, equivalent JSON is accepted as is, and a
// contradiction fails the stream instead of executing ambiguous arguments.
type streamToolCalls struct {
	byItem    map[string]*streamToolCall
	nextIndex int
}

type streamToolCall struct {
	index     int
	arguments strings.Builder
}

type toolArgumentsSource string

const (
	toolArgumentsFromArgumentsDone toolArgumentsSource = "response.function_call_arguments.done"
	toolArgumentsFromOutputDone    toolArgumentsSource = "response.output_item.done"
	toolArgumentsFromCompleted     toolArgumentsSource = "response.completed"
)

func newStreamToolCalls() *streamToolCalls {
	return &streamToolCalls{byItem: make(map[string]*streamToolCall)}
}

// register starts tracking a function_call item and returns its opening chunk,
// including any arguments the gateway already attached to the item.
func (calls *streamToolCalls) register(item responses.ResponseFunctionToolCall) (agentschema.ToolCall, error) {
	if _, exists := calls.byItem[item.ID]; exists {
		return agentschema.ToolCall{}, fmt.Errorf("openai responses stream: duplicate function call item %q", item.ID)
	}
	call := &streamToolCall{index: calls.nextIndex}
	calls.nextIndex++
	calls.byItem[item.ID] = call
	call.arguments.WriteString(item.Arguments)
	index := call.index
	return agentschema.ToolCall{
		Index:    &index,
		ID:       item.CallID,
		Type:     "function",
		Function: agentschema.FunctionCall{Name: item.Name, Arguments: item.Arguments},
	}, nil
}

func (calls *streamToolCalls) appendDelta(itemID, delta string) (agentschema.ToolCall, error) {
	call, ok := calls.byItem[itemID]
	if !ok {
		return agentschema.ToolCall{}, fmt.Errorf("openai responses stream: arguments arrived before function call item %q", itemID)
	}
	call.arguments.WriteString(delta)
	index := call.index
	return agentschema.ToolCall{Index: &index, Function: agentschema.FunctionCall{Arguments: delta}}, nil
}

// completeArguments reconciles the final arguments of a known item and returns
// the chunk that fills in text the stream has not delivered yet, if any.
func (calls *streamToolCalls) completeArguments(itemID, final string, source toolArgumentsSource) ([]agentschema.ToolCall, error) {
	call, ok := calls.byItem[itemID]
	if !ok {
		return nil, fmt.Errorf("openai responses stream: %s arrived before function call item %q", source, itemID)
	}
	return call.reconcile(itemID, final, source)
}

// completeItem reconciles a finished function_call item. Items the stream never
// announced are registered here so gateways that omit output_item.added still
// produce complete tool calls.
func (calls *streamToolCalls) completeItem(item responses.ResponseFunctionToolCall, source toolArgumentsSource) ([]agentschema.ToolCall, error) {
	call, ok := calls.byItem[item.ID]
	if !ok {
		opening, err := calls.register(item)
		if err != nil {
			return nil, err
		}
		slog.Info("Recovered unannounced OpenAI Responses function call", "item_id", item.ID, "source", source, "arguments_bytes", len(item.Arguments))
		return []agentschema.ToolCall{opening}, nil
	}
	return call.reconcile(item.ID, item.Arguments, source)
}

func (call *streamToolCall) reconcile(itemID, final string, source toolArgumentsSource) ([]agentschema.ToolCall, error) {
	streamed := call.arguments.String()
	switch {
	case final == "" || final == streamed:
		return nil, nil
	case strings.HasPrefix(final, streamed):
		remainder := final[len(streamed):]
		call.arguments.WriteString(remainder)
		slog.Info("Recovered OpenAI Responses tool arguments missing from deltas", "item_id", itemID, "source", source,
			"streamed_bytes", len(streamed), "recovered_bytes", len(remainder))
		index := call.index
		return []agentschema.ToolCall{{Index: &index, Function: agentschema.FunctionCall{Arguments: remainder}}}, nil
	case equivalentJSON(streamed, final):
		return nil, nil
	default:
		return nil, fmt.Errorf("openai responses stream: %s arguments for function call item %q contradict streamed deltas", source, itemID)
	}
}

func equivalentJSON(left, right string) bool {
	leftValue, leftOK := decodeJSONValue(left)
	rightValue, rightOK := decodeJSONValue(right)
	return leftOK && rightOK && reflect.DeepEqual(leftValue, rightValue)
}

func decodeJSONValue(text string) (any, bool) {
	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil || decoder.More() {
		return nil, false
	}
	return value, true
}
