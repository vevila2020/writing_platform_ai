package automationapp

import (
	agentexecution "github.com/vevila2020/writing_platform_ai/internal/agents/execution"
	"path/filepath"
	"strings"

	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/agents/session"
	"github.com/vevila2020/writing_platform_ai/internal/book"
	projectdomain "github.com/vevila2020/writing_platform_ai/internal/project"
)

// automationWorkspaceSnapshot binds asynchronous trigger evaluation and any
// resulting automation run to one workspace runtime. The Host keeps the
// referenced generation alive with an Operation for as long as it is used.
type automationWorkspaceSnapshot struct {
	projectID        string
	projectType      projectdomain.Type
	stateRoot        string
	workspace        string
	novaDir          string
	cfg              config.Config
	bookState        *book.State
	bookService      *book.Service
	sessionStore     *session.Store
	executionRuntime *agentexecution.Runtime
}

// runtimeSnapshot captures the currently selected immutable runtime through
// the Host boundary.
func (s *Service) runtimeSnapshot() (*automationWorkspaceSnapshot, error) {
	if s == nil || s.host == nil {
		return nil, ErrNoWorkspace
	}
	runtime, err := s.host.CurrentRuntime()
	if err != nil {
		return nil, err
	}
	return snapshotFromRuntime(runtime), nil
}

func canonicalAutomationWorkspace(workspace string) string {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return ""
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return filepath.Clean(workspace)
	}
	if canonical, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(canonical)
	}
	return filepath.Clean(abs)
}
