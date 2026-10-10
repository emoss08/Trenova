package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type capturingPlanStore struct {
	created []*agent.AgentPlan
}

func (s *capturingPlanStore) Create(
	_ context.Context,
	plan *agent.AgentPlan,
) (*agent.AgentPlan, error) {
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
	opened   []*serviceports.RunRequest
}

func (fx *handOffEffects) Complete(
	_ *agentruntime.Turn,
	req *serviceports.ChatCompletionRequest,
) (agentruntime.ModelReply, error) {
	return fx.svc.runtime.StreamCompletion(fx.ctx, req, fx.Emit)
}

func (fx *handOffEffects) Dispatch(
	t *agentruntime.Turn,
	call agentruntime.DispatchCall,
) agentruntime.ToolOutcome {
	return fx.svc.runtime.DispatchStep(fx.ctx, t.Request(), call)
}

func (*handOffEffects) Find(*agentruntime.Turn, map[string]any) agentruntime.FindAnswer {
	return agentruntime.FindAnswer{}
}

func (*handOffEffects) Emit(serviceports.StreamEvent) {}

func (fx *handOffEffects) Observe(
	_ *agentruntime.Turn,
	call *serviceports.ToolCall,
	outcome agentruntime.ToolOutcome,
) agentruntime.ToolOutcome {
	return fx.svc.runtime.ObserveCall(nil, call, outcome)
}

func (*handOffEffects) NewCallID() string { return agentruntime.NewCallID() }

func (fx *handOffEffects) Delegate(
	t *agentruntime.Turn,
	call agentruntime.DelegateCall,
) agentruntime.DelegateRun {
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

	return agentruntime.DelegateRun{Definition: opened.Request.Definition, Result: result}
}

func (*handOffEffects) Supports(string) bool { return true }

