package proposalrecorder

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func actionOn(
	tool string,
	resource permission.Resource,
	id pulid.ID,
) agentruntimeports.PendingAction {
	action := pendingAction(tool)
	action.Target = &agentruntimeports.ProposalTarget{Resource: resource, ID: id}

	return action
}

func recordTriggered(
	t *testing.T,
	plans *capturingPlans,
	trigger agent.RunTrigger,
	actions ...agentruntimeports.PendingAction,
) *RecordResult {
	t.Helper()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	result, err := recorderOn(nil, &capturingStore{}).WithPlans(plans).
		Record(t.Context(), &RecordRequest{
			Actor: &permissionports.RequestActor{OrganizationID: orgID, BusinessUnitID: buID},
			Definition: &agentdefinition.Definition{
				ID:   pulid.MustNew("agdef_"),
				Name: "Credential desk",
			},
			Run: &agent.AgentRun{
				ID:             pulid.MustNew("arun_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Trigger:        trigger,
			},
			Actions:   actions,
			PlanTitle: "Credential renewals and Jane Doe dispatch hold",
			Evidence:  messageEvidence,
		})
	require.NoError(t, err)

	return result
}

func TestRecord_AnUnattendedRunFilesOnePlanForEachRecordItWorked(t *testing.T) {
	t.Parallel()

	jane, mike, sarah := pulid.MustNew("wrk_"), pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	plans := &capturingPlans{}
	result := recordTriggered(t, plans, agent.RunTriggerEvent,
		actionOn("worker_place_dispatch_hold", permission.ResourceWorker, jane),
		actionOn("worker_request_credential_renewal", permission.ResourceWorker, mike),
		actionOn("worker_request_credential_renewal", permission.ResourceWorker, jane),
		actionOn("worker_request_credential_renewal", permission.ResourceWorker, sarah),
	)

	assert.Nil(t, result.Plan)
	require.Len(t, plans.created, 1, "only the driver with two writes needs a plan")
	require.Len(t, result.Records, 1)
	plan := plans.created[0]
	assert.Equal(t, jane, plan.TargetID)
	assert.Equal(t, 2, plan.StepCount)
	assert.Equal(t, "Credential desk: 2 changes", plan.Title,
		"the name the agent gave the whole run is not put on one record's share of it")

	require.Len(t, result.Proposals, 4)
	assert.Equal(t, plan.ID, *result.Proposals[0].PlanID)
	assert.Equal(t, 1, result.Proposals[0].PlanStep)
	assert.Equal(t, plan.ID, *result.Proposals[2].PlanID)
	assert.Equal(t, 2, result.Proposals[2].PlanStep)
	assert.Nil(t, result.Proposals[1].PlanID, "another driver's one request stands alone")
	assert.Nil(t, result.Proposals[3].PlanID)
}

func TestRecord_OneSequenceStaysOnePlan(t *testing.T) {
	t.Parallel()

	jane, mike := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	shipment, invoice := pulid.MustNew("shp_"), pulid.MustNew("inv_")

	t.Run("a conversation's writes are one decision whatever they touch", func(t *testing.T) {
		t.Parallel()

		plans := &capturingPlans{}
		result := recordTriggered(t, plans, agent.RunTriggerChat,
			actionOn("worker_request_credential_renewal", permission.ResourceWorker, jane),
			actionOn("worker_request_credential_renewal", permission.ResourceWorker, mike),
		)

		require.NotNil(t, result.Plan)
		assert.Empty(t, result.Records)
		assert.Equal(t, "Credential renewals and Jane Doe dispatch hold", result.Plan.Title)
		assert.Equal(t, 2, result.Plan.StepCount)
	})

	t.Run("an unattended run across kinds of record is one sequence", func(t *testing.T) {
		t.Parallel()

		plans := &capturingPlans{}
		result := recordTriggered(t, plans, agent.RunTriggerEvent,
			actionOn("shipment_add_comment", permission.ResourceShipment, shipment),
			actionOn("invoice_add_note", permission.ResourceInvoice, invoice),
		)

		require.NotNil(t, result.Plan)
		assert.Empty(t, result.Records)
		assert.Equal(t, 2, result.Plan.StepCount)
	})

	t.Run("an unattended run with a write about no record is one sequence", func(t *testing.T) {
		t.Parallel()

		plans := &capturingPlans{}
		result := recordTriggered(t, plans, agent.RunTriggerScheduled,
			actionOn("worker_request_credential_renewal", permission.ResourceWorker, jane),
			actionOn("worker_request_credential_renewal", permission.ResourceWorker, mike),
			pendingAction("report_create"),
		)

		require.NotNil(t, result.Plan)
		assert.Empty(t, result.Records)
		assert.Equal(t, 3, result.Plan.StepCount)
	})
}
