package canonical

import (
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func testContextStateFragment(revision, content string) agentschema.ContextFragment {
	return agentschema.ContextFragment{
		Source: "test.workspace", Purpose: "provide current workspace state", Resource: "workspace",
		Revision: revision, StateID: "workspace", Stability: agentschema.ContextSessionState, Placement: agentschema.ContextStateMessage,
		Rendering: agentschema.ContextRenderVerbatim, Content: content, HardLimit: 64 << 10,
	}
}
