package advance

import (
	"fmt"
	"sort"
	"strings"
)

func render(state State) (map[string]string, error) {
	t := textEN
	if state.Context.Language == "zh-CN" {
		t = textZH
	}
	var progress, characters, tracking strings.Builder
	section := func(b *strings.Builder, title string, items []string) {
		fmt.Fprintf(b, "## %s\n\n", title)
		if len(items) == 0 {
			items = []string{t.None}
		}
		for _, item := range items {
			fmt.Fprintf(b, "- %s\n", item)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&progress, "# %s\n\n", t.Progress)
	if state.BaselineProgress != "" {
		fmt.Fprintf(&progress, "## %s\n\n%s\n\n", t.Imported, strings.TrimSpace(state.BaselineProgress))
	}
	for _, chapter := range state.Chapters {
		if chapter.Summary != "" {
			fmt.Fprintf(&progress, "## %s\n\n%s\n\n", chapter.Path, chapter.Summary)
		}
	}
	section(&progress, t.Position, []string{state.Context.Position})
	section(&progress, t.Commitments, state.Context.NextCommitments)
	fmt.Fprintf(&characters, "# %s\n\n", t.Characters)
	names := make([]string, 0, len(state.Characters))
	for name := range state.Characters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		character := state.Characters[name]
		fmt.Fprintf(&characters, "## %s\n\n- %s: %s\n- %s: %s\n- %s: %s\n", name, t.Location, character.Location, t.Condition, character.Condition, t.Goal, character.Goal)
		for _, field := range []struct {
			name   string
			values []string
		}{{t.Possessions, character.Possessions}, {t.Relationships, character.Relationships}, {t.Knowledge, character.Knowledge}} {
			value := strings.Join(field.values, "; ")
			if value == "" {
				value = t.None
			}
			fmt.Fprintf(&characters, "- %s: %s\n", field.name, value)
		}
		characters.WriteString("\n")
	}
	fmt.Fprintf(&tracking, "# %s\n\n", t.Tracking)
	section(&tracking, t.Position, []string{state.Context.Position})
	section(&tracking, t.Constraints, state.Context.Constraints)
	active := []string{}
	for _, name := range state.Context.ActiveCharacters {
		character := state.Characters[name]
		active = append(active, name+"｜"+character.Location+"｜"+character.Condition+"｜"+character.Goal)
	}
	section(&tracking, t.Characters, active)
	threads := []string{}
	for _, id := range state.Context.ActiveThreads {
		thread := state.Threads[id]
		threads = append(threads, thread.ID+"｜"+thread.Label+"｜"+thread.Detail)
	}
	section(&tracking, t.Threads, threads)
	author, reader := []string{}, []string{}
	for _, id := range state.Context.ActiveSecrets {
		secret := state.Secrets[id]
		author = append(author, id+"｜"+secret.AuthorTruth)
		if secret.ReaderKnowledge != "" {
			reader = append(reader, id+"｜"+secret.ReaderKnowledge)
		}
	}
	section(&tracking, t.AuthorTruth, author)
	section(&tracking, t.ReaderKnowledge, reader)
	section(&tracking, t.Commitments, state.Context.NextCommitments)
	section(&tracking, t.Risks, state.Context.Risks)
	recent := []string{}
	for _, chapter := range state.Chapters[max(0, len(state.Chapters)-3):] {
		if chapter.Summary != "" {
			recent = append(recent, chapter.Path+"："+chapter.Summary)
		}
	}
	section(&tracking, t.Recent, recent)
	if len(state.Unresolved) > 0 {
		issues := []string{}
		for _, issue := range state.Unresolved {
			issues = append(issues, issue.Path+"："+issue.Reason)
		}
		section(&tracking, t.Conflicts, issues)
	}
	if tracking.Len() > 12<<10 {
		return nil, fmt.Errorf("continuity working set exceeds 12 KiB; compress or retire context explicitly before retrying")
	}
	if characters.Len() > 2<<20 || progress.Len() > 2<<20 {
		return nil, fmt.Errorf("generated progress or character states exceed 2 MiB; consolidate historical content before retrying")
	}
	return map[string]string{viewPaths[0]: progress.String(), viewPaths[1]: characters.String(), viewPaths[2]: tracking.String()}, nil
}
