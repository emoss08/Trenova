package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingPlanStore struct {
	created []*agentplan.AgentPlan
}

func (s *capturingPlanStore) Create(
	_ context.Context,
	plan *agentplan.AgentPlan,
) (*agentplan.AgentPlan, error) {
	plan.ID = pulid.MustNew("apl_")
	s.created = append(s.created, plan)

	return plan, nil
}

// handOffEffects drives a conversation turn in process the way the turn
// workflow does, including the hand-off: the delegate's turn is opened by the
// assistant and driven through the same effects.
type handOffEffects struct {
	ctx      context.Context
	t        *testing.T
	svc      *Service
	parent   *agentdefinition.Definition
	threadID pulid.ID
	opened   []*agentruntimeports.RunRequest
}

func (fx *handOffEffects) Complete(
	_ *agentruntime.Turn,
	req *llmports.ChatCompletionRequest,
) (modelstream.Reply, error) {
	return fx.svc.runtime.StreamCompletion(fx.ctx, req, fx.Emit)
}

func (fx *handOffEffects) Dispatch(
	t *agentruntime.Turn,
	call toolcall.Call,
) toolcall.Outcome {
	return fx.svc.runtime.DispatchStep(fx.ctx, t.Request(), call)
}

func (fx *handOffEffects) DispatchReads(
	t *agentruntime.Turn,
	calls []toolcall.Call,
) []toolcall.Outcome {
	outcomes := make([]toolcall.Outcome, 0, len(calls))
	for _, call := range calls {
		outcomes = append(outcomes, fx.Dispatch(t, call))
	}

	return outcomes
}

func (*handOffEffects) Find(*agentruntime.Turn, map[string]any) toolaccess.FindAnswer {
	return toolaccess.FindAnswer{}
}

func (*handOffEffects) Emit(kernelports.StreamEvent) {}

func (fx *handOffEffects) Observe(
	_ *agentruntime.Turn,
	call *llmports.ToolCall,
	outcome toolcall.Outcome,
) toolcall.Outcome {
	return fx.svc.runtime.ObserveCall(nil, call, outcome)
}

func (*handOffEffects) NewCallID() string { return agentruntime.NewCallID() }

func (fx *handOffEffects) Delegate(
	t *agentruntime.Turn,
	calls []delegation.Call,
) []delegation.Run {
	runs := make([]delegation.Run, 0, len(calls))
	for _, call := range calls {
		runs = append(runs, fx.delegate(t, call))
	}

	return runs
}

func (fx *handOffEffects) delegate(
	t *agentruntime.Turn,
	call delegation.Call,
) delegation.Run {
	opened, err := fx.svc.OpenDelegate(fx.ctx, &OpenDelegateRequest{
		Parent:    fx.parent,
		Actor:     t.Request().Actor,
		ThreadID:  fx.threadID,
		StepOwner: t.Request().StepOwner,
		Call:      call,
	})
	require.NoError(fx.t, err)
	fx.opened = append(fx.opened, opened.Request)

	turn := fx.svc.runtime.RestoreTurn(opened.Request, opened.Turn)
	turn.ReserveCallIDs(call.CallIDs)
	result, err := fx.svc.runtime.Drive(turn, fx)
	require.NoError(fx.t, err)

	return delegation.Run{Definition: opened.Request.Definition, Result: result}
}

func (*handOffEffects) Interject(
	*agentruntime.Turn,
	*worldwatch.Check,
) worldwatch.Interjections {
	return worldwatch.Interjections{}
}

func (*handOffEffects) Now() int64 { return timeutils.NowUnix() }

func savedWithIDs(messages []conversation.Message) []conversation.Message {
	saved := make([]conversation.Message, len(messages))
	for idx := range messages {
		saved[idx] = messages[idx]
		saved[idx].ID = pulid.MustNew("amsg_")
	}

	return saved
}

func callNamed(t *testing.T, saved []conversation.Message, tool string) string {
	t.Helper()

	for idx := range saved {
		for _, call := range saved[idx].ToolCalls {
			if call.Name == tool {
				return call.ID
			}
		}
	}
	require.Failf(t, "no call", "the turn made no %s call", tool)

	return ""
}

