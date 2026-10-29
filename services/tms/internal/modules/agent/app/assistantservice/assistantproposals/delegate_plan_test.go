package assistantproposals

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type threadPlans struct {
	plans []*agentplan.AgentPlan
}

func (s *threadPlans) ListByThread(
	context.Context,
	agentplanports.ListAgentPlansByThreadRequest,
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
	runs := &assistanttest.StubRunRepo{ByID: map[pulid.ID]*agent.AgentRun{
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
	proposals := &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{
		{ID: pulid.MustNew("ap_"), RunID: parentRun.ID, PlanID: &planID, PlanStep: 1},
		{ID: pulid.MustNew("ap_"), RunID: delegateRun.ID, PlanID: &planID, PlanStep: 2},
	}}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	svc := newProposalReadsBehind(
		runs,
		proposals,
		&assistanttest.StubConversationRepo{
			Thread: thread,
		},
		assistanttest.ShadowSwitches{Definition: delegate},
	)
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
	agentplanports.ExpireAgentPlansByThreadRequest,
) (int, error) {
	return 0, nil
}
