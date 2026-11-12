package book

import (
	"reflect"
	"testing"
)

func TestSummaryIncludesGroupAndChapterPlansInPlanningOrder(t *testing.T) {
	service := NewService(t.TempDir())
	for _, path := range []string{
		"setting/chapter-plans/ch00010-crossing.md",
		"setting/chapter-groups/group02-journey.md",
		"setting/chapter-plans/ch00002-choice.md",
		"setting/chapter-groups/group01-opening.md",
	} {
		if err := service.Create(path, "file", "# "+path+"\n\nThe plan."); err != nil {
			t.Fatal(err)
		}
	}
	summary, err := service.Summary()
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(summary.ChapterPlans))
	for _, plan := range summary.ChapterPlans {
		paths = append(paths, plan.Path)
	}
	want := []string{"setting/chapter-groups/group01-opening.md", "setting/chapter-groups/group02-journey.md", "setting/chapter-plans/ch00002-choice.md", "setting/chapter-plans/ch00010-crossing.md"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("planning paths = %v, want %v", paths, want)
	}
}
