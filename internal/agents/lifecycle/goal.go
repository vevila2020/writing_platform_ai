package lifecycle

import (
	"context"
	"fmt"
	"strings"

	agentchat "github.com/vevila2020/writing_platform_ai/internal/agents/chat"
	"github.com/vevila2020/writing_platform_ai/internal/agents/modelio"
	agentrun "github.com/vevila2020/writing_platform_ai/internal/agents/run"

	publicgoal "github.com/vevila2020/writing_platform_ai/agent/engine/goal"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

// NewGoalManager returns InkDock's adapter for the public revisioned Goal
// capability. Native owns evaluation and state transitions; this adapter only
// prepares InkDock continuation HostData. External runtimes do not call it.
func NewGoalManager() publicgoal.GoalManager {
	return denovaGoalManager{delegate: publicgoal.Standard()}
}

type denovaGoalManager struct{ delegate publicgoal.GoalManager }

func (manager denovaGoalManager) Identity() agentschema.CapabilityIdentity {
	return agentschema.CapabilityIdentity{Kind: "denova.goal.standard", Version: 3}
}

func (manager denovaGoalManager) Apply(ctx context.Context, request publicgoal.GoalApplyRequest) (publicgoal.GoalState, error) {
	return manager.delegate.Apply(ctx, request)
}

func (manager denovaGoalManager) Prepare(ctx context.Context, request publicgoal.GoalPrepareRequest) (publicgoal.GoalPreparation, error) {
	return manager.delegate.Prepare(ctx, request)
}

func (manager denovaGoalManager) AfterRun(ctx context.Context, request publicgoal.GoalAfterRunRequest) (publicgoal.GoalAfterRunDecision, error) {
	decision, err := manager.delegate.AfterRun(modelio.WithTraceSource(ctx, "goal_evaluation"), request)
	if err != nil || decision.Verdict != publicgoal.GoalVerdictContinue {
		return decision, err
	}
	data, err := DecodeTurnHostData(request.Input)
	if err != nil {
		return decision, fmt.Errorf("prepare InkDock Goal continuation: %w", err)
	}
	message := strings.TrimSpace(decision.Input.Text)
	if message == "" {
		return decision, fmt.Errorf("prepare InkDock Goal continuation: message is empty")
	}
	// Preserve only locale and durable routing. References, selections, review
	// feedback, and explicit Skills belonged to the completed caller turn and
	// must not be replayed as a new autonomous instruction.
	next, err := TurnInput(TurnNext, agentchat.ChatRequest{
		Message: message, Locale: data.Caller.Locale,
		InputVisibility: agentrun.InputModelOnly,
	}, agentrun.Options{
		AutomationTaskID: data.AutomationID,
		TurnID:           data.TurnID,
		MaintenanceTask:  data.MaintenanceTask,
		Mode:             data.Mode,
		WriteMode:        data.WriteMode,
		WriteScope:       data.WriteScope,
		RestoreData:      data.RestoreData,
	})
	if err != nil {
		return decision, fmt.Errorf("prepare InkDock Goal continuation input: %w", err)
	}
	// Agent assigns the durable command identity for automatic continuations.
	next.IdempotencyKey = ""
	decision.Input = next
	return decision, nil
}

var _ publicgoal.GoalManager = denovaGoalManager{}
