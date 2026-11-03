package agents

import (
	agentchat "github.com/vevila2020/writing_platform_ai/internal/agents/chat"
	agentlifecycle "github.com/vevila2020/writing_platform_ai/internal/agents/lifecycle"
	agentrun "github.com/vevila2020/writing_platform_ai/internal/agents/run"

	"github.com/vevila2020/writing_platform_ai/agent"
)

// Product lifecycle vocabulary lives in lifecycle so the execution host can
// depend on it without introducing a package cycle through Agent builders.
type TurnHostData = agentlifecycle.TurnHostData
type TurnKind = agentlifecycle.TurnKind
type DefinitionRequest = agentlifecycle.DefinitionRequest
type DefinitionResolver = agentlifecycle.DefinitionResolver
type DefinitionResolverFunc = agentlifecycle.DefinitionResolverFunc

const (
	TurnStart    = agentlifecycle.TurnStart
	TurnSteer    = agentlifecycle.TurnSteer
	TurnFollowUp = agentlifecycle.TurnFollowUp
	TurnNext     = agentlifecycle.TurnNext
)

func TurnInput(kind TurnKind, request agentchat.ChatRequest, options agentrun.Options) (agent.Input, error) {
	return agentlifecycle.TurnInput(kind, request, options)
}

func DecodeTurnHostData(input agent.Input) (TurnHostData, error) {
	return agentlifecycle.DecodeTurnHostData(input)
}

func DecodeTurnHostDataFromPrepare(request agent.PrepareRequest) (TurnHostData, error) {
	return agentlifecycle.DecodeTurnHostDataFromPrepare(request)
}

func NewSource(resolver DefinitionResolver) (agent.Source, error) {
	return agentlifecycle.NewSource(resolver)
}
