package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/vevila2020/writing_platform_ai/config"
)

// InvocationSource records why a Skill entered the turn context. The zero value
// keeps the historical explicit semantics for existing callers.
type InvocationSource string

const (
	InvocationSourceExplicit InvocationSource = "explicit"
	InvocationSourceKeyword  InvocationSource = "keyword"
)

// Invocation is a Skill deterministically resolved from the current user
// message, either through an explicit /<skill-name> mention or through a
// genre-card keyword match.
type Invocation struct {
	Name          string
	Instructions  string
	BaseDirectory string
	Source        InvocationSource
	Keyword       string
}

// IsKeyword reports whether the runtime, not the user, selected this Skill.
func (invocation Invocation) IsKeyword() bool {
	return invocation.Source == InvocationSourceKeyword
}

// TurnInvocations separates user intent from automatic genre routing so the
// two can be projected and budgeted differently.
type TurnInvocations struct {
	Explicit []Invocation
	Keyword  []Invocation
}

// ExplicitResolver is the optional conversation capability used before the
// first model call. Implementations must return only Skills enabled for the
// active Agent.
type ExplicitResolver interface {
	ResolveExplicitSkills(context.Context, string) ([]Invocation, error)
}

// TurnSkillResolver is the optional conversation capability that returns both
// explicitly requested Skills and at most one keyword-matched genre card from
// a single catalog snapshot.
type TurnSkillResolver interface {
	ResolveTurnSkills(context.Context, string) (TurnInvocations, error)
}

// ResolveConfiguredInvocations resolves explicit Skills against one Agent's
// effective configuration. Unknown slash commands remain ordinary user text.
func ResolveConfiguredInvocations(ctx context.Context, cfg *config.Config, agentKind, message string) ([]Invocation, error) {
	turn, err := ResolveTurnInvocations(ctx, cfg, agentKind, message)
	if err != nil {
		return nil, err
	}
	return turn.Explicit, nil
}

// ResolveTurnInvocations resolves explicit Skills and one keyword-matched genre
// card from the same catalog snapshot. Explicit Skills keep their fail-closed
// size handling; a genre card that cannot fit is skipped by the caller.
func ResolveTurnInvocations(ctx context.Context, cfg *config.Config, agentKind, message string) (TurnInvocations, error) {
	if err := ctx.Err(); err != nil {
		return TurnInvocations{}, err
	}
	if cfg == nil || !config.ResolveAgentTools(cfg, agentKind).Allows(config.AgentToolSkills) {
		return TurnInvocations{}, nil
	}
	mode, overrides, _ := config.ResolveActiveAgentSkillPolicy(cfg, agentKind)
	backend := NewAgentBackendWithPolicy(
		NewDirectories(cfg.SkillsDir, cfg.DataDir(), cfg.Workspace),
		agentKind,
		overrides,
		mode == config.AgentSkillPolicyExplicit,
	)
	maxFragmentBytes := config.ResolveAgentContext(cfg, agentKind).MaxFragmentBytes
	records := backend.activeRecords(ctx)
	if err := ctx.Err(); err != nil {
		return TurnInvocations{}, err
	}

	explicit := make([]Invocation, 0)
	explicitNames := make(map[string]bool)
	for _, skill := range resolveExplicitFromRecords(records, message) {
		invocation, err := explicitInvocation(skill, maxFragmentBytes)
		if err != nil {
			return TurnInvocations{}, err
		}
		explicit = append(explicit, invocation)
		explicitNames[skill.Name] = true
	}

	keyword := []Invocation(nil)
	if !backend.explicitOnly && keywordAutoLoadAgent(agentKind) && strings.TrimSpace(message) != "" {
		if match, ok := matchKeywordSkill(records, message); ok && !explicitNames[match.Skill.Name] {
			if invocation, fits := keywordInvocation(match, maxFragmentBytes); fits {
				keyword = []Invocation{invocation}
			}
		}
	}
	return TurnInvocations{Explicit: explicit, Keyword: keyword}, nil
}

func explicitInvocation(skill Skill, maxFragmentBytes int) (Invocation, error) {
	notice := runtimeNotice(skill.Name)
	instructionsLimit := maxFragmentBytes - len(notice) - 2
	instructions := FormatForModel(skill, instructionsLimit)
	if instructionsLimit <= 0 || !strings.HasPrefix(instructions, "# Skill: "+skill.Name) {
		return Invocation{}, fmt.Errorf("explicit Skill %q cannot fit the configured context fragment limit", skill.Name)
	}
	return Invocation{
		Name: skill.Name, Instructions: instructions, BaseDirectory: skill.BaseDirectory,
		Source: InvocationSourceExplicit,
	}, nil
}

func keywordInvocation(match KeywordMatch, maxFragmentBytes int) (Invocation, bool) {
	notice := keywordRuntimeNotice(match.Skill.Name, match.Keyword)
	instructionsLimit := maxFragmentBytes - len(notice) - 2
	instructions := FormatForModel(match.Skill, instructionsLimit)
	if instructionsLimit <= 0 || !strings.HasPrefix(instructions, "# Skill: "+match.Skill.Name) {
		return Invocation{}, false
	}
	return Invocation{
		Name: match.Skill.Name, Instructions: instructions, BaseDirectory: match.Skill.BaseDirectory,
		Source: InvocationSourceKeyword, Keyword: match.Keyword,
	}, true
}

// ModelContent returns the complete model-visible invocation block.
func (invocation Invocation) ModelContent() string {
	name := strings.TrimSpace(invocation.Name)
	if invocation.IsKeyword() {
		return keywordRuntimeNotice(name, strings.TrimSpace(invocation.Keyword)) + "\n\n" + strings.TrimSpace(invocation.Instructions)
	}
	return runtimeNotice(name) + "\n\n" + strings.TrimSpace(invocation.Instructions)
}

func runtimeNotice(name string) string {
	return "[InkDock runtime] The user explicitly selected /" + name + ", so the runtime loaded this Skill before the first model request. " +
		"Do not call the `skill` tool only to load the same Skill again; follow the instructions below and continue."
}

func keywordRuntimeNotice(name, keyword string) string {
	return "[InkDock runtime] The current user message matched the genre keyword " + fmt.Sprintf("%q", keyword) + ", so the runtime preloaded this genre card before the first model request. " +
		"This is an automatic genre match, not an explicit user command. Treat the card as optional guidance, not as a replacement for the user's request, " +
		"and do not call the `skill` tool only to load the same Skill again."
}
