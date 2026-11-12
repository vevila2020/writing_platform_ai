package advance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/vevila2020/writing_platform_ai/internal/book"
	"github.com/vevila2020/writing_platform_ai/internal/portablepath"
	workspacechange "github.com/vevila2020/writing_platform_ai/internal/workspace/change"
)

// Service uses the product's existing content transaction service. Validate is
// required for commit and runs again against the prepared prose and plan.
type Service struct {
	changes  *workspacechange.Service
	Validate func(context.Context, string, string) error
}

func NewService(changes *workspacechange.Service, validate func(context.Context, string, string) error) *Service {
	return &Service{changes: changes, Validate: validate}
}

type fileSnapshot struct{ Content, Revision string }
type snapshot struct {
	state       State
	files       map[string]fileSnapshot
	chapters    []string
	preparation Preparation
}

func (s *Service) Prepare(ctx context.Context, chapterPath, planPath string) (Preparation, error) {
	current, err := s.snapshot(ctx, chapterPath, planPath)
	return current.preparation, err
}

// CanDraft enforces only Advance's structural prerequisites. It never prevents
// edits to the current draft, and it never applies this policy to other Skills.
func (s *Service) CanDraft(ctx context.Context, chapterPath, planPath string) error {
	prepared, err := s.Prepare(ctx, chapterPath, planPath)
	if err != nil {
		return err
	}
	if prepared.NeedsInitialization {
		return fmt.Errorf("initialize Advance from existing prose before drafting another chapter")
	}
	if len(prepared.ViewConflicts) > 0 {
		return fmt.Errorf("continuity views changed; reconcile them through prepare_advance and commit_advance before drafting: %s", strings.Join(prepared.ViewConflicts, ", "))
	}
	if len(prepared.Uncommitted) > 0 {
		return fmt.Errorf("commit or revise preceding chapters before drafting: %s", strings.Join(prepared.Uncommitted, ", "))
	}
	if len(prepared.Unresolved) > 0 {
		return fmt.Errorf("resolve the recorded later-chapter conflicts before drafting")
	}
	return nil
}

