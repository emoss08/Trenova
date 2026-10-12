package customerpaymentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/customerpaymentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// The write repositories carry no expectations until the preview has run,
// so any write it attempted would fail the test.
func TestPreviewPostAndApply_IsWhatPostAndApplyRecords(t *testing.T) {
	t.Parallel()

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	customerID := pulid.MustNew("cus_")
	stored := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		CustomerID:         customerID,
		Number:             "INV-1001",
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		TotalAmount:        decimal.NewFromInt(100),
		TotalAmountMinor:   10000,
		AppliedAmountMinor: 2000,
		SettlementStatus:   invoice.SettlementStatusPartiallyPaid,
	}

	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, invoiceports.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			copied := *stored
			return &copied, nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{
			AccountingBasis:               tenant.AccountingBasisAccrual,
			RevenueRecognitionPolicy:      tenant.RevenueRecognitionOnInvoicePost,
			JournalPostingMode:            tenant.JournalPostingModeAutomatic,
			DefaultCashAccountID:          pulid.MustNew("gla_"),
			DefaultUnappliedCashAccountID: pulid.MustNew("gla_"),
			DefaultARAccountID:            pulid.MustNew("gla_"),
		}, nil)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: pulid.MustNew("fy_"),
		}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)

	svc := &Service{
		db:             fakePaymentDB{},
		repo:           paymentRepo,
		invoiceRepo:    invoiceRepo,
		accountingRepo: accountingRepo,
		journalRepo:    journalRepo,
		generator:      testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator: NewValidator(
			ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo},
		),
		auditService: &mocks.NoopAuditService{},
	}
	request := func() *customerpaymentports.PostCustomerPaymentRequest {
		return &customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:     customerID,
			PaymentDate:    100,
			AccountingDate: 100,
			AmountMinor:    10000,
			PaymentMethod:  customerpayment.MethodACH,
			CurrencyCode:   "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{{
				InvoiceID:          stored.ID,
				AppliedAmountMinor: 6000,
			}},
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		}
	}
	actor := testutil.NewSessionActor(userID, orgID, buID)

	preview, err := svc.PreviewPostAndApply(t.Context(), request(), actor)
	require.NoError(t, err)

	var savedInvoice *invoice.Invoice
	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			copied := *entity
			savedInvoice = &copied
			return &copied, nil
		})
	var savedPayment *customerpayment.Payment
	paymentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			entity.ID = pulid.MustNew("cpay_")
			copied := *entity
			return &copied, nil
		})
	paymentRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			copied := *entity
			savedPayment = &copied
			return &copied, nil
		})
	journalRepo.EXPECT().CreatePosting(mock.Anything, mock.Anything).Return(nil)

	_, err = svc.PostAndApply(t.Context(), request(), actor)
	require.NoError(t, err)
	require.NotNil(t, savedInvoice)
	require.NotNil(t, savedPayment)

	assert.Equal(t, savedPayment.AmountMinor, preview.Payment.AmountMinor)
	assert.Equal(t, savedPayment.AppliedAmountMinor, preview.Payment.AppliedAmountMinor)
	assert.Equal(t, savedPayment.UnappliedAmountMinor, preview.Payment.UnappliedAmountMinor)
	assert.Equal(t, int64(4000), preview.Payment.UnappliedAmountMinor)

	require.Len(t, preview.InvoicesAfter, 1)
	assert.Equal(t, savedInvoice.AppliedAmountMinor, preview.InvoicesAfter[0].AppliedAmountMinor)
	assert.Equal(t, savedInvoice.SettlementStatus, preview.InvoicesAfter[0].SettlementStatus)
	assert.Equal(t, int64(8000), preview.InvoicesBefore[0].OpenBalanceMinor())
	assert.Equal(t, int64(2000), preview.InvoicesAfter[0].OpenBalanceMinor())
}
