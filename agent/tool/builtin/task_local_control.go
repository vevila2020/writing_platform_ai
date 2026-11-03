package builtin

import (
	"context"

	"github.com/vevila2020/writing_platform_ai/agent"
	agentevent "github.com/vevila2020/writing_platform_ai/agent/lifecycle/event"
)

// Interrupt uses the existing resumable Session pause. It never promotes a
// queued message and never substitutes tree abort for an exact Run control.
func (tasks *LocalTasks) Interrupt(ctx context.Context, ref TaskRef, request agent.SuspendRequest) (agentevent.CommandReceipt, error) {
	_, session, err := tasks.open(ctx, ref)
	if err != nil {
		return agentevent.CommandReceipt{}, err
	}
	request.RunID = ref.Run
	suspension, err := session.SuspendAndClose(ctx, request)
	return suspension.Receipt, err
}
