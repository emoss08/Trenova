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
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	storedInvoice := &invoice.Invoice{ID: pulid.MustNew("inv_"), OrganizationID: orgID, BusinessUnitID: buID, CustomerID: customerID, Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmountMinor: 10000, AppliedAmountMinor: 0}
	invoiceRepo.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
		copy := *storedInvoice
		return &copy, nil
	})
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: pulid.MustNew("fy_")}, nil)
	v := NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo})

	_, _, me := v.ValidatePostAndApply(t.Context(), &customerpayment.Payment{OrganizationID: orgID, BusinessUnitID: buID, CustomerID: customerID, PaymentDate: 100, AccountingDate: 100, AmountMinor: 12000, CurrencyCode: "USD", PaymentMethod: customerpayment.MethodACH, Applications: []*customerpayment.Application{{InvoiceID: storedInvoice.ID, AppliedAmountMinor: 12000}}}, repositories.GetInvoiceByIDRequest{TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID}}, &tenant.AccountingControl{CurrencyMode: tenant.CurrencyModeSingleCurrency, FunctionalCurrencyCode: "USD", DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_")})

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
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	storedPayment := (*customerpayment.Payment)(nil)
	storedInvoice := &invoice.Invoice{ID: invoiceID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: pulid.MustNew("cus_"), Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(100), TotalAmountMinor: 10000, AppliedAmount: decimal.Zero, AppliedAmountMinor: 0, SettlementStatus: invoice.SettlementStatusUnpaid}
	invoiceRepo.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
		copy := *storedInvoice
		return &copy, nil
	})
	invoiceRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
		copy := *entity
		storedInvoice = &copy
		return &copy, nil
	})
	paymentRepo.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew("cpay_")
		}
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	paymentRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisCash, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnCashReceipt, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultRevenueAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	var lastPosting repositories.CreateJournalPostingParams
	journalRepo.EXPECT().CreatePosting(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
		lastPosting = params
		lastPosting.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
		return nil
	})
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	posted, err := svc.PostAndApply(t.Context(), &serviceports.PostCustomerPaymentRequest{CustomerID: storedInvoice.CustomerID, PaymentDate: 100, AccountingDate: 100, AmountMinor: 10000, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: invoiceID, AppliedAmountMinor: 10000}}, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, posted)
	require.NotNil(t, storedPayment)
	assert.Equal(t, int64(10000), lastPosting.Lines[0].DebitAmount)
	assert.Equal(t, int64(10000), lastPosting.Lines[1].CreditAmount)
	assert.Equal(t, tenant.JournalSourceEventCustomerPaymentPosted.String(), lastPosting.SourceEventType)
	assert.Equal(t, invoice.SettlementStatusPaid, storedInvoice.SettlementStatus)
	assert.Equal(t, permission.ResourceCustomerPayment, permission.ResourceCustomerPayment)
}

