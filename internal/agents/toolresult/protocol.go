package toolresult

import (
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
	agenttool "github.com/vevila2020/writing_platform_ai/agent/tool"
)

func validToolCall(call agentschema.ToolCall) bool {
	_, err := agenttool.NormalizeToolCallForModelContext(call, nil)
	return err == nil
}

func assistantHasIndependentContent(message *agentschema.Message) bool {
	if message == nil {
		return false
	}
	return message.Content != "" || message.Name != "" || message.ReasoningContent != "" ||
		len(message.MultiContent) > 0 || len(message.AssistantGenMultiContent) > 0
}