func (*handOffEffects) Interject(*agentruntime.Turn, *agentruntime.WorldCheck) agentruntime.Interjections {
	return agentruntime.Interjections{}
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

/*
The owner's transcript: "void and recreate" became two unordered proposals
with separate approvals. The Dispatch desk proposes voiding the old shipment
and hands the Shipment Desk the copy, naming the shipment rather than
retyping it. The Shipment Desk proposes duplicate_shipment from the id it was
handed, and the turn files both as one plan, voided first, under one
approval, each write still the proposal of the agent that filed it.
*/
func TestHandOff_VoidAndRecreateIsOneTwoStepPlanInTheOrderAsked(t *testing.T) {
	t.Parallel()

	f := newDelegateFixture()
	f.parent.ToolNames = []string{"void_shipment"}
	f.delegate.ToolNames = []string{"duplicate_shipment"}
	void := &agentruntimetest.StubActionTool{ToolName: "void_shipment", Tier: agent.TierPropose}
	duplicate := &agentruntimetest.StubActionTool{
		ToolName: "duplicate_shipment",
		Tier:     agent.TierPropose,
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		{ToolCalls: []serviceports.ToolCall{{
			ID: "call_void", Name: "void_shipment",
			Arguments: map[string]any{"shipmentId": "shp_old", "reason": "Wrong dates"},
		}}},
		{ToolCalls: []serviceports.ToolCall{{
			ID: "call_hand", Name: "delegate_task",
			Arguments: map[string]any{
				"agentId": f.delegate.ID.String(),
				"task":    "Copy this shipment with pickup moved to next Monday; return its pro.",
				"records": []any{map[string]any{"entityType": "shipment", "id": "shp_old"}},
			},
		}}},
		{ToolCalls: []serviceports.ToolCall{{
			ID: "call_copy", Name: "duplicate_shipment",
			Arguments: map[string]any{"shipmentId": "shp_old"},
		}}},
		{Text: "I proposed the copy of shp_old."},
		{Text: "The void and the copy wait on your approval as one plan."},
	}}
	f.svc.runtime = agentruntime.New(agentruntime.Params{
		Logger:     zap.NewNop(),
		Completion: completion,
		QueryTools: &stubQueryRegistry{},
		ActionTools: &stubActionRegistry{
			Tools: []serviceports.AgentTool{void, duplicate},
		},
		Permissions: &agentruntimetest.StubPermissions{},
	})
	runs, proposals, plans := &stubRunRepo{}, &stubProposalRepo{}, &capturingPlanStore{}
	f.svc.recorder = proposalrecorder.NewWithStores(zap.NewNop(), runs, proposals).
		WithPlans(plans)
	f.svc.shadow = agentshadow.New(agentshadow.Params{
		Control:     stubControlRepo{},
		Runs:        runs,
		Definitions: stubDefinitionRepo{},
	})

	threadID := pulid.MustNew("athr_")
	req := &serviceports.RunRequest{
		Definition: f.parent,
		Actor:      f.request.Actor,
		Context: agentdefinition.RuntimeContext{
			Trigger: agent.RunTriggerChat,
			Delegates: []agentdefinition.RuntimeDelegate{{
				ID:    f.delegate.ID,
				Name:  f.delegate.Name,
				Tools: f.delegate.ToolNames,
			}},
		},
		Input:    "Void SEED-DET-009 and recreate it for next Monday.",
		ThreadID: threadID,
		StepOwner: serviceports.RunStepOwner{
			Kind: serviceports.RunStepOwnerAssistantTurn,
			ID:   pulid.MustNew("atrn_"),
		},
	}
	fx := &handOffEffects{
		ctx: t.Context(), t: t, svc: f.svc, parent: f.parent, threadID: threadID,
	}

	result, err := f.svc.runtime.Drive(f.svc.runtime.OpenTurn(t.Context(), req), fx)
	require.NoError(t, err)

	require.Len(t, fx.opened, 1)
	assert.Contains(t, fx.opened[0].Input, "shipment shp_old",
		"the Shipment Desk was handed the record, not a retyped copy")
	require.Len(t, result.Actions, 1)
	require.Len(t, result.Delegations, 1)
	require.Len(t, result.Delegations[0].Actions, 1)
	assert.Zero(t, void.Calls+duplicate.Calls, "nothing ran before the person approved")

	saved := savedWithIDs(result.Messages)
	persisted, err := f.svc.persistProposals(t.Context(), persistProposalsParams{
		TurnID:      req.StepOwner.ID,
		Definition:  f.parent,
		Thread:      &conversation.Thread{ID: threadID},
		Actor:       req.Actor,
		Saved:       saved,
		Actions:     result.Actions,
		Model:       result.Model,
		Input:       req.Input,
		Taint:       result.Taint,
		Delegations: result.Delegations,
	})
	require.NoError(t, err)

	require.Len(t, plans.created, 1, "one decision for the whole request")
	plan := plans.created[0]
	assert.Equal(t, 2, plan.StepCount)
	require.Len(t, runs.created, 2)
	parentRun, delegateRun := runs.created[0], runs.created[1]
	assert.Equal(t, plan.RunID, parentRun.ID)
	assert.Equal(t, f.delegate.ID, delegateRun.AgentDefinitionID)
	assert.Equal(t, callNamed(t, saved, "delegate_task"), delegateRun.DelegateCallID)

	require.Len(t, proposals.created, 2)
	steps := map[string]*agent.AgentProposal{}
	for _, proposal := range proposals.created {
		steps[proposal.ToolName] = proposal
		require.NotNil(t, proposal.PlanID)
		assert.Equal(t, plan.ID, *proposal.PlanID)
	}
	assert.Equal(t, 1, steps["void_shipment"].PlanStep, "the void runs first, as asked")
	assert.Equal(t, parentRun.ID, steps["void_shipment"].RunID)
	assert.Equal(t, 2, steps["duplicate_shipment"].PlanStep)
	assert.Equal(t, delegateRun.ID, steps["duplicate_shipment"].RunID)

	byTool := map[string]serviceports.AssistantProposal{}
	for _, proposal := range persisted {
		byTool[proposal.ToolName] = proposal
	}
	assert.Equal(t, f.parent.Name, byTool["void_shipment"].AgentName)
	assert.Equal(t, f.delegate.Name, byTool["duplicate_shipment"].AgentName,
		"the card still says which agent proposed it")
}

// #628: a hand-off that only read leaves the delegate's run, so the work it
// did is in AI Control after a reload.
func TestPersistProposals_AReadOnlyHandOffLeavesItsRun(t *testing.T) {
	t.Parallel()

	runs := &stubRunRepo{}
	proposals := &stubProposalRepo{}
	svc := newProposalService(runs, proposals, &stubConversationRepo{})
	params := proposalTestParams(nil, nil)
	params.TurnID = pulid.MustNew("atrn_")
	params.Delegations = []serviceports.DelegatedRun{{
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
	plans []*agent.AgentPlan
}

func (s *threadPlans) ListByThread(
	context.Context,
	repositories.ListAgentPlansByThreadRequest,
) ([]*agent.AgentPlan, error) {
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
	plan := &agent.AgentPlan{
		ID:        pulid.MustNew("apl_"),
		RunID:     parentRun.ID,
		Title:     "Dispatch: 2 changes",
		Status:    agent.PlanStatusPending,
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
	svc.plans = &threadPlans{plans: []*agent.AgentPlan{plan}}

	listed, err := svc.ListThreadPlans(t.Context(), repositories.GetThreadRequest{ID: thread.ID})
	require.NoError(t, err)

	require.Len(t, listed, 1)
	require.NotNil(t, listed[0].Hold, "the delegate's switch holds the whole plan")
	assert.Equal(t, "Shipment Desk", listed[0].Hold.AgentName)
}

func (s *threadPlans) ExpirePendingByThread(
	context.Context,
	repositories.ExpireAgentPlansByThreadRequest,
) (int, error) {
	return 0, nil
}
