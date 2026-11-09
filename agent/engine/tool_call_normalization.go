package engine

import (
	"fmt"
	"reflect"
	"strings"

	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
	agenttool "github.com/vevila2020/writing_platform_ai/agent/tool"
)

// normalizeToolCallBatch removes transport-level duplicate tool calls while
// preserving distinct parallel calls. Some gateway streams announce the same
// call_id as more than one output item; executing both would repeat the side
// effect and a canonical transcript could not pair both results safely. Only
// identical calls are folded. A conflicting reuse of a call_id remains fatal.
func normalizeToolCallBatch(calls []agentschema.ToolCall) ([]agentschema.ToolCall, error) {
	result := make([]agentschema.ToolCall, 0, len(calls))
	positions := make(map[string]int, len(calls))
	for _, call := range calls {
		id := strings.TrimSpace(call.ID)
		if id == "" {
			result = append(result, agentschema.CloneToolCalls([]agentschema.ToolCall{call})[0])
			continue
		}
		if position, exists := positions[id]; exists {
			if equivalentToolCall(result[position], call) {
				continue
			}
			return nil, fmt.Errorf("model returned conflicting tool calls for call ID %q", id)
		}
		positions[id] = len(result)
		result = append(result, agentschema.CloneToolCalls([]agentschema.ToolCall{call})[0])
	}
	return result, nil
}

func equivalentToolCall(left, right agentschema.ToolCall) bool {
	if normalizedToolCallType(left.Type) != normalizedToolCallType(right.Type) ||
		left.Function.Name != right.Function.Name || !reflect.DeepEqual(left.Extra, right.Extra) {
		return false
	}
	leftArguments, leftErr := agenttool.CanonicalizeToolArgumentsJSON(left.Function.Arguments)
	rightArguments, rightErr := agenttool.CanonicalizeToolArgumentsJSON(right.Function.Arguments)
	if leftErr == nil && rightErr == nil {
		return leftArguments == rightArguments
	}
	return left.Function.Arguments == right.Function.Arguments
}

func normalizedToolCallType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "function"
	}
	return value
}
