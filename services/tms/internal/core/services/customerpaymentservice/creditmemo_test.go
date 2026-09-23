package customerpaymentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
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
	"go.uber.org/zap"
)

type creditMemoFixture struct {
	orgID, buID, userID, customerID pulid.ID
	tenantInfo                      pagination.TenantInfo
	actor                           *serviceports.RequestActor
	memo, target                    *invoice.Invoice
	invoiceRepo                     *mocks.MockInvoiceRepository
	paymentRepo                     *mocks.MockCustomerPaymentRepository
	fiscalRepo                      *mocks.MockFiscalPeriodRepository
	svc                             *Service
}

func newCreditMemoFixture(t *testing.T) *creditMemoFixture {
	t.Helper()
	f := &creditMemoFixture{
		orgID:      pulid.MustNew("org_"),
		buID:       pulid.MustNew("bu_"),
		userID:     pulid.MustNew("usr_"),
		customerID: pulid.MustNew("cus_"),
	}
	f.tenantInfo = pagination.TenantInfo{OrgID: f.orgID, BuID: f.buID, UserID: f.userID}
	f.actor = testutil.NewSessionActor(f.userID, f.orgID, f.buID)
	f.memo = &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     f.orgID,
		BusinessUnitID:     f.buID,
		CustomerID:         f.customerID,
		Number:             "CM-1",
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeCreditMemo,
		TotalAmount:        decimal.NewFromInt(-80),
		TotalAmountMinor:   -8000,
		AppliedAmount:      decimal.Zero,
		AppliedAmountMinor: 0,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
	}
	f.target = &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     f.orgID,
		BusinessUnitID:     f.buID,
		CustomerID:         f.customerID,
		Number:             "INV-1",
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		TotalAmount:        decimal.NewFromInt(100),
		TotalAmountMinor:   10000,
		AppliedAmount:      decimal.Zero,
		AppliedAmountMinor: 0,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
	}
	f.invoiceRepo = mocks.NewMockInvoiceRepository(t)
	f.paymentRepo = mocks.NewMockCustomerPaymentRepository(t)
	f.fiscalRepo = mocks.NewMockFiscalPeriodRepository(t)
	f.svc = &Service{
		l:           zap.NewNop(),
		db:          fakePaymentDB{},
		repo:        f.paymentRepo,
		invoiceRepo: f.invoiceRepo,
		validator: NewValidator(
			ValidatorParams{InvoiceRepo: f.invoiceRepo, FiscalPeriodRepo: f.fiscalRepo},
		),
		auditService: &mocks.NoopAuditService{},
	}

	return f
}

func (f *creditMemoFixture) expectPeriod() {
	f.fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: f.orgID, BuID: f.buID, Date: 200}).
		Return(&fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_")}, nil).
		Once()
}

func (f *creditMemoFixture) lock(inv *invoice.Invoice) {
	f.invoiceRepo.EXPECT().
		LockForUpdate(mock.Anything, repositories.GetInvoiceByIDRequest{ID: inv.ID, TenantInfo: f.tenantInfo}).
		Return(inv, nil).
		Once()
}

func (f *creditMemoFixture) applyRequest(amount int64) *serviceports.ApplyCreditMemoRequest {
	return &serviceports.ApplyCreditMemoRequest{
		CreditMemoID:   f.memo.ID,
		AccountingDate: 200,
		TenantInfo:     f.tenantInfo,
		Applications: []*serviceports.CreditMemoApplicationInput{
			{InvoiceID: f.target.ID, AppliedAmountMinor: amount},
		},
	}
}

func TestApplyCreditMemoValidatesTheRequestShape(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)

	tests := []struct {
		name    string
		mutate  func(req *serviceports.ApplyCreditMemoRequest)
		wantErr string
	}{
		{
			"missing memo",
			func(req *serviceports.ApplyCreditMemoRequest) { req.CreditMemoID = pulid.Nil },
			"Credit memo is required",
		},
		{
			"missing accounting date",
			func(req *serviceports.ApplyCreditMemoRequest) { req.AccountingDate = 0 },
			"Accounting date is required",
		},
		{
			"no applications",
			func(req *serviceports.ApplyCreditMemoRequest) { req.Applications = nil },
			"At least one application",
		},
		{
			"zero amount",
			func(req *serviceports.ApplyCreditMemoRequest) { req.Applications[0].AppliedAmountMinor = 0 },
			"greater than zero",
		},
		{"duplicate invoice", func(req *serviceports.ApplyCreditMemoRequest) {
			req.Applications = append(
				req.Applications,
				&serviceports.CreditMemoApplicationInput{
					InvoiceID:          f.target.ID,
					AppliedAmountMinor: 100,
				},
			)
		}, "once"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := f.applyRequest(1000)
			tt.mutate(req)
			_, err := f.svc.ApplyCreditMemo(t.Context(), req, f.actor)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}

	_, err := f.svc.ApplyCreditMemo(t.Context(), f.applyRequest(1000), &serviceports.RequestActor{})
	require.Error(t, err, "an anonymous actor cannot apply credits")
}

