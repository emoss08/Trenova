package bankreceiptservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestImportAndMatchBankReceipt(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	receiptRepo := &fakeBankReceiptRepo{}
	paymentRepo := &fakeMatchedPaymentRepo{
		payment: &customerpayment.Payment{
			ID:             pulid.MustNew("cpay_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			AmountMinor:    10000,
			Status:         customerpayment.StatusPosted,
		},
	}
	svc := &Service{
		repo:         receiptRepo,
		paymentRepo:  paymentRepo,
		auditService: &mocks.NoopAuditService{},
	}

	receipt, err := svc.Import(
		t.Context(),
		&serviceports.ImportBankReceiptRequest{
			ReceiptDate:     100,
			AmountMinor:     10000,
			ReferenceNumber: "BANK-1",
			TenantInfo:      pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.Equal(t, bankreceipt.StatusImported, receipt.Status)

	matched, err := svc.Match(
		t.Context(),
		&serviceports.MatchBankReceiptRequest{
			ReceiptID:  receipt.ID,
			PaymentID:  paymentRepo.payment.ID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	require.NotNil(t, matched)
	assert.Equal(t, bankreceipt.StatusMatched, matched.Status)
	assert.Equal(t, paymentRepo.payment.ID, matched.MatchedCustomerPaymentID)
}

func TestImportAutoMatchesWhenUniqueCandidateExists(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	payment := &customerpayment.Payment{
		ID:              pulid.MustNew("cpay_"),
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		AmountMinor:     10000,
		Status:          customerpayment.StatusPosted,
		ReferenceNumber: "AUTO-1",
	}
	receiptRepo := &fakeBankReceiptRepo{}
	paymentRepo := &fakeMatchedPaymentRepo{payment: payment}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{ReconciliationMode: tenant.ReconciliationModeWarnOnly, NotifyOnReconciliationException: true}, nil).
		Once()
	svc := &Service{
		repo:           receiptRepo,
		paymentRepo:    paymentRepo,
		accountingRepo: accountingRepo,
		auditService:   &mocks.NoopAuditService{},
	}

	receipt, err := svc.Import(
		t.Context(),
		&serviceports.ImportBankReceiptRequest{
			ReceiptDate:     100,
			AmountMinor:     10000,
			ReferenceNumber: "AUTO-1",
			TenantInfo:      pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	assert.Equal(t, bankreceipt.StatusMatched, receipt.Status)
	assert.Equal(t, payment.ID, receipt.MatchedCustomerPaymentID)
}

func TestImportMarksExceptionWhenNoUniqueCandidateExists(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	receiptRepo := &fakeBankReceiptRepo{}
	paymentRepo := &fakeMatchedPaymentRepo{}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{ReconciliationMode: tenant.ReconciliationModeWarnOnly, NotifyOnReconciliationException: false}, nil).
		Once()
	svc := &Service{
		repo:           receiptRepo,
		paymentRepo:    paymentRepo,
		accountingRepo: accountingRepo,
		auditService:   &mocks.NoopAuditService{},
	}

	receipt, err := svc.Import(
		t.Context(),
		&serviceports.ImportBankReceiptRequest{
			ReceiptDate:     100,
			AmountMinor:     10000,
			ReferenceNumber: "AUTO-2",
			TenantInfo:      pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	assert.Equal(t, bankreceipt.StatusException, receipt.Status)
	assert.Contains(t, receipt.ExceptionReason, "No unique")
}

func TestImportCreatesWorkItemForException(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	receiptRepo := &fakeBankReceiptRepo{}
	paymentRepo := &fakeMatchedPaymentRepo{}
	workItemRepo := &fakeBankReceiptWorkItemRepo{}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{ReconciliationMode: tenant.ReconciliationModeWarnOnly, NotifyOnReconciliationException: false}, nil).
		Once()
	svc := &Service{
		repo:           receiptRepo,
		workItemRepo:   workItemRepo,
		paymentRepo:    paymentRepo,
		accountingRepo: accountingRepo,
		auditService:   &mocks.NoopAuditService{},
	}

	_, err := svc.Import(
		t.Context(),
		&serviceports.ImportBankReceiptRequest{
			ReceiptDate:     100,
			AmountMinor:     10000,
			ReferenceNumber: "NO-MATCH",
			TenantInfo:      pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, workItemRepo.item)
	assert.Equal(t, bankreceiptworkitem.StatusOpen, workItemRepo.item.Status)
}

func TestListExceptionsAndSuggestMatches(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	receipt := &bankreceipt.Receipt{
		ID:              pulid.MustNew("brcpt_"),
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		Status:          bankreceipt.StatusException,
		ReferenceNumber: "SG-1",
		AmountMinor:     10000,
	}
	payment := &customerpayment.Payment{
		ID:              pulid.MustNew("cpay_"),
		CustomerID:      pulid.MustNew("cus_"),
		ReferenceNumber: "SG-1",
		AmountMinor:     10000,
		Status:          customerpayment.StatusPosted,
	}
	svc := &Service{
		repo:         &fakeBankReceiptRepo{receipt: receipt},
		paymentRepo:  &fakeMatchedPaymentRepo{payment: payment},
		auditService: &mocks.NoopAuditService{},
	}

	items, err := svc.ListExceptions(t.Context(), pagination.TenantInfo{OrgID: orgID, BuID: buID})
	require.NoError(t, err)
	require.Len(t, items, 1)

	suggestions, err := svc.SuggestMatches(
		t.Context(),
		&serviceports.GetBankReceiptRequest{
			ReceiptID:  receipt.ID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		},
	)
	require.NoError(t, err)
	require.Len(t, suggestions, 1)
	assert.Equal(t, 95, suggestions[0].Score)
}

type fakeBankReceiptRepo struct{ receipt *bankreceipt.Receipt }

func (f *fakeBankReceiptRepo) GetByID(
	context.Context,
	repositories.GetBankReceiptByIDRequest,
) (*bankreceipt.Receipt, error) {
	return f.receipt, nil
}

func (f *fakeBankReceiptRepo) ListExceptions(
	context.Context,
	pagination.TenantInfo,
) ([]*bankreceipt.Receipt, error) {
	if f.receipt == nil {
		return nil, nil
	}
	return []*bankreceipt.Receipt{f.receipt}, nil
}

func (f *fakeBankReceiptRepo) ListByImportBatchID(
	context.Context,
	repositories.ListBankReceiptsByImportBatchRequest,
) ([]*bankreceipt.Receipt, error) {
	if f.receipt == nil {
		return nil, nil
	}

	return []*bankreceipt.Receipt{f.receipt}, nil
}

func (f *fakeBankReceiptRepo) GetSummary(
	context.Context,
	repositories.GetBankReceiptSummaryRequest,
) (*bankreceipt.ReconciliationSummary, error) {
	return &bankreceipt.ReconciliationSummary{}, nil
}

func (f *fakeBankReceiptRepo) Create(
	_ context.Context,
	entity *bankreceipt.Receipt,
) (*bankreceipt.Receipt, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("brcpt_")
	}
	copy := *entity
	f.receipt = &copy
	return &copy, nil
}

func (f *fakeBankReceiptRepo) Update(
	_ context.Context,
	entity *bankreceipt.Receipt,
) (*bankreceipt.Receipt, error) {
	copy := *entity
	f.receipt = &copy
	return &copy, nil
}

type fakeMatchedPaymentRepo struct{ payment *customerpayment.Payment }

func (f *fakeMatchedPaymentRepo) GetByID(
	context.Context,
	repositories.GetCustomerPaymentByIDRequest,
) (*customerpayment.Payment, error) {
	return f.payment, nil
}

func (f *fakeMatchedPaymentRepo) FindMatchCandidates(
	_ context.Context,
	req repositories.FindCustomerPaymentMatchCandidatesRequest,
) ([]*customerpayment.Payment, error) {
	if f.payment != nil && f.payment.ReferenceNumber == req.ReferenceNumber &&
		f.payment.AmountMinor == req.AmountMinor {
		return []*customerpayment.Payment{f.payment}, nil
	}
	return nil, nil
}

func (f *fakeMatchedPaymentRepo) FindSuggestedMatchCandidates(
	ctx context.Context,
	req repositories.FindCustomerPaymentMatchCandidatesRequest,
) ([]*customerpayment.Payment, error) {
	return f.FindMatchCandidates(ctx, req)
}

func (f *fakeMatchedPaymentRepo) Create(
	context.Context,
	*customerpayment.Payment,
) (*customerpayment.Payment, error) {
	return nil, nil
}

func (f *fakeMatchedPaymentRepo) Update(
	context.Context,
	*customerpayment.Payment,
) (*customerpayment.Payment, error) {
	return nil, nil
}

type fakeBankReceiptWorkItemRepo struct{ item *bankreceiptworkitem.WorkItem }

func (f *fakeBankReceiptWorkItemRepo) GetByID(
	context.Context,
	repositories.GetBankReceiptWorkItemByIDRequest,
) (*bankreceiptworkitem.WorkItem, error) {
	return f.item, nil
}

func (f *fakeBankReceiptWorkItemRepo) GetActiveByReceiptID(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	return f.item, nil
}

func (f *fakeBankReceiptWorkItemRepo) ListActive(
	context.Context,
	pagination.TenantInfo,
) ([]*bankreceiptworkitem.WorkItem, error) {
	if f.item == nil {
		return nil, nil
	}
	return []*bankreceiptworkitem.WorkItem{f.item}, nil
}

func (f *fakeBankReceiptWorkItemRepo) Create(
	_ context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	copy := *entity
	f.item = &copy
	return &copy, nil
}

func (f *fakeBankReceiptWorkItemRepo) Update(
	_ context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	copy := *entity
	f.item = &copy
	return &copy, nil
}
