package chat

import (
	"fmt"
	"strings"

	agentcontext "github.com/vevila2020/writing_platform_ai/internal/agents/context"
	novaskills "github.com/vevila2020/writing_platform_ai/internal/agents/skills"
)

const (
	explicitSkillFragmentSource = "turn.skill.explicit"
	keywordSkillFragmentSource  = "turn.skill.keyword"
)

func explicitSkillFragments(invocations []novaskills.Invocation) []agentcontext.Fragment {
	fragments := make([]agentcontext.Fragment, 0, len(invocations))
	for index, invocation := range invocations {
		name := strings.TrimSpace(invocation.Name)
		if name == "" || strings.TrimSpace(invocation.Instructions) == "" {
			continue
		}
		fragment := turnFragment(
			fmt.Sprintf("turn_explicit_skill_%d_%s", index+1, name),
			explicitSkillFragmentSource,
			"Explicit Skill: "+name,
			"apply a Skill explicitly selected by the user before the first model call",
			invocation.ModelContent(),
			0,
		)
		fragment.Note = "source=active Skill catalog; explicit user invocation; base_directory=" + strings.TrimSpace(invocation.BaseDirectory)
		fragments = append(fragments, fragment)
	}
	return fragments
}

// keywordSkillFragments renders automatically matched genre cards. They are
// best-effort context: the turn may drop them when the budget is tight.
func keywordSkillFragments(invocations []novaskills.Invocation) []agentcontext.Fragment {
	fragments := make([]agentcontext.Fragment, 0, len(invocations))
	for index, invocation := range invocations {
		name := strings.TrimSpace(invocation.Name)
		if name == "" || strings.TrimSpace(invocation.Instructions) == "" {
			continue
		}
		fragment := turnFragment(
			fmt.Sprintf("turn_keyword_skill_%d_%s", index+1, name),
			keywordSkillFragmentSource,
			"Genre Card: "+name,
			"apply a genre card matched from the user's message keywords",
			invocation.ModelContent(),
			0,
		)
		fragment.Note = fmt.Sprintf(
			"source=active Skill catalog; matched keyword=%q; auto-loaded genre card; base_directory=%s",
			strings.TrimSpace(invocation.Keyword), strings.TrimSpace(invocation.BaseDirectory),
		)
		fragments = append(fragments, fragment)
	}
	return fragments
}

func validateExplicitSkillProjection(result agentcontext.Result, invocations []novaskills.Invocation) error {
	if len(invocations) == 0 {
		return nil
	}
	byID := make(map[string]agentcontext.Fragment, len(result.Fragments))
	for _, fragment := range result.Fragments {
		byID[fragment.ID] = fragment
	}
	for index, invocation := range invocations {
		id := fmt.Sprintf("turn_explicit_skill_%d_%s", index+1, strings.TrimSpace(invocation.Name))
		fragment, ok := byID[id]
		expected := strings.TrimSpace(invocation.ModelContent())
		if !ok || !fragment.Included || fragment.Truncated || strings.TrimSpace(fragment.Content) != expected {
			return fmt.Errorf("explicit Skill %q did not fit the model context; raise the Agent context limits or request fewer Skills", invocation.Name)
		}
	}
	return nil
}

// fragmentsWithoutSource lets the turn retry assembly without best-effort
// keyword cards while preserving every higher-priority fragment.
func fragmentsWithoutSource(fragments []agentcontext.Fragment, source string) []agentcontext.Fragment {
	filtered := make([]agentcontext.Fragment, 0, len(fragments))
	for _, fragment := range fragments {
		if fragment.Source == source {
			continue
		}
		filtered = append(filtered, fragment)
	}
	return filtered
}
