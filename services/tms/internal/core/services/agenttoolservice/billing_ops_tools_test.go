package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReassigner struct {
	guard      *writeGuard
	reassigned *serviceports.ReassignChargeRequest
	canceled   *billingqueue.BillingQueueItem
}

func (f *fakeReassigner) ReassignCharge(
	_ context.Context,
	req *serviceports.ReassignChargeRequest,
	_ *serviceports.RequestActor,
) (*serviceports.ReassignChargeResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.reassigned = req

	return &serviceports.ReassignChargeResult{
		CreatedItemIDs:  []pulid.ID{pulid.MustNew("bqi_")},
		CanceledItemIDs: []pulid.ID{f.canceled.ID},
	}, nil
}

func (f *fakeReassigner) PreviewReassignCharge(
	_ context.Context,
	req *serviceports.ReassignChargeRequest,
	_ *serviceports.RequestActor,
) (*serviceports.ReassignChargePreview, error) {
	share := &shipment.PayerShare{
		PayerID:       req.Allocations[0].BillToCustomerID,
		FreightAmount: decimal.RequireFromString("900"),
		TotalAmount:   decimal.RequireFromString("900"),
	}

	return &serviceports.ReassignChargePreview{
		Item:        &billingqueue.BillingQueueItem{ID: req.ItemID},
		Shipment:    &shipment.Shipment{ProNumber: "PRO-4040"},
		Description: "Freight",
		Shares:      []*shipment.PayerShare{share},
		ToCreate:    []*shipment.PayerShare{share},
		ToCancel:    []*billingqueue.BillingQueueItem{f.canceled},
	}, nil
}

func TestReassignBillingCharge_ShowsTheItemsItOpensAndCancels(t *testing.T) {
	t.Parallel()

	billing := &fakeReassigner{
		guard: &writeGuard{},
		canceled: &billingqueue.BillingQueueItem{
			ID:      pulid.MustNew("bqi_"),
			Number:  "BQ-12",
			Status:  billingqueue.StatusInReview,
			Version: 2,
		},
	}
	tool := newReassignBillingChargeTool(billing)
	payer := pulid.MustNew("cus_")
	params := approvedParams(map[string]any{
		paramItemID:     pulid.MustNew("bqi_").String(),
		paramChargeKind: "Freight",
		paramAllocations: []any{map[string]any{
			paramBillToCustomer: payer.String(), paramAllocMethod: "Percent",
			paramAllocPercent: "100",
		}},
	})

	preview := previewWithoutWrites(t, billing.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)
	assert.Equal(t, string(billingqueue.StatusCanceled),
		fieldByPath(t, previewChange(t, preview, 1), fieldStatus).After)
	assert.Contains(t, preview.Summary, "PRO-4040")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, billing.reassigned)
	require.Len(t, billing.reassigned.Allocations, 1)
	assert.Equal(t, shipment.ChargeAllocationKindFreight,
		billing.reassigned.Allocations[0].ChargeKind)

	require.ErrorIs(t, tool.Execute(t.Context(), executeParams(params.Params)),
		ErrNeedsAPersonsApproval)
	policy := tool.Policy()
	assert.Equal(t, permission.ResourceBillingQueue, policy.Resource)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)

	for name, raw := range map[string]map[string]any{
		"an accessorial without its charge": {
			paramItemID: pulid.MustNew("bqi_").String(), paramChargeKind: "Accessorial",
			paramAllocations: []any{},
		},
		"no allocations": {
			paramItemID: pulid.MustNew("bqi_").String(), paramChargeKind: "Freight",
		},
		"an order charge": {
			paramItemID: pulid.MustNew("bqi_").String(), paramChargeKind: "OrderCharge",
			paramAllocations: []any{},
		},
	} {
		require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
			executeParams(raw)), name)
	}
}

type fakeTransferRuns struct {
	guard    *writeGuard
	refusal  error
	run      *billingtransfer.BillingTransferRun
	canceled bool
	retried  bool
}

func (f *fakeTransferRuns) Cancel(
	context.Context,
	*billingtransferservice.RunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.canceled = true

	return f.run, nil
}

func (f *fakeTransferRuns) PreviewCancel(
	_ context.Context,
	req *billingtransferservice.RunRequest,
) (*billingtransferservice.RunPreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	after := *f.run
	after.CancelRequestedByID = req.TenantInfo.UserID

	return &billingtransferservice.RunPreview{Before: f.run, After: &after}, nil
}

func (f *fakeTransferRuns) Retry(
	context.Context,
	*billingtransferservice.RunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.retried = true

	return &billingtransfer.BillingTransferRun{ID: pulid.MustNew("btr_")}, nil
}

func (f *fakeTransferRuns) PreviewRetry(
	context.Context,
	*billingtransferservice.RunRequest,
) (*billingtransferservice.RunPreview, error) {
	return &billingtransferservice.RunPreview{
		Before: f.run,
		After:  f.run,
		Created: &billingtransfer.BillingTransferRun{
			Status:      billingtransfer.RunStatusQueued,
			Scope:       billingtransfer.RunScopeRetry,
			SourceRunID: f.run.ID,
		},
	}, nil
}

func TestManageBillingTransferRun_StopsOrRetries(t *testing.T) {
	t.Parallel()

	runs := &fakeTransferRuns{
		guard: &writeGuard{},
		run: &billingtransfer.BillingTransferRun{
			ID:             pulid.MustNew("btr_"),
			TotalCount:     400,
			ProcessedCount: 120,
			RetryableCount: 3,
			SkippedCount:   1,
		},
	}
	tool := newManageBillingTransferRunTool(runs)
	stop := executeParams(map[string]any{
		resultRunID:    runs.run.ID.String(),
		paramRunAction: "Stop",
	})

	preview := previewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), stop)
	})
	assert.Contains(t, preview.Summary, "120 of 400")
	require.NoError(t, tool.Execute(t.Context(), stop))
	assert.True(t, runs.canceled)

	retry := executeParams(map[string]any{
		resultRunID:    runs.run.ID.String(),
		paramRunAction: "Retry",
	})
	retryPreview := previewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), retry)
	})
	assert.Contains(t, retryPreview.Summary, "4 shipments")
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, retryPreview, 0).Operation)
	require.NoError(t, tool.Execute(t.Context(), retry))
	assert.True(t, runs.retried)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipment, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)

	runs.refusal = errortypes.NewAuthorizationError("Only the person who started a transfer can stop it")
	refused, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), stop)
	require.NoError(t, err)
	requireWarning(t, refused, agent.PreviewWarningWouldFail)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{resultRunID: runs.run.ID.String(), paramRunAction: "Pause"})))
}
