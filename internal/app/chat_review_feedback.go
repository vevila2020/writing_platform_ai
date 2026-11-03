package app

import (
	"context"

	agentchat "github.com/vevila2020/writing_platform_ai/internal/agents/chat"
	agentrun "github.com/vevila2020/writing_platform_ai/internal/agents/run"
	reviewapp "github.com/vevila2020/writing_platform_ai/internal/app/review"
)

func (service *ChatAppService) resolveReviewFeedback(ctx context.Context, runtime ideChatRuntime, request *agentchat.ChatRequest) error {
	return reviewapp.Resolve(ctx, reviewRuntime(runtime), request)
}

func (service *ChatAppService) consumeResolvedReviewFeedback(ctx context.Context, runtime ideChatRuntime, request agentchat.ChatRequest) error {
	return reviewapp.Consume(ctx, reviewRuntime(runtime), request)
}

func (service *ChatAppService) bindReviewFeedbackInputCommit(options agentrun.Options, runtime ideChatRuntime, request agentchat.ChatRequest) agentrun.Options {
	return reviewapp.BindInputCommit(options, reviewRuntime(runtime), request)
}

func reviewRuntime(runtime ideChatRuntime) reviewapp.Runtime {
	sessionID := ""
	if runtime.sess != nil {
		sessionID = runtime.sess.ID
	}
	return reviewapp.Runtime{
		Workspace: runtime.workspace, StateRoot: runtime.projectStore, SessionID: sessionID,
		BookService: runtime.bookService,
	}
}
