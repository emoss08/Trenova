package workforcetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriverPayReviews struct {
	DriverPayReviewer

	guard   *agenttoolstest.WriteGuard
	dispute *driversettlement.Dispute
	expense *driverpay.Expense
	draft   *driversettlement.Settlement

	startedReview []pulid.ID
	resolved      *driversettlementservice.ResolveDisputeRequest
	reviewed      *driversettlementservice.ReviewExpenseRequest
}

func newFakeDriverPayReviews() *fakeDriverPayReviews {
	workerID := pulid.MustNew("wrk_")
	return &fakeDriverPayReviews{
		guard: &agenttoolstest.WriteGuard{},
		dispute: &driversettlement.Dispute{
			ID:          pulid.MustNew("dsd_"),
			WorkerID:    workerID,
			Status:      driversettlement.DisputeStatusOpen,
			Category:    driversettlement.DisputeCategory("MissingPay"),
			Description: "Detention at the Joliet DC was not paid",
			Version:     1,
		},
		expense: &driverpay.Expense{
			ID:           pulid.MustNew("dexp_"),
			WorkerID:     workerID,
			Status:       driverpay.ExpenseStatusPending,
			AmountMinor:  4_250,
			CurrencyCode: "USD",
			Description:  "Lumper fee",
			Version:      1,
		},
		draft: &driversettlement.Settlement{
			ID:               pulid.MustNew("dstl_"),
			SettlementNumber: "DS-2026-0412",
			CurrencyCode:     "USD",
		},
	}
}

func (f *fakeDriverPayReviews) PlanStartDisputeReview(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	*permissionports.RequestActor,
) (*driversettlementservice.DisputeChange, error) {
	if f.dispute.Status != driversettlement.DisputeStatusOpen {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalidOperation,
			"Only open disputes can be moved to review")
	}
	after := *f.dispute
	after.Status = driversettlement.DisputeStatusInReview
	return &driversettlementservice.DisputeChange{Before: f.dispute, After: &after}, nil
}

func (f *fakeDriverPayReviews) StartDisputeReview(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ *permissionports.RequestActor,
) (*driversettlement.Dispute, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.startedReview = append(f.startedReview, id)
	return f.dispute, nil
}

func (f *fakeDriverPayReviews) PlanResolveDispute(
	_ context.Context,
	req *driversettlementservice.ResolveDisputeRequest,
	actor *permissionports.RequestActor,
) (*driversettlementservice.ResolveDisputePlan, error) {
	if !req.Approve && req.Adjustment != nil {
		return nil, errortypes.NewValidationError("adjustment", errortypes.ErrInvalid,
			"A denied dispute cannot include an adjustment")
	}
	after := *f.dispute
	after.Status = driversettlement.DisputeStatusDenied
	if req.Approve {
		after.Status = driversettlement.DisputeStatusResolved
	}
	after.ResolutionNote = req.ResolutionNote
	after.ResolvedByID = &actor.UserID
	plan := &driversettlementservice.ResolveDisputePlan{
		Dispute: driversettlementservice.DisputeChange{Before: f.dispute, After: &after},
	}
	if req.Adjustment != nil {
		plan.AdjustmentTarget = f.draft
	}
	return plan, nil
}

func (f *fakeDriverPayReviews) ResolveDispute(
	_ context.Context,
	req *driversettlementservice.ResolveDisputeRequest,
	_ *permissionports.RequestActor,
) (*driversettlement.Dispute, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.resolved = req
	resolved := *f.dispute
	resolved.Status = driversettlement.DisputeStatusResolved
	return &resolved, nil
}

func (f *fakeDriverPayReviews) PlanReviewExpense(
	_ context.Context,
	req *driversettlementservice.ReviewExpenseRequest,
	_ *permissionports.RequestActor,
) (*driversettlementservice.ReviewExpensePlan, error) {
	if !req.Approve && req.Note == "" {
		return nil, errortypes.NewValidationError("note", errortypes.ErrRequired,
			"A note is required when rejecting an expense so the driver knows why")
	}
	after := *f.expense
	after.Status = driverpay.ExpenseStatusRejected
	plan := &driversettlementservice.ReviewExpensePlan{
		ExpenseChange: driversettlementservice.ExpenseChange{Before: f.expense, After: &after},
	}
	if req.Approve {
		after.Status = driverpay.ExpenseStatusReimbursed
	}
	return plan, nil
}

func (f *fakeDriverPayReviews) ReviewExpense(
	_ context.Context,
	req *driversettlementservice.ReviewExpenseRequest,
	_ *permissionports.RequestActor,
) (*driverpay.Expense, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.reviewed = req
	return f.expense, nil
}

