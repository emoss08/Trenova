package customerpaymentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestValidatorRejectsOverApplication(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	customerID := pulid.MustNew("cus_")
	invoiceRepo := &fakeCustomerPaymentInvoiceRepo{invoice: &invoice.Invoice{ID: pulid.MustNew("inv_"), OrganizationID: orgID, BusinessUnitID: buID, CustomerID: customerID, Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmountMinor: 10000, AppliedAmountMinor: 0}}
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: pulid.MustNew("fy_")}, nil)
	v := NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo})

	_, _, me := v.ValidatePostAndApply(t.Context(), &customerpayment.Payment{OrganizationID: orgID, BusinessUnitID: buID, CustomerID: customerID, PaymentDate: 100, AccountingDate: 100, AmountMinor: 12000, CurrencyCode: "USD", PaymentMethod: customerpayment.MethodACH, Applications: []*customerpayment.Application{{InvoiceID: invoiceRepo.invoice.ID, AppliedAmountMinor: 12000}}}, repositories.GetInvoiceByIDRequest{TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID}}, &tenant.AccountingControl{CurrencyMode: tenant.CurrencyModeSingleCurrency, FunctionalCurrencyCode: "USD", DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_")})

	require.NotNil(t, me)
	assert.Contains(t, me.Error(), "open balance")
}

func TestPostAndApplyUsesRevenueForCashBasis(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	paymentRepo := &fakeCustomerPaymentRepo{}
	invoiceRepo := &fakeCustomerPaymentInvoiceRepo{invoice: &invoice.Invoice{ID: invoiceID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: pulid.MustNew("cus_"), Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(100), TotalAmountMinor: 10000, AppliedAmount: decimal.Zero, AppliedAmountMinor: 0, SettlementStatus: invoice.SettlementStatusUnpaid}}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisCash, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnCashReceipt, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultRevenueAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := &fakeCustomerPaymentJournalRepo{}
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	posted, err := svc.PostAndApply(t.Context(), &serviceports.PostCustomerPaymentRequest{CustomerID: invoiceRepo.invoice.CustomerID, PaymentDate: 100, AccountingDate: 100, AmountMinor: 10000, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: invoiceID, AppliedAmountMinor: 10000}}, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, posted)
	require.NotNil(t, journalRepo.last)
	assert.Equal(t, int64(10000), journalRepo.last.Lines[0].DebitAmount)
	assert.Equal(t, int64(10000), journalRepo.last.Lines[1].CreditAmount)
	assert.Equal(t, tenant.JournalSourceEventCustomerPaymentPosted.String(), journalRepo.last.SourceEventType)
	assert.Equal(t, invoice.SettlementStatusPaid, invoiceRepo.invoice.SettlementStatus)
	assert.Equal(t, permission.ResourceCustomerPayment, permission.ResourceCustomerPayment)
}

func TestPostAndApplyCreatesUnappliedCashLine(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	paymentRepo := &fakeCustomerPaymentRepo{}
	invoiceRepo := &fakeCustomerPaymentInvoiceRepo{invoice: nil}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisAccrual, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := &fakeCustomerPaymentJournalRepo{}
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	posted, err := svc.PostAndApply(t.Context(), &serviceports.PostCustomerPaymentRequest{CustomerID: pulid.MustNew("cus_"), PaymentDate: 100, AccountingDate: 100, AmountMinor: 10000, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: nil, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, posted)
	assert.Equal(t, int64(0), posted.AppliedAmountMinor)
	assert.Equal(t, int64(10000), posted.UnappliedAmountMinor)
	require.NotNil(t, journalRepo.last)
	require.Len(t, journalRepo.last.Lines, 2)
	assert.Equal(t, int64(10000), journalRepo.last.Lines[0].DebitAmount)
	assert.Equal(t, int64(10000), journalRepo.last.Lines[1].CreditAmount)
}

func TestApplyUnappliedUsesUnappliedCashReclassification(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	paymentID := pulid.MustNew("cpay_")
	invoiceID := pulid.MustNew("inv_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	paymentRepo := &fakeCustomerPaymentRepo{payment: &customerpayment.Payment{ID: paymentID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: pulid.MustNew("cus_"), AmountMinor: 15000, AppliedAmountMinor: 10000, UnappliedAmountMinor: 5000, Status: customerpayment.StatusPosted, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: []*customerpayment.Application{{ID: pulid.MustNew("cpapp_"), InvoiceID: pulid.MustNew("inv_"), AppliedAmountMinor: 10000}}}}
	invoiceRepo := &fakeCustomerPaymentInvoiceRepo{invoices: map[pulid.ID]*invoice.Invoice{invoiceID: {ID: invoiceID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: paymentRepo.payment.CustomerID, Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(50), TotalAmountMinor: 5000, AppliedAmount: decimal.Zero, AppliedAmountMinor: 0, SettlementStatus: invoice.SettlementStatusUnpaid}}}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisAccrual, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 200}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := &fakeCustomerPaymentJournalRepo{}
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	updated, err := svc.ApplyUnapplied(t.Context(), &serviceports.ApplyCustomerPaymentRequest{PaymentID: paymentID, AccountingDate: 200, Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: invoiceID, AppliedAmountMinor: 5000}}, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, int64(15000), updated.AppliedAmountMinor)
	assert.Equal(t, int64(0), updated.UnappliedAmountMinor)
	require.NotNil(t, journalRepo.last)
	assert.Equal(t, "CustomerPaymentApplied", journalRepo.last.SourceEventType)
	require.Len(t, journalRepo.last.Lines, 2)
	assert.Equal(t, int64(5000), journalRepo.last.Lines[0].DebitAmount)
	assert.Equal(t, int64(5000), journalRepo.last.Lines[1].CreditAmount)
}

