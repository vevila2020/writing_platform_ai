package lifecycle

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/book"

	agentcontext "github.com/vevila2020/writing_platform_ai/agent/context"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func writeStoredBookMeta(t *testing.T, storeRoot string, meta book.BookMeta) {
	t.Helper()
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeRoot, "meta.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBookProfileContextSourceInjectsLiveStoredSynopsisForWriting(t *testing.T) {
	workspace, storeRoot := t.TempDir(), t.TempDir()
	cfg := &config.Config{ProjectID: "project-book", Workspace: workspace, ProjectStoreDir: storeRoot}
	source, err := NewBookProfileContextSource(cfg, config.AgentKindIDE, book.NewState(workspace))
	if err != nil {
		t.Fatal(err)
	}
	if source == nil {
		t.Fatal("writing Agent must receive the Book profile source")
	}
	fragments, err := source.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil || len(fragments) != 0 {
		t.Fatalf("missing metadata fragments = %#v err=%v", fragments, err)
	}

	writeStoredBookMeta(t, storeRoot, book.BookMeta{Title: "White Hair", Author: "Author A", Description: "Gu Yan wakes up as Lu Chen's childhood friend."})
	fragments, err = source.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fragments) != 1 {
		t.Fatalf("fragment count = %d", len(fragments))
	}
	fragment := fragments[0]
	if fragment.Revision == "" || fragment.Stability != agentschema.ContextStablePrefix ||
		fragment.Placement != agentschema.ContextLeadingMessage || fragment.Rendering != agentschema.ContextRenderVerbatim ||
		fragment.Role != agentschema.User || fragment.HardLimit <= 0 {
		t.Fatalf("unexpected Book profile fragment: %#v", fragment)
	}
	for _, want := range []string{"White Hair", "Author A", "Gu Yan wakes up as Lu Chen's childhood friend."} {
		if !strings.Contains(fragment.Content, want) {
			t.Fatalf("Book profile is missing %q: %q", want, fragment.Content)
		}
	}
	if strings.Contains(fragment.Content, storeRoot) || strings.Contains(fragment.Content, workspace) {
		t.Fatalf("Book profile leaked a host path: %q", fragment.Content)
	}

	writeStoredBookMeta(t, storeRoot, book.BookMeta{Title: "White Hair", Description: "Gu Yan returns."})
	next, err := source.Materialize(context.Background(), agentcontext.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 1 || next[0].Revision == fragment.Revision || !strings.Contains(next[0].Content, "Gu Yan returns.") {
		t.Fatalf("edited synopsis was not refreshed: %#v", next)
	}

	writeStoredBookMeta(t, storeRoot, book.BookMeta{Title: "White Hair", Description: "  "})
	if empty, err := source.Materialize(context.Background(), agentcontext.ContextRequest{}); err != nil || len(empty) != 0 {
		t.Fatalf("blank synopsis fragments = %#v err=%v", empty, err)
	}
}

func TestBookProfileContextSourceRejectsOversizeSynopsis(t *testing.T) {
	workspace, storeRoot := t.TempDir(), t.TempDir()
	limit := 512
	cfg := &config.Config{Workspace: workspace, ProjectStoreDir: storeRoot, AgentContexts: config.AgentContextSettings{
		IDE: config.AgentContextOverride{MaxFragmentBytes: &limit},
	}}
	writeStoredBookMeta(t, storeRoot, book.BookMeta{Title: "Long", Description: strings.Repeat("x", 1024)})
	source, err := NewBookProfileContextSource(cfg, config.AgentKindIDE, book.NewState(workspace))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Materialize(context.Background(), agentcontext.ContextRequest{}); err == nil || !strings.Contains(err.Error(), "Book profile") {
		t.Fatalf("oversize error = %v", err)
	}
}

func TestBookProfileContextSourceIsScopedToWritingProjects(t *testing.T) {
	workspace, storeRoot := t.TempDir(), t.TempDir()
	cfg := &config.Config{Workspace: workspace, ProjectStoreDir: storeRoot}
	for _, kind := range []string{config.AgentKindGeneral, config.AgentKindInteractiveStory, config.AgentKindImage} {
		source, err := NewBookProfileContextSource(cfg, kind, book.NewState(workspace))
		if err != nil || source != nil {
			t.Fatalf("kind %s source = %#v err=%v", kind, source, err)
		}
	}
	for name, input := range map[string]struct {
		cfg   *config.Config
		state *book.State
	}{
		"no project state": {cfg: cfg},
		"no project store": {cfg: &config.Config{Workspace: workspace}, state: book.NewState(workspace)},
	} {
		source, err := NewBookProfileContextSource(input.cfg, config.AgentKindIDE, input.state)
		if err != nil || source != nil {
			t.Fatalf("%s source = %#v err=%v", name, source, err)
		}
	}
}
