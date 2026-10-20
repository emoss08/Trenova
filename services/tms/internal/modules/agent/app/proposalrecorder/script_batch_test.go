package proposalrecorder

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scriptStep = "script-step-1"

func scriptIntent(tool string, key string) agentports.PendingAction {
	action := pendingAction(tool)
	action.ScriptStepKey = key

	return action
}

func TestRecord_FilesAScriptsIntentsAsOneBatch(t *testing.T) {
	t.Parallel()

	const intents = 7
	actions := make([]agentports.PendingAction, 0, intents)
	for range intents {
		actions = append(actions, scriptIntent("invoice_post", scriptStep))
	}

	plans := &capturingPlans{}
	result := recordActions(t, plans, actions...)

	require.Len(t, plans.created, 1, "N intents are one proposal, never N")
	assert.Nil(t, result.Plan, "the batch is not the turn's own plan")
	require.Len(t, result.Batches, 1)
	batch := result.Batches[0]
	assert.Equal(t, scriptStep, batch.ScriptStepKey)
	assert.True(t, batch.FromScript())
	assert.Equal(t, intents, batch.StepCount)
	assert.Equal(t, "Dispatch coverage: 7 changes from a script", batch.Title)

	require.Len(t, result.Proposals, intents)
	for idx, proposal := range result.Proposals {
		assert.Equal(t, batch.ID, *proposal.PlanID)
		assert.Equal(t, idx+1, proposal.PlanStep, "steps keep the order the script prepared them")
		assert.Equal(t, agent.ProposalStatusPending, proposal.Status)
		assert.Nil(t, proposal.ExecutedAt, "an intent never ran")
		assert.True(t, proposal.ApprovalNeedsReview(), "a script's write always needs its preview")
	}
}

func TestRecord_KeepsEachScriptsBatchApartFromTheTurnsPlan(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	result := recordActions(
		t,
		plans,
		pendingAction("dispatch_assign_move"),
		scriptIntent("invoice_post", scriptStep),
		pendingAction("worker_send_message"),
		scriptIntent("invoice_post", scriptStep),
		scriptIntent("invoice_send", "script-step-2"),
		scriptIntent("invoice_send", "script-step-2"),
	)

	require.Len(t, plans.created, 3)
	require.NotNil(t, result.Plan)
	assert.Empty(t, result.Plan.ScriptStepKey)
	assert.Equal(t, 2, result.Plan.StepCount)
	require.Len(t, result.Batches, 2)
	assert.Equal(t, scriptStep, result.Batches[0].ScriptStepKey)
	assert.Equal(t, "script-step-2", result.Batches[1].ScriptStepKey)

	for _, proposal := range result.Proposals {
		switch proposal.ScriptStepKey {
		case "":
			assert.Equal(t, result.Plan.ID, *proposal.PlanID)
		case scriptStep:
			assert.Equal(t, result.Batches[0].ID, *proposal.PlanID)
		default:
			assert.Equal(t, result.Batches[1].ID, *proposal.PlanID)
		}
	}
}

func TestRecord_LeavesAScriptsSingleIntentAProposal(t *testing.T) {
	t.Parallel()

	internal := pendingAction("dispatch_assign_move")
	internal.Egress = agent.EgressInternal
	intent := scriptIntent("dispatch_assign_move", scriptStep)
	intent.Egress = agent.EgressInternal

	plans := &capturingPlans{}
	result := recordActions(t, plans, internal, intent)

	assert.Empty(t, plans.created, "one write of the turn and one of the script are two decisions")
	assert.Empty(t, result.Batches)
	require.Len(t, result.Proposals, 2)
	assert.Equal(t, scriptStep, result.Proposals[1].ScriptStepKey)
	assert.True(t, result.Proposals[1].ApprovalNeedsReview(), "an internal intent still needs its preview")
	assert.False(t, result.Proposals[0].ApprovalNeedsReview(), "the same write called by the model does not")
}

type scriptLedger struct {
	asked []agentports.GetAgentRunStepRequest
	step  *agent.AgentRunStep
}

func (l *scriptLedger) Get(
	_ context.Context,
	req *agentports.GetAgentRunStepRequest,
) (*agent.AgentRunStep, error) {
	l.asked = append(l.asked, *req)
	if l.step == nil {
		return nil, errors.New("not found")
	}

	return l.step, nil
}

func TestRecord_KeepsTheScriptABatchCameFromOnItsPlan(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	runID := pulid.MustNew("arun_")
	ledger := &scriptLedger{step: &agent.AgentRunStep{
		StepKey: scriptStep,
		Arguments: map[string]any{
			"title":   "Hold Range's uncovered loads",
			"purpose": "Hold every Range load out of Denver with nobody on it.",
			"code":    "for load in loads: trenova.shipment.place_hold(shipmentId=load)",
		},
	}}
	plans := &capturingPlans{}
	result, err := NewWithStores(nil, nil, &capturingStore{}).
		WithPlans(plans).
		WithScriptSteps(ledger).
		Record(t.Context(), &RecordRequest{
			Actor: &permissionports.RequestActor{OrganizationID: orgID, BusinessUnitID: buID},
			Run:   &agent.AgentRun{ID: runID, OrganizationID: orgID, BusinessUnitID: buID},
			Actions: []agentports.PendingAction{
				scriptIntent("shipment_place_hold", scriptStep),
				scriptIntent("shipment_place_hold", scriptStep),
			},
			Evidence: messageEvidence,
		})
	require.NoError(t, err)

	require.Len(t, result.Batches, 1)
	require.NotNil(t, result.Batches[0].Script)
	assert.Equal(t, "Hold Range's uncovered loads", result.Batches[0].Script.Title)
	assert.Contains(t, result.Batches[0].Script.Code, "place_hold")
	require.Len(t, ledger.asked, 1)
	assert.Equal(t, runID, ledger.asked[0].OwnerID, "a run's script is read from the run's ledger")
	assert.Equal(t, scriptStep, ledger.asked[0].StepKey)
}

func TestRecord_FilesABatchWhoseScriptCannotBeRead(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	result, err := NewWithStores(nil, nil, &capturingStore{}).
		WithPlans(plans).
		WithScriptSteps(&scriptLedger{}).
		Record(t.Context(), &RecordRequest{
			Actor: &permissionports.RequestActor{OrganizationID: orgID, BusinessUnitID: buID},
			Run:   &agent.AgentRun{ID: pulid.MustNew("arun_"), OrganizationID: orgID, BusinessUnitID: buID},
			Actions: []agentports.PendingAction{
				scriptIntent("shipment_place_hold", scriptStep),
				scriptIntent("shipment_place_hold", scriptStep),
			},
			Evidence: messageEvidence,
		})
	require.NoError(t, err)

	require.Len(t, result.Batches, 1, "the script is the batch's record, not its authority")
	assert.Nil(t, result.Batches[0].Script)
	assert.Equal(t, scriptStep, result.Batches[0].ScriptStepKey)
}
