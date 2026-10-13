package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func creditMemo(orgID pulid.ID, total int64) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     orgID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		Scope:              invoice.ScopeMemo,
		BillType:           billingqueue.BillTypeCreditMemo,
		TotalAmount:        decimal.NewFromInt(total),
	}
}

func thresholdControl(threshold int64, separate bool) *tenant.InvoiceAdjustmentControl {
	return &tenant.InvoiceAdjustmentControl{
		StandardAdjustmentApprovalPolicy:    tenant.ApprovalPolicyAmountThreshold,
		StandardAdjustmentApprovalThreshold: decimal.NewFromInt(threshold),
		RequireSeparateApprover:             separate,
	}
}

func TestMemoNeedsApproval(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	assert.True(t, memoNeedsApproval(thresholdControl(500, false), creditMemo(orgID, -750)))
	assert.False(t, memoNeedsApproval(thresholdControl(500, false), creditMemo(orgID, -250)))
	assert.True(t, memoNeedsApproval(
		&tenant.InvoiceAdjustmentControl{StandardAdjustmentApprovalPolicy: tenant.ApprovalPolicyAlways},
		creditMemo(orgID, -1),
	))
	assert.False(t, memoNeedsApproval(
		&tenant.InvoiceAdjustmentControl{StandardAdjustmentApprovalPolicy: tenant.ApprovalPolicyNone},
		creditMemo(orgID, -10_000),
	))

	debit := creditMemo(orgID, 750)
	debit.BillType = billingqueue.BillTypeDebitMemo
	assert.False(t, memoNeedsApproval(thresholdControl(500, false), debit), "a debit memo adds to the balance")
	assert.False(t, memoNeedsApproval(nil, creditMemo(orgID, -750)))
}

func TestRefuseAutoPostingMemoAboveTheApprovalThreshold(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	controls := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, orgID).Return(thresholdControl(500, false), nil).Twice()
	svc := &Service{adjustmentCtrlRepo: controls}

	err := svc.refuseAutoPostingMemo(t.Context(), &invoiceports.CreateMemoRequest{AutoPost: true}, creditMemo(orgID, -750))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs approval before it is posted")

	require.NoError(t, svc.refuseAutoPostingMemo(
		t.Context(), &invoiceports.CreateMemoRequest{AutoPost: true}, creditMemo(orgID, -100),
	))
	require.NoError(t, svc.refuseAutoPostingMemo(
		t.Context(), &invoiceports.CreateMemoRequest{AutoPost: false}, creditMemo(orgID, -750),
	))
}

func TestGuardMemoPosterKeepsTheRaiserFromPostingWhenSeparationIsRequired(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	raiser := pulid.MustNew("usr_")
	memo := creditMemo(tenantInfo.OrgID, -750)

	controls := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, tenantInfo.OrgID).Return(thresholdControl(500, true), nil).Twice()
	queue := billingqueueportsmock.NewMockBillingQueueRepository(t)
	queue.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&billingqueue.BillingQueueItem{
			ID:                memo.BillingQueueItemID,
			AdjustmentContext: map[string]any{memoRaisedByKey: raiser.String()},
		}, nil).
		Twice()
	svc := &Service{adjustmentCtrlRepo: controls, billingQueueRepo: queue}

	err := svc.guardMemoPoster(
		t.Context(), memo, testutil.NewSessionActor(raiser, tenantInfo.OrgID, tenantInfo.BuID), tenantInfo,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "someone else must approve and post it")

	require.NoError(t, svc.guardMemoPoster(
		t.Context(), memo, testutil.NewSessionActor(pulid.MustNew("usr_"), tenantInfo.OrgID, tenantInfo.BuID), tenantInfo,
	))
}