func TestApplyCreditMemoRefusesAnAccountingDateOutsideAFiscalPeriod(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(nil, assert.AnError).
		Once()

	_, err := f.svc.ApplyCreditMemo(t.Context(), f.applyRequest(1000), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "fiscal period")
}

func TestApplyCreditMemoSourceMustBeAPostedCreditMemoWithCreditLeft(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(memo *invoice.Invoice)
		wantErr string
	}{
		{
			"an invoice is not a credit memo",
			func(m *invoice.Invoice) { m.BillType = billingqueue.BillTypeInvoice },
			"Only a credit memo",
		},
		{
			"a draft memo",
			func(m *invoice.Invoice) { m.Status = invoice.StatusDraft },
			"posted credit memo",
		},
		{
			"a voided memo",
			func(m *invoice.Invoice) { m.Status = invoice.StatusVoided },
			"posted credit memo",
		},
		{
			"a fully applied memo",
			func(m *invoice.Invoice) { m.AppliedAmountMinor = 8000 },
			"nothing left to apply",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newCreditMemoFixture(t)
			tt.mutate(f.memo)
			f.expectPeriod()
			f.lock(f.memo)

			_, err := f.svc.ApplyCreditMemo(t.Context(), f.applyRequest(1000), f.actor)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestApplyCreditMemoTotalCannotExceedTheRemainingCredit(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.memo.AppliedAmountMinor = 5000
	f.expectPeriod()
	f.lock(f.memo)

	_, err := f.svc.ApplyCreditMemo(t.Context(), f.applyRequest(3500), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceed the credit memo's remaining balance by 500")
}

func TestApplyCreditMemoTargetRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(target *invoice.Invoice)
		amount  int64
		wantErr string
	}{
		{
			"another customer",
			func(inv *invoice.Invoice) { inv.CustomerID = pulid.MustNew("cus_") },
			1000,
			"customer must match",
		},
		{
			"a draft target",
			func(inv *invoice.Invoice) { inv.Status = invoice.StatusDraft },
			1000,
			"Only posted invoices",
		},
		{
			"a voided target",
			func(inv *invoice.Invoice) { inv.Status = invoice.StatusVoided },
			1000,
			"Only posted invoices",
		},
		{
			"a credit memo target",
			func(inv *invoice.Invoice) { inv.BillType = billingqueue.BillTypeCreditMemo },
			1000,
			"invoices and debit memos only",
		},
		{
			"over the open balance",
			func(inv *invoice.Invoice) { inv.AppliedAmountMinor = 9500 },
			1000,
			"exceeds the invoice open balance by 500",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newCreditMemoFixture(t)
			tt.mutate(f.target)
			f.expectPeriod()
			f.lock(f.memo)
			f.lock(f.target)

			_, err := f.svc.ApplyCreditMemo(t.Context(), f.applyRequest(tt.amount), f.actor)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestApplyCreditMemoSettlesInvoicesAndRecordsApplications(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	second := &invoice.Invoice{
		ID:               pulid.MustNew("inv_"),
		OrganizationID:   f.orgID,
		BusinessUnitID:   f.buID,
		CustomerID:       f.customerID,
		Number:           "DM-2",
		Status:           invoice.StatusPosted,
		BillType:         billingqueue.BillTypeDebitMemo,
		TotalAmount:      decimal.NewFromInt(30),
		TotalAmountMinor: 3000,
		SettlementStatus: invoice.SettlementStatusUnpaid,
	}
	f.expectPeriod()
	f.lock(f.memo)
	f.lock(f.target)
	f.lock(second)

	updates := make(map[pulid.ID]*invoice.Invoice, 3)
	f.invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			copy := *entity
			updates[entity.ID] = &copy
			return &copy, nil
		}).
		Times(3)

	var inserted []*customerpayment.CreditMemoApplication
	f.paymentRepo.EXPECT().
		CreateCreditMemoApplications(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, rows []*customerpayment.CreditMemoApplication) error {
			inserted = rows
			return nil
		}).
		Once()

	req := f.applyRequest(5000)
	req.Applications = append(
		req.Applications,
		&serviceports.CreditMemoApplicationInput{InvoiceID: second.ID, AppliedAmountMinor: 3000},
	)

	applied, err := f.svc.ApplyCreditMemo(t.Context(), req, f.actor)

	require.NoError(t, err)
	require.Len(t, applied, 2)
	assert.Same(t, inserted[0], applied[0])

	assert.Equal(t, int64(5000), updates[f.target.ID].AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusPartiallyPaid, updates[f.target.ID].SettlementStatus)
	assert.Equal(t, int64(3000), updates[second.ID].AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusPaid, updates[second.ID].SettlementStatus)
	assert.Equal(t, int64(8000), updates[f.memo.ID].AppliedAmountMinor, "the memo is fully used up")
	assert.Equal(t, int64(0), updates[f.memo.ID].CreditRemainingMinor())

	for idx, row := range applied {
		assert.Equal(t, idx+1, row.LineNumber)
		assert.Equal(t, f.memo.ID, row.CreditMemoInvoiceID)
		assert.Equal(t, customerpayment.CreditApplicationStatusApplied, row.Status)
		assert.Equal(t, int64(200), row.AccountingDate)
		assert.Equal(t, f.userID, row.CreatedByID)
		assert.Equal(t, f.orgID, row.OrganizationID)
	}
	assert.Equal(t, f.target.ID, applied[0].InvoiceID)
	assert.Equal(t, int64(5000), applied[0].AppliedAmountMinor)
	assert.Equal(t, second.ID, applied[1].InvoiceID)
}

func TestUnapplyCreditMemoApplicationRestoresBothBalances(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.memo.AppliedAmountMinor = 5000
	f.memo.AppliedAmount = decimal.NewFromInt(50)
	f.target.AppliedAmountMinor = 5000
	f.target.AppliedAmount = decimal.NewFromInt(50)
	f.target.SettlementStatus = invoice.SettlementStatusPartiallyPaid
	application := &customerpayment.CreditMemoApplication{
		ID:                  pulid.MustNew("cma_"),
		OrganizationID:      f.orgID,
		BusinessUnitID:      f.buID,
		CreditMemoInvoiceID: f.memo.ID,
		InvoiceID:           f.target.ID,
		AppliedAmountMinor:  5000,
		Status:              customerpayment.CreditApplicationStatusApplied,
	}

	f.paymentRepo.EXPECT().
		GetCreditMemoApplicationByID(mock.Anything, repositories.GetCreditMemoApplicationRequest{ID: application.ID, TenantInfo: f.tenantInfo}).
		Return(application, nil).
		Once()
	f.lock(f.memo)
	f.lock(f.target)
	updates := make(map[pulid.ID]*invoice.Invoice, 2)
	f.invoiceRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			copy := *entity
			updates[entity.ID] = &copy
			return &copy, nil
		}).
		Twice()
	f.paymentRepo.EXPECT().
		UpdateCreditMemoApplication(mock.Anything, mock.MatchedBy(func(row *customerpayment.CreditMemoApplication) bool {
			return row.ID == application.ID &&
				row.Status == customerpayment.CreditApplicationStatusUnapplied &&
				row.UnappliedAt != nil &&
				row.UnappliedByID == f.userID &&
				row.UnappliedReason == "Applied to the wrong invoice"
		})).
		RunAndReturn(func(_ context.Context, row *customerpayment.CreditMemoApplication) (*customerpayment.CreditMemoApplication, error) {
			return row, nil
		}).
		Once()

	updated, err := f.svc.UnapplyCreditMemoApplication(
		t.Context(),
		&serviceports.UnapplyCreditMemoApplicationRequest{
			ApplicationID: application.ID,
			Reason:        "  Applied to the wrong invoice ",
			TenantInfo:    f.tenantInfo,
		},
		f.actor,
	)

	require.NoError(t, err)
	assert.Equal(t, customerpayment.CreditApplicationStatusUnapplied, updated.Status)
	assert.Equal(t, int64(0), updates[f.target.ID].AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusUnpaid, updates[f.target.ID].SettlementStatus)
	assert.Equal(t, int64(0), updates[f.memo.ID].AppliedAmountMinor)
	assert.Equal(t, int64(8000), updates[f.memo.ID].CreditRemainingMinor())
}

