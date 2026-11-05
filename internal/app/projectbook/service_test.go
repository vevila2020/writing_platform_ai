package projectbook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vevila2020/writing_platform_ai/internal/book"
	projectdomain "github.com/vevila2020/writing_platform_ai/internal/project"
)

func TestApplyStoredBookMetaPrefersProjectState(t *testing.T) {
	contentRoot, storeRoot := t.TempDir(), t.TempDir()
	writeTestFile(t, filepath.Join(contentRoot, "book.json"), `{"title":"旧书名","author":"旧作者"}`)
	writeTestFile(t, filepath.Join(storeRoot, "meta.json"), `{"title":"新书名","author":"新作者"}`)

	summary := book.WorkspaceSummary{Title: "旧书名", Author: "旧作者"}
	applyStoredBookMeta(projectdomain.Layout{ContentRoot: contentRoot, StoreRoot: storeRoot}, &summary)

	if summary.Title != "新书名" || summary.Author != "新作者" {
		t.Fatalf("summary = %#v, want the Project-state metadata", summary)
	}
}

func TestApplyStoredBookMetaFallsBackToBookJSON(t *testing.T) {
	contentRoot, storeRoot := t.TempDir(), t.TempDir()
	writeTestFile(t, filepath.Join(contentRoot, "book.json"), `{"title":"旧书名","author":"旧作者"}`)

	summary := book.WorkspaceSummary{Title: filepath.Base(contentRoot)}
	applyStoredBookMeta(projectdomain.Layout{ContentRoot: contentRoot, StoreRoot: storeRoot}, &summary)

	if summary.Title != "旧书名" || summary.Author != "旧作者" {
		t.Fatalf("summary = %#v, want the book.json fallback", summary)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