func TestPersistProposals_AReadOnlyHandOffLeavesItsRun(t *testing.T) {
	t.Parallel()

	runs := &stubRunRepo{}
	proposals := &stubProposalRepo{}
	svc := newProposalService(runs, proposals, &stubConversationRepo{})
	params := proposalTestParams(nil, nil)
	params.TurnID = pulid.MustNew("atrn_")
	params.Delegations = []agentruntimeports.DelegatedRun{{
		Definition: &agentdefinition.Definition{
			ID:   pulid.MustNew("agd_"),
			Name: "Shipment Desk",
		},
		CallID: "call_look",
		Input:  "Where is PRO-1001?",
		Model:  "delegate-model",
	}}

	persisted, err := svc.persistProposals(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, persisted)
	assert.Empty(t, proposals.created)
	require.Len(t, runs.created, 1, "the parent filed nothing and opens no run")
	run := runs.created[0]
	assert.Equal(t, params.Delegations[0].Definition.ID, run.AgentDefinitionID)
	assert.Equal(t, agent.RunStatusCompleted, run.Status)
	assert.Equal(t, agent.RunOwnerAssistantTurn, run.ParentOwnerKind)
	assert.Equal(t, params.TurnID, run.ParentOwnerID)
	assert.Equal(t, "call_look", run.DelegateCallID)
	assert.Equal(t, "delegate-model", run.ModelIdentifier)
}

type threadPlans struct {
	plans []*agentplan.AgentPlan
}

func (s *threadPlans) ListByThread(
	context.Context,
	agentports.ListAgentPlansByThreadRequest,
) ([]*agentplan.AgentPlan, error) {
	return s.plans, nil
}

// A plan that carries a step of a delegate in shadow mode arrives on hold,
// though the run it hangs on is live: approving it would run the write the
// switch withholds.
func TestListThreadPlans_HoldsAPlanWithAStepFromAnAgentInShadowMode(t *testing.T) {
	t.Parallel()

	delegate := &agentdefinition.Definition{
		ID:         pulid.MustNew("agdef_"),
		Name:       "Shipment Desk",
		ShadowMode: true,
	}
	parentRun := &agent.AgentRun{ID: pulid.MustNew("arun_")}
	delegateRun := &agent.AgentRun{ID: pulid.MustNew("arun_"), AgentDefinitionID: delegate.ID}
	runs := &stubRunRepo{byID: map[pulid.ID]*agent.AgentRun{
		parentRun.ID:   parentRun,
		delegateRun.ID: delegateRun,
	}}
	plan := &agentplan.AgentPlan{
		ID:        pulid.MustNew("apl_"),
		RunID:     parentRun.ID,
		Title:     "Dispatch: 2 changes",
		Status:    agentplan.PlanStatusPending,
		StepCount: 2,
	}
	planID := plan.ID
	proposals := &stubProposalRepo{byThread: []*agent.AgentProposal{
		{ID: pulid.MustNew("ap_"), RunID: parentRun.ID, PlanID: &planID, PlanStep: 1},
		{ID: pulid.MustNew("ap_"), RunID: delegateRun.ID, PlanID: &planID, PlanStep: 2},
	}}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	svc := newProposalServiceBehind(runs, proposals,
		&stubConversationRepo{thread: thread}, shadowSwitches{definition: delegate})
	svc.plans = &threadPlans{plans: []*agentplan.AgentPlan{plan}}

	listed, err := svc.ListThreadPlans(
		t.Context(),
		conversationports.GetThreadRequest{ID: thread.ID},
	)
	require.NoError(t, err)

	require.Len(t, listed, 1)
	require.NotNil(t, listed[0].Hold, "the delegate's switch holds the whole plan")
	assert.Equal(t, "Shipment Desk", listed[0].Hold.AgentName)
}

func (s *threadPlans) ExpirePendingByThread(
	context.Context,
	agentports.ExpireAgentPlansByThreadRequest,
) (int, error) {
	return 0, nil
}
