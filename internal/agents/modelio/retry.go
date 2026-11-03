package modelio

import (
	"github.com/vevila2020/writing_platform_ai/config"

	agentexecution "github.com/vevila2020/writing_platform_ai/agent/engine/execution"
	agentmodel "github.com/vevila2020/writing_platform_ai/agent/model"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

// ModelExecutionPolicy maps the persisted product retry setting to the shared
// per-response budget. Callers add their tool, iteration and idle policies.
func ModelExecutionPolicy(cfg *config.Config) agentexecution.ExecutionPolicy {
	retries := 5
	if cfg != nil && cfg.ModelMaxRetries >= 0 {
		retries = cfg.ModelMaxRetries
	}
	return agentexecution.ExecutionPolicy{
		Retry:            &agentmodel.RetryConfig{Decide: agentmodel.TransientRetry},
		RetryIdentity:    agentschema.CapabilityIdentity{Kind: "denova.retry.transient", Version: 1},
		ModelMaxAttempts: retries + 1,
	}
}
