package tool

import (
	"context"

	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

// Toolset materializes the immutable tool registry for one model cycle.
type Toolset interface {
	Identity() agentschema.CapabilityIdentity
	PrepareTools(context.Context, ToolRequest) ([]ToolDefinition, error)
}

type ToolRequest struct {
	Session agentschema.SessionView
	Run     agentschema.RunView
	Input   agentschema.Input
}
