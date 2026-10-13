package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (f voidFixture) discardRequest() *invoiceports.DiscardDraftForReviewRequest {
	return &invoiceports.DiscardDraftForReviewRequest{
		BillingQueueItemID: f.queueItemID,
		TenantInfo:         f.tenantInfo,
	}
}

func (f voidFixture) expectLookup(
	repo *invoiceportsmock.MockInvoiceRepository,
	found *invoice.Invoice,
	err error,
) {
	repo.EXPECT().
		GetByBillingQueueItemID(mock.Anything, invoiceports.GetInvoiceByBillingQueueItemIDRequest{
			BillingQueueItemID: f.queueItemID,
			TenantInfo:         f.tenantInfo,
		}).
		Return(found, err).
		Once()
}

func TestDiscardDraftForReviewVoidsTheDraftAndReleasesTheItemForRebilling(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	draft := f.invoice(invoice.StatusDraft)

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	f.expectLookup(repo, draft, nil)
	repo.EXPECT().
		LockForUpdate(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.invoiceID, TenantInfo: f.tenantInfo}).
		Return(draft, nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.Status == invoice.StatusVoided &&
				updated.VoidReason == draftReturnedToReviewReason &&
				updated.VoidDisposition == invoice.VoidDispositionRebill &&
				updated.VoidedByID == f.userID
		})).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()
	repo.EXPECT().
		RevokeDocumentShareTokens(mock.Anything, mock.Anything).
		Return(int64(0), nil).
		Once()

	queueRepo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.MatchedBy(func(req *billingqueueports.ReleaseForInvoiceRequest) bool {
			return req.Rebill &&
				req.InvoiceID == f.invoiceID &&
				req.AnchorItemID == f.queueItemID &&
				req.RenumberFn != nil
		})).
		Return([]*billingqueue.BillingQueueItem{{ID: f.queueItemID, Status: billingqueue.StatusApproved}}, nil).
		Once()

	err := newVoidService(repo, queueRepo).DiscardDraftForReview(t.Context(), f.discardRequest(), f.actor)

	require.NoError(t, err)
}

func TestDiscardDraftForReviewDoesNothingWithoutAnInvoice(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	f.expectLookup(repo, nil, errortypes.NewNotFoundError("Invoice not found"))

	err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		DiscardDraftForReview(t.Context(), f.discardRequest(), f.actor)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "LockForUpdate", mock.Anything, mock.Anything)
}

func TestDiscardDraftForReviewRefusesAPostedInvoice(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	posted := f.invoice(invoice.StatusPosted)
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	f.expectLookup(repo, posted, nil)
	repo.EXPECT().LockForUpdate(mock.Anything, mock.Anything).Return(posted, nil).Once()

	err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		DiscardDraftForReview(t.Context(), f.discardRequest(), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "void it to send the freight back to review")
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestDiscardDraftForReviewPassesLookupFailuresThrough(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	failure := errortypes.NewConflictError("lookup failed")
	f.expectLookup(repo, nil, failure)

	err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		DiscardDraftForReview(t.Context(), f.discardRequest(), f.actor)

	require.ErrorIs(t, err, failure)
}
