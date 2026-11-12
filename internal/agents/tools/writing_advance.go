package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
	agenttool "github.com/vevila2020/writing_platform_ai/agent/tool"
	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/book/advance"
	workspacechange "github.com/vevila2020/writing_platform_ai/internal/workspace/change"
)

type prepareAdvanceInput struct {
	Path     string `json:"path" jsonschema_description:"Workspace-relative chapter prose path under chapters/. May be missing before drafting."`
	PlanPath string `json:"plan_path,omitempty" jsonschema_description:"Detailed plan under setting/chapter-plans/. Omit to discover a unique matching order prefix."`
}

type commitAdvanceInput struct {
	Action         string         `json:"action" jsonschema:"enum=commit,enum=initialize" jsonschema_description:"commit validates final prose and commits its continuity. initialize adopts existing prose and confirmed current facts on first use; it does not draft or rewrite prose."`
	Path           string         `json:"path"`
	PlanPath       string         `json:"plan_path,omitempty"`
	BaseRevision   string         `json:"base_revision" jsonschema_description:"Exact token from prepare_advance after the final prose and plan edit. Re-prepare and reconcile when stale."`
	Update         advance.Update `json:"update"`
	WordMin        int            `json:"word_min,omitempty"`
	WordMax        int            `json:"word_max,omitempty"`
	LengthDecision string         `json:"length_decision,omitempty" jsonschema:"enum=author_accepted" jsonschema_description:"Set author_accepted only after the author explicitly accepts this chapter's out-of-range length. Hard quality failures cannot be overridden."`
}

