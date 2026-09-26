package customerpaymentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type paymentWorld struct {
	orgID, buID, userID pulid.ID
	tenantInfo          pagination.TenantInfo
	actor               *serviceports.RequestActor
	payment             *customerpayment.Payment
	invoices            map[pulid.ID]*invoice.Invoice
	postings            []repositories.CreateJournalPostingParams
	writes              int
	svc                 *Service
}

func newPaymentWorld(
	t *testing.T,
	payment *customerpayment.Payment,
	invoices ...*invoice.Invoice,
) *paymentWorld {
	t.Helper()

	w := &paymentWorld{
		orgID:    payment.OrganizationID,
		buID:     payment.BusinessUnitID,
		userID:   pulid.MustNew("usr_"),
		payment:  payment,
		invoices: make(map[pulid.ID]*invoice.Invoice, len(invoices)),
	}
	w.tenantInfo = pagination.TenantInfo{OrgID: w.orgID, BuID: w.buID, UserID: w.userID}
	w.actor = testutil.NewSessionActor(w.userID, w.orgID, w.buID)
	for _, inv := range invoices {
		w.invoices[inv.ID] = inv
	}

	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	paymentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, repositories.GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error) {
			stored := *w.payment
			return &stored, nil
		}).Maybe()
	paymentRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			w.writes++
			entity.SyncAmounts()
			stored := *entity
			w.payment = &stored
			return &stored, nil
		}).Maybe()
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			stored := *w.invoices[req.ID]
			return &stored, nil
		}).Maybe()
	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			w.writes++
			stored := *entity
			w.invoices[entity.ID] = &stored
			return &stored, nil
		}).Maybe()
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, w.orgID).
		Return(&tenant.AccountingControl{
			AccountingBasis:               tenant.AccountingBasisAccrual,
			RevenueRecognitionPolicy:      tenant.RevenueRecognitionOnInvoicePost,
			JournalPostingMode:            tenant.JournalPostingModeAutomatic,
			DefaultCashAccountID:          pulid.MustNew("gla_"),
			DefaultUnappliedCashAccountID: pulid.MustNew("gla_"),
			DefaultARAccountID:            pulid.MustNew("gla_"),
			DefaultWriteOffAccountID:      pulid.MustNew("gla_"),
		}, nil).Maybe()
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: pulid.MustNew("fy_")}, nil).
		Maybe()
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			w.writes++
			w.postings = append(w.postings, params)
			return nil
		}).Maybe()

	w.svc = &Service{
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

	return w
}

func openInvoiceFor(payment *customerpayment.Payment, totalMinor, appliedMinor int64) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     payment.OrganizationID,
		BusinessUnitID:     payment.BusinessUnitID,
		CustomerID:         payment.CustomerID,
		Number:             "INV-77",
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		CurrencyCode:       "USD",
		TotalAmount:        decimal.NewFromInt(totalMinor / 100),
		TotalAmountMinor:   totalMinor,
		AppliedAmount:      decimal.NewFromInt(appliedMinor / 100),
		AppliedAmountMinor: appliedMinor,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
	}
}

