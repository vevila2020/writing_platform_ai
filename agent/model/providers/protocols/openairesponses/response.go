package openairesponses

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/openai/openai-go/v3/responses"
	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

const (
	ExtraKeyRequestID  = "openai-request-id"
	ExtraKeyProvider   = "provider"
	ExtraKeyProtocol   = "protocol"
	ExtraKeyResponseID = "response_id"
	ExtraKeyModel      = "model"
	ExtraKeyCreated    = "created"
	ExtraKeyStatus     = "response_status"
)

func responseMessage(response *responses.Response, rawResponse *http.Response, config providers.ModelConfig) (*agentschema.Message, error) {
	message := &agentschema.Message{
		Role: agentschema.Assistant,
		ResponseMeta: &agentschema.ResponseMeta{
			FinishReason: responseFinishReason(response),
		},
		Extra: responseIdentityExtra(response, rawResponse, string(config.Provider)),
	}
	var content strings.Builder
	var reasoning strings.Builder
	for _, item := range response.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				switch part.Type {
				case "output_text":
					content.WriteString(part.Text)
				case "refusal":
					content.WriteString(part.Refusal)
				}
			}
		case "function_call":
			call := item.AsFunctionCall()
			message.ToolCalls = append(message.ToolCalls, agentschema.ToolCall{
				ID:   call.CallID,
				Type: "function",
				Function: agentschema.FunctionCall{
					Name:      call.Name,
					Arguments: call.Arguments,
				},
			})
		case "reasoning":
			reasoningItem := item.AsReasoning()
			for _, summary := range reasoningItem.Summary {
				reasoning.WriteString(summary.Text)
			}
			for _, part := range reasoningItem.Content {
				reasoning.WriteString(part.Text)
			}
		}
	}
	message.Content = content.String()
	message.ReasoningContent = reasoning.String()
	if response.Status != "" {
		message.Extra[ExtraKeyStatus] = string(response.Status)
	}
	output, err := responseOutputItems(response.Output)
	if err != nil {
		return nil, err
	}
	if len(output) != 0 {
		continuation, err := providers.NewContinuation(config, output)
		if err != nil {
			return nil, err
		}
		message.Extra[providers.ExtraKeyContinuation] = continuation
	}
	if response.JSON.Usage.Valid() {
		message.ResponseMeta.Usage = responseUsage(response.Usage)
	}
	return message, nil
}

func responseUsage(usage responses.ResponseUsage) *agentschema.TokenUsage {
	return &agentschema.TokenUsage{
		PromptTokens:     int(usage.InputTokens),
		CompletionTokens: int(usage.OutputTokens),
		TotalTokens:      int(usage.TotalTokens),
		PromptTokenDetails: agentschema.PromptTokenDetails{
			CachedTokens: int(usage.InputTokensDetails.CachedTokens),
		},
		CompletionTokensDetails: agentschema.CompletionTokensDetails{
			ReasoningTokens: int(usage.OutputTokensDetails.ReasoningTokens),
		},
	}
}

func responseFinishReason(response *responses.Response) string {
	if response == nil {
		return ""
	}
	if response.Status == responses.ResponseStatusIncomplete {
		if reason := strings.TrimSpace(response.IncompleteDetails.Reason); reason != "" {
			return reason
		}
		return "incomplete"
	}
	for _, item := range response.Output {
		if item.Type == "function_call" {
			return "tool_calls"
		}
	}
	if response.Status != "" && response.Status != responses.ResponseStatusCompleted {
		return string(response.Status)
	}
	return "stop"
}

func responseIdentityExtra(response *responses.Response, rawResponse *http.Response, provider string) map[string]any {
	result := map[string]any{
		ExtraKeyProvider: provider,
		ExtraKeyProtocol: string(providers.ProtocolOpenAIResponses),
	}
	if requestID := responseRequestID(rawResponse); requestID != "" {
		result[ExtraKeyRequestID] = requestID
	}
	if response == nil {
		return result
	}
	if response.ID != "" {
		result[ExtraKeyResponseID] = response.ID
	}
	if response.Model != "" {
		result[ExtraKeyModel] = string(response.Model)
	}
	if response.CreatedAt != 0 {
		result[ExtraKeyCreated] = int64(response.CreatedAt)
	}
	return result
}

func responseOutputItems(items []responses.ResponseOutputItemUnion) ([]any, error) {
	if len(items) == 0 {
		return nil, nil
	}
	result := make([]any, 0, len(items))
	for index, item := range items {
		var value any
		if err := json.Unmarshal([]byte(item.RawJSON()), &value); err != nil {
			return nil, fmt.Errorf("openai responses: preserve output item %d: %w", index, err)
		}
		result = append(result, value)
	}
	return result, nil
}

