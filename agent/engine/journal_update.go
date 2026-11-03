package engine

import (
	"encoding/json"

	agentcanonical "github.com/vevila2020/writing_platform_ai/agent/session/canonical"
)

// CanonicalUpdate describes one existing product boundary. The Session lock
// spans the host commit so inbox acceptance cannot race its logical revision.
type CanonicalUpdate struct {
	Stage            agentcanonical.CommitStage
	Snapshot         TurnSnapshot
	State            json.RawMessage
	Hash             string
	Tool             *PersistedTool
	CapabilityStates map[string]json.RawMessage
}
