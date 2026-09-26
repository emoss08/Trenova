package bankreceiptservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
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

// The receipt and work item Update expectations are registered only after
// the preview, so a write during it fails the test.
func TestPreviewMatch_IsWhatMatchSaves(t *testing.T) {
	t.Parallel()

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	stored := &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		AmountMinor:     10000,
		ReferenceNumber: "BANK-1",
		Status:          bankreceipt.StatusException,
	}
	item := &bankreceiptworkitem.WorkItem{
		ID:            pulid.MustNew("brwi_"),
		BankReceiptID: stored.ID,
		Status:        bankreceiptworkitem.StatusOpen,
	}
	payment := &customerpayment.Payment{
		ID:          pulid.MustNew("cpay_"),
		AmountMinor: 10000,
		Status:      customerpayment.StatusPosted,
	}

	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	receiptRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, repositories.GetBankReceiptByIDRequest) (*bankreceipt.BankReceipt, error) {
			copied := *stored
			return &copied, nil
		})
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	paymentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(payment, nil)
	workItems := mocks.NewMockBankReceiptWorkItemRepository(t)
	workItems.EXPECT().
		GetActiveByReceiptID(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, pagination.TenantInfo, pulid.ID) (*bankreceiptworkitem.WorkItem, error) {
			copied := *item
			return &copied, nil
		})

	svc := &Service{
		repo:         receiptRepo,
		workItemRepo: workItems,
		paymentRepo:  paymentRepo,
		auditService: &mocks.NoopAuditService{},
	}
	request := &serviceports.MatchBankReceiptRequest{
		ReceiptID:  stored.ID,
		PaymentID:  payment.ID,
		TenantInfo: tenantInfo,
	}
	actor := testutil.NewSessionActor(userID, orgID, buID)

	preview, err := svc.PreviewMatch(t.Context(), request, actor)
	require.NoError(t, err)

	var savedReceipt *bankreceipt.BankReceipt
	receiptRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			savedReceipt = entity
			return entity, nil
		})
	var savedItem *bankreceiptworkitem.WorkItem
	workItems.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceiptworkitem.WorkItem) (*bankreceiptworkitem.WorkItem, error) {
			savedItem = entity
			return entity, nil
		})

	_, err = svc.Match(t.Context(), request, actor)
	require.NoError(t, err)

	assert.Equal(t, bankreceipt.StatusException, preview.ReceiptBefore.Status)
	assert.Equal(t, savedReceipt.Status, preview.ReceiptAfter.Status)
	assert.Equal(t, savedReceipt.MatchedCustomerPaymentID, preview.ReceiptAfter.MatchedCustomerPaymentID)
	assert.Equal(t, savedReceipt.MatchedByID, preview.ReceiptAfter.MatchedByID)
	require.NotNil(t, savedItem)
	assert.Equal(t, savedItem.Status, preview.WorkItemAfter.Status)
	assert.Equal(t, savedItem.ResolutionType, preview.WorkItemAfter.ResolutionType)
	assert.Equal(t, bankreceiptworkitem.StatusOpen, preview.WorkItemBefore.Status)
}

func TestPreviewMatchPayment_RefusesAnAmountMismatch(t *testing.T) {
	t.Parallel()

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	receiptRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&bankreceipt.BankReceipt{AmountMinor: 10000, Status: bankreceipt.StatusException}, nil)
	svc := &Service{repo: receiptRepo}

	_, err := svc.PreviewMatchPayment(t.Context(), &PreviewMatchPaymentRequest{
		ReceiptID:  pulid.MustNew("brcpt_"),
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Payment:    &customerpayment.Payment{AmountMinor: 9000, Status: customerpayment.StatusPosted},
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match")
}
