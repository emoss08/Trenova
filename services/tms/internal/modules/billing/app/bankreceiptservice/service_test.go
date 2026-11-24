package bankreceiptservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptworkitemportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerpaymentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestImportAndMatchBankReceipt(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	paymentID := pulid.MustNew("cpay_")
	receiptRepo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
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
	receiptRepo.EXPECT().ListMatchedToPayment(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	receiptRepo.EXPECT().
		LockForUpdate(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ bankreceiptports.GetBankReceiptByIDRequest) (*bankreceipt.BankReceipt, error) {
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
		l:              zap.NewNop(),
		workItemRepo:   noWorkItems(t),
		db:             passthroughDB{},
		repo:           receiptRepo,
		accountingRepo: reconciliationDisabled(t),
		paymentRepo:    paymentRepo,
		auditService:   &mocks.NoopAuditService{},
		candidates:     &scoring{},
		waker:          &countingWaker{},
	}

	receipt, err := svc.Import(
		t.Context(),
		&bankreceiptports.ImportBankReceiptRequest{
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
		&bankreceiptports.MatchBankReceiptRequest{
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
	receiptRepo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
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
	paymentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(payment, nil).Once()
	receiptRepo.EXPECT().ListMatchedToPayment(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	receiptRepo.EXPECT().
		LockForUpdate(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ bankreceiptports.GetBankReceiptByIDRequest) (*bankreceipt.BankReceipt, error) {
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
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{ReconciliationMode: accountingcontrol.ReconciliationModeWarnOnly, NotifyOnReconciliationException: true}, nil).
		Once()
	svc := &Service{
		l:              zap.NewNop(),
		workItemRepo:   noWorkItems(t),
		db:             passthroughDB{},
		repo:           receiptRepo,
		paymentRepo:    paymentRepo,
		accountingRepo: accountingRepo,
		auditService:   &mocks.NoopAuditService{},
		candidates: &scoring{payments: []bankreceipt.PaymentEvidence{{
			ID:          payment.ID,
			Reference:   payment.ReferenceNumber,
			AmountMinor: payment.AmountMinor,
		}}},
		waker: &countingWaker{},
	}

	receipt, err := svc.Import(
		t.Context(),
		&bankreceiptports.ImportBankReceiptRequest{
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
	receiptRepo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
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
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			copy := *entity
			receipt = &copy
			return &copy, nil
		}).
		Once()
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{ReconciliationMode: accountingcontrol.ReconciliationModeWarnOnly, NotifyOnReconciliationException: false}, nil).
		Twice()
	svc := &Service{
		l:              zap.NewNop(),
		workItemRepo:   noWorkItems(t),
		db:             passthroughDB{},
		repo:           receiptRepo,
		paymentRepo:    paymentRepo,
		accountingRepo: accountingRepo,
		auditService:   &mocks.NoopAuditService{},
		candidates:     &scoring{},
		waker:          &countingWaker{},
	}

	receipt, err := svc.Import(
		t.Context(),
		&bankreceiptports.ImportBankReceiptRequest{
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
	receiptRepo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	paymentRepo := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	workItemRepo := bankreceiptworkitemportsmock.NewMockBankReceiptWorkItemRepository(t)
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
	accountingRepo := tenantportsmock.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&accountingcontrol.AccountingControl{ReconciliationMode: accountingcontrol.ReconciliationModeWarnOnly, NotifyOnReconciliationException: false}, nil).
		Twice()
	svc := &Service{
		l:              zap.NewNop(),
		db:             passthroughDB{},
		repo:           receiptRepo,
		workItemRepo:   workItemRepo,
		paymentRepo:    paymentRepo,
		accountingRepo: accountingRepo,
		auditService:   &mocks.NoopAuditService{},
		candidates:     &scoring{},
		waker:          &countingWaker{},
	}

	_, err := svc.Import(
		t.Context(),
		&bankreceiptports.ImportBankReceiptRequest{
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

func TestListExceptions_ReadsThePageFromTheRepository(t *testing.T) {
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
	repo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	svc := &Service{l: zap.NewNop(), repo: repo}
	repo.EXPECT().
		ListExceptions(mock.Anything, mock.Anything).
		Return(&bankreceiptports.BankReceiptExceptionPage{
			Items: []*bankreceipt.BankReceipt{receipt},
			Total: 1,
		}, nil).
		Once()

	page, err := svc.ListExceptions(t.Context(), &bankreceiptports.ListBankReceiptExceptionsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, 1, page.Total)
}

func TestPaymentsOfReceiptAmount_AsksForPaymentsOfTheReceiptsAmountUpToTheLimit(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	receipt := &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		Status:          bankreceipt.StatusException,
		ReferenceNumber: "ACH-88213",
		AmountMinor:     120_000,
	}
	payment := &customerpayment.Payment{
		ID:          pulid.MustNew("cpay_"),
		AmountMinor: 120_000,
		Status:      customerpayment.StatusPosted,
	}
	repo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	payments := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	svc := &Service{db: passthroughDB{}, repo: repo, paymentRepo: payments}

	repo.EXPECT().
		GetByID(mock.Anything, bankreceiptports.GetBankReceiptByIDRequest{ID: receipt.ID, TenantInfo: tenant}).
		Return(receipt, nil).
		Once()
	payments.EXPECT().
		FindSuggestedMatchCandidates(mock.Anything, customerpaymentports.FindCustomerPaymentMatchCandidatesRequest{
			TenantInfo:  tenant,
			AmountMinor: 120_000,
			Limit:       11,
		}).
		Return([]*customerpayment.Payment{payment}, nil).
		Once()

	found, err := svc.PaymentsOfReceiptAmount(
		t.Context(),
		&bankreceiptports.ReceiptAmountPaymentsRequest{
			ReceiptID:  receipt.ID,
			Limit:      11,
			TenantInfo: tenant,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []*customerpayment.Payment{payment}, found,
		"every payment of the amount, whatever its reference, so the caller judges the references")
}

func noWorkItems(t *testing.T) *bankreceiptworkitemportsmock.MockBankReceiptWorkItemRepository {
	t.Helper()

	repo := bankreceiptworkitemportsmock.NewMockBankReceiptWorkItemRepository(t)
	repo.EXPECT().
		GetActiveByReceiptID(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("no active work item")).
		Maybe()
	repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, item *bankreceiptworkitem.WorkItem) (*bankreceiptworkitem.WorkItem, error) {
			return item, nil
		}).
		Maybe()

	return repo
}

func reconciliationDisabled(t *testing.T) *tenantportsmock.MockAccountingControlRepository {
	t.Helper()

	repo := tenantportsmock.NewMockAccountingControlRepository(t)
	repo.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(&accountingcontrol.AccountingControl{ReconciliationMode: accountingcontrol.ReconciliationModeDisabled}, nil).
		Maybe()

	return repo
}
