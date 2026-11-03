package agentchat

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vevila2020/writing_platform_ai/config"
	agents "github.com/vevila2020/writing_platform_ai/internal/agents"
	agentexecution "github.com/vevila2020/writing_platform_ai/internal/agents/execution"
	agentruntime "github.com/vevila2020/writing_platform_ai/internal/agents/runtime"
	"github.com/vevila2020/writing_platform_ai/internal/agents/session"
	agenttool "github.com/vevila2020/writing_platform_ai/internal/agents/tool"
	"github.com/vevila2020/writing_platform_ai/internal/analytics"
	"github.com/vevila2020/writing_platform_ai/internal/book"
	projectdomain "github.com/vevila2020/writing_platform_ai/internal/project"
)

type activeViewTestHost struct{}

func (activeViewTestHost) BeginUsageRun(config.Config, string) analytics.Run { return analytics.Run{} }

func (activeViewTestHost) AgentEngines() *agentruntime.Engines { return nil }

func (activeViewTestHost) BaseRuntime() (config.Config, *agentexecution.Runtime) {
	return config.Config{}, nil
}

func (activeViewTestHost) ProjectVersionService(string) (*book.VersionService, error) {
	return nil, nil
}

func (activeViewTestHost) CurrentWorkspace() string { return "" }

func (activeViewTestHost) OnVerifiedMutations(context.Context, string, *book.VersionService, config.Config, []agenttool.Mutation, agenttool.Verification) {
}

func (activeViewTestHost) ProjectAgentHostCapabilities(context.Context, projectdomain.Type, *config.Config, string) (agents.AgentHostCapabilities, error) {
	return agents.AgentHostCapabilities{}, nil
}

func TestActiveViewProjectsPendingInterruptionForIdleAgentChat(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := projectdomain.NewRegistry(filepath.Join(root, "denova"))
	record, err := registry.Add(workspace, projectdomain.TypeGeneral, "")
	if err != nil {
		t.Fatal(err)
	}
	layout, err := registry.EnsureStore(record)
	if err != nil {
		t.Fatal(err)
	}
	store, err := session.NewStore(layout.SessionsDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	sess, err := store.GetOrCreate("session-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.MarkInterrupted("Draft a scene", "Partial scene", "user_requested"); err != nil {
		t.Fatal(err)
	}
	pending := sess.PendingInterruption()
	if pending == nil {
		t.Fatal("expected pending interruption fixture")
	}

	service := NewService(activeViewTestHost{}, registry)
	service.projects[record.ID] = &projectRuntime{
		projectID: record.ID, projectType: projectdomain.TypeGeneral,
		agentKind: "general", stateRoot: layout.StoreRoot, workspace: workspace,
		store: store,
	}

	view := service.ActiveView(context.Background(), Binding{ProjectID: record.ID, SessionID: sess.ID})
	if view.PendingInterruptionID != pending.ID {
		t.Fatalf("pending interruption id = %q, want %q", view.PendingInterruptionID, pending.ID)
	}
}

func (activeViewTestHost) PageInstructions(context.Context, string, string, string) (string, error) {
	return "Create content in the current page.", nil
}
