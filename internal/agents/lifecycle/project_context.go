package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/book"

	agentcontext "github.com/vevila2020/writing_platform_ai/agent/context"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

type projectInstructionDefinition struct {
	resource string
	heading  string
	source   string
	purpose  string
}

const writingContextCacheAnchor = "Writing context follows."

var projectInstructionDefinitions = [...]projectInstructionDefinition{
	{
		resource: book.AgentInstructionsFileName,
		heading:  "# Project instructions",
		source:   "workspace AGENTS.md",
		purpose:  "apply stable workspace-level project and workflow instructions",
	},
	{
		resource: book.CreatorFileName,
		heading:  book.CreatorInstructionsHeading,
		source:   "workspace CREATOR.md",
		purpose:  "apply stable workspace-level creative instructions",
	},
}

// NewProjectInstructionsContextSource puts a constant writing anchor before
// root AGENTS.md and CREATOR.md so turn changes retain a stable first message.
func NewProjectInstructionsContextSource(cfg *config.Config, agentKind string, state *book.State) (agentcontext.ContextSource, error) {
	if state == nil {
		return nil, nil
	}
	agentKind = strings.TrimSpace(agentKind)
	if agentKind == "" {
		return nil, fmt.Errorf("project instruction context requires an Agent kind")
	}
	workspace := strings.TrimSpace(state.Workspace())
	projectID := ""
	if cfg != nil {
		projectID = strings.TrimSpace(cfg.ProjectID)
	}
	limit := config.ResolveAgentContext(cfg, agentKind).MaxFragmentBytes
	encoded, err := json.Marshal(struct {
		AgentKind string
		ProjectID string
		Resources []string
		Limit     int
	}{agentKind, projectID, []string{book.AgentInstructionsFileName, book.CreatorFileName}, limit})
	if err != nil {
		return nil, fmt.Errorf("encode project instruction context identity: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return projectInstructionsContextSource{
		files: book.NewService(workspace),
		limit: limit, agentKind: agentKind,
		identity: agentschema.CapabilityIdentity{
			Kind: "denova.project_instructions.context", Version: 4, ConfigHash: hex.EncodeToString(digest[:]),
		},
	}, nil
}

type projectInstructionsContextSource struct {
	files     *book.Service
	limit     int
	agentKind string
	identity  agentschema.CapabilityIdentity
}

func (source projectInstructionsContextSource) Identity() agentschema.CapabilityIdentity {
	return source.identity
}

func (source projectInstructionsContextSource) Materialize(context.Context, agentcontext.ContextRequest) ([]agentschema.ContextFragment, error) {
	fragments := make([]agentschema.ContextFragment, 0, len(projectInstructionDefinitions)+1)
	if source.agentKind == config.AgentKindIDE {
		fragments = append(fragments, agentschema.ContextFragment{
			Source: "writing.context.cache_anchor", Purpose: "keep the first model content stable across writing turns",
			Resource: "writing-context-cache-anchor", Revision: "1",
			Stability: agentschema.ContextStablePrefix, Placement: agentschema.ContextLeadingMessage, Rendering: agentschema.ContextRenderVerbatim, Role: agentschema.User,
			Content: writingContextCacheAnchor, HardLimit: source.limit,
		})
	}
	for _, definition := range projectInstructionDefinitions {
		raw, err := source.files.ReadFile(definition.resource)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read project instruction %s: %w", definition.resource, err)
		}
		body := strings.TrimSpace(raw)
		if body == "" {
			continue
		}
		content := book.ProjectInstructionsContent(definition.heading, body)
		if len(content) > source.limit {
			return nil, fmt.Errorf("%s exceeds the %d-byte project instruction limit", definition.resource, source.limit)
		}
		digest := sha256.Sum256([]byte(body))
		fragments = append(fragments, agentschema.ContextFragment{
			Source: definition.source, Purpose: definition.purpose,
			Resource: definition.resource, Revision: hex.EncodeToString(digest[:]),
			Stability: agentschema.ContextStablePrefix, Placement: agentschema.ContextLeadingMessage, Rendering: agentschema.ContextRenderVerbatim, Role: agentschema.User,
			Content: content, HardLimit: source.limit,
		})
	}
	return fragments, nil
}