func TestStartSettlementDisputeReview_OnlyAnOpenDispute(t *testing.T) {
	t.Parallel()

	reviews := newFakeDriverPayReviews()
	tool := newStartSettlementDisputeReviewTool(reviews)
	params := agenttoolstest.ExecuteParams(
		map[string]any{toolparams.ParamDisputeID: reviews.dispute.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviews.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		string(driversettlement.DisputeStatusInReview),
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			changepreview.FieldStatus,
		).After,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{reviews.dispute.ID}, reviews.startedReview)
	assert.Equal(t, permission.ResourceSettlementDispute, tool.Policy().Resource)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	reviews.dispute.Status = driversettlement.DisputeStatusInReview
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}

func TestResolveSettlementDispute_ClassifiesByWhetherPayMoves(t *testing.T) {
	t.Parallel()

	reviews := newFakeDriverPayReviews()
	tool := newResolveSettlementDisputeTool(reviews)
	approve := map[string]any{
		toolparams.ParamDisputeID:  reviews.dispute.ID.String(),
		paramDecision:              "Approve",
		paramResolutionNote:        "Detention confirmed from the gate log",
		paramAdjustmentAmountMoney: "75.00",
		paramAdjustmentLabel:       "Detention at Joliet DC",
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviews.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(approve),
			)
		},
	)
	assert.Contains(t, preview.Summary, "approve the pay dispute and tell the driver")
	assert.Contains(t, preview.Summary, "settlement DS-2026-0412")
	require.Len(t, preview.Changes, 2)

	policy := tool.Policy()
	assert.Equal(
		t,
		agent.EgressMoney,
		policy.Classified(agenttoolstest.ExecuteParams(approve)).Egress,
	)
	deny := map[string]any{
		toolparams.ParamDisputeID: reviews.dispute.ID.String(),
		paramDecision:             "Deny",
		paramResolutionNote:       "The load was paid on DS-2026-0398",
	}
	assert.Equal(
		t,
		agent.EgressDriverVisible,
		policy.Classified(agenttoolstest.ExecuteParams(deny)).Egress,
	)
	assert.Equal(t, permission.OpApprove, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)

	require.ErrorIs(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(approve)),
		toolspec.ErrNeedsAPersonsApproval)
	assert.Nil(t, reviews.resolved)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(approve)))
	require.NotNil(t, reviews.resolved)
	require.NotNil(t, reviews.resolved.Adjustment)
	assert.Equal(t, int64(7_500), reviews.resolved.Adjustment.AmountMinor)
	assert.True(t, reviews.resolved.Approve)

	deniedWithPay := map[string]any{
		toolparams.ParamDisputeID:  reviews.dispute.ID.String(),
		paramDecision:              "Deny",
		paramResolutionNote:        "x",
		paramAdjustmentAmountMoney: "10.00",
		paramAdjustmentLabel:       "x",
	}
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(deniedWithPay)), "validation runs the service's plan")
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			toolparams.ParamDisputeID:  reviews.dispute.ID.String(),
			paramDecision:              "Approve",
			paramResolutionNote:        "x",
			paramAdjustmentAmountMoney: "10.00",
		})), "an adjustment says what it is for")
}

func TestReviewDriverExpense_ARejectionSaysWhy(t *testing.T) {
	t.Parallel()

	reviews := newFakeDriverPayReviews()
	tool := newReviewDriverExpenseTool(reviews)
	approve := map[string]any{
		paramExpenseID: reviews.expense.ID.String(),
		paramDecision:  "Approve",
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviews.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(approve),
			)
		},
	)
	assert.Contains(
		t,
		preview.Summary,
		"Would reimburse the 42.50 USD expense on a new off-cycle settlement",
	)
	require.Len(t, preview.Changes, 2)

	reject := map[string]any{
		paramExpenseID: reviews.expense.ID.String(),
		paramDecision:  "Reject",
	}
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(reject)))
	reject[changepreview.FieldNote] = "No receipt attached"
	preview = agenttoolstest.PreviewWithoutWrites(
		t,
		reviews.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(reject),
			)
		},
	)
	assert.Contains(t, preview.Summary, "reject the 42.50 USD expense")
	assert.Len(t, preview.Changes, 1)

	policy := tool.Policy()
	assert.Equal(
		t,
		agent.EgressMoney,
		policy.Classified(agenttoolstest.ExecuteParams(approve)).Egress,
	)
	assert.Equal(
		t,
		agent.EgressDriverVisible,
		policy.Classified(agenttoolstest.ExecuteParams(reject)).Egress,
	)
	assert.Equal(t, permission.ResourceDriverExpense, policy.Resource)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(reject)))
	require.NotNil(t, reviews.reviewed)
	assert.False(t, reviews.reviewed.Approve)
	assert.Equal(t, "No receipt attached", reviews.reviewed.Note)
}
