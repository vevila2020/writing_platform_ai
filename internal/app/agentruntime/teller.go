package agentruntime

import (
	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/agents/prompts"
	interactiveapp "github.com/vevila2020/writing_platform_ai/internal/app/interactive"
	"github.com/vevila2020/writing_platform_ai/internal/interactive/teller"
	"github.com/vevila2020/writing_platform_ai/internal/style"
)

func WritingTellerForConfig(cfg *config.Config) prompts.IDEStoryTeller {
	if cfg == nil || cfg.DataDir() == "" {
		return prompts.IDEStoryTeller{}
	}
	tellerID := cfg.IDEStoryTellerID
	if tellerID == "" {
		tellerID = style.DefaultID
	}
	return WritingTeller(interactiveapp.LoadWritingTeller(cfg.DataDir(), tellerID))
}

func WritingTeller(definition teller.Definition) prompts.IDEStoryTeller {
	if definition.ID == "" {
		return prompts.IDEStoryTeller{}
	}
	return prompts.IDEStoryTeller{
		ID: definition.ID, Name: definition.Name, Description: definition.Description,
		Prompt: definition.PromptForTargets("system", "turn_context"),
	}
}
