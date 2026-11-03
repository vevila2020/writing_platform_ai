package interactive

import (
	"context"
	"encoding/json"

	agentsession "github.com/vevila2020/writing_platform_ai/agent/session"
	"github.com/vevila2020/writing_platform_ai/internal/agents/conversationjournal"
	"github.com/vevila2020/writing_platform_ai/internal/agents/sessionjournal"
)

// LoadCapability reads a branch's runtime state from the Story journal.
func (s *Store) LoadCapability(ctx context.Context, storyID string, key agentsession.Key, capability string) (json.RawMessage, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	handle, err := s.refreshStoryJournalLocked(storyID, false)
	if err != nil {
		return nil, false, err
	}
	return handle.projection.AgentSessions.Capability(key, capability)
}

// UpdateCapabilities atomically serializes related peer-runtime state with Story mutations.
// The selected executor owns the capability; this creates no Native Agent.
func (s *Store) UpdateCapabilities(ctx context.Context, storyID string, key agentsession.Key, upgrade string, update func(sessionjournal.CapabilityReader) (map[string]json.RawMessage, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	release, err := s.acquireStoryMutationLeaseLocked(storyID)
	if err != nil {
		return err
	}
	defer release()
	handle, err := s.refreshStoryJournalLocked(storyID, true)
	if err != nil {
		return err
	}
	records, err := handle.projection.AgentSessions.CapabilityChanges(key, update)
	if err != nil || len(records) == 0 {
		return err
	}
	payloads := make([]json.RawMessage, len(records))
	for i, record := range records {
		payloads[i], err = json.Marshal(record)
		if err != nil {
			return err
		}
	}
	head := handle.journal.Head()
	_, err = handle.journal.AppendWithBackup(ctx, conversationjournal.Guard{Cursor: head.Cursor, RecordSHA256: head.RecordSHA256}, upgrade, payloads...)
	return err
}
