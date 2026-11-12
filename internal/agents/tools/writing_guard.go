package tools

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strings"

	"github.com/vevila2020/writing_platform_ai/internal/book/advance"
	workspacechange "github.com/vevila2020/writing_platform_ai/internal/workspace/change"
)

type writingSkillContextKey struct{}

// ContextWithWritingSkill binds product workflow policy to one admitted turn.
// It is inherited by child tools and never alters the independent Agent loop.
func ContextWithWritingSkill(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, writingSkillContextKey{}, strings.TrimSpace(name))
}

func guardAdvanceMutation(ctx context.Context, changes workspaceChangeService, workspace, name string) error {
	if skill, _ := ctx.Value(writingSkillContextKey{}).(string); skill != "novel-serial" {
		return nil
	}
	name = path.Clean(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if filepath.IsAbs(name) {
		rel, err := filepath.Rel(workspace, filepath.FromSlash(name))
		if err != nil {
			return err
		}
		name = filepath.ToSlash(rel)
	}
	for _, managed := range []string{advance.StatePath, "setting/progress.md", "setting/character-states.md", "setting/tracking.md"} {
		if strings.EqualFold(name, managed) {
			return fmt.Errorf("Advance continuity documents are generated; use prepare_advance and commit_advance instead of write/edit")
		}
	}
	if !strings.HasPrefix(strings.ToLower(name), "chapters/") {
		return nil
	}
	plan := discoverChapterPlanPath(workspace, name)
	if plan == "" {
		return fmt.Errorf("Advance requires one unambiguous detailed chapter plan before prose; create the matching setting/chapter-plans file first")
	}
	checked, err := runWritingCheck(ctx, workspace, checkWritingInput{Action: "plan", Path: plan})
	if err != nil {
		return err
	}
	if checked.Blocking > 0 {
		return fmt.Errorf("Advance chapter plan is incomplete; run check_writing action=plan and fill the required fields before prose")
	}
	// Existing prose remains editable so authors can repair an unresolved later
	// chapter without first pretending its continuity has already been fixed.
	if content, _, err := changes.ReadFile(name); err == nil && strings.TrimSpace(content) != "" {
		return nil
	}
	service, ok := changes.(*workspacechange.Service)
	if !ok {
		return fmt.Errorf("Advance requires the product workspace change service")
	}
	if err := advance.NewService(service, nil).CanDraft(ctx, name, plan); err != nil {
		slog.WarnContext(ctx, "Advance draft prerequisite rejected", "path", name, "error", err)
		return err
	}
	return nil
}
