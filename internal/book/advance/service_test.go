package advance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	workspacechange "github.com/vevila2020/writing_platform_ai/internal/workspace/change"
)

func fixture(t *testing.T, workspace, path, content string) {
	t.Helper()
	abs := filepath.Join(workspace, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func update() Update {
	return Update{Summary: "The traveller receives a key and chooses the dark stairway.", Context: Context{Language: "en-US", Position: "Night, at the stairway.", ActiveCharacters: []string{"Lin"}, ActiveThreads: []string{"F001"}, ActiveSecrets: []string{"S001"}, NextCommitments: []string{"Open the stairway door."}}, Characters: []Character{{Name: "Lin", Location: "Stairway", Condition: "Tired but unhurt", Goal: "Find the sender", Possessions: []string{"Key"}, Knowledge: []string{"The guard recognized the letter."}}}, Threads: []Thread{{ID: "F001", Label: "The sealed letter", Status: "active", Detail: "The sender is still unknown."}}, Secrets: []Secret{{ID: "S001", AuthorTruth: "The guard sent the letter.", ReaderKnowledge: "The guard recognized the seal."}}}
}
func preparedRequest(t *testing.T, s *Service, path, plan string) CommitRequest {
	t.Helper()
	p, err := s.Prepare(context.Background(), path, plan)
	if err != nil {
		t.Fatal(err)
	}
	return CommitRequest{Action: "commit", Path: path, PlanPath: plan, BaseRevision: p.BaseRevision, Update: update(), Metadata: workspacechange.ChangeMetadata{Origin: workspacechange.OriginAgent, ChangeGroupID: "advance-" + strings.ReplaceAll(path, "/", "-")}}
}

func TestChapterCommitRetryRevisionAndUndo(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	changes, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	validated := 0
	s := NewService(changes, func(context.Context, string, string) error { validated++; return nil })
	path, plan := "chapters/ch00001-night.md", "setting/chapter-plans/ch00001-night.md"
	fixture(t, root, path, "Final prose.")
	fixture(t, root, plan, "Final plan.")
	req := preparedRequest(t, s, path, plan)
	receipt, err := s.Commit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.StateRevision != 1 || len(receipt.ChangedPaths) != 4 || receipt.Mutation == nil || validated != 1 {
		t.Fatalf("incomplete commit: %#v", receipt)
	}
	for _, name := range viewPaths {
		content, _, err := changes.ReadFile(name)
		if err != nil || !strings.Contains(content, "Lin") && name == viewPaths[1] {
			t.Fatalf("missing generated view %s: %v", name, err)
		}
	}
	retry, err := s.Commit(ctx, req)
	if err != nil || retry.Status != "unchanged" || retry.StateRevision != 1 {
		t.Fatalf("retry: %#v %v", retry, err)
	}
	fixture(t, root, plan, "A revised chapter plan.")
	if _, err := s.Commit(ctx, req); err == nil {
		t.Fatal("idempotent retry ignored a changed plan")
	}
	fixture(t, root, plan, "Final plan.")
	fixture(t, root, path, "Revised final prose.")
	stale := req
	stale.Update.Summary = "A corrected outcome."
	if _, err := s.Commit(ctx, stale); err == nil {
		t.Fatal("stale chapter revision was accepted")
	}
	revision := preparedRequest(t, s, path, plan)
	revision.Update.Summary = "Lin keeps the letter and climbs the stairs."
	revision.Metadata.ChangeGroupID = "revision"
	if _, err := s.Commit(ctx, revision); err != nil {
		t.Fatal(err)
	}
	if _, err := changes.Undo(ctx, workspacechange.HistoryRequest{GroupID: "revision"}); err != nil {
		t.Fatal(err)
	}
	p, err := s.Prepare(ctx, path, plan)
	if err != nil {
		t.Fatal(err)
	}
	if p.Committed {
		t.Fatal("undo must reveal that prose and continuity no longer agree")
	}
}

func TestCommitRejectsProseChangedDuringValidation(t *testing.T) {
	root := t.TempDir()
	path, plan := "chapters/ch00001-night.md", "setting/chapter-plans/ch00001-night.md"
	fixture(t, root, path, "Final prose.")
	fixture(t, root, plan, "Final plan.")
	changes, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(changes, func(context.Context, string, string) error {
		fixture(t, root, path, "A concurrent author edit.")
		return nil
	})
	if _, err := s.Commit(context.Background(), preparedRequest(t, s, path, plan)); err == nil {
		t.Fatal("concurrent edit was accepted")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(StatePath))); !os.IsNotExist(err) {
		t.Fatal("failed commit wrote continuity state")
	}
}

func TestExistingBookInitializationPreservesProgressAndBlocksUncommittedPredecessor(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	changes, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(changes, func(context.Context, string, string) error { return nil })
	path, plan := "chapters/ch00001-night.md", "setting/chapter-plans/ch00002-door.md"
	fixture(t, root, path, "Existing prose.")
	fixture(t, root, viewPaths[0], "Original historical progress.")
	fixture(t, root, plan, "Next plan.")
	p, err := s.Prepare(ctx, "chapters/ch00002-door.md", plan)
	if err != nil || !p.NeedsInitialization {
		t.Fatalf("initialization: %#v %v", p, err)
	}
	req := preparedRequest(t, s, path, "")
	req.Action = "initialize"
	if _, err := s.Commit(ctx, req); err != nil {
		t.Fatal(err)
	}
	progress, _, _ := changes.ReadFile(viewPaths[0])
	if !strings.Contains(progress, "Original historical progress.") {
		t.Fatal("initialization lost user progress")
	}
	fixture(t, root, path, "A changed predecessor.")
	if err := s.CanDraft(ctx, "chapters/ch00002-door.md", plan); err == nil {
		t.Fatal("stale predecessor was ignored")
	}
}

func TestContinuityViewsInformationBoundariesAndRetirement(t *testing.T) {
	root := t.TempDir()
	changes, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(changes, func(context.Context, string, string) error { return nil })
	path, plan := "chapters/ch00001-night.md", "setting/chapter-plans/ch00001-night.md"
	fixture(t, root, path, "Final prose.")
	fixture(t, root, plan, "Plan.")
	req := preparedRequest(t, s, path, plan)
	req.Update.Secrets[0].ReaderKnowledge = ""
	if _, err := s.Commit(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	tracking, _, _ := changes.ReadFile(viewPaths[2])
	reader := strings.Split(tracking, "## Reader knowledge")[1]
	reader = strings.Split(reader, "## Next-chapter commitments")[0]
	if strings.Contains(reader, "guard sent") {
		t.Fatal("hidden author truth leaked into reader knowledge")
	}
	req = preparedRequest(t, s, path, plan)
	req.Update.Context.NextCommitments = nil
	if _, err := s.Commit(context.Background(), req); err == nil {
		t.Fatal("unfulfilled commitment was silently removed")
	}
	fixture(t, root, viewPaths[2], "An author correction.")
	p, err := s.Prepare(context.Background(), path, plan)
	if err != nil || len(p.ViewConflicts) != 1 {
		t.Fatalf("manual correction not detected: %#v %v", p, err)
	}
	data, _, _ := changes.ReadFile(StatePath)
	var state State
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		t.Fatal(err)
	}
	if state.Secrets["S001"].ReaderKnowledge != "" {
		t.Fatal("unrevealed secret changed")
	}
}

func TestEarlierRevisionRecordsLaterConflictsAndRecovers(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	changes, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(changes, func(context.Context, string, string) error { return nil })
	first, second := "chapters/ch00001-night.md", "chapters/ch00002-door.md"
	firstPlan, secondPlan := "setting/chapter-plans/ch00001-night.md", "setting/chapter-plans/ch00002-door.md"
	for _, name := range []string{first, firstPlan} {
		fixture(t, root, name, "First final text.")
	}
	if _, err := s.Commit(ctx, preparedRequest(t, s, first, firstPlan)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{second, secondPlan} {
		fixture(t, root, name, "Second final text.")
	}
	if _, err := s.Commit(ctx, preparedRequest(t, s, second, secondPlan)); err != nil {
		t.Fatal(err)
	}
	fixture(t, root, first, "An earlier choice is changed.")
	revision := preparedRequest(t, s, first, firstPlan)
	revision.Update.Unresolved = []Issue{{Path: second, Reason: "The later scene still relies on the earlier choice."}}
	if _, err := s.Commit(ctx, revision); err != nil {
		t.Fatal(err)
	}
	if err := s.CanDraft(ctx, "chapters/ch00003-next.md", ""); err == nil {
		t.Fatal("unresolved later chapter allowed continuation")
	}
	fixture(t, root, second, "The later scene is reconciled.")
	repair := preparedRequest(t, s, second, secondPlan)
	if _, err := s.Commit(ctx, repair); err != nil {
		t.Fatal(err)
	}
	reopened, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	restored := NewService(reopened, nil)
	if err := restored.CanDraft(ctx, "chapters/ch00003-next.md", ""); err != nil {
		t.Fatalf("reconciled book did not resume after reload: %v", err)
	}
}

func TestExistingLaterProseRequiresInitialization(t *testing.T) {
	root := t.TempDir()
	changes, err := workspacechange.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	fixture(t, root, "chapters/ch00002-door.md", "Existing later prose.")
	s := NewService(changes, nil)
	if err := s.CanDraft(context.Background(), "chapters/ch00001-night.md", ""); err == nil {
		t.Fatal("existing later manuscript was silently excluded from initialization")
	}
}

func BenchmarkPrepareLongManuscript(b *testing.B) {
	root := b.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "chapters"), 0755); err != nil {
		b.Fatal(err)
	}
	for i := 1; i <= 1000; i++ {
		name := filepath.Join(root, "chapters", fmt.Sprintf("ch%05d-chapter.md", i))
		if err := os.WriteFile(name, []byte(strings.Repeat("A scene with a choice and consequences.\n", 50)), 0644); err != nil {
			b.Fatal(err)
		}
	}
	changes, err := workspacechange.NewService(root)
	if err != nil {
		b.Fatal(err)
	}
	s := NewService(changes, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Prepare(context.Background(), "chapters/ch01001-next.md", ""); err != nil {
			b.Fatal(err)
		}
	}
}