func (s *Service) snapshot(ctx context.Context, chapterPath, planPath string) (snapshot, error) {
	current := snapshot{files: map[string]fileSnapshot{}}
	if s == nil || s.changes == nil {
		return current, fmt.Errorf("Advance content service is unavailable")
	}
	chapterPath, err := canonicalPath(chapterPath, "chapters/")
	if err != nil {
		return current, err
	}
	if planPath != "" {
		planPath, err = canonicalPath(planPath, "setting/chapter-plans/")
		if err != nil {
			return current, err
		}
	}
	err = s.changes.WithConsistentWorkspaceSnapshot(ctx, func() error {
		total := 0
		for _, name := range append([]string{StatePath, chapterPath}, viewPaths...) {
			file, err := s.read(name)
			if err != nil {
				return err
			}
			current.files[name] = file
			total += len(file.Content)
		}
		if planPath != "" {
			file, err := s.read(planPath)
			if err != nil {
				return err
			}
			current.files[planPath] = file
			total += len(file.Content)
		}
		if current.files[StatePath].Revision != "missing" {
			if err := json.Unmarshal([]byte(current.files[StatePath].Content), &current.state); err != nil {
				return fmt.Errorf("read continuity state: %w", err)
			}
			if current.state.Schema != Schema || current.state.Revision < 1 {
				return fmt.Errorf("unsupported continuity state schema or revision")
			}
			if err := validateState(current.state); err != nil {
				return fmt.Errorf("invalid continuity state: %w", err)
			}
		}
		chapterRoot := filepath.Join(s.changes.Workspace(), "chapters")
		if err := portablepath.PreflightTree(chapterRoot); err != nil && !os.IsNotExist(err) {
			return err
		}
		root, err := os.OpenRoot(s.changes.Workspace())
		if err != nil {
			return err
		}
		defer root.Close()
		err = filepath.WalkDir(chapterRoot, func(abs string, entry os.DirEntry, scanErr error) error {
			if os.IsNotExist(scanErr) {
				return nil
			}
			if scanErr != nil {
				return scanErr
			}
			if strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !isTextPath(entry.Name()) {
				return nil
			}
			rel, err := filepath.Rel(s.changes.Workspace(), abs)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			file, err := readSnapshotFile(root, rel)
			if err != nil {
				return err
			}
			if strings.TrimSpace(file.Content) != "" {
				if _, read := current.files[rel]; !read {
					total += len(file.Content)
				}
				if total > 64<<20 {
					return fmt.Errorf("Advance source text exceeds 64 MiB; reduce the active project size with a backed-up workspace operation before retrying")
				}
				// Earlier prose is needed only for its revision and manuscript order.
				// Keep the selected chapter body for validation and initialization.
				if rel != chapterPath {
					file.Content = ""
				}
				current.files[rel] = file
				current.chapters = append(current.chapters, rel)
			}
			return nil
		})
		if err != nil {
			return err
		}
		if total > 64<<20 {
			return fmt.Errorf("Advance snapshot exceeds 64 MiB; reduce the active project size with a backed-up workspace operation before retrying")
		}
		sort.Slice(current.chapters, func(i, j int) bool { return book.CompareManuscriptPaths(current.chapters[i], current.chapters[j]) < 0 })
		return nil
	})
	if err != nil {
		return current, err
	}
	p := Preparation{Initialized: current.state.Schema != "", Path: chapterPath, PlanPath: planPath, Context: cloneContext(current.state.Context), Unresolved: append([]Issue{}, current.state.Unresolved...), ViewConflicts: []string{}, Uncommitted: []string{}, Characters: []Character{}, Threads: []Thread{}, Secrets: []Secret{}}
	known := map[string]Chapter{}
	for _, chapter := range current.state.Chapters {
		known[chapter.Path] = chapter
	}
	for _, name := range current.chapters {
		if name == chapterPath {
			chapter := known[name]
			p.Committed = chapter.Revision == current.files[name].Revision && chapter.PlanPath == planPath && (planPath == "" || chapter.PlanRevision == current.files[planPath].Revision)
			continue
		}
		if chapter, ok := known[name]; ok {
			if chapter.Revision != current.files[name].Revision {
				p.Uncommitted = append(p.Uncommitted, name)
			}
		} else if !p.Initialized || book.CompareManuscriptPaths(name, chapterPath) < 0 {
			p.Uncommitted = append(p.Uncommitted, name)
		}
	}
	for _, chapter := range current.state.Chapters {
		if chapter.Path != chapterPath && current.files[chapter.Path].Revision == "" {
			p.Uncommitted = append(p.Uncommitted, chapter.Path)
		}
	}
	p.NeedsInitialization = !p.Initialized && len(p.Uncommitted) > 0
	if p.Initialized {
		views, err := render(current.state)
		if err != nil {
			return current, err
		}
		for _, name := range viewPaths {
			if current.files[name].Content != views[name] {
				p.ViewConflicts = append(p.ViewConflicts, name)
			}
		}
	}
	for _, name := range p.Context.ActiveCharacters {
		p.Characters = append(p.Characters, current.state.Characters[name])
	}
	for _, id := range p.Context.ActiveThreads {
		p.Threads = append(p.Threads, current.state.Threads[id])
	}
	for _, id := range p.Context.ActiveSecrets {
		p.Secrets = append(p.Secrets, current.state.Secrets[id])
	}
	p.RecentChapters = append([]Chapter{}, current.state.Chapters[max(0, len(current.state.Chapters)-3):]...)
	revisions := map[string]string{}
	for name, file := range current.files {
		revisions[name] = file.Revision
	}
	encoded, _ := json.Marshal(struct {
		Path, Plan string
		Revisions  map[string]string
	}{chapterPath, planPath, revisions})
	p.BaseRevision = digest(encoded)
	current.preparation = p
	return current, nil
}