func postedPayment(appliedMinor, amountMinor int64, apps ...*customerpayment.Application) *customerpayment.Payment {
	return &customerpayment.Payment{
		ID:                   pulid.MustNew("cpay_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		CustomerID:           pulid.MustNew("cus_"),
		AmountMinor:          amountMinor,
		AppliedAmountMinor:   appliedMinor,
		UnappliedAmountMinor: amountMinor - appliedMinor,
		Status:               customerpayment.StatusPosted,
		PaymentMethod:        customerpayment.MethodACH,
		ReferenceNumber:      "ACH-9001",
		CurrencyCode:         "USD",
		Applications:         apps,
	}
}

func journalLinesOf(posting repositories.CreateJournalPostingParams) []serviceports.JournalLinePreview {
	lines := make([]serviceports.JournalLinePreview, 0, len(posting.Lines))
	for _, line := range posting.Lines {
		lines = append(lines, serviceports.JournalLinePreview{
			GLAccountID: line.GLAccountID,
			Description: line.Description,
			DebitMinor:  line.DebitAmount,
			CreditMinor: line.CreditAmount,
		})
	}

	return lines
}

func TestPreviewApplyUnapplied_IsWhatApplyUnappliedWrites(t *testing.T) {
	t.Parallel()

	payment := postedPayment(10000, 15000, &customerpayment.Application{
		ID: pulid.MustNew("cpapp_"), InvoiceID: pulid.MustNew("inv_"), AppliedAmountMinor: 10000,
	})
	target := openInvoiceFor(payment, 6000, 0)
	w := newPaymentWorld(t, payment, target)
	req := &serviceports.ApplyCustomerPaymentRequest{
		PaymentID:      payment.ID,
		AccountingDate: 200,
		Applications: []*serviceports.CustomerPaymentApplicationInput{
			{InvoiceID: target.ID, AppliedAmountMinor: 4500, ShortPayAmountMinor: 500},
		},
		TenantInfo: w.tenantInfo,
	}

	preview, err := w.svc.PreviewApplyUnapplied(t.Context(), req, w.actor)
	require.NoError(t, err)
	require.Zero(t, w.writes)

	assert.Equal(t, int64(14500), preview.PaymentAfter.AppliedAmountMinor)
	assert.Equal(t, int64(500), preview.PaymentAfter.UnappliedAmountMinor)
	assert.Equal(t, int64(10000), preview.PaymentBefore.AppliedAmountMinor)
	require.Len(t, preview.InvoicesAfter, 1)
	assert.Equal(t, int64(5000), preview.InvoicesAfter[0].AppliedAmountMinor)
	assert.Equal(t, "Posted", preview.Journal.EntryStatus)

	applied, err := w.svc.ApplyUnapplied(t.Context(), req, w.actor)
	require.NoError(t, err)
	assert.Equal(t, applied.AppliedAmountMinor, preview.PaymentAfter.AppliedAmountMinor)
	assert.Equal(t, applied.UnappliedAmountMinor, preview.PaymentAfter.UnappliedAmountMinor)
	assert.Equal(t, w.invoices[target.ID].AppliedAmountMinor,
		preview.InvoicesAfter[0].AppliedAmountMinor)
	require.Len(t, w.postings, 1)
	assert.Equal(t, journalLinesOf(w.postings[0])[1:], preview.Journal.Lines[1:])
	assert.Equal(t, w.postings[0].Lines[0].DebitAmount, preview.Journal.Lines[0].DebitMinor)
}

func TestPreviewApplyUnapplied_RefusesWhatApplyRefuses(t *testing.T) {
	t.Parallel()

	payment := postedPayment(15000, 15000)
	target := openInvoiceFor(payment, 6000, 0)
	w := newPaymentWorld(t, payment, target)

	_, err := w.svc.PreviewApplyUnapplied(t.Context(), &serviceports.ApplyCustomerPaymentRequest{
		PaymentID:      payment.ID,
		AccountingDate: 200,
		Applications: []*serviceports.CustomerPaymentApplicationInput{
			{InvoiceID: target.ID, AppliedAmountMinor: 100},
		},
		TenantInfo: w.tenantInfo,
	}, w.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no unapplied amount")
	assert.Zero(t, w.writes)
}

func TestPreviewReverse_IsWhatReverseWrites(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	payment := postedPayment(10000, 15000, &customerpayment.Application{
		ID: pulid.MustNew("cpapp_"), InvoiceID: invoiceID, AppliedAmountMinor: 10000,
	})
	paid := openInvoiceFor(payment, 10000, 10000)
	paid.ID = invoiceID
	paid.SettlementStatus = invoice.SettlementStatusPaid
	w := newPaymentWorld(t, payment, paid)
	req := &serviceports.ReverseCustomerPaymentRequest{
		PaymentID:      payment.ID,
		AccountingDate: 300,
		Reason:         "Chargeback from the bank",
		TenantInfo:     w.tenantInfo,
	}

	preview, err := w.svc.PreviewReverse(t.Context(), req, w.actor)
	require.NoError(t, err)
	require.Zero(t, w.writes)
	assert.Equal(t, customerpayment.StatusReversed, preview.PaymentAfter.Status)
	assert.Equal(t, "Chargeback from the bank", preview.PaymentAfter.ReversalReason)
	assert.Equal(t, invoice.SettlementStatusUnpaid, preview.InvoicesAfter[0].SettlementStatus)

	_, err = w.svc.Reverse(t.Context(), req, w.actor)
	require.NoError(t, err)
	require.Len(t, w.postings, 1)
	assert.Equal(t, journalLinesOf(w.postings[0]), preview.Journal.Lines)
	assert.Equal(t, w.invoices[invoiceID].AppliedAmountMinor,
		preview.InvoicesAfter[0].AppliedAmountMinor)
	assert.Equal(t, w.payment.Status, preview.PaymentAfter.Status)
}

func TestPreviewReverse_RefusesAReversedPayment(t *testing.T) {
	t.Parallel()

	payment := postedPayment(0, 15000)
	payment.Status = customerpayment.StatusReversed
	w := newPaymentWorld(t, payment)

	_, err := w.svc.PreviewReverse(t.Context(), &serviceports.ReverseCustomerPaymentRequest{
		PaymentID:      payment.ID,
		AccountingDate: 300,
		TenantInfo:     w.tenantInfo,
	}, w.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only posted customer payments can be reversed")
}

func TestPreviewApplyCreditMemo_IsWhatApplyCreditMemoWrites(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.expectPeriod()
	f.invoiceRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{ID: f.memo.ID, TenantInfo: f.tenantInfo}).
		RunAndReturn(func(context.Context, repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			stored := *f.memo
			return &stored, nil
		}).Once()
	f.invoiceRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{ID: f.target.ID, TenantInfo: f.tenantInfo}).
		RunAndReturn(func(context.Context, repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			stored := *f.target
			return &stored, nil
		}).Once()

	preview, err := f.svc.PreviewApplyCreditMemo(t.Context(), f.applyRequest(3000), f.actor)
	require.NoError(t, err)

	assert.Equal(t, int64(0), preview.CreditMemoBefore.AppliedAmountMinor)
	assert.Equal(t, f.memo.CreditRemainingMinor()-3000, preview.CreditMemoAfter.CreditRemainingMinor())
	require.Len(t, preview.InvoicesAfter, 1)
	assert.Equal(t, int64(3000), preview.InvoicesAfter[0].AppliedAmountMinor)
	assert.Equal(t, int64(0), preview.InvoicesBefore[0].AppliedAmountMinor)
	require.Len(t, preview.Applications, 1)
	assert.Equal(t, customerpayment.CreditApplicationStatusApplied, preview.Applications[0].Status)
	assert.Equal(t, f.userID, preview.Applications[0].CreatedByID)
	assert.Nil(t, preview.ApplicationBefore)
}

func TestPreviewUnapplyCreditMemoApplication_RestoresBothBalances(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.memo.AppliedAmountMinor = -3000
	f.target.AppliedAmountMinor = 3000
	application := &customerpayment.CreditMemoApplication{
		ID:                  pulid.MustNew("cmapp_"),
		CreditMemoInvoiceID: f.memo.ID,
		InvoiceID:           f.target.ID,
		AppliedAmountMinor:  3000,
		Status:              customerpayment.CreditApplicationStatusApplied,
	}
	f.paymentRepo.EXPECT().
		GetCreditMemoApplicationByID(mock.Anything, repositories.GetCreditMemoApplicationRequest{ID: application.ID, TenantInfo: f.tenantInfo}).
		Return(application, nil).Once()
	f.invoiceRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{ID: f.memo.ID, TenantInfo: f.tenantInfo}).
		Return(f.memo, nil).Once()
	f.invoiceRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{ID: f.target.ID, TenantInfo: f.tenantInfo}).
		Return(f.target, nil).Once()

	preview, err := f.svc.PreviewUnapplyCreditMemoApplication(
		t.Context(),
		&serviceports.UnapplyCreditMemoApplicationRequest{
			ApplicationID: application.ID,
			Reason:        " Applied to the wrong invoice ",
			TenantInfo:    f.tenantInfo,
		},
		f.actor,
	)
	require.NoError(t, err)

	assert.Equal(t, customerpayment.CreditApplicationStatusApplied, preview.ApplicationBefore.Status)
	assert.Equal(t, customerpayment.CreditApplicationStatusUnapplied, preview.Applications[0].Status)
	assert.Equal(t, "Applied to the wrong invoice", preview.Applications[0].UnappliedReason)
	assert.Equal(t, int64(0), preview.InvoicesAfter[0].AppliedAmountMinor)
	assert.Equal(t, int64(3000), preview.InvoicesBefore[0].AppliedAmountMinor)
	assert.Equal(t, int64(0), preview.CreditMemoAfter.AppliedAmountMinor)
}
