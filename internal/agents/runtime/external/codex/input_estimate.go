package codex

import (
	"github.com/vevila2020/writing_platform_ai/internal/agents/runtime/external"

	agentmodel "github.com/vevila2020/writing_platform_ai/agent/model"
	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
)

// InputEstimator uses the same resolved model identity as Run. Unrecognized CLI
// aliases use the shared image fallback rather than compressed file size.
func (c *Client) InputEstimator(input external.Input) agentmodel.InputEstimator {
	model := ""
	if input.Selection.Codex != nil {
		model = input.Selection.Codex.Model
	}
	if input.Selection.ModelProfileID() != "" {
		model = c.apiModel
	}
	return (providers.ModelConfig{Model: model}).InputEstimator()
}
