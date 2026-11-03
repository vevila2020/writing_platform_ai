package tool

import (
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func testToolDefinition(tool Tool) ToolDefinition {
	return ToolDefinition{Tool: tool, Descriptor: ToolDescriptor{
		Source: ToolSourceRead, Execution: ToolExecutionParallelRead,
		MutationScope: ToolMutationNone, PostCheck: ToolPostCheckNone,
		Recovery: ToolRecoveryReadOnly, ResultProjection: agentschema.ToolResultBoundedModelContext,
		ResultRetention: agentschema.ToolResultDeferred,
		Steering:        SteeringFinishCurrent, MaxResultBytes: 1 << 20,
	}}
}