func TestReverseCreatesReversingPaymentEntry(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	paymentID := pulid.MustNew("cpay_")
	invoiceID := pulid.MustNew("inv_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	paymentRepo := &fakeCustomerPaymentRepo{payment: &customerpayment.Payment{ID: paymentID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: pulid.MustNew("cus_"), AmountMinor: 15000, AppliedAmountMinor: 10000, UnappliedAmountMinor: 5000, Status: customerpayment.StatusPosted, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: []*customerpayment.Application{{ID: pulid.MustNew("cpapp_"), InvoiceID: invoiceID, AppliedAmountMinor: 10000}}}}
	invoiceRepo := &fakeCustomerPaymentInvoiceRepo{invoices: map[pulid.ID]*invoice.Invoice{invoiceID: {ID: invoiceID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: paymentRepo.payment.CustomerID, Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(100), TotalAmountMinor: 10000, AppliedAmount: decimal.RequireFromString("100.00"), AppliedAmountMinor: 10000, SettlementStatus: invoice.SettlementStatusPaid}}}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisAccrual, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 300}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := &fakeCustomerPaymentJournalRepo{}
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	reversed, err := svc.Reverse(t.Context(), &serviceports.ReverseCustomerPaymentRequest{PaymentID: paymentID, AccountingDate: 300, Reason: "chargeback", TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, reversed)
	assert.Equal(t, customerpayment.StatusReversed, reversed.Status)
	require.NotNil(t, journalRepo.last)
	assert.Equal(t, tenant.JournalSourceEventCustomerPaymentReversed.String(), journalRepo.last.SourceEventType)
	require.Len(t, journalRepo.last.Lines, 3)
	assert.Equal(t, int64(10000), journalRepo.last.Lines[0].DebitAmount)
	assert.Equal(t, int64(5000), journalRepo.last.Lines[1].DebitAmount)
	assert.Equal(t, int64(15000), journalRepo.last.Lines[2].CreditAmount)
	assert.Equal(t, invoice.SettlementStatusUnpaid, invoiceRepo.invoices[invoiceID].SettlementStatus)
}

type fakePaymentDB struct{}

func (fakePaymentDB) DB() *bun.DB                          { return nil }
func (fakePaymentDB) DBForContext(context.Context) bun.IDB { return nil }
func (fakePaymentDB) WithTx(ctx context.Context, _ ports.TxOptions, fn func(context.Context, bun.Tx) error) error {
	return fn(ctx, bun.Tx{})
}
func (fakePaymentDB) HealthCheck(context.Context) error { return nil }
func (fakePaymentDB) IsHealthy(context.Context) bool    { return true }
func (fakePaymentDB) Close() error                      { return nil }

type fakeCustomerPaymentRepo struct{ payment *customerpayment.Payment }

func (f *fakeCustomerPaymentRepo) GetByID(context.Context, repositories.GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error) {
	return f.payment, nil
}
func (f *fakeCustomerPaymentRepo) Create(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
	entity.SyncAmounts()
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("cpay_")
	}
	copy := *entity
	f.payment = &copy
	return &copy, nil
}
func (f *fakeCustomerPaymentRepo) Update(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
	entity.SyncAmounts()
	copy := *entity
	f.payment = &copy
	return &copy, nil
}

type fakeCustomerPaymentInvoiceRepo struct {
	invoice  *invoice.Invoice
	invoices map[pulid.ID]*invoice.Invoice
}

func (f *fakeCustomerPaymentInvoiceRepo) List(context.Context, *repositories.ListInvoicesRequest) (*pagination.ListResult[*invoice.Invoice], error) {
	return nil, nil
}
func (f *fakeCustomerPaymentInvoiceRepo) GetByID(_ context.Context, req repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
	if f.invoices != nil {
		if inv, ok := f.invoices[req.ID]; ok {
			copy := *inv
			return &copy, nil
		}
		return nil, errortypes.NewNotFoundError("invoice not found")
	}
	if f.invoice == nil {
		return nil, errortypes.NewNotFoundError("invoice not found")
	}
	copy := *f.invoice
	return &copy, nil
}
func (f *fakeCustomerPaymentInvoiceRepo) GetByBillingQueueItemID(context.Context, repositories.GetInvoiceByBillingQueueItemIDRequest) (*invoice.Invoice, error) {
	return nil, nil
}
func (f *fakeCustomerPaymentInvoiceRepo) CountPostedReconciliationDiscrepancies(context.Context, repositories.CountPostedInvoiceReconciliationDiscrepanciesRequest) (int, error) {
	return 0, nil
}
func (f *fakeCustomerPaymentInvoiceRepo) Create(context.Context, *invoice.Invoice) (*invoice.Invoice, error) {
	return nil, nil
}
func (f *fakeCustomerPaymentInvoiceRepo) Update(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
	if f.invoices != nil {
		f.invoices[entity.ID] = entity
		return entity, nil
	}
	f.invoice = entity
	return entity, nil
}

type fakeCustomerPaymentJournalRepo struct {
	last *repositories.CreateJournalPostingParams
}

func (f *fakeCustomerPaymentJournalRepo) CreatePosting(_ context.Context, params repositories.CreateJournalPostingParams) error {
	copyParams := params
	copyParams.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
	f.last = &copyParams
	return nil
}
