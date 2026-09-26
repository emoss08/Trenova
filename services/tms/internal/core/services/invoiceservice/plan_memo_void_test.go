package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPreviewVoidDraftShowsItVoidedWithoutWriting(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	draft := f.invoice(invoice.StatusDraft)
	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(draft, nil).Once()

	req := f.request(invoice.VoidDispositionRebill)
	req.Reason = "  Billed the wrong customer  "
	preview, err := newVoidService(repo, mocks.NewMockBillingQueueRepository(t)).
		PreviewVoid(t.Context(), req, f.actor)

	require.NoError(t, err)
	assert.False(t, preview.Posted)
	assert.Nil(t, preview.Reversal)
	assert.Equal(t, invoice.StatusDraft, preview.Before.Status)
	assert.Equal(t, invoice.StatusVoided, preview.After.Status)
	assert.Equal(t, "Billed the wrong customer", preview.After.VoidReason)
	assert.Equal(t, invoice.VoidDispositionRebill, preview.After.VoidDisposition)
	assert.Equal(t, f.userID, preview.After.VoidedByID)
}

func TestPreviewVoidPostedPreviewsTheFullReversalWithoutSubmittingIt(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	posted := f.invoice(invoice.StatusPosted)
	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Once()

	figures := &servicesports.InvoiceAdjustmentPreview{
		InvoiceID:         f.invoiceID,
		Kind:              invoiceadjustment.KindFullReversal,
		CreditTotalAmount: decimal.NewFromInt(-100),
		RequiresApproval:  true,
	}
	adjustments := mocks.NewMockInvoiceAdjustmentService(t)
	adjustments.EXPECT().
		Preview(mock.Anything, mock.MatchedBy(func(req *servicesports.InvoiceAdjustmentRequest) bool {
			return req.InvoiceID == f.invoiceID &&
				req.Kind == invoiceadjustment.KindFullReversal &&
				req.IdempotencyKey == "invoice-void:"+f.invoiceID.String() &&
				req.Reason == "Billed the wrong customer"
		}), f.actor).
		Return(figures, nil).
		Once()

	svc := newVoidService(repo, mocks.NewMockBillingQueueRepository(t))
	svc.adjustmentService = adjustments

	preview, err := svc.PreviewVoid(t.Context(), f.request(invoice.VoidDispositionDoNotRebill), f.actor)

	require.NoError(t, err)
	assert.True(t, preview.Posted)
	assert.Same(t, figures, preview.Reversal)
	assert.Equal(t, invoice.StatusPosted, preview.After.Status,
		"a posted invoice voids only when its reversal executes")
	assert.Equal(t, invoice.VoidDispositionDoNotRebill, preview.After.VoidDisposition)
	adjustments.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything, mock.Anything)
}

func TestPreviewVoidRefusesWhatVoidRefuses(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	posted := f.invoice(invoice.StatusPosted)
	posted.AppliedAmountMinor = 500
	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Once()
	svc := newVoidService(repo, mocks.NewMockBillingQueueRepository(t))
	svc.adjustmentService = mocks.NewMockInvoiceAdjustmentService(t)

	_, err := svc.PreviewVoid(t.Context(), f.request(invoice.VoidDispositionRebill), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unapply the customer payments and credit memos")
}

func TestPreviewMemoBuildsTheMemoWithoutMintingANumberOrWriting(t *testing.T) {
	t.Parallel()

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	customerID := pulid.MustNew("cus_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	customers := mocks.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&customer.Customer{
		ID:             customerID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "AMD",
		Code:           "AMD01",
		BillingProfile: &customer.CustomerBillingProfile{
			PaymentTerm: customer.PaymentTermNet15,
		},
	}, nil).Once()
	billing := mocks.NewMockBillingControlRepository(t)
	billing.EXPECT().GetByOrgID(mock.Anything, orgID).
		Return(&tenant.BillingControl{DefaultPaymentTerm: tenant.PaymentTermNet30}, nil).Once()

	svc := &Service{
		l:                zap.NewNop(),
		db:               fakeInvoiceDB{},
		repo:             mocks.NewMockInvoiceRepository(t),
		billingQueueRepo: mocks.NewMockBillingQueueRepository(t),
		customerRepo:     customers,
		billingRepo:      billing,
		validator: &Validator{
			validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().Build(),
		},
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "CM-9"},
	}

	req := memoRequest(tenantInfo, customerID, billingqueue.BillTypeCreditMemo)
	req.InvoiceDate = 1_700_000_000
	memo, err := svc.PreviewMemo(t.Context(), req)

	require.NoError(t, err)
	assert.Empty(t, memo.Number, "a number is minted only when the memo is made")
	assert.Equal(t, invoice.StatusDraft, memo.Status)
	assert.Equal(t, invoice.ScopeMemo, memo.Scope)
	assert.Equal(t, invoice.MemoKindManual, memo.MemoKind)
	assert.True(t, memo.TotalAmount.Equal(decimal.RequireFromString("-75.5")),
		"got %s", memo.TotalAmount)
	require.Len(t, memo.Lines, 2)
}