func (s *Service) Commit(ctx context.Context, request CommitRequest) (Receipt, error) {
	current, err := s.snapshot(ctx, request.Path, request.PlanPath)
	if err != nil {
		return Receipt{}, err
	}
	request.Path, request.PlanPath = current.preparation.Path, current.preparation.PlanPath
	submission, _ := json.Marshal(struct {
		Action, Path, Plan, Base string
		Update                   Update
	}{request.Action, request.Path, request.PlanPath, request.BaseRevision, request.Update})
	submissionID := digest(submission)
	if current.state.LastSubmission == submissionID && current.preparation.Committed && len(current.preparation.ViewConflicts) == 0 && len(current.preparation.Uncommitted) == 0 {
		return Receipt{Schema: Schema, Status: "unchanged", Path: request.Path, StateRevision: current.state.Revision, ChangedPaths: []string{}}, nil
	}
	if request.BaseRevision == "" || current.preparation.BaseRevision != request.BaseRevision {
		return Receipt{}, fmt.Errorf("Advance inputs changed; prepare again, reconcile the differences and retry")
	}
	if request.Action != "commit" && request.Action != "initialize" {
		return Receipt{}, fmt.Errorf("action must be initialize or commit")
	}
	file := current.files[request.Path]
	if file.Revision == "missing" || strings.TrimSpace(file.Content) == "" {
		return Receipt{}, fmt.Errorf("final chapter prose is missing or empty")
	}
	if request.Action == "initialize" {
		if current.preparation.Initialized {
			return Receipt{}, fmt.Errorf("Advance is already initialized; use commit to revise continuity")
		}
		if len(current.chapters) == 0 || current.chapters[len(current.chapters)-1] != request.Path {
			return Receipt{}, fmt.Errorf("initialize must adopt the latest existing prose in manuscript order")
		}
		current.state = State{Schema: Schema, Characters: map[string]Character{}, Threads: map[string]Thread{}, Secrets: map[string]Secret{}, ImportedThrough: request.Path, BaselineProgress: current.files[viewPaths[0]].Content}
		for _, name := range current.chapters {
			current.state.Chapters = append(current.state.Chapters, Chapter{Path: name, Revision: current.files[name].Revision})
		}
	} else {
		if current.preparation.NeedsInitialization {
			return Receipt{}, fmt.Errorf("initialize existing prose and continuity first")
		}
		if request.PlanPath == "" || current.files[request.PlanPath].Revision == "missing" {
			return Receipt{}, fmt.Errorf("a validated chapter plan is required before commit")
		}
		if s.Validate == nil {
			return Receipt{}, fmt.Errorf("chapter validator is unavailable")
		}
		if err := s.Validate(ctx, request.Path, request.PlanPath); err != nil {
			return Receipt{}, err
		}
		if current.state.Schema == "" {
			current.state = State{Schema: Schema, Characters: map[string]Character{}, Threads: map[string]Thread{}, Secrets: map[string]Secret{}}
		}
		_, revising := findChapter(current.state.Chapters, request.Path)
		if !revising && (len(current.preparation.Uncommitted) > 0 || len(current.state.Unresolved) > 0) {
			return Receipt{}, fmt.Errorf("preceding chapters have uncommitted changes or unresolved continuity conflicts")
		}
		if !revising && len(current.state.Chapters) > 0 && book.CompareManuscriptPaths(request.Path, current.state.Chapters[len(current.state.Chapters)-1].Path) <= 0 {
			return Receipt{}, fmt.Errorf("new chapters must follow the committed manuscript; revise existing chapters explicitly")
		}
		if revising && current.state.ImportedThrough != "" && book.CompareManuscriptPaths(request.Path, current.state.ImportedThrough) <= 0 && request.Update.BaselineProgress == nil {
			return Receipt{}, fmt.Errorf("revising imported prose requires corrected baseline_progress; preserve still-valid historical progress")
		}
	}
	if err := applyUpdate(&current.state, request.Update); err != nil {
		return Receipt{}, err
	}
	for _, name := range current.preparation.Uncommitted {
		if request.Action != "initialize" && !slices.ContainsFunc(current.state.Unresolved, func(issue Issue) bool { return issue.Path == name }) {
			return Receipt{}, fmt.Errorf("record unresolved conflict for changed chapter %s", name)
		}
	}
	record := Chapter{Path: request.Path, Revision: file.Revision, PlanPath: request.PlanPath, Summary: request.Update.Summary, RetiredContext: request.Update.RetiredContext}
	if request.PlanPath != "" {
		record.PlanRevision = current.files[request.PlanPath].Revision
	}
	if index, ok := findChapter(current.state.Chapters, request.Path); ok {
		current.state.Chapters[index] = record
	} else {
		current.state.Chapters = append(current.state.Chapters, record)
	}
	sort.Slice(current.state.Chapters, func(i, j int) bool {
		return book.CompareManuscriptPaths(current.state.Chapters[i].Path, current.state.Chapters[j].Path) < 0
	})
	current.state.Revision++
	current.state.LastSubmission = submissionID
	if err := validateState(current.state); err != nil {
		return Receipt{}, err
	}
	views, err := render(current.state)
	if err != nil {
		return Receipt{}, err
	}
	encoded, err := json.MarshalIndent(current.state, "", "  ")
	if err != nil {
		return Receipt{}, err
	}
	if len(encoded) > 16<<20 {
		return Receipt{}, fmt.Errorf("continuity state exceeds 16 MiB; consolidate historical summaries before retrying")
	}
	views[StatePath] = string(encoded) + "\n"
	batch := workspacechange.ReplaceFilesRequest{Metadata: request.Metadata}
	for name, content := range views {
		batch.Files = append(batch.Files, workspacechange.FileReplacement{Path: name, Content: content, BaseRevision: current.files[name].Revision})
	}
	for name, file := range current.files {
		if _, changed := views[name]; !changed {
			batch.Guards = append(batch.Guards, workspacechange.FileRevision{Path: name, Revision: file.Revision})
		}
	}
	group, err := s.changes.ReplaceFiles(ctx, batch)
	if err != nil {
		return Receipt{}, err
	}
	paths := make([]string, 0, len(views))
	for name := range views {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	slog.InfoContext(ctx, "Advance chapter committed", "path", request.Path, "revision", current.state.Revision, "group_id", group.ID)
	receipt := Receipt{Schema: Schema, Status: "committed", Path: request.Path, StateRevision: current.state.Revision, ChangeGroupID: group.ID, ChangedPaths: paths}
	for _, change := range group.ChangeSets {
		if change.Path == StatePath {
			mutation := workspacechange.NewToolReceipt(change)
			mutation.AffectedPaths = paths
			receipt.Mutation = &mutation
			break
		}
	}
	return receipt, nil
}

func (s *Service) read(name string) (fileSnapshot, error) {
	_, err := book.SafePath(s.changes.Workspace(), name)
	if err != nil {
		return fileSnapshot{}, err
	}
	root, err := os.OpenRoot(s.changes.Workspace())
	if err != nil {
		return fileSnapshot{}, err
	}
	defer root.Close()
	return readSnapshotFile(root, name)
}

// Scanned manuscript paths have already passed a single portable tree preflight.
// Rooted reads avoid rescanning every sibling for every chapter and contain races.
func readSnapshotFile(root *os.Root, name string) (fileSnapshot, error) {
	info, err := root.Lstat(filepath.FromSlash(name))
	if errors.Is(err, os.ErrNotExist) {
		return fileSnapshot{Revision: "missing"}, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > 16<<20 {
		return fileSnapshot{}, fmt.Errorf("Advance input %s must be a regular text file no larger than 16 MiB", name)
	}
	file, err := root.Open(filepath.FromSlash(name))
	if err != nil {
		return fileSnapshot{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return fileSnapshot{}, err
	}
	if !os.SameFile(info, opened) {
		return fileSnapshot{}, fmt.Errorf("Advance input %s changed while opening", name)
	}
	data, err := io.ReadAll(io.LimitReader(file, (16<<20)+1))
	if err != nil {
		return fileSnapshot{}, err
	}
	if len(data) > 16<<20 {
		return fileSnapshot{}, fmt.Errorf("Advance input %s exceeds 16 MiB", name)
	}
	if !utf8.Valid(data) {
		return fileSnapshot{}, fmt.Errorf("Advance input %s is not valid UTF-8", name)
	}
	return fileSnapshot{Content: string(data), Revision: workspacechange.Revision(data)}, nil
}

func canonicalPath(name, prefix string) (string, error) {
	name = strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	if !strings.HasPrefix(name, prefix) || path.Clean(name) != name || !isTextPath(name) {
		return "", fmt.Errorf("expected a canonical relative text path under %s", prefix)
	}
	return name, nil
}
func isTextPath(name string) bool {
	return strings.EqualFold(path.Ext(name), ".md") || strings.EqualFold(path.Ext(name), ".txt")
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func findChapter(chapters []Chapter, name string) (int, bool) {
	for i, ch := range chapters {
		if ch.Path == name {
			return i, true
		}
	}
	return 0, false
}
