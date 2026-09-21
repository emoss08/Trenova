package proposalrecorder

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingPlans struct {
	created []*agent.AgentPlan
}

func (s *capturingPlans) Create(
	_ context.Context,
	plan *agent.AgentPlan,
) (*agent.AgentPlan, error) {
	plan.ID = pulid.MustNew("apl_")
	s.created = append(s.created, plan)

	return plan, nil
}

func pendingAction(tool string) serviceports.PendingAction {
	return serviceports.PendingAction{
		ToolName:  tool,
		Arguments: map[string]any{},
		Rationale: "because " + tool,
		Tier:      agent.TierActWithApproval,
	}
}

func recordActions(
	t *testing.T,
	plans *capturingPlans,
	actions ...serviceports.PendingAction,
) *RecordResult {
	t.Helper()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	store := &capturingStore{}
	result, err := NewWithStores(nil, nil, store).WithPlans(plans).Record(t.Context(), &RecordRequest{
		Actor:      &serviceports.RequestActor{OrganizationID: orgID, BusinessUnitID: buID},
		Definition: &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Dispatch coverage"},
		Run: &agent.AgentRun{
			ID:             pulid.MustNew("arun_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Summary:        "Cover the two open moves",
		},
		Actions: actions,
		Evidence: func(serviceports.PendingAction, pulid.ID) []agent.EvidenceRef {
			return []agent.EvidenceRef{{Type: "message", ID: "amsg_1"}}
		},
	})
	require.NoError(t, err)

	return result
}

// Two or more pending writes in one run are the agent asking for a sequence;
// the plan keeps them in the order it asked and carries one expiry for all.
func TestRecord_GroupsSeveralPendingProposalsIntoAPlan(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	executed := pendingAction("add_shipment_comment")
	executed.Executed = true
	result := recordActions(
		t,
		plans,
		pendingAction("assign_move"),
		executed,
		pendingAction("notify_driver"),
	)

	require.NotNil(t, result.Plan)
	require.Len(t, plans.created, 1)
	plan := plans.created[0]
	assert.Equal(t, "Dispatch coverage: 2 changes", plan.Title)
	assert.Equal(t, "Cover the two open moves", plan.Summary)
	assert.Equal(t, 2, plan.StepCount, "a write that already ran is not a step anyone decides")
	assert.Equal(t, agent.PlanStatusPending, plan.Status)
	assert.Positive(t, plan.ExpiresAt)

	require.Len(t, result.Proposals, 3)
	assert.Equal(t, plan.ID, *result.Proposals[0].PlanID)
	assert.Equal(t, 1, result.Proposals[0].PlanStep)
	assert.Nil(t, result.Proposals[1].PlanID, "the executed write stands outside the plan")
	assert.Equal(t, 2, result.Proposals[2].PlanStep)
	assert.Equal(t, plan.ExpiresAt, result.Proposals[2].ExpiresAt)
}

func TestRecord_LeavesASingleProposalOnItsOwn(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	result := recordActions(t, plans, pendingAction("assign_move"))

	assert.Nil(t, result.Plan)
	assert.Empty(t, plans.created)
	require.Len(t, result.Proposals, 1)
	assert.Nil(t, result.Proposals[0].PlanID)
	assert.Equal(t, 0, result.Proposals[0].PlanStep)
}