func TestUnapplyCreditMemoApplicationRefusesAnUnappliedRow(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	application := &customerpayment.CreditMemoApplication{
		ID:     pulid.MustNew("cma_"),
		Status: customerpayment.CreditApplicationStatusUnapplied,
	}
	f.paymentRepo.EXPECT().
		GetCreditMemoApplicationByID(mock.Anything, mock.Anything).
		Return(application, nil).
		Once()

	_, err := f.svc.UnapplyCreditMemoApplication(
		t.Context(),
		&serviceports.UnapplyCreditMemoApplicationRequest{
			ApplicationID: application.ID,
			TenantInfo:    f.tenantInfo,
		},
		f.actor,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been unapplied")
}

func TestUnapplyCreditMemoApplicationRefusesAVoidedTarget(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.target.Status = invoice.StatusVoided
	application := &customerpayment.CreditMemoApplication{
		ID:                  pulid.MustNew("cma_"),
		CreditMemoInvoiceID: f.memo.ID,
		InvoiceID:           f.target.ID,
		AppliedAmountMinor:  5000,
		Status:              customerpayment.CreditApplicationStatusApplied,
	}
	f.paymentRepo.EXPECT().
		GetCreditMemoApplicationByID(mock.Anything, mock.Anything).
		Return(application, nil).
		Once()
	f.lock(f.memo)
	f.lock(f.target)

	_, err := f.svc.UnapplyCreditMemoApplication(
		t.Context(),
		&serviceports.UnapplyCreditMemoApplicationRequest{
			ApplicationID: application.ID,
			TenantInfo:    f.tenantInfo,
		},
		f.actor,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "has been voided")
}
