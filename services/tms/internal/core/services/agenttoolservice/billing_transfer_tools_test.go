package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTransferredDuringPreview = errors.New("a preview transferred shipments")

type fakeTransferPlanner struct {
	plan      *serviceports.BillingTransferPlan
	planned   *serviceports.PlanBillingTransfersRequest
	response  *serviceports.BulkTransferToBillingResponse
	requested *serviceports.BulkTransferShipmentToBillingRequest
	preview   bool
}

func (f *fakeTransferPlanner) PlanBillingTransfers(
	_ context.Context,
	req *serviceports.PlanBillingTransfersRequest,
) (*serviceports.BillingTransferPlan, error) {
	f.planned = req

	return f.plan, nil
}

func (f *fakeTransferPlanner) BulkTransferToBilling(
	_ context.Context,
	req *serviceports.BulkTransferShipmentToBillingRequest,
	_ *serviceports.RequestActor,
) (*serviceports.BulkTransferToBillingResponse, error) {
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
	planner := &fakeTransferPlanner{preview: true, plan: &serviceports.BillingTransferPlan{
		Transfer: 2,
		Refused:  1,
		Decisions: []serviceports.BillingTransferDecision{
			{
				ShipmentID: ready, ProNumber: "PRO-1", Status: shipment.StatusReadyToInvoice,
				Outcome: serviceports.BillingTransferOutcomeTransfer, AutoApprove: true,
				Version: 4,
			},
			{
				ShipmentID: completed, ProNumber: "PRO-2", Status: shipment.StatusCompleted,
				Outcome: serviceports.BillingTransferOutcomeMarkReadyAndTransfer,
			},
			{
				ShipmentID: blocked, ProNumber: "PRO-3", Status: shipment.StatusReadyToInvoice,
				Outcome:     serviceports.BillingTransferOutcomeRefused,
				FailureCode: serviceports.BillingTransferFailureRequirementsUnmet,
				Reason:      "Shipment billing requirements must be resolved before transfer",
				MissingRequirements: []serviceports.ShipmentBillingRequirement{
					{DocumentTypeName: "Proof of Delivery"},
				},
			},
		},
	}}
	tool := newTransferToBillingTool(planner, &fakeRunStarter{}).(*transferToBillingTool)
	params := executeParams(map[string]any{
		paramShipmentIDs: []any{
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
	assert.Contains(t, fieldByPath(t, &refused, "outcome").After, "RequirementsUnmet")
	assert.Equal(t, "Proof of Delivery", fieldByPath(t, &refused, "missingDocuments").After)

	queued := preview.Changes[1]
	assert.Equal(t, ready, queued.EntityID)
	assert.Equal(t, "PRO-1", queued.Label)
	assert.Equal(t, "ReadyForReview", fieldByPath(t, &queued, "billingTransferStatus").After)
	assert.Contains(t, fieldByPath(t, &queued, "outcome").After, "auto-approve")
	require.NotNil(t, queued.Version)
	assert.Equal(t, int64(4), *queued.Version)

	marked := preview.Changes[2]
	assert.Equal(t, "Completed", fieldByPath(t, &marked, "status").Before)
	assert.Equal(t, "ReadyToInvoice", fieldByPath(t, &marked, "status").After)
}

func TestTransferToBilling_PreviewWarnsWhenNothingWouldGo(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	planner := &fakeTransferPlanner{preview: true, plan: &serviceports.BillingTransferPlan{
		Returned: 1,
		Decisions: []serviceports.BillingTransferDecision{{
			ShipmentID:  id,
			Outcome:     serviceports.BillingTransferOutcomeReturnToOperations,
			FailureCode: serviceports.BillingTransferFailureReturnToOperations,
			Reason:      "Shipment must be corrected in operations",
		}},
	}}
	tool := newTransferToBillingTool(planner, &fakeRunStarter{}).(*transferToBillingTool)

	preview, err := tool.Preview(t.Context(), executeParams(map[string]any{
		paramShipmentIDs: []any{id.String()},
	}))
	require.NoError(t, err)

	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Contains(t, preview.Summary, "1 would stay with operations")
}

// A transfer the synchronous path takes runs in the call, with the same
// shipments, bill type and choice about completed ones the preview checked.
func TestTransferToBilling_TransfersASmallSetInTheCall(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	planner := &fakeTransferPlanner{response: &serviceports.BulkTransferToBillingResponse{
		TotalCount:   2,
		SuccessCount: 1,
		Results: []serviceports.BulkTransferToBillingResult{
			{ShipmentID: first, ProNumber: "PRO-1", Success: true},
			{
				ShipmentID:  second,
				ProNumber:   "PRO-2",
				FailureCode: serviceports.BillingTransferFailureRateValidation,
			},
		},
	}}
	runs := &fakeRunStarter{}
	tool := newTransferToBillingTool(planner, runs).(*transferToBillingTool)

	result, err := tool.ExecuteWithResult(t.Context(), executeParams(map[string]any{
		paramShipmentIDs: []any{first.String(), second.String()},
		paramBillType:    "DebitMemo",
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
	tool := newTransferToBillingTool(planner, runs).(*transferToBillingTool)
	params := executeParams(map[string]any{
		paramShipmentIDs: shipmentIDs(serviceports.MaxBulkTransferToBillingShipments + 1),
	})

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, runs.started)
	assert.Nil(t, planner.requested, "the synchronous transfer is not used")
	assert.Equal(t, billingtransfer.RunScopeSelected, runs.started.Scope)
	assert.Len(t, runs.started.ShipmentIDs, serviceports.MaxBulkTransferToBillingShipments+1)
	assert.Equal(t, params.Actor.UserID, runs.started.TenantInfo.UserID)
	assert.Equal(t, params.OrganizationID, runs.started.TenantInfo.OrgID)
	assert.NotEmpty(t, result.IDs[resultRunID])
}

// One background transfer runs per person at a time; the refusal reaches the
// caller rather than a second run starting.
func TestTransferToBilling_ReportsARunAlreadyGoing(t *testing.T) {
	t.Parallel()

	busy := errors.New("You already have a transfer running")
	tool := newTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{err: busy})

	err := tool.Execute(t.Context(), executeParams(map[string]any{
		paramShipmentIDs: shipmentIDs(serviceports.MaxBulkTransferToBillingShipments + 1),
	}))
	require.ErrorIs(t, err, busy)
}

func TestTransferToBilling_PreviewOfALargeSetChecksTheFirstHundred(t *testing.T) {
	t.Parallel()

	planner := &fakeTransferPlanner{preview: true, plan: &serviceports.BillingTransferPlan{}}
	tool := newTransferToBillingTool(planner, &fakeRunStarter{}).(*transferToBillingTool)

	preview, err := tool.Preview(t.Context(), executeParams(map[string]any{
		paramShipmentIDs: shipmentIDs(250),
	}))
	require.NoError(t, err)

	assert.Len(t, planner.planned.ShipmentIDs, serviceports.MaxBulkTransferToBillingShipments)
	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "background transfer of 250 shipments")
}

func TestTransferToBilling_RefusesArgumentsItCannotRun(t *testing.T) {
	t.Parallel()

	tool := newTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{}).(*transferToBillingTool)

	for name, params := range map[string]map[string]any{
		"no shipments":  {paramShipmentIDs: []any{}},
		"not an id":     {paramShipmentIDs: []any{"PRO-1"}},
		"bad bill type": {paramShipmentIDs: shipmentIDs(1), paramBillType: "Invoices"},
		"too many": {
			paramShipmentIDs: shipmentIDs(serviceports.MaxBillingTransferCandidateIDs + 1),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, tool.Validate(t.Context(), executeParams(params)))
		})
	}
}

// The shipments are a record subset: the approval form lists them and lets a
// person untick some, and the policy keeps a transfer a proposal until the
// agent earns more.
func TestTransferToBilling_DeclaresItsShipmentsASubsetAPersonMayNarrow(t *testing.T) {
	t.Parallel()

	tool := newTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{})

	fields := serviceports.ProposalFields(tool, map[string]any{paramShipmentIDs: shipmentIDs(2)})
	var subset *toolschema.Field
	for idx := range fields {
		if fields[idx].Name == paramShipmentIDs {
			subset = &fields[idx]
		}
	}
	require.NotNil(t, subset)
	assert.Equal(t, toolschema.KindRecordSubset, subset.Kind)
	assert.Equal(t, permission.ResourceShipment.String(), subset.Resource)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipment, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
}
