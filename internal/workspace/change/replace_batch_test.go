package change

import (
	"context"
	"errors"
	"testing"
)

func TestReplaceFilesGuardsRecoveryAndUndo(t *testing.T) {
	for _, stage := range []string{operationStageAfterPrepare, operationStageAfterPathVisible, operationStageBeforeCommit} {
		t.Run(stage, func(t *testing.T) {
			service, chapter := newTestServiceWithFile(t, "Final prose.")
			writeTestFile(t, service.workspace, "setting/progress.md", "Old progress.")
			request := ReplaceFilesRequest{
				Files:    []FileReplacement{{Path: "setting/progress.md", Content: "New progress.", BaseRevision: Revision([]byte("Old progress."))}, {Path: "setting/tracking.md", Content: "New continuity.", BaseRevision: "missing"}},
				Guards:   []FileRevision{{Path: chapter, Revision: Revision([]byte("Final prose."))}},
				Metadata: ChangeMetadata{Origin: OriginAgent, ChangeGroupID: "chapter-commit"},
			}
			service.groupOperationHook = func(current, _ string) error {
				if current == stage {
					return errSimulatedOperationCrash
				}
				return nil
			}
			if _, err := service.ReplaceFiles(context.Background(), request); !errors.Is(err, errSimulatedOperationCrash) {
				t.Fatalf("injected crash = %v", err)
			}
			reloaded, err := NewService(service.workspace)
			if err != nil {
				t.Fatal(err)
			}
			if readTestFile(t, service.workspace, "setting/progress.md") != "New progress." || readTestFile(t, service.workspace, "setting/tracking.md") != "New continuity." {
				t.Fatal("batch was not recovered")
			}
			if _, err := reloaded.Undo(context.Background(), HistoryRequest{GroupID: "chapter-commit"}); err != nil {
				t.Fatal(err)
			}
			if readTestFile(t, service.workspace, "setting/progress.md") != "Old progress." {
				t.Fatal("undo lost the original content")
			}
		})
	}
}

func TestReplaceFilesRejectsStaleInputsWithoutPartialWrites(t *testing.T) {
	service, chapter := newTestServiceWithFile(t, "Changed prose.")
	_, err := service.ReplaceFiles(context.Background(), ReplaceFilesRequest{
		Files:  []FileReplacement{{Path: "setting/progress.md", Content: "Should not be written.", BaseRevision: "missing"}},
		Guards: []FileRevision{{Path: chapter, Revision: Revision([]byte("Old prose."))}},
	})
	assertChangeErrorCode(t, err, ErrorCodeRevisionConflict)
	if _, _, err := service.ReadFile("setting/progress.md"); err == nil {
		t.Fatal("stale batch wrote a file")
	}
}

func TestReplaceFilesRecoveryPreservesChangedGuard(t *testing.T) {
	service, chapter := newTestServiceWithFile(t, "Final prose.")
	service.groupOperationHook = func(stage, _ string) error {
		if stage == operationStageAfterPrepare {
			return errSimulatedOperationCrash
		}
		return nil
	}
	_, err := service.ReplaceFiles(context.Background(), ReplaceFilesRequest{Files: []FileReplacement{{Path: "setting/tracking.md", Content: "Prepared state.", BaseRevision: "missing"}}, Guards: []FileRevision{{Path: chapter, Revision: Revision([]byte("Final prose."))}}})
	if !errors.Is(err, errSimulatedOperationCrash) {
		t.Fatal(err)
	}
	writeTestFile(t, service.workspace, chapter, "An author's intervening correction.")
	if _, err := NewService(service.workspace); err != nil {
		t.Fatal(err)
	}
	if readTestFile(t, service.workspace, chapter) != "An author's intervening correction." {
		t.Fatal("recovery overwrote author prose")
	}
	if _, _, err := service.ReadFile("setting/tracking.md"); err == nil {
		t.Fatal("recovery committed stale continuity")
	}
}
