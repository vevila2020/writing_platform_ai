package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"denova/config"
	"denova/internal/book"

	agentcontext "github.com/alfredxw/denova/agent/context"
	agentschema "github.com/alfredxw/denova/agent/schema"
)

const bookProfileResource = "book profile"

// NewBookProfileContextSource exposes the author-entered Book title, author, and
// synopsis to the writing Agent. The synopsis lives in Project Store metadata,
// outside the workspace file tools, so without this source the model invents
// characters that contradict the premise the author wrote at Book creation.
//
// Only the writing Agent receives it: Game stories own their premise through
// story setup, and other Agent kinds do not write book prose. It returns nil
// when the runtime has no bound Project Store.
func NewBookProfileContextSource(cfg *config.Config, agentKind string, state *book.State) (agentcontext.ContextSource, error) {
	if state == nil || cfg == nil || strings.TrimSpace(agentKind) != config.AgentKindIDE {
		return nil, nil
	}
	storeRoot := strings.TrimSpace(cfg.ProjectStoreDir)
	if storeRoot == "" {
		return nil, nil
	}
	limit := config.ResolveAgentContext(cfg, agentKind).MaxFragmentBytes
	encoded, err := json.Marshal(struct {
		AgentKind string
		ProjectID string
		Limit     int
	}{agentKind, strings.TrimSpace(cfg.ProjectID), limit})
	if err != nil {
		return nil, fmt.Errorf("encode Book profile context identity: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return bookProfileContextSource{
		contentRoot: strings.TrimSpace(state.Workspace()),
		storeRoot:   storeRoot,
		limit:       limit,
		identity: agentschema.CapabilityIdentity{
			Kind: "denova.book_profile.context", Version: 1, ConfigHash: hex.EncodeToString(digest[:]),
		},
	}, nil
}

type bookProfileContextSource struct {
	contentRoot string
	storeRoot   string
	limit       int
	identity    agentschema.CapabilityIdentity
}

func (source bookProfileContextSource) Identity() agentschema.CapabilityIdentity {
	return source.identity
}

// Materialize reads metadata on every request so edits in Book settings apply to
// the next turn. A blank synopsis produces no fragment; an oversize profile is
// rejected instead of being silently truncated.
func (source bookProfileContextSource) Materialize(context.Context, agentcontext.ContextRequest) ([]agentschema.ContextFragment, error) {
	meta, err := book.ReadStoredMeta(source.contentRoot, source.storeRoot)
	if err != nil {
		return nil, fmt.Errorf("read Book profile: %w", err)
	}
	synopsis := strings.TrimSpace(meta.Description)
	if synopsis == "" {
		return nil, nil
	}
	content := bookProfileContent(meta, synopsis)
	if len(content) > source.limit {
		return nil, fmt.Errorf("the Book profile exceeds the %d-byte context limit; shorten the Book synopsis", source.limit)
	}
	digest := sha256.Sum256([]byte(content))
	return []agentschema.ContextFragment{{
		Source: "Book metadata", Purpose: "keep prose consistent with the author-entered title and synopsis",
		Resource: bookProfileResource, Revision: hex.EncodeToString(digest[:]),
		Stability: agentschema.ContextStablePrefix, Placement: agentschema.ContextLeadingMessage, Rendering: agentschema.ContextRenderVerbatim, Role: agentschema.User,
		Content: content, HardLimit: source.limit,
	}}, nil
}

func bookProfileContent(meta book.BookMeta, synopsis string) string {
	var sb strings.Builder
	sb.WriteString("# Book profile\n\n")
	if title := strings.TrimSpace(meta.Title); title != "" {
		sb.WriteString("Title: " + title + "\n")
	}
	if author := strings.TrimSpace(meta.Author); author != "" {
		sb.WriteString("Author: " + author + "\n")
	}
	sb.WriteString("\nSynopsis:\n" + synopsis + "\n\n")
	sb.WriteString("The author wrote this profile in Book settings. Treat its character names, relationships, and premise as established canon. " +
		"Explicit user requests and confirmed workspace canon take precedence. When existing chapters contradict the profile, point out the conflict to the author instead of silently choosing one.")
	return sb.String()
}
