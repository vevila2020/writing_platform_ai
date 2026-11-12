package skills

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestBundledWritingWorkflowsIncludeNovelSerial(t *testing.T) {
	builtin := filepath.Join("..", "..", "..", "skills")
	for _, reference := range []string{
		"workflow-plan.md",
		"workflow-chapter.md",
		"workflow-daily.md",
		"workflow-revision.md",
		"chapter-plan-template.md",
		"tracking-template.md",
		"continuity-and-quality.md",
		"review-rubric.md",
		"context-and-recall.md",
	} {
		path := filepath.Join(builtin, "novel-serial", "references", reference)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("novel-serial reference %s is missing: %v", reference, err)
		}
	}

	backend := NewAgentBackend(NewDirectories(builtin, "", ""), "ide", nil)
	available, err := backend.List(context.Background())
	if err != nil {
		t.Fatalf("list bundled skills: %v", err)
	}

	for _, item := range available {
		if item.Name != "novel-serial" {
			continue
		}
		if !slices.Contains(item.Capabilities, CapabilityWritingWorkflow) {
			t.Fatalf("novel-serial must opt into %q: %#v", CapabilityWritingWorkflow, item.Capabilities)
		}
		if !agentMatches(item.Agent, "ide") {
			t.Fatalf("novel-serial must be available to the IDE agent, got %q", item.Agent)
		}
		return
	}
	t.Fatal("bundled novel-serial skill was not discovered")
}
