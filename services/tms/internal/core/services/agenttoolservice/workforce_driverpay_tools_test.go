package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriverPayReviews struct {
	driverPayReviewer

	guard   *writeGuard
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
		guard: &writeGuard{},
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
	*serviceports.RequestActor,
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
	_ *serviceports.RequestActor,
) (*driversettlement.Dispute, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.startedReview = append(f.startedReview, id)
	return f.dispute, nil
}

func (f *fakeDriverPayReviews) PlanResolveDispute(
	_ context.Context,
	req *driversettlementservice.ResolveDisputeRequest,
	actor *serviceports.RequestActor,
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
	_ *serviceports.RequestActor,
) (*driversettlement.Dispute, error) {
	if err := f.guard.write(); err != nil {
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
	_ *serviceports.RequestActor,
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
	_ *serviceports.RequestActor,
) (*driverpay.Expense, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.reviewed = req
	return f.expense, nil
}

func TestStartSettlementDisputeReview_OnlyAnOpenDispute(t *testing.T) {
	t.Parallel()

	reviews := newFakeDriverPayReviews()
	tool := newStartSettlementDisputeReviewTool(reviews)
	params := executeParams(map[string]any{paramDisputeID: reviews.dispute.ID.String()})

	preview := previewWithoutWrites(t, reviews.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, string(driversettlement.DisputeStatusInReview),
		fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{reviews.dispute.ID}, reviews.startedReview)
	assert.Equal(t, permission.ResourceSettlementDispute, tool.Policy().Resource)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	reviews.dispute.Status = driversettlement.DisputeStatusInReview
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
}

func TestResolveSettlementDispute_ClassifiesByWhetherPayMoves(t *testing.T) {
	t.Parallel()

	reviews := newFakeDriverPayReviews()
	tool := newResolveSettlementDisputeTool(reviews)
	approve := map[string]any{
		paramDisputeID:             reviews.dispute.ID.String(),
		paramDecision:              "Approve",
		paramResolutionNote:        "Detention confirmed from the gate log",
		paramAdjustmentAmountMoney: "75.00",
		paramAdjustmentLabel:       "Detention at Joliet DC",
	}

	preview := previewWithoutWrites(t, reviews.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), executeParams(approve))
	})
	assert.Contains(t, preview.Summary, "approve the pay dispute and tell the driver")
	assert.Contains(t, preview.Summary, "settlement DS-2026-0412")
	require.Len(t, preview.Changes, 2)

	policy := tool.Policy()
	assert.Equal(t, agent.EgressMoney, policy.Classified(executeParams(approve)).Egress)
	deny := map[string]any{
		paramDisputeID:      reviews.dispute.ID.String(),
		paramDecision:       "Deny",
		paramResolutionNote: "The load was paid on DS-2026-0398",
	}
	assert.Equal(t, agent.EgressDriverVisible, policy.Classified(executeParams(deny)).Egress)
	assert.Equal(t, permission.OpApprove, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)

	require.ErrorIs(t, tool.Execute(t.Context(), executeParams(approve)),
		ErrNeedsAPersonsApproval)
	assert.Nil(t, reviews.resolved)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(approve)))
	require.NotNil(t, reviews.resolved)
	require.NotNil(t, reviews.resolved.Adjustment)
	assert.Equal(t, int64(7_500), reviews.resolved.Adjustment.AmountMinor)
	assert.True(t, reviews.resolved.Approve)

	deniedWithPay := map[string]any{
		paramDisputeID:             reviews.dispute.ID.String(),
		paramDecision:              "Deny",
		paramResolutionNote:        "x",
		paramAdjustmentAmountMoney: "10.00",
		paramAdjustmentLabel:       "x",
	}
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(deniedWithPay)), "validation runs the service's plan")
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramDisputeID:             reviews.dispute.ID.String(),
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

	preview := previewWithoutWrites(t, reviews.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), executeParams(approve))
	})
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
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(reject)))
	reject[fieldNote] = "No receipt attached"
	preview = previewWithoutWrites(t, reviews.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), executeParams(reject))
	})
	assert.Contains(t, preview.Summary, "reject the 42.50 USD expense")
	assert.Len(t, preview.Changes, 1)

	policy := tool.Policy()
	assert.Equal(t, agent.EgressMoney, policy.Classified(executeParams(approve)).Egress)
	assert.Equal(t, agent.EgressDriverVisible, policy.Classified(executeParams(reject)).Egress)
	assert.Equal(t, permission.ResourceDriverExpense, policy.Resource)

	require.NoError(t, tool.Execute(t.Context(), approvedParams(reject)))
	require.NotNil(t, reviews.reviewed)
	assert.False(t, reviews.reviewed.Approve)
	assert.Equal(t, "No receipt attached", reviews.reviewed.Note)
}
