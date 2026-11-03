package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	agentchat "github.com/vevila2020/writing_platform_ai/internal/agents/chat"
	agentcontext "github.com/vevila2020/writing_platform_ai/internal/agents/context"
	agentrun "github.com/vevila2020/writing_platform_ai/internal/agents/run"
	"github.com/vevila2020/writing_platform_ai/internal/book"

	publiccontext "github.com/vevila2020/writing_platform_ai/agent/context"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

const minimumInkDockContextHardLimit = publiccontext.DefaultLifecycleHardLimit

// ConversationContextConfig adapts InkDock's product context projection to the
// public Agent transcript boundary. Conversation owns current workspace/story
// state; Agent remains the only owner of model history.
type ConversationContextConfig struct {
	Conversation agentchat.Conversation
	BookService  *book.Service
	Request      agentchat.ChatRequest
	Options      agentrun.Options
	Identity     agentschema.CapabilityIdentity
	OnPrepared   func(agentchat.AgentContextPreparation)
}

type conversationContextSource struct {
	config ConversationContextConfig
}

func NewConversationContextSource(config ConversationContextConfig) (publiccontext.ContextSource, error) {
	if config.Conversation == nil {
		return nil, errors.New("InkDock Conversation ContextSource requires a conversation")
	}
	if strings.TrimSpace(config.Identity.Kind) == "" || config.Identity.Version == 0 {
		return nil, errors.New("InkDock Conversation ContextSource requires a stable identity")
	}
	config.Request = agentchat.CaptureChatRequestCallerInput(config.Request)
	return &conversationContextSource{config: config}, nil
}

func (source *conversationContextSource) Identity() agentschema.CapabilityIdentity {
	if source == nil {
		return agentschema.CapabilityIdentity{}
	}
	return source.config.Identity
}

func (source *conversationContextSource) Materialize(ctx context.Context, request publiccontext.ContextRequest) ([]agentschema.ContextFragment, error) {
	if source == nil || source.config.Conversation == nil {
		return nil, errors.New("InkDock Conversation ContextSource is unavailable")
	}
	if err := bindConversationCycle(source.config.Conversation, source.config.Options.AgentKind, request.Run); err != nil {
		return nil, err
	}
	if err := bindAgentCompaction(source.config.Conversation, request.Compaction); err != nil {
		return nil, err
	}
	prepared, err := agentchat.PrepareAgentContext(
		ctx,
		source.config.Conversation,
		source.config.Request,
		source.config.BookService,
		source.config.Options.Workspace,
		request.Run.StartedAt,
	)
	if err != nil {
		return nil, err
	}
	fragments, err := projectConversationContext(
		prepared, request, agentcontext.ModelContextBudgetFor(source.config.Conversation),
	)
	if err != nil {
		return nil, err
	}
	if source.config.OnPrepared != nil {
		source.config.OnPrepared(prepared)
	}
	return fragments, nil
}

func bindConversationCycle(conversation agentchat.Conversation, agentKind string, run agentschema.RunView) error {
	identity := agentrun.CycleIdentity{
		CommandID: agentrun.CommandID(run.CommandID), OperationID: agentrun.OperationID(run.ID), Cycle: run.Cycle,
	}
	if !agentrun.ValidCycleIdentity(identity) {
		return errors.New("InkDock Conversation ContextSource requires an exact Agent cycle identity")
	}
	if binder, ok := conversation.(agentrun.CycleIdentityBinder); ok {
		binder.BindAgentCycleIdentity(identity)
	}
	if binder, ok := conversation.(agentrun.AgentKindBinder); ok {
		binder.BindAgentKind(agentKind)
	}
	return nil
}

func projectConversationContext(
	prepared agentchat.AgentContextPreparation,
	request publiccontext.ContextRequest,
	budget agentcontext.Budget,
) ([]agentschema.ContextFragment, error) {
	modelUser := lastUserMessage(prepared.ModelContext.Messages)
	if modelUser == nil {
		return nil, errors.New("InkDock context assembly produced no final user message")
	}
	hardLimit := max(minimumInkDockContextHardLimit, budget.MaxTotalBytes+len(request.Input.Text)+minimumInkDockContextHardLimit)
	hardLimit = max(hardLimit, len(modelUser.Content))
	fragments, err := publiccontext.ExportLifecycleFragments(prepared.ModelContext.Context)
	if err != nil {
		return nil, fmt.Errorf("export InkDock model context to Agent lifecycle: %w", err)
	}
	fragments = append(fragments, agentschema.ContextFragment{
		Source: "denova.turn.context", Purpose: "preserve the exact localized InkDock turn assembly",
		Resource: firstContextValue(request.Run.CommandID, request.Run.ID, "turn"), Revision: request.Run.ID,
		Stability: agentschema.ContextTurn, Placement: agentschema.ContextFinalUserMessage, Rendering: agentschema.ContextRenderVerbatim,
		Content: modelUser.Content, HardLimit: hardLimit,
	})
	return fragments, nil
}

func firstContextValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func lastUserMessage(messages []*agentschema.Message) *agentschema.Message {
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index] != nil && messages[index].Role == agentschema.User && !publiccontext.IsContextStateMessage(messages[index]) && messages[index].TaskCompletion == nil {
			return messages[index].Clone()
		}
	}
	return nil
}
