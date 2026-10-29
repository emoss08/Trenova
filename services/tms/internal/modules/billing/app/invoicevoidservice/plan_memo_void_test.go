package invoicevoidservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/billingtest/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPreviewVoidDraftShowsItVoidedWithoutWriting(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	draft := f.Invoice(invoice.StatusDraft)
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(draft, nil).Once()

	req := f.Request(invoice.VoidDispositionRebill)
	req.Reason = "  Billed the wrong customer  "
	preview, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		PreviewVoid(t.Context(), req, f.Actor)

	require.NoError(t, err)
	assert.False(t, preview.Posted)
	assert.Nil(t, preview.Reversal)
	assert.Equal(t, invoice.StatusDraft, preview.Before.Status)
	assert.Equal(t, invoice.StatusVoided, preview.After.Status)
	assert.Equal(t, "Billed the wrong customer", preview.After.VoidReason)
	assert.Equal(t, invoice.VoidDispositionRebill, preview.After.VoidDisposition)
	assert.Equal(t, f.UserID, preview.After.VoidedByID)
}

func TestPreviewVoidPostedPreviewsTheFullReversalWithoutSubmittingIt(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	posted := f.Invoice(invoice.StatusPosted)
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Once()

	figures := &invoiceadjustmentports.InvoiceAdjustmentPreview{
		InvoiceID:         f.InvoiceID,
		Kind:              invoiceadjustment.KindFullReversal,
		CreditTotalAmount: decimal.NewFromInt(-100),
		RequiresApproval:  true,
	}
	adjustments := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentService(t)
	adjustments.EXPECT().
		Preview(mock.Anything, mock.MatchedBy(func(req *invoiceadjustmentports.InvoiceAdjustmentRequest) bool {
			return req.InvoiceID == f.InvoiceID &&
				req.Kind == invoiceadjustment.KindFullReversal &&
				req.IdempotencyKey == "invoice-void:"+f.InvoiceID.String() &&
				req.Reason == "Billed the wrong customer"
		}), f.Actor).
		Return(figures, nil).
		Once()

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.adjustmentService = adjustments

	preview, err := svc.PreviewVoid(
		t.Context(),
		f.Request(invoice.VoidDispositionDoNotRebill),
		f.Actor,
	)

	require.NoError(t, err)
	assert.True(t, preview.Posted)
	assert.Same(t, figures, preview.Reversal)
	assert.Equal(t, invoice.StatusPosted, preview.After.Status,
		"a posted invoice voids only when its reversal executes")
	assert.Equal(t, invoice.VoidDispositionDoNotRebill, preview.After.VoidDisposition)
	adjustments.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything, mock.Anything)
}

func TestPreviewVoidRefusesWhatVoidRefuses(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	posted := f.Invoice(invoice.StatusPosted)
	posted.AppliedAmountMinor = 500
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Once()
	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.adjustmentService = invoiceadjustmentportsmock.NewMockInvoiceAdjustmentService(t)

	_, err := svc.PreviewVoid(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unapply the customer payments and credit memos")
}
