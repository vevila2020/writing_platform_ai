package advance

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/vevila2020/writing_platform_ai/internal/book"
)

var threadIDPattern = regexp.MustCompile(`^F[0-9]+$`)

func applyUpdate(state *State, update Update) error {
	if strings.TrimSpace(update.Summary) == "" || len(update.Summary) > 3072 {
		return fmt.Errorf("chapter summary must contain 1-3072 UTF-8 bytes")
	}
	for _, group := range []struct {
		old, new []string
		name     string
	}{{state.Context.Constraints, update.Context.Constraints, "constraint"}, {state.Context.Risks, update.Context.Risks, "risk"}, {state.Context.NextCommitments, update.Context.NextCommitments, "commitment"}} {
		for _, previous := range group.old {
			if !slices.Contains(group.new, previous) && !slices.ContainsFunc(update.RetiredContext, func(reason string) bool {
				return strings.Contains(reason, previous) && strings.TrimSpace(reason) != previous
			}) {
				return fmt.Errorf("removed %s requires an explicit retired_context entry containing the original text and reason: %s", group.name, previous)
			}
		}
	}
	seen := map[string]bool{}
	for _, character := range update.Characters {
		if seen[character.Name] {
			return fmt.Errorf("duplicate character update: %s", character.Name)
		}
		seen[character.Name] = true
		character.Possessions = append([]string{}, character.Possessions...)
		character.Relationships = append([]string{}, character.Relationships...)
		character.Knowledge = append([]string{}, character.Knowledge...)
		state.Characters[character.Name] = character
	}
	seen = map[string]bool{}
	for _, thread := range update.Threads {
		if seen[thread.ID] {
			return fmt.Errorf("duplicate thread update: %s", thread.ID)
		}
		seen[thread.ID] = true
		state.Threads[thread.ID] = thread
	}
	seen = map[string]bool{}
	for _, secret := range update.Secrets {
		if seen[secret.ID] {
			return fmt.Errorf("duplicate secret update: %s", secret.ID)
		}
		seen[secret.ID] = true
		state.Secrets[secret.ID] = secret
	}
	state.Context = cloneContext(update.Context)
	state.Unresolved = update.Unresolved
	if update.BaselineProgress != nil {
		state.BaselineProgress = *update.BaselineProgress
	}
	return nil
}

// cloneContext also keeps empty lists as JSON arrays, so a prepared context can
// be submitted unchanged through the tool's array schema.
func cloneContext(source Context) Context {
	source.Constraints = append([]string{}, source.Constraints...)
	source.ActiveCharacters = append([]string{}, source.ActiveCharacters...)
	source.ActiveThreads = append([]string{}, source.ActiveThreads...)
	source.ActiveSecrets = append([]string{}, source.ActiveSecrets...)
	source.NextCommitments = append([]string{}, source.NextCommitments...)
	source.Risks = append([]string{}, source.Risks...)
	return source
}

