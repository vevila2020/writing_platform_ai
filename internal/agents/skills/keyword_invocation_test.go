package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/config"
)

func writeGenreCardSkill(t *testing.T, root, dirName, skillName string, agents string, keywords []string, body string) {
	t.Helper()
	dir := filepath.Join(root, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	var frontmatter strings.Builder
	frontmatter.WriteString("---\n")
	frontmatter.WriteString("name: " + skillName + "\n")
	frontmatter.WriteString("description: " + skillName + " genre card\n")
	frontmatter.WriteString("category: writing\n")
	frontmatter.WriteString("capabilities:\n  - genre-card\n")
	frontmatter.WriteString("agent: " + agents + "\n")
	frontmatter.WriteString("keywords:\n")
	for _, keyword := range keywords {
		frontmatter.WriteString("  - " + keyword + "\n")
	}
	frontmatter.WriteString("---\n\n")
	frontmatter.WriteString(body)
	if err := os.WriteFile(filepath.Join(dir, SkillFileName), []byte(frontmatter.String()), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func TestResolveKeywordInvocationPrefersTheLongestKeyword(t *testing.T) {
	root := t.TempDir()
	writeGenreCardSkill(t, root, "urban", "genre-urban", "general,ide", []string{"都市"}, "URBAN CARD")
	writeGenreCardSkill(t, root, "urban-power", "genre-urban-power", "general,ide", []string{"都市异能"}, "URBAN POWER CARD")
	backend := NewAgentBackend([]Directory{{Scope: ScopeUser, Path: root}}, "ide", nil)

	match, ok := backend.ResolveKeywordInvocation(context.Background(), "我想写一个都市异能开局")
	if !ok || match.Skill.Name != "genre-urban-power" || match.Keyword != "都市异能" {
		t.Fatalf("match=%#v ok=%v", match, ok)
	}
}

func TestResolveKeywordInvocationUsesASCIIWordBoundaries(t *testing.T) {
	root := t.TempDir()
	writeGenreCardSkill(t, root, "infinite", "genre-infinite", "general,ide", []string{"infinite flow"}, "INFINITE CARD")
	backend := NewAgentBackend([]Directory{{Scope: ScopeUser, Path: root}}, "ide", nil)

	if match, ok := backend.ResolveKeywordInvocation(context.Background(), "the loop repeats infinitely"); ok {
		t.Fatalf("unexpected match inside a longer word: %#v", match)
	}
	match, ok := backend.ResolveKeywordInvocation(context.Background(), "写一个 Infinite Flow 副本")
	if !ok || match.Skill.Name != "genre-infinite" {
		t.Fatalf("match=%#v ok=%v", match, ok)
	}
}

func TestResolveKeywordInvocationSkipsNonGenreCardsAndDisabledSkills(t *testing.T) {
	root := t.TempDir()
	writeGenreCardSkill(t, root, "urban", "genre-urban", "general,ide", []string{"都市"}, "URBAN CARD")
	writeSkillFileForAgents(t, root, "craft", "novel-craft", "Craft", "general,ide")
	backend := NewAgentBackend([]Directory{{Scope: ScopeUser, Path: root}}, "ide", nil)
	if _, ok := backend.ResolveKeywordInvocation(context.Background(), "novel-craft 都市"); !ok {
		t.Fatal("expected the genre card to remain matchable")
	}
	backend = NewAgentBackend([]Directory{{Scope: ScopeUser, Path: root}}, "ide", map[string]bool{"genre-urban": false})
	if match, ok := backend.ResolveKeywordInvocation(context.Background(), "都市"); ok {
		t.Fatalf("disabled skill matched: %#v", match)
	}
}

func TestResolveKeywordInvocationRespectsAgentAndExplicitOnlyPolicy(t *testing.T) {
	root := t.TempDir()
	writeGenreCardSkill(t, root, "urban", "genre-urban", "general,ide", []string{"都市异能"}, "URBAN CARD")

	for _, agentKind := range []string{"interactive_story", "image"} {
		backend := NewAgentBackend([]Directory{{Scope: ScopeUser, Path: root}}, agentKind, nil)
		if match, ok := backend.ResolveKeywordInvocation(context.Background(), "都市异能"); ok {
			t.Fatalf("agent %s matched %#v", agentKind, match)
		}
	}
	backend := NewAgentBackendWithPolicy([]Directory{{Scope: ScopeUser, Path: root}}, "ide", map[string]bool{"genre-urban": true}, true)
	if match, ok := backend.ResolveKeywordInvocation(context.Background(), "都市异能"); ok {
		t.Fatalf("explicit-only catalog matched %#v", match)
	}
}

func TestResolveTurnInvocationsDoesNotDuplicateAnExplicitGenreCard(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "builtin-skills")
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeGenreCardSkill(t, skillsDir, "urban", "genre-urban", "general,ide", []string{"都市异能"}, "URBAN CARD")

	turn := mustResolveTurnSkills(t, skillsDir, workspace, "ide", "/genre-urban 写一个都市异能开局")
	if len(turn.Explicit) != 1 || turn.Explicit[0].Name != "genre-urban" {
		t.Fatalf("explicit=%#v", turn.Explicit)
	}
	if len(turn.Keyword) != 0 {
		t.Fatalf("keyword=%#v, want none", turn.Keyword)
	}
}

func TestResolveTurnInvocationsLabelsKeywordCards(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, "builtin-skills")
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeGenreCardSkill(t, skillsDir, "urban", "genre-urban", "general,ide", []string{"都市异能"}, "URBAN CARD")

	turn := mustResolveTurnSkills(t, skillsDir, workspace, "ide", "写一个都市异能开局")
	if len(turn.Explicit) != 0 || len(turn.Keyword) != 1 {
		t.Fatalf("turn=%#v", turn)
	}
	invocation := turn.Keyword[0]
	if invocation.Name != "genre-urban" || invocation.Keyword != "都市异能" || !invocation.IsKeyword() {
		t.Fatalf("invocation=%#v", invocation)
	}
	content := invocation.ModelContent()
	if !strings.Contains(content, "automatic genre match") || !strings.Contains(content, "都市异能") || !strings.Contains(content, "URBAN CARD") {
		t.Fatalf("keyword ModelContent=%q", content)
	}
	if explicit := (Invocation{Name: "alpha", Instructions: "# Skill: alpha"}).ModelContent(); strings.Contains(explicit, "automatic genre match") {
		t.Fatalf("zero source must stay explicit: %q", explicit)
	}
}

func mustResolveTurnSkills(t *testing.T, skillsDir, workspace, agentKind, message string) TurnInvocations {
	t.Helper()
	cfg := &config.Config{
		SkillsDir:  skillsDir,
		InkDockDir: filepath.Join(filepath.Dir(workspace), "data"),
		Workspace:  workspace,
	}
	turn, err := ResolveTurnInvocations(context.Background(), cfg, agentKind, message)
	if err != nil {
		t.Fatalf("ResolveTurnInvocations() error = %v", err)
	}
	return turn
}
