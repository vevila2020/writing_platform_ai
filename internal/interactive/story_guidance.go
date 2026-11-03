package interactive

import (
	"strings"

	agentexecution "github.com/vevila2020/writing_platform_ai/agent/engine/execution"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

const userGuidanceKey = "denova.user_guidance"

// UserGuidanceMessage is an additional accepted player instruction for an
// unfinished Turn. It belongs to the product context, not Native Context State.
func UserGuidanceMessage(commandID, text string, attachments []agentschema.Attachment) *agentschema.Message {
	message := agentschema.UserMessage(text)
	message.Attachments = attachments
	message.Extra = map[string]any{userGuidanceKey: commandID}
	return message
}

func UserGuidanceCommand(message *agentschema.Message) string {
	if message == nil || message.Role != agentschema.User {
		return ""
	}
	id, _ := message.Extra[userGuidanceKey].(string)
	if strings.TrimSpace(id) == "" || agentexecution.ValidateIdempotencyKey(id) != nil {
		return ""
	}
	return id
}
