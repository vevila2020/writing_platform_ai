package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
	agenttool "github.com/vevila2020/writing_platform_ai/agent/tool"
	agentbuiltin "github.com/vevila2020/writing_platform_ai/agent/tool/builtin"
	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/book/advance"
	workspacechange "github.com/vevila2020/writing_platform_ai/internal/workspace/change"
)

const advanceTestPlan = "- Core event: A traveller enters the sealed stairway.\n- Word range: 5-500\n- Target emotion: From doubt to a costly determination.\n- Protagonist goal / key choice: Enter alone to protect the guard.\n- Chapter-end hook: Footsteps approach from below.\n"
const advanceTestProse = "Lin pressed the cold brass key into the lock. The guard watched her hand without speaking. She opened the door, and a light moved at the bottom of the stairs.\n"

func runAdvanceTestTool(t *testing.T, ctx context.Context, definition agenttool.ToolDefinition, input any) (agentschema.ToolResult, error) {
	t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return definition.Tool.Run(ctx, string(data))
}

func TestAdvanceToolsDraftCheckCommitAndRetry(t *testing.T) {
	ctx := ContextWithWritingSkill(context.Background(), "novel-serial")
	cfg := &config.Config{Workspace: t.TempDir(), ProjectStoreDir: t.TempDir()}
	changes, err := workspacechange.ForWorkspaceAt(cfg.Workspace, cfg.ProjectStoreDir)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := newWorkspaceMutationAdapter(changes)
	if err != nil {
		t.Fatal(err)
	}
	path, plan := "chapters/ch00001-night.md", "setting/chapter-plans/ch00001-night.md"
	writeWritingCheckFixture(t, cfg.Workspace, plan, advanceTestPlan)
	if _, err := adapter.Write(ctx, agentbuiltin.WriteRequest{Path: path, Content: advanceTestProse}); err != nil {
		t.Fatal(err)
	}
	definitions, err := newAdvanceTools(cfg, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	prepare, commit := definitions[0], definitions[1]
	if prepare.Descriptor.MutationScope != agenttool.ToolMutationNone || commit.Descriptor.MutationScope != agenttool.ToolMutationWorkspace {
		t.Fatal("Advance tools have incorrect mutation scopes")
	}
	prepareChapter := func() advance.Preparation {
		t.Helper()
		result, err := runAdvanceTestTool(t, ctx, prepare, prepareAdvanceInput{Path: path})
		if err != nil {
			t.Fatal(err)
		}
		var preparation advance.Preparation
		if err := json.Unmarshal([]byte(result.ModelContent), &preparation); err != nil {
			t.Fatal(err)
		}
		if preparation.PlanPath != plan {
			t.Fatalf("plan discovery: %#v", preparation)
		}
		return preparation
	}
	input := commitAdvanceInput{Action: "commit", Path: path, Update: advance.Update{Summary: "Lin opens the stairway door.", Context: advance.Context{Language: "en-US", Position: "Night, at the stairway door."}}}
	input.Update.Context.Constraints = []string{}
	input.Update.Context.ActiveCharacters = []string{}
	input.Update.Context.ActiveThreads = []string{}
	input.Update.Context.ActiveSecrets = []string{}
	input.Update.Context.NextCommitments = []string{}
	input.Update.Context.Risks = []string{}
	input.Update.Characters = []advance.Character{}
	input.Update.Threads = []advance.Thread{}
	input.Update.Secrets = []advance.Secret{}
	input.Update.RetiredContext = []string{}
	input.Update.Unresolved = []advance.Issue{}
	writeWritingCheckFixture(t, cfg.Workspace, path, "TODO")
	input.BaseRevision = prepareChapter().BaseRevision
	if _, err := runAdvanceTestTool(t, ctx, commit, input); err == nil {
		t.Fatal("placeholder prose was committed")
	}
	if _, err := os.Stat(filepath.Join(cfg.Workspace, filepath.FromSlash(advance.StatePath))); !os.IsNotExist(err) {
		t.Fatal("rejected chapter wrote state")
	}
	writeWritingCheckFixture(t, cfg.Workspace, path, advanceTestProse)
	input.BaseRevision = prepareChapter().BaseRevision
	result, err := runAdvanceTestTool(t, ctx, commit, input)
	if err != nil {
		t.Fatal(err)
	}
	receipt, ok := workspacechange.ParseToolReceipt("commit_advance", result.ModelContent)
	if !ok || receipt.Path != advance.StatePath || len(receipt.AffectedPaths) != 4 || strings.Contains(result.ModelContent, cfg.Workspace) {
		t.Fatalf("invalid portable mutation receipt: %s", result.ModelContent)
	}
	checked, err := runWritingCheck(ctx, cfg.Workspace, checkWritingInput{Action: "tracking", Path: "setting/tracking.md"})
	if err != nil || checked.Blocking != 0 {
		t.Fatalf("generated English tracking failed checks: %#v %v", checked, err)
	}
	retry, err := runAdvanceTestTool(t, ctx, commit, input)
	if err != nil || !strings.Contains(retry.ModelContent, `"workspace_mutated":false`) {
		t.Fatalf("retry was not a no-op: %s %v", retry.ModelContent, err)
	}
	writeWritingCheckFixture(t, cfg.Workspace, path, advanceTestProse+"Lin stepped inside.\n")
	if _, err := runAdvanceTestTool(t, ctx, commit, input); err == nil {
		t.Fatal("stale final prose accepted")
	}
}

func TestAdvanceMutationGuardAndWorkflowScope(t *testing.T) {
	root := t.TempDir()
	changes, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := newWorkspaceMutationAdapter(changes)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ContextWithWritingSkill(context.Background(), "novel-serial")
	path, plan := "chapters/ch00001-night.md", "setting/chapter-plans/ch00001-night.md"
	if _, err := adapter.Write(ctx, agentbuiltin.WriteRequest{Path: path, Content: advanceTestProse}); err == nil {
		t.Fatal("missing plan allowed prose")
	}
	writeWritingCheckFixture(t, root, plan, "- Word target: 20\n- Target emotion: TBD\n")
	if _, err := adapter.Write(ctx, agentbuiltin.WriteRequest{Path: path, Content: advanceTestProse}); err == nil {
		t.Fatal("placeholder plan allowed prose")
	}
	writeWritingCheckFixture(t, root, plan, advanceTestPlan)
	if _, err := adapter.Write(ctx, agentbuiltin.WriteRequest{Path: path, Content: advanceTestProse}); err != nil {
		t.Fatal(err)
	}
	for _, managed := range []string{"setting/tracking.md", "setting/../setting/progress.md", "setting/character-states.md", "SETTING/CONTINUITY.JSON", filepath.Join(root, "setting", "tracking.md")} {
		if _, err := adapter.Write(ctx, agentbuiltin.WriteRequest{Path: managed, Content: "Bypass"}); err == nil {
			t.Fatalf("managed view write allowed: %s", managed)
		}
	}
	writeWritingCheckFixture(t, root, "setting/chapter-plans/ch00002-door.md", advanceTestPlan)
	if _, err := adapter.Write(ctx, agentbuiltin.WriteRequest{Path: "chapters/ch00002-door.md", Content: advanceTestProse}); err == nil {
		t.Fatal("uncommitted predecessor allowed next chapter")
	}
	other := ContextWithWritingSkill(context.Background(), "novel-lite")
	if _, err := adapter.Write(other, agentbuiltin.WriteRequest{Path: "chapters/ch00002-door.md", Content: advanceTestProse}); err != nil {
		t.Fatalf("Advance policy leaked into another workflow: %v", err)
	}
}

func TestAdvancePreparationBudgetRetainsUsableCheckpoint(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{Workspace: t.TempDir(), ProjectStoreDir: t.TempDir(), AgentToolResultLimitKB: 1}
	changes, err := workspacechange.ForWorkspaceAt(cfg.Workspace, cfg.ProjectStoreDir)
	if err != nil {
		t.Fatal(err)
	}
	path, plan := "chapters/ch00001-night.md", "setting/chapter-plans/ch00001-night.md"
	writeWritingCheckFixture(t, cfg.Workspace, path, advanceTestProse)
	writeWritingCheckFixture(t, cfg.Workspace, plan, advanceTestPlan)
	s := advance.NewService(changes, func(context.Context, string, string) error { return nil })
	p, err := s.Prepare(ctx, path, plan)
	if err != nil {
		t.Fatal(err)
	}
	update := advance.Update{Summary: "Lin opens the door.", Context: advance.Context{Language: "en-US", Position: strings.Repeat("An existing story position. ", 50)}}
	if _, err := s.Commit(ctx, advance.CommitRequest{Action: "commit", Path: path, PlanPath: plan, BaseRevision: p.BaseRevision, Update: update}); err != nil {
		t.Fatal(err)
	}
	definitions, err := newAdvanceTools(cfg, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runAdvanceTestTool(t, ctx, definitions[0], prepareAdvanceInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint struct {
		BaseRevision   string `json:"base_revision"`
		ContextOmitted bool   `json:"context_omitted"`
		ContextSource  string `json:"context_source"`
	}
	if err := json.Unmarshal([]byte(result.ModelContent), &checkpoint); err != nil {
		t.Fatal(err)
	}
	if !checkpoint.ContextOmitted || checkpoint.BaseRevision == "" || checkpoint.ContextSource != advance.StatePath || len(result.ModelContent) > 1024 {
		t.Fatalf("unusable bounded checkpoint: %s", result.ModelContent)
	}
	update.Context.Position = "Night, at the door."
	if _, err := s.Commit(ctx, advance.CommitRequest{Action: "commit", Path: path, PlanPath: plan, BaseRevision: checkpoint.BaseRevision, Update: update}); err != nil {
		t.Fatalf("checkpoint could not compact the working set: %v", err)
	}
}

func TestAdvanceReadOnlyToolAssembly(t *testing.T) {
	definitions, err := newAdvanceTools(&config.Config{Workspace: t.TempDir()}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 {
		t.Fatal("read-only capability exposed a commit tool")
	}
}
