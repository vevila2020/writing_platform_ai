package chat

import (
	"context"
	"strings"
	"testing"

	agentcontext "github.com/vevila2020/writing_platform_ai/internal/agents/context"
	novaskills "github.com/vevila2020/writing_platform_ai/internal/agents/skills"
)

type keywordSkillConversation struct {
	explicitSkillParityConversation
	turn novaskills.TurnInvocations
}

func (conversation keywordSkillConversation) ResolveTurnSkills(context.Context, string) (novaskills.TurnInvocations, error) {
	return conversation.turn, nil
}

func keywordGenreCard(body string) novaskills.Invocation {
	return novaskills.Invocation{
		Name: "genre-urban", Instructions: "# Skill: genre-urban\n\n" + body,
		Source: novaskills.InvocationSourceKeyword, Keyword: "都市异能",
	}
}

func TestKeywordGenreCardIsInjectedFromTheKeywordFragmentSource(t *testing.T) {
	conversation := keywordSkillConversation{
		explicitSkillParityConversation: explicitSkillParityConversation{budget: agentcontext.DefaultBudget()},
		turn: novaskills.TurnInvocations{Keyword: []novaskills.Invocation{keywordGenreCard("URBAN_GENRE_BODY")}},
	}
	prepared, err := prepareTurnContext(context.Background(), turnContextPreparationInput{
		Conversation: conversation, Request: ChatRequest{Message: "写一个都市异能开局"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.KeywordSkills) != 1 || len(prepared.ExplicitSkills) != 0 {
		t.Fatalf("explicit=%#v keyword=%#v", prepared.ExplicitSkills, prepared.KeywordSkills)
	}
	final := lastUserMessageContent(prepared.ModelContext.Messages)
	if !strings.Contains(final, "URBAN_GENRE_BODY") || !strings.Contains(final, "automatic genre match") {
		t.Fatalf("final user context=%s", final)
	}
	found := false
	for _, fragment := range prepared.ModelContext.Context.Fragments {
		if fragment.Source == keywordSkillFragmentSource {
			found = true
		}
	}
	if !found {
		t.Fatalf("keyword fragment source missing: %#v", prepared.ModelContext.Context.Fragments)
	}
}

func TestKeywordGenreCardIsDroppedInsteadOfFailingTheTurn(t *testing.T) {
	budget := agentcontext.DefaultBudget()
	budget.MaxTotalBytes = 4096
	conversation := keywordSkillConversation{
		explicitSkillParityConversation: explicitSkillParityConversation{budget: budget},
		turn: novaskills.TurnInvocations{Keyword: []novaskills.Invocation{
			keywordGenreCard(strings.Repeat("URBAN_BODY ", 1200)),
		}},
	}
	prepared, err := prepareTurnContext(context.Background(), turnContextPreparationInput{
		Conversation: conversation, Request: ChatRequest{Message: "写一个都市异能开局"},
	})
	if err != nil {
		t.Fatalf("keyword card must be best-effort: %v", err)
	}
	if len(prepared.KeywordSkills) != 0 {
		t.Fatalf("keyword Skills should be dropped, got %#v", prepared.KeywordSkills)
	}
	if final := lastUserMessageContent(prepared.ModelContext.Messages); strings.Contains(final, "URBAN_BODY") {
		t.Fatalf("dropped keyword card leaked into context: %s", final)
	}
	for _, fragment := range prepared.ModelContext.Context.Fragments {
		if fragment.Source == keywordSkillFragmentSource {
			t.Fatalf("dropped keyword fragment still projected: %#v", fragment)
		}
	}
}

func TestExplicitSkillStillFailsClosedUnderTheSameBudget(t *testing.T) {
	budget := agentcontext.DefaultBudget()
	budget.MaxTotalBytes = 4096
	conversation := keywordSkillConversation{
		explicitSkillParityConversation: explicitSkillParityConversation{budget: budget},
		turn: novaskills.TurnInvocations{Explicit: []novaskills.Invocation{{
			Name: "alpha", Instructions: "# Skill: alpha\n\n" + strings.Repeat("ALPHA_BODY ", 1200),
		}}},
	}
	_, err := prepareTurnContext(context.Background(), turnContextPreparationInput{
		Conversation: conversation, Request: ChatRequest{Message: "/alpha"},
	})
	if err == nil || !strings.Contains(err.Error(), "source="+explicitSkillFragmentSource) {
		t.Fatalf("explicit Skill budget error=%v", err)
	}
}
