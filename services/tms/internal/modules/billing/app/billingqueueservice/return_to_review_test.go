package billingqueueservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (f *holdFixture) returnToReview(t *testing.T) (*billingqueue.BillingQueueItem, error) {
	t.Helper()

	return f.svc.UpdateStatus(t.Context(), &billingqueueports.UpdateBillingQueueStatusRequest{
		ItemID:     f.item.ID,
		NewStatus:  billingqueue.StatusInReview,
		TenantInfo: f.tenantInfo,
	}, userActor(f.tenantInfo))
}

func TestUpdateStatus_ReturningAnApprovedItemToReviewDiscardsItsDraft(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.item.Status = billingqueue.StatusApproved
	f.item.Number = "INV-1001"
	f.item.Version = 3
	f.expectRead()
	f.invoices.EXPECT().
		DiscardDraftForReview(mock.Anything, &invoiceports.DiscardDraftForReviewRequest{
			BillingQueueItemID: f.item.ID,
			TenantInfo:         f.tenantInfo,
		}, mock.Anything).
		Return(nil).
		Once()
	f.repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *billingqueueports.GetBillingQueueItemByIDRequest) (*billingqueue.BillingQueueItem, error) {
			released := *f.item
			released.Number = "INV-1002"
			released.Version = 4
			return &released, nil
		}).
		Once()
	f.repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *billingqueue.BillingQueueItem) bool {
			return entity.Status == billingqueue.StatusInReview &&
				entity.Number == "INV-1002" &&
				entity.Version == 4
		})).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).
		Once()

	updated, err := f.returnToReview(t)

	require.NoError(t, err)
	assert.Equal(t, billingqueue.StatusInReview, updated.Status)
	assert.Equal(t, "INV-1002", updated.Number)
}

func TestUpdateStatus_ReturningToReviewIsRefusedWhenTheInvoiceIsPosted(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.item.Status = billingqueue.StatusApproved
	f.expectRead()
	refused := errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		"Invoice is already Posted",
	)
	f.invoices.EXPECT().
		DiscardDraftForReview(mock.Anything, mock.Anything, mock.Anything).
		Return(refused).
		Once()

	updated, err := f.returnToReview(t)

	require.ErrorIs(t, err, refused)
	assert.Nil(t, updated)
	f.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateStatus_MovingIntoReviewFromElsewhereLeavesInvoicesAlone(t *testing.T) {
	t.Parallel()

	f := newHoldFixture(t, billingqueue.BillTypeInvoice)
	f.item.Status = billingqueue.StatusOnHold
	f.expectRead()
	f.repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).
		Once()

	_, err := f.returnToReview(t)

	require.NoError(t, err)
	f.invoices.AssertNotCalled(t, "DiscardDraftForReview", mock.Anything, mock.Anything, mock.Anything)
}