func newAdvanceTools(cfg *config.Config, metadata WorkspaceMetadataProvider, writable bool) ([]agenttool.ToolDefinition, error) {
	prepare, err := agenttool.InferTool("prepare_advance", "Read the Advance chapter checkpoint and bounded continuity working set. Returns an exact base_revision token, active character/foreshadowing/information-boundary records, recent chapter summaries, stale preceding chapters and modified generated views. Call before drafting and again after final review edits. This does not draft, initialize, or commit a chapter. Read the selected writing Skill for the semantic transaction contract.", func(ctx context.Context, input prepareAdvanceInput) (string, error) {
		service, err := advanceService(cfg, nil)
		if err != nil {
			return "", err
		}
		plan := strings.TrimSpace(input.PlanPath)
		if plan == "" {
			plan = discoverChapterPlanPath(cfg.Workspace, input.Path)
		}
		prepared, err := service.Prepare(ctx, input.Path, plan)
		if err != nil {
			return "", err
		}
		data, err := json.Marshal(prepared)
		if err == nil && len(data) > catalogToolResultMaxBytes(cfg) {
			// Always return the revision token needed to compact the working set.
			// Omitting whole named sections is explicit; source facts stay intact.
			data, err = json.Marshal(struct {
				BaseRevision        string   `json:"base_revision"`
				Initialized         bool     `json:"initialized"`
				Committed           bool     `json:"committed"`
				Path                string   `json:"path"`
				PlanPath            string   `json:"plan_path,omitempty"`
				NeedsInitialization bool     `json:"needs_initialization"`
				ViewConflicts       []string `json:"view_conflicts"`
				UncommittedCount    int      `json:"uncommitted_count"`
				UnresolvedCount     int      `json:"unresolved_count"`
				ContextSource       string   `json:"context_source"`
				ContextOmitted      bool     `json:"context_omitted"`
				Message             string   `json:"message"`
			}{BaseRevision: prepared.BaseRevision, Initialized: prepared.Initialized, Committed: prepared.Committed, Path: prepared.Path, PlanPath: prepared.PlanPath, NeedsInitialization: prepared.NeedsInitialization, ViewConflicts: prepared.ViewConflicts, UncommittedCount: len(prepared.Uncommitted), UnresolvedCount: len(prepared.Unresolved), ContextSource: advance.StatePath, ContextOmitted: true, Message: "Working context exceeds the configured result budget. Read relevant records and conflicts from context_source and manuscript files before submitting a smaller active set. No source facts were truncated."})
			if err == nil && len(data) > catalogToolResultMaxBytes(cfg) {
				return "", fmt.Errorf("Advance checkpoint exceeds the configured tool result budget; increase the existing tool result limit before retrying")
			}
		}
		return string(data), err
	})
	if err != nil {
		return nil, err
	}
	definition, err := defineTool(prepare, boundedReadDescriptor(agenttool.ToolSourceOther, config.AgentToolWritingCheck, agentschema.ToolResultRecoveryRerun))
	if err != nil {
		return nil, err
	}
	tools := []agenttool.ToolDefinition{definition}
	if !writable {
		return tools, nil
	}
	commit, err := agenttool.InferTool("commit_advance", "Commit one final chapter's semantic continuity update after review. Rechecks the detailed plan and prose, rejects stale inputs and uncommitted preceding chapters, and writes canonical book continuity plus generated progress, character-state and tracking views as one recoverable workspace change group. Upsert only changed character/thread/secret records; context replaces the working set. Never directly edit setting/continuity.json or its generated views. initialize adopts an existing book once while preserving earlier progress and before-images. Retry identical successful requests safely; prepare again after any subsequent edit. This book artifact is not an Agent recovery journal.", func(ctx context.Context, input commitAdvanceInput) (agentschema.ToolResult, error) {
		validator := func(ctx context.Context, path, plan string) error {
			for _, check := range []checkWritingInput{{Action: "plan", Path: plan}, {Action: "chapter", Path: path, PlanPath: plan, WordMin: input.WordMin, WordMax: input.WordMax}} {
				result, err := runWritingCheck(ctx, cfg.Workspace, check)
				if err != nil {
					return err
				}
				if result.Blocking > 0 {
					data, _ := json.Marshal(result)
					return fmt.Errorf("chapter commit blocked: %s", data)
				}
				for _, finding := range result.Findings {
					if finding.Type == "word_count" && input.LengthDecision != "author_accepted" {
						return fmt.Errorf("chapter length is outside the effective range; revise it or ask the author before setting length_decision=author_accepted")
					}
				}
			}
			return nil
		}
		service, err := advanceService(cfg, validator)
		if err != nil {
			return agentschema.ToolResult{}, err
		}
		plan := strings.TrimSpace(input.PlanPath)
		if plan == "" {
			plan = discoverChapterPlanPath(cfg.Workspace, input.Path)
		}
		receipt, err := service.Commit(ctx, advance.CommitRequest{Action: input.Action, Path: input.Path, PlanPath: plan, BaseRevision: input.BaseRevision, Update: input.Update, Metadata: workspaceChangeMetadata(ctx, metadata)})
		if err != nil {
			slog.WarnContext(ctx, "Advance chapter commit rejected", "path", input.Path, "error", err)
			return agentschema.ToolResult{}, err
		}
		var payload any = receipt
		if receipt.Mutation != nil {
			payload = receipt.Mutation
		} else if receipt.Status == "unchanged" {
			payload = workspaceWriteNoChangeReceipt{
				Schema: "workspace_change.tool_noop.v1", Status: "unchanged", Path: advance.StatePath,
				Message: "This chapter submission is already committed. Continue without retrying.",
			}
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return agentschema.ToolResult{}, err
		}
		result := agentschema.TextToolResult(string(data))
		result.Details = data
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	definition, err = defineTool(commit, workspaceWriteDescriptor(agenttool.ToolSourceWrite, config.AgentToolWorkspaceWrite, agenttool.ToolRecoveryReconcilable))
	if err != nil {
		return nil, err
	}
	return append(tools, definition), nil
}

func advanceService(cfg *config.Config, validate func(context.Context, string, string) error) (*advance.Service, error) {
	changes, err := workspacechange.ForWorkspaceAt(cfg.Workspace, cfg.ProjectStoreDir)
	if err != nil {
		return nil, err
	}
	return advance.NewService(changes, validate), nil
}
