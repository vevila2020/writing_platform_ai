package engine

import (
	"encoding/json"
	"testing"

	agentmodel "github.com/vevila2020/writing_platform_ai/agent/model"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func TestValidateFinalAssistantMessage(t *testing.T) {
	tests := []struct {
		name    string
		message *agentschema.Message
		finish  agentmodel.ModelFinishReasonClass
		wantErr bool
	}{
		{name: "text", message: agentschema.AssistantMessage("answer", nil), finish: agentmodel.ModelFinishReasonOther},
		{name: "multimodal", message: &agentschema.Message{
			Role: agentschema.Assistant,
			AssistantGenMultiContent: []json.RawMessage{
				json.RawMessage(`{"type":"output_text","text":"answer"}`),
			},
		}, finish: agentmodel.ModelFinishReasonOther},
		{name: "thinking only", message: &agentschema.Message{
			Role: agentschema.Assistant, ReasoningContent: "partial thought",
		}, finish: agentmodel.ModelFinishReasonOther, wantErr: true},
		{name: "incomplete thinking only", message: &agentschema.Message{
			Role: agentschema.Assistant, ReasoningContent: "partial thought",
		}, finish: agentmodel.ModelFinishReasonOutputLimit},
		{name: "missing", wantErr: true},
		{name: "tool call", message: agentschema.AssistantMessage("", []agentschema.ToolCall{{
			ID: "call", Type: "function", Function: agentschema.FunctionCall{Name: "read"},
		}}), finish: agentmodel.ModelFinishReasonOther, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateFinalAssistantMessage(test.message, test.finish)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestValidateAssistantTurnOutputUsesCurrentTurnTranscript(t *testing.T) {
	emptyFinal := agentschema.AssistantMessage("", nil)
	previousTurn := agentschema.AssistantMessage("previous turn", nil)
	currentReply := agentschema.AssistantMessage("current reply", nil)
	currentUser := agentschema.UserMessage("continue")
	thinkingOnly := &agentschema.Message{Role: agentschema.Assistant, ReasoningContent: "partial thought"}
	tests := []struct {
		name       string
		message    *agentschema.Message
		transcript []*agentschema.Message
		searchFrom int
		wantErr    bool
	}{
		{
			name: "accumulated current turn content", message: emptyFinal,
			transcript: []*agentschema.Message{previousTurn, currentUser, currentReply, emptyFinal}, searchFrom: 2,
		},
		{
			name: "previous turn content excluded", message: emptyFinal,
			transcript: []*agentschema.Message{previousTurn, currentUser, emptyFinal}, searchFrom: 2, wantErr: true,
		},
		{
			name: "thinking only", message: thinkingOnly,
			transcript: []*agentschema.Message{currentUser, thinkingOnly}, searchFrom: 1, wantErr: true,
		},
		{
			name: "missing final assistant", transcript: []*agentschema.Message{currentUser, currentReply}, searchFrom: 1, wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateAssistantTurnOutput(test.message, test.transcript, test.searchFrom, agentmodel.ModelFinishReasonOther)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestCompletionFinalAssistantSearchesCurrentTurnOnly(t *testing.T) {
	toolOnly := agentschema.AssistantMessage("", []agentschema.ToolCall{{
		ID: "submit", Type: "function", Function: agentschema.FunctionCall{Name: "submit"},
	}})
	transcript := []*agentschema.Message{
		agentschema.AssistantMessage("previous turn", nil),
		agentschema.UserMessage("continue"),
		agentschema.AssistantMessage("accepted prose", nil),
		toolOnly,
	}
	if got := completionFinalAssistant(transcript, toolOnly, 2); got.Content != "accepted prose" {
		t.Fatalf("completion content = %q, want current turn prose", got.Content)
	}
	if got := completionFinalAssistant(transcript, toolOnly, 3); got.Content != "" {
		t.Fatalf("completion borrowed content before search boundary: %q", got.Content)
	}
}