func TestPostAndApplyCreatesUnappliedCashLine(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	storedPayment := (*customerpayment.Payment)(nil)
	paymentRepo.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew("cpay_")
		}
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	paymentRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisAccrual, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	var lastPosting repositories.CreateJournalPostingParams
	journalRepo.EXPECT().CreatePosting(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
		lastPosting = params
		lastPosting.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
		return nil
	})
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	posted, err := svc.PostAndApply(t.Context(), &serviceports.PostCustomerPaymentRequest{CustomerID: pulid.MustNew("cus_"), PaymentDate: 100, AccountingDate: 100, AmountMinor: 10000, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: nil, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, posted)
	require.NotNil(t, storedPayment)
	assert.Equal(t, int64(0), posted.AppliedAmountMinor)
	assert.Equal(t, int64(10000), posted.UnappliedAmountMinor)
	require.Len(t, lastPosting.Lines, 2)
	assert.Equal(t, int64(10000), lastPosting.Lines[0].DebitAmount)
	assert.Equal(t, int64(10000), lastPosting.Lines[1].CreditAmount)
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
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	storedPayment := &customerpayment.Payment{ID: paymentID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: pulid.MustNew("cus_"), AmountMinor: 15000, AppliedAmountMinor: 10000, UnappliedAmountMinor: 5000, Status: customerpayment.StatusPosted, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: []*customerpayment.Application{{ID: pulid.MustNew("cpapp_"), InvoiceID: pulid.MustNew("inv_"), AppliedAmountMinor: 10000}}}
	storedInvoices := map[pulid.ID]*invoice.Invoice{invoiceID: {ID: invoiceID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: storedPayment.CustomerID, Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(50), TotalAmountMinor: 5000, AppliedAmount: decimal.Zero, AppliedAmountMinor: 0, SettlementStatus: invoice.SettlementStatusUnpaid}}
	paymentRepo.EXPECT().GetByID(mock.Anything, repositories.GetCustomerPaymentByIDRequest{ID: paymentID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}).RunAndReturn(func(_ context.Context, _ repositories.GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error) {
		copy := *storedPayment
		return &copy, nil
	})
	paymentRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	invoiceRepo.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
		copy := *storedInvoices[invoiceID]
		return &copy, nil
	})
	invoiceRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
		copy := *entity
		storedInvoices[entity.ID] = &copy
		return &copy, nil
	})
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisAccrual, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 200}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	var lastPosting repositories.CreateJournalPostingParams
	journalRepo.EXPECT().CreatePosting(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
		lastPosting = params
		lastPosting.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
		return nil
	})
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	updated, err := svc.ApplyUnapplied(t.Context(), &serviceports.ApplyCustomerPaymentRequest{PaymentID: paymentID, AccountingDate: 200, Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: invoiceID, AppliedAmountMinor: 5000}}, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, int64(15000), updated.AppliedAmountMinor)
	assert.Equal(t, int64(0), updated.UnappliedAmountMinor)
	assert.Equal(t, "CustomerPaymentApplied", lastPosting.SourceEventType)
	require.Len(t, lastPosting.Lines, 2)
	assert.Equal(t, int64(5000), lastPosting.Lines[0].DebitAmount)
	assert.Equal(t, int64(5000), lastPosting.Lines[1].CreditAmount)
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
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	storedPayment := &customerpayment.Payment{ID: paymentID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: pulid.MustNew("cus_"), AmountMinor: 15000, AppliedAmountMinor: 10000, UnappliedAmountMinor: 5000, Status: customerpayment.StatusPosted, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: []*customerpayment.Application{{ID: pulid.MustNew("cpapp_"), InvoiceID: invoiceID, AppliedAmountMinor: 10000}}}
	storedInvoices := map[pulid.ID]*invoice.Invoice{invoiceID: {ID: invoiceID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: storedPayment.CustomerID, Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(100), TotalAmountMinor: 10000, AppliedAmount: decimal.RequireFromString("100.00"), AppliedAmountMinor: 10000, SettlementStatus: invoice.SettlementStatusPaid}}
	paymentRepo.EXPECT().GetByID(mock.Anything, repositories.GetCustomerPaymentByIDRequest{ID: paymentID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}).RunAndReturn(func(_ context.Context, _ repositories.GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error) {
		copy := *storedPayment
		return &copy, nil
	})
	paymentRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	invoiceRepo.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
		copy := *storedInvoices[invoiceID]
		return &copy, nil
	})
	invoiceRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
		copy := *entity
		storedInvoices[entity.ID] = &copy
		return &copy, nil
	})
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisAccrual, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 300}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	var lastPosting repositories.CreateJournalPostingParams
	journalRepo.EXPECT().CreatePosting(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
		lastPosting = params
		lastPosting.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
		return nil
	})
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	reversed, err := svc.Reverse(t.Context(), &serviceports.ReverseCustomerPaymentRequest{PaymentID: paymentID, AccountingDate: 300, Reason: "chargeback", TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, reversed)
	assert.Equal(t, customerpayment.StatusReversed, reversed.Status)
	assert.Equal(t, tenant.JournalSourceEventCustomerPaymentReversed.String(), lastPosting.SourceEventType)
	require.Len(t, lastPosting.Lines, 3)
	assert.Equal(t, int64(10000), lastPosting.Lines[0].DebitAmount)
	assert.Equal(t, int64(5000), lastPosting.Lines[1].DebitAmount)
	assert.Equal(t, int64(15000), lastPosting.Lines[2].CreditAmount)
	assert.Equal(t, invoice.SettlementStatusUnpaid, storedInvoices[invoiceID].SettlementStatus)
}

func TestPostAndApplyRecognizesShortPay(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	periodID := pulid.MustNew("fp_")
	fyID := pulid.MustNew("fy_")
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	storedPayment := (*customerpayment.Payment)(nil)
	storedInvoice := &invoice.Invoice{ID: invoiceID, OrganizationID: orgID, BusinessUnitID: buID, CustomerID: pulid.MustNew("cus_"), Status: invoice.StatusPosted, BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(100), TotalAmountMinor: 10000, AppliedAmount: decimal.Zero, AppliedAmountMinor: 0, SettlementStatus: invoice.SettlementStatusUnpaid}
	invoiceRepo.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
		copy := *storedInvoice
		return &copy, nil
	})
	invoiceRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
		copy := *entity
		storedInvoice = &copy
		return &copy, nil
	})
	paymentRepo.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew("cpay_")
		}
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	paymentRepo.EXPECT().Update(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		copy := *entity
		storedPayment = &copy
		return &copy, nil
	})
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{AccountingBasis: tenant.AccountingBasisAccrual, RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost, JournalPostingMode: tenant.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_"), DefaultWriteOffAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	var lastPosting repositories.CreateJournalPostingParams
	journalRepo.EXPECT().CreatePosting(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
		lastPosting = params
		lastPosting.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
		return nil
	})
	svc := &Service{db: fakePaymentDB{}, repo: paymentRepo, invoiceRepo: invoiceRepo, accountingRepo: accountingRepo, journalRepo: journalRepo, generator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"}, validator: NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo}), auditService: &mocks.NoopAuditService{}}

	posted, err := svc.PostAndApply(t.Context(), &serviceports.PostCustomerPaymentRequest{CustomerID: storedInvoice.CustomerID, PaymentDate: 100, AccountingDate: 100, AmountMinor: 9000, PaymentMethod: customerpayment.MethodACH, CurrencyCode: "USD", Applications: []*serviceports.CustomerPaymentApplicationInput{{InvoiceID: invoiceID, AppliedAmountMinor: 9000, ShortPayAmountMinor: 1000}}, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, posted)
	require.NotNil(t, storedPayment)
	require.Len(t, lastPosting.Lines, 4)
	assert.Equal(t, int64(9000), lastPosting.Lines[0].DebitAmount)
	assert.Equal(t, int64(9000), lastPosting.Lines[1].CreditAmount)
	assert.Equal(t, int64(1000), lastPosting.Lines[2].CreditAmount)
	assert.Equal(t, int64(1000), lastPosting.Lines[3].DebitAmount)
	assert.Equal(t, invoice.SettlementStatusPaid, storedInvoice.SettlementStatus)
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
