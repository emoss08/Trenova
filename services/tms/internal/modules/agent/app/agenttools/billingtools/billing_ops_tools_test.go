package billingtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReassigner struct {
	guard      *agenttoolstest.WriteGuard
	reassigned *billingqueueports.ReassignChargeRequest
	canceled   *billingqueue.BillingQueueItem
}

func (f *fakeReassigner) ReassignCharge(
	_ context.Context,
	req *billingqueueports.ReassignChargeRequest,
	_ *permissionports.RequestActor,
) (*billingqueueports.ReassignChargeResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.reassigned = req

	return &billingqueueports.ReassignChargeResult{
		CreatedItemIDs:  []pulid.ID{pulid.MustNew("bqi_")},
		CanceledItemIDs: []pulid.ID{f.canceled.ID},
	}, nil
}

func (f *fakeReassigner) PreviewReassignCharge(
	_ context.Context,
	req *billingqueueports.ReassignChargeRequest,
	_ *permissionports.RequestActor,
) (*billingqueueports.ReassignChargePreview, error) {
	share := &shipment.PayerShare{
		PayerID:       req.Allocations[0].BillToCustomerID,
		FreightAmount: decimal.RequireFromString("900"),
		TotalAmount:   decimal.RequireFromString("900"),
	}

	return &billingqueueports.ReassignChargePreview{
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
		guard: &agenttoolstest.WriteGuard{},
		canceled: &billingqueue.BillingQueueItem{
			ID:      pulid.MustNew("bqi_"),
			Number:  "BQ-12",
			Status:  billingqueue.StatusInReview,
			Version: 2,
		},
	}
	tool := NewReassignBillingChargeTool(billing)
	payer := pulid.MustNew("cus_")
	params := agenttoolstest.ApprovedParams(map[string]any{
		common.ParamItemID: pulid.MustNew("bqi_").String(),
		paramChargeKind:    "Freight",
		common.ParamAllocations: []any{map[string]any{
			common.ParamBillToCustomer: payer.String(), common.ParamAllocMethod: "Percent",
			common.ParamAllocPercent: "100",
		}},
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, billing.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, agent.PreviewOperationCreate, agenttoolstest.PreviewChange(t, preview, 0).Operation)
	assert.Equal(t, string(billingqueue.StatusCanceled),
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 1), common.FieldStatus).After)
	assert.Contains(t, preview.Summary, "PRO-4040")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, billing.reassigned)
	require.Len(t, billing.reassigned.Allocations, 1)
	assert.Equal(t, shipment.ChargeAllocationKindFreight,
		billing.reassigned.Allocations[0].ChargeKind)

	require.ErrorIs(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(params.Params)),
		common.ErrNeedsAPersonsApproval)
	policy := tool.Policy()
	assert.Equal(t, permission.ResourceBillingQueue, policy.Resource)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)

	for name, raw := range map[string]map[string]any{
		"an accessorial without its charge": {
			common.ParamItemID: pulid.MustNew("bqi_").String(), paramChargeKind: "Accessorial",
			common.ParamAllocations: []any{},
		},
		"no allocations": {
			common.ParamItemID: pulid.MustNew("bqi_").String(), paramChargeKind: "Freight",
		},
		"an order charge": {
			common.ParamItemID: pulid.MustNew("bqi_").String(), paramChargeKind: "OrderCharge",
			common.ParamAllocations: []any{},
		},
	} {
		require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(raw)), name)
	}
}

type fakeTransferRuns struct {
	guard    *agenttoolstest.WriteGuard
	refusal  error
	run      *billingtransfer.BillingTransferRun
	canceled bool
	retried  bool
}

func (f *fakeTransferRuns) Cancel(
	context.Context,
	*billingtransferservice.RunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	if err := f.guard.Write(); err != nil {
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
	if err := f.guard.Write(); err != nil {
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
		guard: &agenttoolstest.WriteGuard{},
		run: &billingtransfer.BillingTransferRun{
			ID:             pulid.MustNew("btr_"),
			TotalCount:     400,
			ProcessedCount: 120,
			RetryableCount: 3,
			SkippedCount:   1,
		},
	}
	tool := NewManageBillingTransferRunTool(runs)
	stop := agenttoolstest.ExecuteParams(map[string]any{
		resultRunID:    runs.run.ID.String(),
		paramRunAction: "Stop",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), stop)
	})
	assert.Contains(t, preview.Summary, "120 of 400")
	require.NoError(t, tool.Execute(t.Context(), stop))
	assert.True(t, runs.canceled)

	retry := agenttoolstest.ExecuteParams(map[string]any{
		resultRunID:    runs.run.ID.String(),
		paramRunAction: "Retry",
	})
	retryPreview := agenttoolstest.PreviewWithoutWrites(t, runs.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), retry)
	})
	assert.Contains(t, retryPreview.Summary, "4 shipments")
	assert.Equal(t, agent.PreviewOperationCreate, agenttoolstest.PreviewChange(t, retryPreview, 0).Operation)
	require.NoError(t, tool.Execute(t.Context(), retry))
	assert.True(t, runs.retried)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipment, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)

	runs.refusal = errortypes.NewAuthorizationError("Only the person who started a transfer can stop it")
	refused, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), stop)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, refused, agent.PreviewWarningWouldFail)
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{resultRunID: runs.run.ID.String(), paramRunAction: "Pause"})))
}
