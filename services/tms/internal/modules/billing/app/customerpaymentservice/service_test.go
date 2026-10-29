package customerpaymentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerpaymentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidatorRejectsOverApplication(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	customerID := pulid.MustNew("cus_")
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	storedInvoice := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		CustomerID:         customerID,
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		TotalAmountMinor:   10000,
		AppliedAmountMinor: 0,
	}
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ invoiceports.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			copy := *storedInvoice
			return &copy, nil
		})
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).
		Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), FiscalYearID: pulid.MustNew("fy_")}, nil)
	v := NewValidator(ValidatorParams{InvoiceRepo: invoiceRepo, FiscalPeriodRepo: fiscalRepo})

	_, _, me := v.ValidatePostAndApply(
		t.Context(),
		&customerpayment.Payment{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			CustomerID:     customerID,
			PaymentDate:    100,
			AccountingDate: 100,
			AmountMinor:    12000,
			CurrencyCode:   "USD",
			PaymentMethod:  customerpayment.MethodACH,
			Applications: []*customerpayment.Application{
				{InvoiceID: storedInvoice.ID, AppliedAmountMinor: 12000},
			},
		},
		invoiceports.GetInvoiceByIDRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
		&accountingcontrol.AccountingControl{
			CurrencyMode:                  accountingcontrol.CurrencyModeSingleCurrency,
			FunctionalCurrencyCode:        "USD",
			DefaultCashAccountID:          pulid.MustNew("gla_"),
			DefaultUnappliedCashAccountID: pulid.MustNew("gla_"),
		},
	)

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
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	storedPayment := (*customerpayment.Payment)(nil)
	storedInvoice := &invoice.Invoice{
		ID:                 invoiceID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		CustomerID:         pulid.MustNew("cus_"),
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		TotalAmount:        decimal.NewFromInt(100),
		TotalAmountMinor:   10000,
		AppliedAmount:      decimal.Zero,
		AppliedAmountMinor: 0,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
	}
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ invoiceports.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			copy := *storedInvoice
			return &copy, nil
		})
	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			copy := *entity
			storedInvoice = &copy
			return &copy, nil
		})
	paymentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("cpay_")
			}
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	paymentRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{AccountingBasis: accountingcontrol.AccountingBasisCash, RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnCashReceipt, JournalPostingMode: accountingcontrol.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultRevenueAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).
		Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	var lastPosting journalentryports.CreateJournalPostingParams
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			lastPosting = params
			lastPosting.Lines = append([]journalentryports.JournalPostingLine(nil), params.Lines...)
			return nil
		})
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

	posted, err := svc.PostAndApply(
		t.Context(),
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:     storedInvoice.CustomerID,
			PaymentDate:    100,
			AccountingDate: 100,
			AmountMinor:    10000,
			PaymentMethod:  customerpayment.MethodACH,
			CurrencyCode:   "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: invoiceID, AppliedAmountMinor: 10000},
			},
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, posted)
	require.NotNil(t, storedPayment)
	assert.Equal(t, int64(10000), lastPosting.Lines[0].DebitAmount)
	assert.Equal(t, int64(10000), lastPosting.Lines[1].CreditAmount)
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventCustomerPaymentPosted.String(),
		lastPosting.SourceEventType,
	)
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
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	storedPayment := (*customerpayment.Payment)(nil)
	paymentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("cpay_")
			}
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	paymentRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{AccountingBasis: accountingcontrol.AccountingBasisAccrual, RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost, JournalPostingMode: accountingcontrol.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).
		Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	var lastPosting journalentryports.CreateJournalPostingParams
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			lastPosting = params
			lastPosting.Lines = append([]journalentryports.JournalPostingLine(nil), params.Lines...)
			return nil
		})
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

	posted, err := svc.PostAndApply(
		t.Context(),
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:     pulid.MustNew("cus_"),
			PaymentDate:    100,
			AccountingDate: 100,
			AmountMinor:    10000,
			PaymentMethod:  customerpayment.MethodACH,
			CurrencyCode:   "USD",
			Applications:   nil,
			TenantInfo:     pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

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
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	storedPayment := &customerpayment.Payment{
		ID:                   paymentID,
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		CustomerID:           pulid.MustNew("cus_"),
		AmountMinor:          15000,
		AppliedAmountMinor:   10000,
		UnappliedAmountMinor: 5000,
		Status:               customerpayment.StatusPosted,
		PaymentMethod:        customerpayment.MethodACH,
		CurrencyCode:         "USD",
		Applications: []*customerpayment.Application{
			{
				ID:                 pulid.MustNew("cpapp_"),
				InvoiceID:          pulid.MustNew("inv_"),
				AppliedAmountMinor: 10000,
			},
		},
	}
	storedInvoices := map[pulid.ID]*invoice.Invoice{
		invoiceID: {
			ID:                 invoiceID,
			OrganizationID:     orgID,
			BusinessUnitID:     buID,
			CustomerID:         storedPayment.CustomerID,
			Status:             invoice.StatusPosted,
			BillType:           billingqueue.BillTypeInvoice,
			TotalAmount:        decimal.NewFromInt(50),
			TotalAmountMinor:   5000,
			AppliedAmount:      decimal.Zero,
			AppliedAmountMinor: 0,
			SettlementStatus:   invoice.SettlementStatusUnpaid,
		},
	}
	paymentRepo.EXPECT().
		GetByID(mock.Anything, customerpaymentports.GetCustomerPaymentByIDRequest{ID: paymentID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}).
		RunAndReturn(func(_ context.Context, _ customerpaymentports.GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error) {
			copy := *storedPayment
			return &copy, nil
		})
	paymentRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ invoiceports.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			copy := *storedInvoices[invoiceID]
			return &copy, nil
		})
	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			copy := *entity
			storedInvoices[entity.ID] = &copy
			return &copy, nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{AccountingBasis: accountingcontrol.AccountingBasisAccrual, RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost, JournalPostingMode: accountingcontrol.JournalPostingModeAutomatic, DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 200}).
		Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	var lastPosting journalentryports.CreateJournalPostingParams
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			lastPosting = params
			lastPosting.Lines = append([]journalentryports.JournalPostingLine(nil), params.Lines...)
			return nil
		})
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

	updated, err := svc.ApplyUnapplied(
		t.Context(),
		&customerpaymentports.ApplyCustomerPaymentRequest{
			PaymentID:      paymentID,
			AccountingDate: 200,
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: invoiceID, AppliedAmountMinor: 5000},
			},
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

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
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	storedPayment := &customerpayment.Payment{
		ID:                   paymentID,
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		CustomerID:           pulid.MustNew("cus_"),
		AmountMinor:          15000,
		AppliedAmountMinor:   10000,
		UnappliedAmountMinor: 5000,
		Status:               customerpayment.StatusPosted,
		PaymentMethod:        customerpayment.MethodACH,
		CurrencyCode:         "USD",
		Applications: []*customerpayment.Application{
			{ID: pulid.MustNew("cpapp_"), InvoiceID: invoiceID, AppliedAmountMinor: 10000},
		},
	}
	storedInvoices := map[pulid.ID]*invoice.Invoice{
		invoiceID: {
			ID:                 invoiceID,
			OrganizationID:     orgID,
			BusinessUnitID:     buID,
			CustomerID:         storedPayment.CustomerID,
			Status:             invoice.StatusPosted,
			BillType:           billingqueue.BillTypeInvoice,
			TotalAmount:        decimal.NewFromInt(100),
			TotalAmountMinor:   10000,
			AppliedAmount:      decimal.RequireFromString("100.00"),
			AppliedAmountMinor: 10000,
			SettlementStatus:   invoice.SettlementStatusPaid,
		},
	}
	paymentRepo.EXPECT().
		GetByID(mock.Anything, customerpaymentports.GetCustomerPaymentByIDRequest{ID: paymentID, TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}}).
		RunAndReturn(func(_ context.Context, _ customerpaymentports.GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error) {
			copy := *storedPayment
			return &copy, nil
		})
	paymentRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ invoiceports.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			copy := *storedInvoices[invoiceID]
			return &copy, nil
		})
	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			copy := *entity
			storedInvoices[entity.ID] = &copy
			return &copy, nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{AccountingBasis: accountingcontrol.AccountingBasisAccrual, RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost, JournalPostingMode: accountingcontrol.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 300}).
		Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	var lastPosting journalentryports.CreateJournalPostingParams
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			lastPosting = params
			lastPosting.Lines = append([]journalentryports.JournalPostingLine(nil), params.Lines...)
			return nil
		})
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

	reversed, err := svc.Reverse(
		t.Context(),
		&customerpaymentports.ReverseCustomerPaymentRequest{
			PaymentID:      paymentID,
			AccountingDate: 300,
			Reason:         "chargeback",
			TenantInfo:     pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, reversed)
	assert.Equal(t, customerpayment.StatusReversed, reversed.Status)
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventCustomerPaymentReversed.String(),
		lastPosting.SourceEventType,
	)
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
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	storedPayment := (*customerpayment.Payment)(nil)
	storedInvoice := &invoice.Invoice{
		ID:                 invoiceID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		CustomerID:         pulid.MustNew("cus_"),
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		TotalAmount:        decimal.NewFromInt(100),
		TotalAmountMinor:   10000,
		AppliedAmount:      decimal.Zero,
		AppliedAmountMinor: 0,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
	}
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ invoiceports.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			copy := *storedInvoice
			return &copy, nil
		})
	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			copy := *entity
			storedInvoice = &copy
			return &copy, nil
		})
	paymentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("cpay_")
			}
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	paymentRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
			entity.SyncAmounts()
			copy := *entity
			storedPayment = &copy
			return &copy, nil
		})
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{AccountingBasis: accountingcontrol.AccountingBasisAccrual, RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost, JournalPostingMode: accountingcontrol.JournalPostingModeAutomatic, DefaultCashAccountID: pulid.MustNew("gla_"), DefaultUnappliedCashAccountID: pulid.MustNew("gla_"), DefaultARAccountID: pulid.MustNew("gla_"), DefaultWriteOffAccountID: pulid.MustNew("gla_")}, nil)
	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: 100}).
		Return(&fiscalperiod.FiscalPeriod{ID: periodID, FiscalYearID: fyID}, nil)
	journalRepo := journalentryportsmock.NewMockJournalPostingRepository(t)
	var lastPosting journalentryports.CreateJournalPostingParams
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params journalentryports.CreateJournalPostingParams) error {
			lastPosting = params
			lastPosting.Lines = append([]journalentryports.JournalPostingLine(nil), params.Lines...)
			return nil
		})
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

	posted, err := svc.PostAndApply(
		t.Context(),
		&customerpaymentports.PostCustomerPaymentRequest{
			CustomerID:     storedInvoice.CustomerID,
			PaymentDate:    100,
			AccountingDate: 100,
			AmountMinor:    9000,
			PaymentMethod:  customerpayment.MethodACH,
			CurrencyCode:   "USD",
			Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
				{InvoiceID: invoiceID, AppliedAmountMinor: 9000, ShortPayAmountMinor: 1000},
			},
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

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

func (fakePaymentDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}