func validateState(state State) error {
	if state.Characters == nil || state.Threads == nil || state.Secrets == nil {
		return fmt.Errorf("continuity state requires character, thread and secret collections")
	}
	if state.Context.Language != "zh-CN" && state.Context.Language != "en-US" {
		return fmt.Errorf("context.language must be zh-CN or en-US")
	}
	if strings.TrimSpace(state.Context.Position) == "" || len(state.Context.Position) > 2048 {
		return fmt.Errorf("context.position requires a concise current story position")
	}
	for _, group := range []struct {
		values []string
		limit  int
		name   string
	}{
		{state.Context.Constraints, 6, "constraints"}, {state.Context.ActiveCharacters, 20, "active_characters"}, {state.Context.ActiveThreads, 8, "active_threads"}, {state.Context.ActiveSecrets, 12, "active_secrets"}, {state.Context.NextCommitments, 6, "next_commitments"}, {state.Context.Risks, 6, "risks"},
	} {
		if len(group.values) > group.limit {
			return fmt.Errorf("context.%s exceeds %d entries; consolidate or ask the author to prioritize", group.name, group.limit)
		}
		seen := map[string]bool{}
		for _, value := range group.values {
			if strings.TrimSpace(value) == "" || seen[value] || len(value) > 2048 {
				return fmt.Errorf("context.%s contains empty, duplicate or oversized entries", group.name)
			}
			seen[value] = true
		}
	}
	for _, name := range state.Context.ActiveCharacters {
		if _, ok := state.Characters[name]; !ok {
			return fmt.Errorf("active character has no snapshot: %s", name)
		}
	}
	for _, id := range state.Context.ActiveThreads {
		if thread, ok := state.Threads[id]; !ok || thread.Status != "active" {
			return fmt.Errorf("active thread is missing or retired: %s", id)
		}
	}
	for _, id := range state.Context.ActiveSecrets {
		if _, ok := state.Secrets[id]; !ok {
			return fmt.Errorf("active secret is missing: %s", id)
		}
	}
	folded := map[string]bool{}
	for key, character := range state.Characters {
		if key != character.Name || strings.TrimSpace(key) == "" || folded[strings.ToLower(key)] {
			return fmt.Errorf("invalid or duplicate character name: %s", key)
		}
		folded[strings.ToLower(key)] = true
		data, _ := json.Marshal(character)
		if len(data) > 8192 {
			return fmt.Errorf("character snapshot %s exceeds 8192 bytes", key)
		}
		if character.Location == "" || character.Condition == "" || character.Goal == "" {
			return fmt.Errorf("character %s requires current location, condition and goal", key)
		}
	}
	for key, thread := range state.Threads {
		if key != thread.ID || !threadIDPattern.MatchString(key) || strings.TrimSpace(thread.Label) == "" || strings.TrimSpace(thread.Detail) == "" {
			return fmt.Errorf("thread %s requires stable F-id, label and actual detail", key)
		}
		switch thread.Status {
		case "active", "resolved", "abandoned":
		default:
			return fmt.Errorf("thread %s has invalid status", key)
		}
		data, _ := json.Marshal(thread)
		if len(data) > 4096 {
			return fmt.Errorf("thread %s exceeds 4096 bytes", key)
		}
	}
	for key, secret := range state.Secrets {
		if key != secret.ID || strings.TrimSpace(key) == "" || strings.TrimSpace(secret.AuthorTruth) == "" || len(secret.AuthorTruth)+len(secret.ReaderKnowledge) > 4096 {
			return fmt.Errorf("secret %s requires an ID, author truth and at most 4096 bytes", key)
		}
	}
	seen := map[string]bool{}
	for _, chapter := range state.Chapters {
		canonical, err := canonicalPath(chapter.Path, "chapters/")
		if err != nil || canonical != chapter.Path || seen[strings.ToLower(chapter.Path)] || !strings.HasPrefix(chapter.Revision, "sha256:") {
			return fmt.Errorf("invalid or duplicate committed chapter: %s", chapter.Path)
		}
		seen[strings.ToLower(chapter.Path)] = true
		if chapter.PlanPath != "" {
			canonical, err = canonicalPath(chapter.PlanPath, "setting/chapter-plans/")
			if err != nil || canonical != chapter.PlanPath || !strings.HasPrefix(chapter.PlanRevision, "sha256:") {
				return fmt.Errorf("invalid committed chapter plan: %s", chapter.PlanPath)
			}
		}
		if len(chapter.Summary) > 3072 {
			return fmt.Errorf("chapter summary exceeds 3072 bytes: %s", chapter.Path)
		}
	}
	if !slices.IsSortedFunc(state.Chapters, func(a, b Chapter) int { return book.CompareManuscriptPaths(a.Path, b.Path) }) {
		return fmt.Errorf("committed chapters are out of manuscript order")
	}
	for _, issue := range state.Unresolved {
		if _, err := canonicalPath(issue.Path, "chapters/"); err != nil || strings.TrimSpace(issue.Reason) == "" {
			return fmt.Errorf("continuity conflicts require a chapter path and reason")
		}
	}
	return nil
}
