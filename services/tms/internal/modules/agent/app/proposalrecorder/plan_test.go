package proposalrecorder

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingPlans struct {
	created []*agentplan.AgentPlan
}

func (s *capturingPlans) Create(
	_ context.Context,
	plan *agentplan.AgentPlan,
) (*agentplan.AgentPlan, error) {
	plan.ID = pulid.MustNew("apl_")
	s.created = append(s.created, plan)

	return plan, nil
}

func pendingAction(tool string) agentruntimeports.PendingAction {
	return agentruntimeports.PendingAction{
		ToolName:  tool,
		Arguments: map[string]any{},
		Rationale: "because " + tool,
		Tier:      agent.TierActWithApproval,
	}
}

func recordActions(
	t *testing.T,
	plans *capturingPlans,
	actions ...agentruntimeports.PendingAction,
) *RecordResult {
	t.Helper()

	return recordNamedActions(t, plans, "", actions...)
}

func recordNamedActions(
	t *testing.T,
	plans *capturingPlans,
	planTitle string,
	actions ...agentruntimeports.PendingAction,
) *RecordResult {
	t.Helper()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	store := &capturingStore{}
	result, err := NewWithStores(
		nil,
		nil,
		store,
	).WithPlans(plans).
		Record(t.Context(), &RecordRequest{
			Actor: &permissionports.RequestActor{OrganizationID: orgID, BusinessUnitID: buID},
			Definition: &agentdefinition.Definition{
				ID:   pulid.MustNew("agdef_"),
				Name: "Dispatch coverage",
			},
			Run: &agent.AgentRun{
				ID:             pulid.MustNew("arun_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Summary:        "Cover the two open moves",
			},
			Actions:   actions,
			PlanTitle: planTitle,
			Evidence:  messageEvidence,
		})
	require.NoError(t, err)

	return result
}

// Two or more pending writes in one run are the agent asking for a sequence;
// the plan keeps them in the order it asked and carries one expiry for all.
func TestRecord_GroupsSeveralPendingProposalsIntoAPlan(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	executed := pendingAction("shipment_add_comment")
	executed.Executed = true
	result := recordActions(
		t,
		plans,
		pendingAction("dispatch_assign_move"),
		executed,
		pendingAction("worker_send_message"),
	)

	require.NotNil(t, result.Plan)
	require.Len(t, plans.created, 1)
	plan := plans.created[0]
	assert.Equal(t, "Dispatch coverage: 2 changes", plan.Title)
	assert.Equal(t, "Cover the two open moves", plan.Summary)
	assert.Equal(t, 2, plan.StepCount, "a write that already ran is not a step anyone decides")
	assert.Equal(t, agentplan.PlanStatusPending, plan.Status)
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
	result := recordActions(t, plans, pendingAction("dispatch_assign_move"))

	assert.Nil(t, result.Plan)
	assert.Empty(t, plans.created)
	require.Len(t, result.Proposals, 1)
	assert.Nil(t, result.Proposals[0].PlanID)
	assert.Equal(t, 0, result.Proposals[0].PlanStep)
}

// A proposal the agent withdrew later in the same turn is recorded Superseded:
// never pending and never a plan step. With it gone, its replacement stands
// alone rather than in a plan that approved both.
func TestRecord_AWithdrawnProposalIsSupersededAndOutOfThePlan(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	withdrawn := pendingAction("shipment_create")
	withdrawn.Withdrawn = true
	result := recordActions(t, plans, withdrawn, pendingAction("shipment_create"))

	assert.Nil(t, result.Plan)
	assert.Empty(t, plans.created)
	require.Len(t, result.Proposals, 2)
	assert.Equal(t, agent.ProposalStatusSuperseded, result.Proposals[0].Status)
	assert.Nil(t, result.Proposals[0].PlanID)
	assert.Equal(t, agent.ProposalStatusPending, result.Proposals[1].Status)
}
