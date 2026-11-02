package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"denova/config"
	"denova/internal/book"
)

func TestBookDescriptionLimitRejectsOversizeSynopsisWithoutSideEffects(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	application, err := New(ctx, &config.Config{OpenAIModel: "test-model", DenovaDir: root})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	atLimit := strings.Repeat("简", book.MaxDescriptionRunes)
	oversize := atLimit + "介"

	if _, err := application.CreateBook(ctx, root, "Oversize Book", "", oversize, ""); !errors.Is(err, book.ErrDescriptionTooLong) {
		t.Fatalf("create oversize error = %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "*", "Oversize Book"))
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "Oversize Book")); len(matches) != 0 || statErr == nil {
		t.Fatalf("rejected Book left a workspace behind: %v", matches)
	}

	created, err := application.CreateBook(ctx, root, "Limit Book", "", atLimit, "")
	if err != nil {
		t.Fatalf("create at limit: %v", err)
	}
	if _, err := application.BookAssets().UpdateInfo(created.Workspace, "Limit Book", "", oversize); !errors.Is(err, book.ErrDescriptionTooLong) {
		t.Fatalf("update oversize error = %v", err)
	}
	meta, err := application.BookAssets().Info(created.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Description != atLimit {
		t.Fatalf("rejected update changed the stored synopsis: %d runes", len([]rune(meta.Description)))
	}
}