func responseRequestID(response *http.Response) string {
	if response == nil {
		return ""
	}
	for _, name := range []string{"x-request-id", "openai-request-id", "request-id", "x-ms-request-id"} {
		if value := strings.TrimSpace(response.Header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

type streamState struct {
	rawResponse  *http.Response
	config       providers.ModelConfig
	pendingExtra map[string]any
	toolCalls    *streamToolCalls
}

func newStreamState(rawResponse *http.Response, config providers.ModelConfig) *streamState {
	return &streamState{
		rawResponse: rawResponse,
		config:      config,
		toolCalls:   newStreamToolCalls(),
	}
}

func (state *streamState) convert(event responses.ResponseStreamEventUnion) (*agentschema.Message, error) {
	switch event.Type {
	case "response.created":
		created := event.AsResponseCreated()
		state.pendingExtra = responseIdentityExtra(&created.Response, state.rawResponse, string(state.config.Provider))
		return nil, nil
	case "response.output_text.delta", "response.refusal.delta":
		return state.attachPending(&agentschema.Message{Role: agentschema.Assistant, Content: event.Delta}), nil
	case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
		return state.attachPending(&agentschema.Message{Role: agentschema.Assistant, ReasoningContent: event.Delta}), nil
	case "response.output_item.added":
		added := event.AsResponseOutputItemAdded()
		switch added.Item.Type {
		case "message", "reasoning":
			return nil, nil
		case "function_call":
			call, err := state.toolCalls.register(added.Item.AsFunctionCall())
			if err != nil {
				return nil, err
			}
			return state.toolCallMessage([]agentschema.ToolCall{call}), nil
		default:
			return nil, fmt.Errorf("openai responses stream: unsupported output item type %q", added.Item.Type)
		}
	case "response.function_call_arguments.delta":
		call, err := state.toolCalls.appendDelta(event.ItemID, event.Delta)
		if err != nil {
			return nil, err
		}
		return state.toolCallMessage([]agentschema.ToolCall{call}), nil
	case "response.function_call_arguments.done":
		done := event.AsResponseFunctionCallArgumentsDone()
		calls, err := state.toolCalls.completeArguments(done.ItemID, done.Arguments, toolArgumentsFromArgumentsDone)
		if err != nil {
			return nil, err
		}
		return state.toolCallMessage(calls), nil
	case "response.output_item.done":
		done := event.AsResponseOutputItemDone()
		if done.Item.Type != "function_call" {
			return nil, nil
		}
		calls, err := state.toolCalls.completeItem(done.Item.AsFunctionCall(), toolArgumentsFromOutputDone)
		if err != nil {
			return nil, err
		}
		return state.toolCallMessage(calls), nil
	case "response.completed", "response.incomplete":
		response := event.Response
		message := &agentschema.Message{
			Role: agentschema.Assistant,
			ResponseMeta: &agentschema.ResponseMeta{
				FinishReason: responseFinishReason(&response),
			},
			Extra: map[string]any{},
		}
		for _, item := range response.Output {
			if item.Type != "function_call" {
				continue
			}
			calls, err := state.toolCalls.completeItem(item.AsFunctionCall(), toolArgumentsFromCompleted)
			if err != nil {
				return nil, err
			}
			message.ToolCalls = append(message.ToolCalls, calls...)
		}
		if response.JSON.Usage.Valid() {
			message.ResponseMeta.Usage = responseUsage(response.Usage)
		}
		output, err := responseOutputItems(response.Output)
		if err != nil {
			return nil, err
		}
		if len(output) != 0 {
			continuation, err := providers.NewContinuation(state.config, output)
			if err != nil {
				return nil, err
			}
			message.Extra[providers.ExtraKeyContinuation] = continuation
		}
		message.Extra[ExtraKeyStatus] = string(response.Status)
		return state.attachPending(message), nil
	case "response.failed":
		return nil, responseFailure(&event.Response, state.rawResponse)
	case "error":
		message := strings.TrimSpace(event.Message)
		if event.Code != "" {
			message = event.Code + ": " + message
		}
		return nil, &providers.APIError{
			RequestID: responseRequestID(state.rawResponse),
			Message:   strings.TrimSpace(message),
		}
	// response.metadata carries provider-side annotations (for example moderation
	// metadata) that do not affect content, tool calls, or continuation replay.
	case "response.in_progress", "response.queued", "response.metadata",
		"response.content_part.added", "response.content_part.done",
		"response.output_text.done", "response.refusal.done",
		"response.reasoning_summary_part.added", "response.reasoning_summary_part.done",
		"response.reasoning_summary_text.done", "response.reasoning_text.done":
		return nil, nil
	default:
		return nil, fmt.Errorf("openai responses stream: unsupported event type %q", event.Type)
	}
}

func (state *streamState) toolCallMessage(calls []agentschema.ToolCall) *agentschema.Message {
	if len(calls) == 0 {
		return nil
	}
	return state.attachPending(&agentschema.Message{Role: agentschema.Assistant, ToolCalls: calls})
}

func (state *streamState) attachPending(message *agentschema.Message) *agentschema.Message {
	if message == nil || len(state.pendingExtra) == 0 {
		return message
	}
	if message.Extra == nil {
		message.Extra = make(map[string]any, len(state.pendingExtra))
	}
	for key, value := range state.pendingExtra {
		if _, exists := message.Extra[key]; !exists {
			message.Extra[key] = value
		}
	}
	state.pendingExtra = nil
	return message
}
