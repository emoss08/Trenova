package bankreceiptservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerpaymentportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMatchRefusesAPaymentAlreadyMatchedToAnotherDeposit(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	receipt := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), AmountMinor: 5_000, Status: bankreceipt.StatusException}
	payment := &customerpayment.Payment{ID: pulid.MustNew("cpay_"), AmountMinor: 5_000, Status: customerpayment.StatusPosted}
	other := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), ReferenceNumber: "DEP-7", MatchedCustomerPaymentID: payment.ID}

	receipts := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	receipts.EXPECT().LockForUpdate(mock.Anything, mock.Anything).Return(receipt, nil).Once()
	receipts.EXPECT().ListMatchedToPayment(mock.Anything, bankreceiptports.ListBankReceiptsMatchedToPaymentRequest{
		PaymentID: payment.ID, TenantInfo: tenantInfo,
	}).Return([]*bankreceipt.BankReceipt{other}, nil).Once()
	payments := customerpaymentportsmock.NewMockCustomerPaymentRepository(t)
	payments.EXPECT().GetByID(mock.Anything, mock.Anything).Return(payment, nil).Once()
	svc := &Service{db: passthroughDB{}, repo: receipts, paymentRepo: payments, auditService: &mocks.NoopAuditService{}}

	_, err := svc.Match(t.Context(), &bankreceiptports.MatchBankReceiptRequest{
		ReceiptID: receipt.ID, PaymentID: payment.ID, TenantInfo: tenantInfo,
	}, testutil.NewSessionActor(pulid.MustNew("usr_"), tenantInfo.OrgID, tenantInfo.BuID))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already matched to bank receipt DEP-7")
	receipts.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestReleasePaymentMatchSendsTheDepositBackToReconciliation(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	paymentID := pulid.MustNew("cpay_")
	matchedAt := int64(1_700_000_000)
	receipt := &bankreceipt.BankReceipt{
		ID: pulid.MustNew("brcpt_"), OrganizationID: tenantInfo.OrgID, BusinessUnitID: tenantInfo.BuID,
		Status: bankreceipt.StatusMatched, MatchedCustomerPaymentID: paymentID,
		MatchedAt: &matchedAt, MatchedByID: pulid.MustNew("usr_"),
	}
	receipts := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	receipts.EXPECT().ListMatchedToPayment(mock.Anything, mock.Anything).Return([]*bankreceipt.BankReceipt{receipt}, nil).Once()
	receipts.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *bankreceipt.BankReceipt) bool {
			return entity.Status == bankreceipt.StatusException &&
				entity.MatchedCustomerPaymentID.IsNil() &&
				entity.MatchedAt == nil &&
				entity.ExceptionReason == "The customer payment ACH-1 matched to this deposit was reversed: NSF"
		})).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			return entity, nil
		}).
		Once()
	svc := &Service{l: zap.NewNop(), db: passthroughDB{}, repo: receipts, auditService: &mocks.NoopAuditService{}}

	released, err := svc.ReleasePaymentMatch(t.Context(), &bankreceiptports.ReleasePaymentMatchRequest{
		PaymentID: paymentID, PaymentReference: "ACH-1", Reason: "NSF", TenantInfo: tenantInfo,
	}, testutil.NewSessionActor(pulid.MustNew("usr_"), tenantInfo.OrgID, tenantInfo.BuID))

	require.NoError(t, err)
	require.Len(t, released, 1)
}
