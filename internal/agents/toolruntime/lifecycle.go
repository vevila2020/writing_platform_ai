package toolruntime

import (
	"context"

	agentrun "github.com/vevila2020/writing_platform_ai/internal/agents/run"
	agenttool "github.com/vevila2020/writing_platform_ai/internal/agents/tool"
)

// recordToolExecution feeds bounded product diagnostics. Canonical Tool
// mutations are committed separately through Agent effects.
func recordToolExecution(ctx context.Context, record agenttool.ExecutionRecord) {
	if observer := agentrun.ObserverFromContext(ctx); observer != nil {
		observer.RecordToolExecution(record)
	}
}
