// Package lifecycle adapts InkDock's application boundary to the public
// Agent -> Session -> Run lifecycle.
package lifecycle

import (
	"context"
	"errors"

	agentrun "github.com/vevila2020/writing_platform_ai/internal/agents/run"

	"github.com/vevila2020/writing_platform_ai/agent"
	agentevent "github.com/vevila2020/writing_platform_ai/agent/lifecycle/event"
	agentinteraction "github.com/vevila2020/writing_platform_ai/agent/lifecycle/interaction"
	agenttrace "github.com/vevila2020/writing_platform_ai/agent/lifecycle/trace"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
	agentsession "github.com/vevila2020/writing_platform_ai/agent/session"
	agentcanonical "github.com/vevila2020/writing_platform_ai/agent/session/canonical"
)

// Lifecycle-facing aliases keep application packages on InkDock's adapter
// boundary while preserving the public lifecycle API unchanged.
type Agent = agent.Agent
type Session = agent.Session
type Run = agent.Run
type Input = agent.Input
type Event = agentevent.Event
type Result = agent.Result
type Snapshot = agentevent.SessionSnapshot
type Observation = agentevent.Observation
type InteractionResponse = agentinteraction.InteractionResponse
type SessionKey = agentsession.Key
type SessionSelector = agentsession.Selector

type CanonicalAdapter = agentcanonical.CanonicalAdapter
type CanonicalAdapterFuncs = agentcanonical.CanonicalAdapterFuncs
type InputCommitRequest = agentcanonical.InputCommitRequest
type OutputCommitRequest = agentcanonical.OutputCommitRequest
type CommitReceipt = agentcanonical.CommitReceipt
type OutputCommitReceipt = agentcanonical.OutputCommitReceipt
type OutputProjection = agentcanonical.OutputProjection
type ContextCommitRequest = agentcanonical.ContextCommitRequest
type EffectRequest = agentcanonical.EffectRequest
type EffectResult = agentcanonical.EffectResult

// Config declares InkDock-owned Session storage and optional integrations.
type Config struct {
	Store             agentsession.Store
	Trace             agenttrace.TraceSink
	RunIDGenerator    agent.RunIDGenerator
	CacheKeyGenerator agentschema.CacheKeyGenerator
}

// DefaultRunIDGenerator is InkDock's application-owned execution identity
// policy. Agent treats the returned value as opaque.
func DefaultRunIDGenerator(agent.RunIDRequest) (string, error) {
	return agentrun.NewID("run"), nil
}

func New(ctx context.Context, source agent.Source, config Config) (*Agent, error) {
	if config.Store == nil {
		return nil, errors.New("InkDock Agent lifecycle Store is required")
	}
	if source == nil {
		return nil, errors.New("InkDock Agent lifecycle Source is required")
	}
	runIDs := config.RunIDGenerator
	if runIDs == nil {
		runIDs = DefaultRunIDGenerator
	}
	options := []agent.Option{
		agent.WithSessionStore(config.Store), agent.WithRunIDGenerator(runIDs),
	}
	if config.CacheKeyGenerator != nil {
		options = append(options, agent.WithCacheKeyGenerator(config.CacheKeyGenerator))
	}
	if config.Trace != nil {
		options = append(options, agent.WithTrace(config.Trace))
	}
	return agent.New(ctx, source, options...)
}
