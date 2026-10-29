package billingtools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTransferredDuringPreview = errors.New("a preview transferred shipments")

type fakeTransferPlanner struct {
	plan       *shipmentbillingports.BillingTransferPlan
	planned    *shipmentbillingports.PlanBillingTransfersRequest
	response   *shipmentbillingports.BulkTransferToBillingResponse
	requested  *shipmentbillingports.BulkTransferShipmentToBillingRequest
	preview    bool
	candidates *shipmentbillingports.BillingTransferCandidateIDsResponse
	listed     *shipmentbillingports.ListBillingTransferCandidateIDsRequest
}

func (f *fakeTransferPlanner) ListBillingTransferCandidateIDs(
	_ context.Context,
	req *shipmentbillingports.ListBillingTransferCandidateIDsRequest,
) (*shipmentbillingports.BillingTransferCandidateIDsResponse, error) {
	f.listed = req
	if f.candidates == nil {
		return &shipmentbillingports.BillingTransferCandidateIDsResponse{}, nil
	}

	return f.candidates, nil
}

func (f *fakeTransferPlanner) PlanBillingTransfers(
	_ context.Context,
	req *shipmentbillingports.PlanBillingTransfersRequest,
) (*shipmentbillingports.BillingTransferPlan, error) {
	f.planned = req

	return f.plan, nil
}

func (f *fakeTransferPlanner) BulkTransferToBilling(
	_ context.Context,
	req *shipmentbillingports.BulkTransferShipmentToBillingRequest,
	_ *permissionports.RequestActor,
) (*shipmentbillingports.BulkTransferToBillingResponse, error) {
	if f.preview {
		return nil, errTransferredDuringPreview
	}
	f.requested = req

	return f.response, nil
}

type fakeRunStarter struct {
	started *billingtransferservice.StartRunRequest
	err     error
}

func (f *fakeRunStarter) Start(
	_ context.Context,
	req *billingtransferservice.StartRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.started = req

	return &billingtransfer.BillingTransferRun{ID: pulid.MustNew("btr_")}, nil
}

func shipmentIDs(n int) []any {
	ids := make([]any, 0, n)
	for range n {
		ids = append(ids, pulid.MustNew("shp_").String())
	}

	return ids
}

func TestTransferToBilling_PreviewSaysWhatHappensToEachShipment(t *testing.T) {
	t.Parallel()

	ready, completed, blocked := pulid.MustNew("shp_"), pulid.MustNew("shp_"), pulid.MustNew("shp_")
	planner := &fakeTransferPlanner{preview: true, plan: &shipmentbillingports.BillingTransferPlan{
		Transfer: 2,
		Refused:  1,
		Decisions: []shipmentbillingports.BillingTransferDecision{
			{
				ShipmentID: ready, ProNumber: "PRO-1", Status: shipment.StatusReadyToInvoice,
				Outcome: shipmentbillingports.BillingTransferOutcomeTransfer, AutoApprove: true,
				Version: 4,
			},
			{
				ShipmentID: completed, ProNumber: "PRO-2", Status: shipment.StatusCompleted,
				Outcome: shipmentbillingports.BillingTransferOutcomeMarkReadyAndTransfer,
			},
			{
				ShipmentID: blocked, ProNumber: "PRO-3", Status: shipment.StatusReadyToInvoice,
				Outcome:     shipmentbillingports.BillingTransferOutcomeRefused,
				FailureCode: shipmentbillingports.BillingTransferFailureRequirementsUnmet,
				Reason:      "Shipment billing requirements must be resolved before transfer",
				MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{
					{DocumentTypeName: "Proof of Delivery"},
				},
			},
		},
	}}
	tool := NewTransferToBillingTool(planner, &fakeRunStarter{}).(*TransferToBillingTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamShipmentIDs: []any{
			ready.String(), completed.String(), blocked.String(), ready.String(),
		},
		paramMarkCompletedReady: true,
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, []pulid.ID{ready, completed, blocked}, planner.planned.ShipmentIDs,
		"each shipment is checked once")
	assert.True(t, planner.planned.MarkCompletedReadyToInvoice)
	assert.Contains(t, preview.Summary, "3 shipments")
	assert.Contains(t, preview.Summary, "2 would transfer, 1 would be refused")
	assert.Empty(t, preview.Warnings)
	require.Len(t, preview.Changes, 3)

	refused := preview.Changes[0]
	assert.Equal(t, blocked, refused.EntityID, "a refusal is listed first")
	assert.Equal(t, permission.ResourceShipment, refused.Resource)
	assert.Contains(
		t,
		agenttoolstest.FieldByPath(t, &refused, "outcome").After,
		"RequirementsUnmet",
	)
	assert.Equal(
		t,
		"Proof of Delivery",
		agenttoolstest.FieldByPath(t, &refused, "missingDocuments").After,
	)

	queued := preview.Changes[1]
	assert.Equal(t, ready, queued.EntityID)
	assert.Equal(t, "PRO-1", queued.Label)
	assert.Equal(
		t,
		"ReadyForReview",
		agenttoolstest.FieldByPath(t, &queued, "billingTransferStatus").After,
	)
	assert.Contains(t, agenttoolstest.FieldByPath(t, &queued, "outcome").After, "auto-approve")
	require.NotNil(t, queued.Version)
	assert.Equal(t, int64(4), *queued.Version)

	marked := preview.Changes[2]
	assert.Equal(t, "Completed", agenttoolstest.FieldByPath(t, &marked, "status").Before)
	assert.Equal(t, "ReadyToInvoice", agenttoolstest.FieldByPath(t, &marked, "status").After)
}

