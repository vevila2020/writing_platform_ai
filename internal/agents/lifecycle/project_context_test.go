package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/book"

	agentcontext "github.com/vevila2020/writing_platform_ai/agent/context"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func TestProjectInstructionsContextSourceIsOrderedVerbatimAndLive(t *testing.T) {
	workspace := t.TempDir()
	state := book.NewState(workspace)
	source, err := NewProjectInstructionsContextSource(&config.Config{Workspace: workspace}, config.AgentKindIDE, state)
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := source.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 1 {
		t.Fatalf("empty project should produce only the cache anchor: %#v", fragments)
	}
	anchor := fragments[0]
	if anchor.Source != "writing.context.cache_anchor" || anchor.Resource != "writing-context-cache-anchor" ||
		anchor.Content != writingContextCacheAnchor || anchor.Stability != agentschema.ContextStablePrefix ||
		anchor.Placement != agentschema.ContextLeadingMessage || anchor.Rendering != agentschema.ContextRenderVerbatim || anchor.Role != agentschema.User {
		t.Fatalf("unexpected writing cache anchor: %#v", anchor)
	}

	if err := os.WriteFile(filepath.Join(workspace, book.AgentInstructionsFileName), []byte("Run focused verification after changes."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, book.CreatorFileName), []byte("Use close third-person narration."), 0o644); err != nil {
		t.Fatal(err)
	}
	fragments, err = source.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 3 || fragments[0] != anchor {
		t.Fatalf("fragments = %#v, want the same anchor before both project instructions", fragments)
	}
	wants := []struct {
		resource string
		source   string
		heading  string
		body     string
	}{
		{book.AgentInstructionsFileName, "workspace AGENTS.md", "# Project instructions", "Run focused verification after changes."},
		{book.CreatorFileName, "workspace CREATOR.md", "# Creative instructions", "Use close third-person narration."},
	}
	for index, want := range wants {
		fragment := fragments[index+1]
		if fragment.Source != want.source || fragment.Resource != want.resource || fragment.Revision == "" ||
			fragment.Placement != agentschema.ContextLeadingMessage || fragment.Rendering != agentschema.ContextRenderVerbatim || fragment.Role != agentschema.User {
			t.Fatalf("unexpected project instruction fragment %d: %#v", index, fragment)
		}
		if !strings.HasPrefix(fragment.Content, want.heading+"\n\n"+want.body) || strings.Contains(fragment.Content, "Source:") ||
			!strings.HasSuffix(fragment.Content, "A later explicit user request takes precedence.") {
			t.Fatalf("unexpected model-visible project instruction %d: %q", index, fragment.Content)
		}
	}

	if err := os.WriteFile(filepath.Join(workspace, book.CreatorFileName), []byte("Use first-person narration."), 0o644); err != nil {
		t.Fatal(err)
	}
	next, err := source.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if next[0] != anchor || next[1] != fragments[1] {
		t.Fatalf("unchanged anchor and AGENTS.md fragment should preserve exact cache bytes: before=%#v after=%#v", fragments[:2], next[:2])
	}
	if next[2].Revision == fragments[2].Revision || !strings.Contains(next[2].Content, "first-person") {
		t.Fatalf("live CREATOR.md instructions were not refreshed: %#v", next[2])
	}
	lowerLimit := 64 * 1024
	limited, err := NewProjectInstructionsContextSource(&config.Config{AgentContexts: config.AgentContextSettings{
		IDE: config.AgentContextOverride{MaxFragmentBytes: &lowerLimit},
	}}, config.AgentKindIDE, state)
	if err != nil {
		t.Fatal(err)
	}
	if limited.Identity() == source.Identity() {
		t.Fatal("project instruction identity must change with its configured hard limit")
	}
}

func TestWritingCacheAnchorPrecedesEmptyCreatorAndBookProfile(t *testing.T) {
	workspace, storeRoot := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, book.CreatorFileName), []byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{ProjectID: "empty-book", Workspace: workspace, ProjectStoreDir: storeRoot}
	project, err := NewProjectInstructionsContextSource(cfg, config.AgentKindIDE, book.NewState(workspace))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := NewBookProfileContextSource(cfg, config.AgentKindIDE, book.NewState(workspace))
	if err != nil {
		t.Fatal(err)
	}
	combined, err := agentcontext.CombineContextSources(project, profile)
	if err != nil {
		t.Fatal(err)
	}
	fragments, err := combined.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 1 || fragments[0].Source != "writing.context.cache_anchor" || fragments[0].Content != writingContextCacheAnchor {
		t.Fatalf("empty Book leading context = %#v, want the constant anchor as the first and only message", fragments)
	}
	first := fragments[0]
	writeStoredBookMeta(t, storeRoot, book.BookMeta{Title: "New book", Description: "A new premise."})
	if err := os.WriteFile(filepath.Join(workspace, book.CreatorFileName), []byte("Use close third person."), 0o644); err != nil {
		t.Fatal(err)
	}
	fragments, err = combined.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) < 2 || fragments[0] != first {
		t.Fatalf("turn-scoped/project changes rotated the first context message: %#v", fragments)
	}
}

func TestProjectInstructionsContextSourceRejectsEachOversizeFile(t *testing.T) {
	for _, resource := range []string{book.AgentInstructionsFileName, book.CreatorFileName} {
		t.Run(resource, func(t *testing.T) {
			workspace := t.TempDir()
			if err := os.WriteFile(filepath.Join(workspace, resource), []byte(strings.Repeat("x", 1024)), 0o644); err != nil {
				t.Fatal(err)
			}
			limit := 512
			cfg := &config.Config{Workspace: workspace, AgentContexts: config.AgentContextSettings{
				IDE: config.AgentContextOverride{MaxFragmentBytes: &limit},
			}}
			source, err := NewProjectInstructionsContextSource(cfg, config.AgentKindIDE, book.NewState(workspace))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := source.Materialize(context.Background(), agentcontext.ContextRequest{}); err == nil ||
				!strings.Contains(err.Error(), resource) || !strings.Contains(err.Error(), "project instruction limit") {
				t.Fatalf("oversize error = %v", err)
			}
		})
	}
}

func TestProjectInstructionsContextIdentityUsesProjectIDInsteadOfRuntimeRoot(t *testing.T) {
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	first, err := NewProjectInstructionsContextSource(
		&config.Config{ProjectID: "project-portable", Workspace: firstRoot},
		config.AgentKindIDE,
		book.NewState(firstRoot),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewProjectInstructionsContextSource(
		&config.Config{ProjectID: "project-portable", Workspace: secondRoot},
		config.AgentKindIDE,
		book.NewState(secondRoot),
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Identity() != second.Identity() {
		t.Fatalf("context identity changed after moving the same Project: first=%#v second=%#v", first.Identity(), second.Identity())
	}

	otherProject, err := NewProjectInstructionsContextSource(
		&config.Config{ProjectID: "project-other", Workspace: secondRoot},
		config.AgentKindIDE,
		book.NewState(secondRoot),
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Identity() == otherProject.Identity() {
		t.Fatal("context identity must distinguish different Projects")
	}
}
