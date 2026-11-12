package change

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/vevila2020/writing_platform_ai/internal/portablepath"
)

// FileRevision guards a read dependency that must remain unchanged throughout
// a multi-file commit. Paths are canonical workspace-relative paths.
type FileRevision struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

type FileReplacement struct {
	Path         string
	Content      string
	BaseRevision string
}

// ReplaceFilesRequest is one recoverable content transaction, not an Agent
// recovery journal. Existing before-images, review and undo apply to every file.
type ReplaceFilesRequest struct {
	Files    []FileReplacement
	Guards   []FileRevision
	Metadata ChangeMetadata
}

// ReplaceFiles validates the entire batch before writing and reuses the durable
// group-operation protocol to roll an interrupted transaction forward.
func (s *Service) ReplaceFiles(ctx context.Context, req ReplaceFilesRequest) (ChangeGroup, error) {
	if s == nil {
		return ChangeGroup{}, newError(ErrorCodeConflict, "change service is nil", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.contextError(ctx); err != nil {
		return ChangeGroup{}, err
	}
	if err := s.reconcilePendingDurabilityLocked(); err != nil {
		return ChangeGroup{}, err
	}
	if len(req.Files) == 0 || len(req.Files) > 32 || len(req.Guards) > 20000 {
		return ChangeGroup{}, newError(ErrorCodeInvalidEdit, "file batch requires 1-32 files and at most 20000 read guards", nil)
	}
	metadata := normalizeMetadata(req.Metadata)
	if metadata.ChangeGroupID == "" {
		metadata.ChangeGroupID = newID("group")
	}
	seen := map[string]bool{}
	plans := make([]groupChangePlan, 0, len(req.Files))
	total := 0
	for _, file := range req.Files {
		rel, err := s.visibleRelPath(file.Path)
		if err != nil {
			return ChangeGroup{}, err
		}
		if seen[rel] {
			return ChangeGroup{}, newError(ErrorCodeInvalidEdit, "file batch contains duplicate paths", map[string]any{"path": rel})
		}
		seen[rel] = true
		total += len(file.Content)
		if !utf8.ValidString(file.Content) || len(file.Content) > maxWorkspaceMutationFileBytes || total > maxWorkspaceMutationScanBytes {
			return ChangeGroup{}, newError(ErrorCodeInvalidEdit, "file batch exceeds text or size limits", nil)
		}
		expected, err := requireBaseRevision(rel, file.BaseRevision)
		if err != nil {
			return ChangeGroup{}, err
		}
		before, mode, readErr := s.readVisibleFileWithMode(rel)
		exists := readErr == nil
		if readErr != nil {
			var typed *Error
			if !errors.As(readErr, &typed) || typed.Code != ErrorCodeNotFound {
				return ChangeGroup{}, readErr
			}
		}
		if err := requireRevision(rel, expected, stateRevision(before, exists)); err != nil {
			return ChangeGroup{}, err
		}
		after := []byte(file.Content)
		if exists && bytes.Equal(before, after) {
			continue
		}
		review := ReviewStatusPending
		if metadata.AutoAccept {
			review = ReviewStatusAccepted
		}
		edits := []AppliedEdit{{ID: newID("edit"), OldString: string(before), NewString: file.Content, ReviewStatus: review, Hunks: []Hunk{{ID: newID("hunk"), BeforeEnd: len(before), AfterEnd: len(after)}}}}
		change := newChangeSet(rel, before, after, exists, true, edits, metadata)
		change.BeforeMode = uint32(mode)
		if exists {
			change.AfterMode = uint32(mode)
		}
		plans = append(plans, groupChangePlan{ChangeSet: change, Metadata: metadata, Before: before, After: after})
	}
	guards := make([]FileRevision, len(req.Guards))
	parents := map[string]map[string]string{}
	for i, guard := range req.Guards {
		rel := guard.Path
		if err := validateGuardPath(rel); err != nil {
			return ChangeGroup{}, err
		}
		if seen[rel] {
			return ChangeGroup{}, newError(ErrorCodeInvalidEdit, "read guard overlaps a replacement", map[string]any{"path": rel})
		}
		seen[rel] = true
		parent := path.Dir(rel)
		if _, checked := parents[parent]; !checked {
			if parent != "." {
				if err := portablepath.CheckPath(s.workspace, parent); err != nil {
					return ChangeGroup{}, err
				}
			}
			entries, err := os.ReadDir(filepath.Join(s.workspace, filepath.FromSlash(parent)))
			if err != nil && !os.IsNotExist(err) {
				return ChangeGroup{}, err
			}
			names := map[string]string{}
			for _, entry := range entries {
				key := portablepath.FoldKey(entry.Name())
				if previous := names[key]; previous != "" && previous != entry.Name() {
					return ChangeGroup{}, newError(ErrorCodeConflict, "read guard directory has a portable filename collision", map[string]any{"path": parent})
				}
				names[key] = entry.Name()
			}
			parents[parent] = names
		}
		if matched := parents[parent][portablepath.FoldKey(path.Base(rel))]; matched != "" && matched != path.Base(rel) {
			return ChangeGroup{}, newError(ErrorCodeConflict, "read guard path has a portable filename collision", map[string]any{"path": rel})
		}
		guards[i] = FileRevision{Path: rel, Revision: guard.Revision}
	}
	if err := s.verifyFileGuards(guards); err != nil {
		return ChangeGroup{}, err
	}
	if len(plans) == 0 {
		return ChangeGroup{}, newError(ErrorCodeNoChange, "file batch already matches the requested content", nil)
	}
	if err := s.commitGroupOperationLocked(ctx, operationKindReplace, metadata.ChangeGroupID, plans, operationProjection{}, guards); err != nil {
		slog.ErrorContext(ctx, "Workspace file batch failed", "group_id", metadata.ChangeGroupID, "error", err)
		return ChangeGroup{}, err
	}
	slog.InfoContext(ctx, "Workspace file batch committed", "group_id", metadata.ChangeGroupID, "files", len(plans))
	return s.groupDetailLocked(metadata.ChangeGroupID, false)
}

func validateGuardPath(name string) error {
	if name == "." || path.Clean(name) != name || filepath.IsAbs(name) {
		return newError(ErrorCodeConflict, "read guard requires a canonical workspace-relative path", map[string]any{"path": name})
	}
	if err := portablepath.Validate(name); err != nil {
		return err
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") {
			return newError(ErrorCodeConflict, "hidden workspace paths cannot be guarded", map[string]any{"path": name})
		}
	}
	return nil
}

func (s *Service) verifyFileGuards(guards []FileRevision) error {
	for _, guard := range guards {
		expected, err := requireBaseRevision(guard.Path, guard.Revision)
		if err != nil {
			return err
		}
		content, exists, err := s.readVisibleState(guard.Path)
		if err != nil {
			return err
		}
		if err := requireRevision(guard.Path, expected, stateRevision(content, exists)); err != nil {
			return err
		}
	}
	return nil
}