func TestTransferToBilling_PreviewWarnsWhenNothingWouldGo(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	planner := &fakeTransferPlanner{preview: true, plan: &shipmentbillingports.BillingTransferPlan{
		Returned: 1,
		Decisions: []shipmentbillingports.BillingTransferDecision{{
			ShipmentID:  id,
			Outcome:     shipmentbillingports.BillingTransferOutcomeReturnToOperations,
			FailureCode: shipmentbillingports.BillingTransferFailureReturnToOperations,
			Reason:      "Shipment must be corrected in operations",
		}},
	}}
	tool := NewTransferToBillingTool(planner, &fakeRunStarter{}).(*TransferToBillingTool)

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamShipmentIDs: []any{id.String()},
	}))
	require.NoError(t, err)

	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	assert.Contains(t, preview.Summary, "1 would stay with operations")
}

// A transfer the synchronous path takes runs in the call, with the same
// shipments, bill type and choice about completed ones the preview checked.
func TestTransferToBilling_TransfersASmallSetInTheCall(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	planner := &fakeTransferPlanner{response: &shipmentbillingports.BulkTransferToBillingResponse{
		TotalCount:   2,
		SuccessCount: 1,
		Results: []shipmentbillingports.BulkTransferToBillingResult{
			{ShipmentID: first, ProNumber: "PRO-1", Success: true},
			{
				ShipmentID:  second,
				ProNumber:   "PRO-2",
				FailureCode: shipmentbillingports.BillingTransferFailureRateValidation,
			},
		},
	}}
	runs := &fakeRunStarter{}
	tool := NewTransferToBillingTool(planner, runs).(*TransferToBillingTool)

	result, err := tool.ExecuteWithResult(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamShipmentIDs: []any{first.String(), second.String()},
		toolparams.ParamBillType:    "DebitMemo",
	}))
	require.NoError(t, err)

	assert.Equal(t, []pulid.ID{first, second}, planner.requested.ShipmentIDs)
	assert.Equal(t, billingqueue.BillTypeDebitMemo, planner.requested.BillType)
	assert.False(t, planner.requested.MarkCompletedReadyToInvoice)
	assert.Nil(t, runs.started)
	assert.Equal(t, "1 of 2 transferred; refused: PRO-2 RateValidation", result.Name)
}

