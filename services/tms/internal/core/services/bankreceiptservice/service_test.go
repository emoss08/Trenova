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
	paymentID := pulid.MustNew("cpay_")
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	var receipt *bankreceipt.BankReceipt
	receiptRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			if copy.ID.IsNil() {
				copy.ID = pulid.MustNew("brcpt_")
			}
			receipt = &copy
			return &copy, nil
		}).
		Once()
	receiptRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ repositories.GetBankReceiptByIDRequest) (*bankreceipt.BankReceipt, error) {
			return receipt, nil
		}).
		Once()
	receiptRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			receipt = &copy
			return &copy, nil
		}).
		Once()
	payment := &customerpayment.Payment{
		ID:             paymentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		AmountMinor:    10000,
		Status:         customerpayment.StatusPosted,
	}
	paymentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(payment, nil).Once()
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
			PaymentID:  paymentID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	require.NotNil(t, matched)
	assert.Equal(t, bankreceipt.StatusMatched, matched.Status)
	assert.Equal(t, paymentID, matched.MatchedCustomerPaymentID)
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
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	var createdReceipt *bankreceipt.BankReceipt
	receiptRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			if copy.ID.IsNil() {
				copy.ID = pulid.MustNew("brcpt_")
			}
			createdReceipt = &copy
			return &copy, nil
		}).
		Once()
	paymentRepo.EXPECT().
		FindSuggestedMatchCandidates(mock.Anything, mock.Anything).
		Return([]*customerpayment.Payment{payment}, nil).
		Once()
	paymentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(payment, nil).Once()
	receiptRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ repositories.GetBankReceiptByIDRequest) (*bankreceipt.BankReceipt, error) {
			return createdReceipt, nil
		}).
		Once()
	receiptRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			createdReceipt = &copy
			return &copy, nil
		}).
		Once()
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
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	var receipt *bankreceipt.BankReceipt
	receiptRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			if copy.ID.IsNil() {
				copy.ID = pulid.MustNew("brcpt_")
			}
			receipt = &copy
			return &copy, nil
		}).
		Once()
	paymentRepo.EXPECT().
		FindSuggestedMatchCandidates(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()
	receiptRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			receipt = &copy
			return &copy, nil
		}).
		Once()
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
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	workItemRepo := mocks.NewMockBankReceiptWorkItemRepository(t)
	var workItem *bankreceiptworkitem.WorkItem
	receiptRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			if copy.ID.IsNil() {
				copy.ID = pulid.MustNew("brcpt_")
			}
			return &copy, nil
		}).
		Once()
	paymentRepo.EXPECT().
		FindSuggestedMatchCandidates(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()
	receiptRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			return &copy, nil
		}).
		Once()
	workItemRepo.EXPECT().
		GetActiveByReceiptID(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()
	workItemRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceiptworkitem.WorkItem) (*bankreceiptworkitem.WorkItem, error) {
			copy := *entity
			workItem = &copy
			return &copy, nil
		}).
		Once()
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
	require.NotNil(t, workItem)
	assert.Equal(t, bankreceiptworkitem.StatusOpen, workItem.Status)
}

func TestListExceptionsAndSuggestMatches(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	receipt := &bankreceipt.BankReceipt{
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
		repo:         mocks.NewMockBankReceiptRepository(t),
		paymentRepo:  mocks.NewMockCustomerPaymentRepository(t),
		auditService: &mocks.NoopAuditService{},
	}
	svc.repo.(*mocks.MockBankReceiptRepository).EXPECT().
		ListExceptions(mock.Anything, mock.Anything).
		Return([]*bankreceipt.BankReceipt{receipt}, nil).
		Once()
	svc.repo.(*mocks.MockBankReceiptRepository).EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(receipt, nil).
		Once()
	svc.paymentRepo.(*mocks.MockCustomerPaymentRepository).EXPECT().
		FindSuggestedMatchCandidates(mock.Anything, mock.Anything).
		Return([]*customerpayment.Payment{payment}, nil).
		Once()

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
