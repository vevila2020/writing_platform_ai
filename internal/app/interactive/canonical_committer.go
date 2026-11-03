package interactiveapp

import (
	"context"
	"errors"

	agentchat "github.com/vevila2020/writing_platform_ai/internal/agents/chat"
	agentlifecycle "github.com/vevila2020/writing_platform_ai/internal/agents/lifecycle"
	agentrun "github.com/vevila2020/writing_platform_ai/internal/agents/run"
	"github.com/vevila2020/writing_platform_ai/internal/agents/session"

	agentcanonical "github.com/vevila2020/writing_platform_ai/agent/session/canonical"
)

type CanonicalCommitterConfig struct {
	Conversation *Conversation
	Options      agentrun.Options
}

type canonicalConversationCommitter struct{ config CanonicalCommitterConfig }

func NewCanonicalCommitter(config CanonicalCommitterConfig) (agentlifecycle.ConversationCommitter, error) {
	if config.Conversation == nil {
		return nil, errors.New("InkDock Game committer requires a conversation")
	}
	config.Options = config.Options.Normalize(config.Options.Workspace)
	return &canonicalConversationCommitter{config: config}, nil
}

// NewAgentConversationCommitter exposes only the reusable lifecycle seam to
// the execution host; game turn and lore persistence remain app-owned.
func (conversation *Conversation) NewAgentConversationCommitter(
	options agentrun.Options,
) (agentlifecycle.ConversationCommitter, error) {
	return NewCanonicalCommitter(CanonicalCommitterConfig{
		Conversation: conversation, Options: options,
	})
}

func (committer *canonicalConversationCommitter) MaterializeInput(
	ctx context.Context,
	request agentcanonical.InputCommitRequest,
) (agentcanonical.CommitReceipt, error) {
	receipt, err := committer.config.Conversation.MaterializeAgentCanonicalInput(
		ctx,
		request.Input.Text,
		request.Input.Attachments,
		request.Checkpoint,
	)
	if err != nil {
		return agentcanonical.CommitReceipt{}, err
	}
	return agentcanonical.CommitReceipt{Revision: receipt.Revision}, nil
}

func (committer *canonicalConversationCommitter) ApplyPreparedContext(
	ctx context.Context,
	prepared agentchat.AgentContextPreparation,
) error {
	return committer.config.Conversation.CommitModelInput(ctx, prepared.OriginalMessage, prepared.ModelContext)
}

func (committer *canonicalConversationCommitter) CommitContext(
	ctx context.Context,
	request agentcanonical.ContextCommitRequest,
) (agentcanonical.CommitReceipt, error) {
	revision, err := committer.config.Conversation.CommitAgentCanonicalContext(ctx, request)
	if err != nil {
		return agentcanonical.CommitReceipt{}, err
	}
	return agentcanonical.CommitReceipt{Revision: revision}, nil
}

func (committer *canonicalConversationCommitter) CommitOutput(
	ctx context.Context,
	prepared agentchat.AgentContextPreparation,
	request agentcanonical.OutputCommitRequest,
) (agentcanonical.OutputCommitReceipt, error) {
	options := committer.config.Options
	metadata := session.MessageMetadata{
		RunID: request.Identity.RunID, AgentKind: options.AgentKind,
		AgentName: options.RootAgentName, RootAgentName: options.RootAgentName,
	}
	if options.RootAgentName != "" {
		metadata.RunPath = []string{options.RootAgentName}
	}
	receipt, err := committer.config.Conversation.CommitAgentCanonicalOutput(
		ctx, request.Message.Clone(), metadata, request.Checkpoint,
	)
	if err != nil {
		return agentcanonical.OutputCommitReceipt{}, err
	}
	if prepared.ResumeInterruption != nil {
		if err := committer.config.Conversation.ResolveInterruption(prepared.ResumeInterruption.ID); err != nil {
			return agentcanonical.OutputCommitReceipt{}, err
		}
	}
	canonical, err := settledTurnContextWindow(request.ContextMessages, request.ActiveUserIndex, receipt.Turn.Narrative, request.Message.Extra)
	if err != nil {
		return agentcanonical.OutputCommitReceipt{}, err
	}
	return agentcanonical.OutputCommitReceipt{
		Revision: receipt.Revision,
		Transcript: &agentcanonical.OutputProjection{
			Content: receipt.Turn.Narrative, Thinking: receipt.Turn.Thinking, ContextMessages: canonical,
		},
	}, nil
}

var _ agentlifecycle.ConversationCommitter = (*canonicalConversationCommitter)(nil)
var _ agentlifecycle.ConversationContextCommitter = (*canonicalConversationCommitter)(nil)