// More than the synchronous transfer takes is handed to the background run
// the dialog uses, as the person who asked, and the result names the run.
func TestTransferToBilling_HandsALargeSetToABackgroundRun(t *testing.T) {
	t.Parallel()

	planner := &fakeTransferPlanner{}
	runs := &fakeRunStarter{}
	tool := NewTransferToBillingTool(planner, runs).(*TransferToBillingTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamShipmentIDs: shipmentIDs(
			shipmentbillingports.MaxBulkTransferToBillingShipments + 1,
		),
	})

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, runs.started)
	assert.Nil(t, planner.requested, "the synchronous transfer is not used")
	assert.Equal(t, billingtransfer.RunScopeSelected, runs.started.Scope)
	assert.Len(
		t,
		runs.started.ShipmentIDs,
		shipmentbillingports.MaxBulkTransferToBillingShipments+1,
	)
	assert.Equal(t, params.Actor.UserID, runs.started.TenantInfo.UserID)
	assert.Equal(t, params.OrganizationID, runs.started.TenantInfo.OrgID)
	assert.NotEmpty(t, result.IDs[resultRunID])
}

// One background transfer runs per person at a time; the refusal reaches the
// caller rather than a second run starting.
func TestTransferToBilling_ReportsARunAlreadyGoing(t *testing.T) {
	t.Parallel()

	busy := errors.New("You already have a transfer running")
	tool := NewTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{err: busy})

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamShipmentIDs: shipmentIDs(
			shipmentbillingports.MaxBulkTransferToBillingShipments + 1,
		),
	}))
	require.ErrorIs(t, err, busy)
}

func TestTransferToBilling_PreviewOfALargeSetChecksTheFirstHundred(t *testing.T) {
	t.Parallel()

	planner := &fakeTransferPlanner{
		preview: true,
		plan:    &shipmentbillingports.BillingTransferPlan{},
	}
	tool := NewTransferToBillingTool(planner, &fakeRunStarter{}).(*TransferToBillingTool)

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamShipmentIDs: shipmentIDs(250),
	}))
	require.NoError(t, err)

	assert.Len(
		t,
		planner.planned.ShipmentIDs,
		shipmentbillingports.MaxBulkTransferToBillingShipments,
	)
	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "background transfer of 250 shipments")
}

func TestTransferToBilling_RefusesArgumentsItCannotRun(t *testing.T) {
	t.Parallel()

	tool := NewTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{}).(*TransferToBillingTool)

	for name, params := range map[string]map[string]any{
		"no shipments":  {toolparams.ParamShipmentIDs: []any{}},
		"not an id":     {toolparams.ParamShipmentIDs: []any{"PRO-1"}},
		"bad bill type": {toolparams.ParamShipmentIDs: shipmentIDs(1), toolparams.ParamBillType: "Invoices"},
		"too many": {
			toolparams.ParamShipmentIDs: shipmentIDs(shipmentbillingports.MaxBillingTransferCandidateIDs + 1),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(params)))
		})
	}
}

// The shipments are a record subset: the approval form lists them and lets a
// person untick some, and the policy keeps a transfer a proposal until the
// agent earns more.
func TestTransferToBilling_DeclaresItsShipmentsASubsetAPersonMayNarrow(t *testing.T) {
	t.Parallel()

	tool := NewTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{})

	proposed := shipmentIDs(2)
	fields := agenttoolports.ProposalFields(
		tool,
		map[string]any{toolparams.ParamShipmentIDs: proposed},
	)
	var subset *toolschema.Field
	for idx := range fields {
		if fields[idx].Name == toolparams.ParamShipmentIDs {
			subset = &fields[idx]
			continue
		}
		assert.Nil(t, fields[idx].Choices, "only a subset field lists records")
	}
	require.NotNil(t, subset)
	assert.Equal(t, toolschema.KindRecordSubset, subset.Kind)
	assert.Equal(t, permission.ResourceShipment.String(), subset.Resource)
	assert.Equal(t, []toolschema.Choice{
		{ID: proposed[0].(string), Label: proposed[0].(string)},
		{ID: proposed[1].(string), Label: proposed[1].(string)},
	}, subset.Choices, "every shipment proposed is listed for the approver to untick")
	assert.LessOrEqual(
		t,
		shipmentbillingports.MaxBillingTransferCandidateIDs,
		toolschema.MaxSubsetChoices,
		"a transfer lists every shipment it may hold",
	)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipment, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
}
