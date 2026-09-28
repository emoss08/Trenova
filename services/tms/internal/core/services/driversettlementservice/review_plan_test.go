package driversettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type disputeReader struct {
	repositories.SettlementDisputeRepository
	dispute *driversettlement.Dispute
}

func (r *disputeReader) GetByID(
	context.Context,
	repositories.GetSettlementDisputeByIDRequest,
) (*driversettlement.Dispute, error) {
	copied := *r.dispute
	return &copied, nil
}

type expenseReader struct {
	repositories.DriverExpenseRepository
	expense *driverpay.Expense
}

func (r *expenseReader) GetByID(
	context.Context,
	repositories.GetDriverExpenseByIDRequest,
) (*driverpay.Expense, error) {
	copied := *r.expense
	return &copied, nil
}

type draftReader struct {
	repositories.DriverSettlementRepository
	draft *driversettlement.Settlement
}

func (r *draftReader) GetOpenDraftForWorker(
	context.Context,
	repositories.GetOpenDraftForWorkerRequest,
) (*driversettlement.Settlement, error) {
	if r.draft == nil {
		return nil, errortypes.NewNotFoundError("no open draft")
	}
	return r.draft, nil
}

type receiptControl struct {
	repositories.DashControlRepository
	required bool
}

func (c *receiptControl) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.DashControl, error) {
	return &tenant.DashControl{RequireExpenseReceipt: c.required}, nil
}

func reviewer() *serviceports.RequestActor {
	return &serviceports.RequestActor{
		UserID:        pulid.MustNew("usr_"),
		PrincipalType: serviceports.PrincipalTypeUser,
	}
}

func TestPlanDisputeDecisions(t *testing.T) {
	t.Parallel()

	dispute := &driversettlement.Dispute{
		ID:       pulid.MustNew("sdsp_"),
		WorkerID: pulid.MustNew("wrk_"),
		Status:   driversettlement.DisputeStatusOpen,
	}
	draft := planFixture(driversettlement.StatusDraft)
	svc := &Service{
		disputeRepo:    &disputeReader{dispute: dispute},
		settlementRepo: &draftReader{draft: draft},
	}

	review, err := svc.PlanStartDisputeReview(t.Context(), pagination.TenantInfo{}, dispute.ID,
		reviewer())
	require.NoError(t, err)
	assert.Equal(t, driversettlement.DisputeStatusInReview, review.After.Status)
	assert.Equal(t, driversettlement.DisputeStatusOpen, dispute.Status)

	_, err = svc.PlanStartDisputeReview(t.Context(), pagination.TenantInfo{}, dispute.ID, nil)
	require.Error(t, err, "a review is started by a person")

	plan, err := svc.PlanResolveDispute(t.Context(), &ResolveDisputeRequest{
		DisputeID:      dispute.ID,
		Approve:        true,
		ResolutionNote: "Short-paid the detention",
		Adjustment:     &AdjustmentLineInput{Description: "Detention", AmountMinor: 7500},
	}, reviewer())
	require.NoError(t, err)
	assert.Equal(t, driversettlement.DisputeStatusResolved, plan.Dispute.After.Status)
	require.NotNil(t, plan.AdjustmentTarget)
	assert.Equal(t, draft.ID, plan.AdjustmentTarget.ID)

	_, err = svc.PlanResolveDispute(t.Context(), &ResolveDisputeRequest{
		DisputeID:      dispute.ID,
		ResolutionNote: "Paid as agreed",
		Adjustment:     &AdjustmentLineInput{Description: "x", AmountMinor: 1},
	}, reviewer())
	require.Error(t, err, "a denial carries no adjustment")
}

func TestPlanReviewExpense(t *testing.T) {
	t.Parallel()

	expense := &driverpay.Expense{
		ID:       pulid.MustNew("dexp_"),
		WorkerID: pulid.MustNew("wrk_"),
		Status:   driverpay.ExpenseStatusPending,
	}
	svc := &Service{
		expenseRepo:     &expenseReader{expense: expense},
		settlementRepo:  &draftReader{},
		dashControlRepo: &receiptControl{required: true},
	}

	_, err := svc.PlanReviewExpense(t.Context(), &ReviewExpenseRequest{
		ExpenseID: expense.ID, Approve: true,
	}, reviewer())
	require.Error(t, err, "a receipt is required before approving")

	_, err = svc.PlanReviewExpense(t.Context(), &ReviewExpenseRequest{
		ExpenseID: expense.ID,
	}, reviewer())
	require.Error(t, err, "a rejection says why")

	rejected, err := svc.PlanReviewExpense(t.Context(), &ReviewExpenseRequest{
		ExpenseID: expense.ID, Note: "Personal meal",
	}, reviewer())
	require.NoError(t, err)
	assert.Equal(t, driverpay.ExpenseStatusRejected, rejected.After.Status)
	assert.Equal(t, driverpay.ExpenseStatusPending, expense.Status)

	svc.dashControlRepo = &receiptControl{}
	approved, err := svc.PlanReviewExpense(t.Context(), &ReviewExpenseRequest{
		ExpenseID: expense.ID, Approve: true,
	}, reviewer())
	require.NoError(t, err)
	assert.Equal(t, driverpay.ExpenseStatusReimbursed, approved.After.Status)
	assert.Nil(t, approved.ReimbursementTarget, "no open draft: an off-cycle one is made")
}
