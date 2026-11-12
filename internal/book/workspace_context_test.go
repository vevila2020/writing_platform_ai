package book

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/internal/book/lore"
)

func TestWorkspaceContextIndexesFilesWithoutInjectingTheirBodies(t *testing.T) {
	dir := t.TempDir()
	state := NewState(dir)
	if err := state.InitWorkspace(); err != nil {
		t.Fatalf("initialize workspace: %v", err)
	}

	files := map[string]string{
		state.IdeasPath(): "IDEAS_BODY_SHOULD_BE_LAZY",
		filepath.Join(state.SettingDir(), "outline.md"):            "OUTLINE_BODY_SHOULD_BE_LAZY",
		filepath.Join(state.SettingDir(), "progress.md"):           "PROGRESS_BODY_SHOULD_BE_LAZY",
		filepath.Join(state.SettingDir(), CharacterStatesFileName): "CHARACTER_BODY_SHOULD_BE_LAZY",
		state.TrackingPath(): "TRACKING_BODY_SHOULD_BE_LAZY",
		filepath.Join(state.ChapterGroupDir(), "group01-wasteland.md"): "GROUP_BODY_SHOULD_BE_LAZY",
		filepath.Join(state.ChapterPlanDir(), "ch00001-opening.md"):    "CHAPTER_PLAN_BODY_SHOULD_BE_LAZY",
		filepath.Join(dir, "chapters", "ch00001-opening.md"):           "CHAPTER_BODY_SHOULD_BE_LAZY",
	}
	if err := os.MkdirAll(state.ChapterPlanDir(), 0o755); err != nil {
		t.Fatalf("create chapter plan directory: %v", err)
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	store := lore.NewStore(dir)
	if _, err := store.Create(lore.ItemInput{
		ID: "resident", Type: "character", Name: "Resident Hero",
		LoadMode: lore.LoadModeResident, Content: "RESIDENT_LORE_BODY_STAYS_INLINE",
	}); err != nil {
		t.Fatalf("create resident Lore: %v", err)
	}
	if _, err := store.Create(lore.ItemInput{
		ID: "manual", Type: "location", Name: "Hidden Archive",
		LoadMode: lore.LoadModeManual, Content: "ON_DEMAND_LORE_BODY_STAYS_LAZY",
	}); err != nil {
		t.Fatalf("create on-demand Lore: %v", err)
	}

	context := state.WorkspaceContext()
	for _, required := range []string{
		"## Workspace Source Index",
		`"ideas.md"`,
		`"setting/outline.md"`,
		"## Resident",
		"RESIDENT_LORE_BODY_STAYS_INLINE",
		"Hidden Archive",
	} {
		if !strings.Contains(context.Stable, required) {
			t.Fatalf("stable workspace context missing %q:\n%s", required, context.Stable)
		}
	}
	for _, required := range []string{
		"## Current Writing Source Index",
		`"setting/progress.md"`,
		`"setting/character-states.md"`,
		`"setting/tracking.md"`,
		`"setting/chapter-groups/group01-wasteland.md"`,
		`"setting/chapter-plans/ch00001-opening.md"`,
		`"chapters/ch00001-opening.md"`,
	} {
		if !strings.Contains(context.Dynamic, required) {
			t.Fatalf("dynamic workspace context missing %q:\n%s", required, context.Dynamic)
		}
	}
	for _, forbidden := range []string{
		"IDEAS_BODY_SHOULD_BE_LAZY",
		"OUTLINE_BODY_SHOULD_BE_LAZY",
		"PROGRESS_BODY_SHOULD_BE_LAZY",
		"CHARACTER_BODY_SHOULD_BE_LAZY",
		"TRACKING_BODY_SHOULD_BE_LAZY",
		"GROUP_BODY_SHOULD_BE_LAZY",
		"CHAPTER_PLAN_BODY_SHOULD_BE_LAZY",
		"CHAPTER_BODY_SHOULD_BE_LAZY",
		"ON_DEMAND_LORE_BODY_STAYS_LAZY",
	} {
		if strings.Contains(context.Markdown(), forbidden) {
			t.Fatalf("workspace context injected lazy body %q:\n%s", forbidden, context.Markdown())
		}
	}
}

func TestWorkspaceContextOmitsIdeasTemplateAndIndexesEditedIdeasByPath(t *testing.T) {
	dir := t.TempDir()
	state := NewState(dir)
	if err := state.InitWorkspace(); err != nil {
		t.Fatalf("initialize workspace: %v", err)
	}
	if context := state.WorkspaceContext(); context.Markdown() != "" {
		t.Fatalf("an untouched workspace should use the new-work hint, got:\n%s", context.Markdown())
	}

	if err := os.WriteFile(state.IdeasPath(), []byte("A revenge story on a flooded planet."), 0o644); err != nil {
		t.Fatal(err)
	}
	context := state.WorkspaceContext()
	if !strings.Contains(context.Stable, `"ideas.md"`) {
		t.Fatalf("edited ideas path missing from stable index:\n%s", context.Stable)
	}
	if strings.Contains(context.Stable, "A revenge story on a flooded planet.") {
		t.Fatalf("ideas body must be loaded on demand:\n%s", context.Stable)
	}
}

func TestWorkspaceContextPathIndexesStayStableAcrossBodyChanges(t *testing.T) {
	dir := t.TempDir()
	state := NewState(dir)
	if err := state.InitWorkspace(); err != nil {
		t.Fatalf("initialize workspace: %v", err)
	}
	outline := filepath.Join(state.SettingDir(), "outline.md")
	progress := filepath.Join(state.SettingDir(), "progress.md")
	if err := os.WriteFile(outline, []byte("outline revision one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(progress, []byte("progress revision one"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := state.WorkspaceContext()

	if err := os.WriteFile(outline, []byte("a materially different outline revision"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(progress, []byte("a materially different progress revision"), 0o644); err != nil {
		t.Fatal(err)
	}
	after := state.WorkspaceContext()
	if before != after {
		t.Fatalf("body-only edits should not rotate path indexes:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestWorkspaceContextBoundsRecentChapterAndGroupPaths(t *testing.T) {
	dir := t.TempDir()
	state := NewState(dir)
	if err := state.InitWorkspace(); err != nil {
		t.Fatalf("initialize workspace: %v", err)
	}
	for index := 1; index <= workspaceContextChapterGroupLimit+1; index++ {
		name := fmt.Sprintf("group%02d.md", index)
		if err := os.WriteFile(filepath.Join(state.ChapterGroupDir(), name), []byte("plan"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for index := 1; index <= workspaceContextChapterLimit+1; index++ {
		name := fmt.Sprintf("ch%05d.md", index)
		if err := os.WriteFile(filepath.Join(dir, "chapters", name), []byte("prose"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dynamic := state.WorkspaceContext().Dynamic
	for _, omitted := range []string{"group01.md", "ch00001.md"} {
		if strings.Contains(dynamic, omitted) {
			t.Fatalf("old path %q should fall outside the bounded index:\n%s", omitted, dynamic)
		}
	}
	for _, included := range []string{"group02.md", "group03.md", "ch00002.md", "ch00013.md"} {
		if !strings.Contains(dynamic, included) {
			t.Fatalf("recent path %q missing from bounded index:\n%s", included, dynamic)
		}
	}
}

func TestWorkspaceContextBoundsRecentChapterPlanPaths(t *testing.T) {
	dir := t.TempDir()
	state := NewState(dir)
	if err := state.InitWorkspace(); err != nil {
		t.Fatalf("initialize workspace: %v", err)
	}
	if err := os.MkdirAll(state.ChapterPlanDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= workspaceContextChapterPlanLimit+1; index++ {
		name := fmt.Sprintf("ch%05d-plan.md", index)
		if err := os.WriteFile(filepath.Join(state.ChapterPlanDir(), name), []byte("plan"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	dynamic := state.WorkspaceContext().Dynamic
	if strings.Contains(dynamic, "ch00001-plan.md") {
		t.Fatalf("old chapter plan should fall outside the bounded index:\n%s", dynamic)
	}
	for _, included := range []string{"ch00002-plan.md", "ch00003-plan.md", "ch00004-plan.md"} {
		if !strings.Contains(dynamic, included) {
			t.Fatalf("recent chapter plan %q missing from bounded index:\n%s", included, dynamic)
		}
	}
}

func TestWorkspaceContextIndexesRecentResearchReportsOnly(t *testing.T) {
	dir := t.TempDir()
	state := NewState(dir)
	if err := state.InitWorkspace(); err != nil {
		t.Fatalf("initialize workspace: %v", err)
	}
	researchDir := filepath.Join(state.SettingDir(), "research")
	if err := os.MkdirAll(researchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= workspaceContextResearchLimit+2; index++ {
		name := fmt.Sprintf("2026-11-%02d-report-%02d.md", index, index)
		if err := os.WriteFile(filepath.Join(researchDir, name), []byte("RESEARCH_BODY_SHOULD_BE_LAZY_"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(researchDir, "notes.txt"), []byte("not a report"), 0o644); err != nil {
		t.Fatal(err)
	}

	context := state.WorkspaceContext()
	if !strings.Contains(context.Dynamic, "Recent research reports") {
		t.Fatalf("research section missing:\n%s", context.Dynamic)
	}
	if !strings.Contains(context.Dynamic, `"setting/research/2026-11-08-report-08.md"`) {
		t.Fatalf("latest research report missing:\n%s", context.Dynamic)
	}
	for _, omitted := range []string{"report-01.md", "report-02.md", "notes.txt"} {
		if strings.Contains(context.Dynamic, omitted) {
			t.Fatalf("unexpected research entry %q:\n%s", omitted, context.Dynamic)
		}
	}
	if strings.Contains(context.Markdown(), "RESEARCH_BODY_SHOULD_BE_LAZY") {
		t.Fatalf("research body must be loaded on demand:\n%s", context.Markdown())
	}
}
