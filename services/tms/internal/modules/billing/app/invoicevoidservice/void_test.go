package invoicevoidservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/latechargeportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newVoidService(
	repo *invoiceportsmock.MockInvoiceRepository,
	queueRepo *billingqueueportsmock.MockBillingQueueRepository,
) *Service {
	return &Service{
		l:                zap.NewNop(),
		db:               invoiceservicetest.FakeTx{},
		repo:             repo,
		billingQueueRepo: queueRepo,
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "INV-2002"},
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       zap.NewNop(),
				AuditService: &mocks.NoopAuditService{},
				Realtime:     &mocks.NoopRealtimeService{},
			},
		),
	}
}

func TestVoidInvoiceRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	svc := newVoidService(
		invoiceportsmock.NewMockInvoiceRepository(t),
		billingqueueportsmock.NewMockBillingQueueRepository(t),
	)

	_, err := svc.VoidInvoice(t.Context(), nil, f.Actor)
	require.Error(t, err)

	req := f.Request(invoice.VoidDispositionRebill)
	req.Reason = "   "
	_, err = svc.VoidInvoice(t.Context(), req, f.Actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "why the invoice is being voided")

	req = f.Request(invoice.VoidDisposition("Later"))
	_, err = svc.VoidInvoice(t.Context(), req, f.Actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rebill or DoNotRebill")
}

func TestVoidInvoiceDraftRebillReleasesQueueItemsForRebilling(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	draft := f.Invoice(invoice.StatusDraft)

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.InvoiceID, TenantInfo: f.TenantInfo}).
		Return(draft, nil).
		Once()
	repo.EXPECT().
		LockForUpdate(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.InvoiceID, TenantInfo: f.TenantInfo}).
		Return(draft, nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.Status == invoice.StatusVoided &&
				updated.VoidReason == "Billed the wrong customer" &&
				updated.VoidDisposition == invoice.VoidDispositionRebill &&
				updated.VoidedByID == f.UserID &&
				updated.VoidedAt != nil
		})).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()

	released := &billingqueue.BillingQueueItem{
		ID:     f.QueueItemID,
		Status: billingqueue.StatusApproved,
	}
	queueRepo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().
		RevokeDocumentShareTokens(mock.Anything, mock.Anything).
		Return(int64(0), nil).
		Once()
	queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.MatchedBy(func(req *billingqueueports.ReleaseForInvoiceRequest) bool {
			return req.Rebill &&
				req.InvoiceID == f.InvoiceID &&
				req.AnchorItemID == f.QueueItemID &&
				req.CanceledByID != nil && *req.CanceledByID == f.UserID &&
				req.RenumberFn != nil &&
				req.CancelReason == "Invoice voided: Billed the wrong customer"
		})).
		Return([]*billingqueue.BillingQueueItem{released}, nil).
		Once()

	result, err := newVoidService(
		repo,
		queueRepo,
	).VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, invoice.StatusVoided, result.Invoice.Status)
	assert.False(t, result.PendingApproval)
	assert.True(t, result.AdjustmentID.IsNil(), "a draft never needs a reversal")
	assert.Equal(t, []pulid.ID{f.QueueItemID}, result.ReleasedQueueItemIDs)
}

func TestVoidInvoiceDraftDoNotRebillCancelsQueueItems(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	draft := f.Invoice(invoice.StatusDraft)

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(draft, nil).Once()
	repo.EXPECT().LockForUpdate(mock.Anything, mock.Anything).Return(draft, nil).Once()
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()

	queueRepo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().
		RevokeDocumentShareTokens(mock.Anything, mock.Anything).
		Return(int64(0), nil).
		Once()
	queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.MatchedBy(func(req *billingqueueports.ReleaseForInvoiceRequest) bool {
			return !req.Rebill && req.InvoiceID == f.InvoiceID
		})).
		Return([]*billingqueue.BillingQueueItem{{ID: f.QueueItemID, Status: billingqueue.StatusCanceled}}, nil).
		Once()

	result, err := newVoidService(
		repo,
		queueRepo,
	).VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionDoNotRebill), f.Actor)

	require.NoError(t, err)
	assert.Equal(t, invoice.VoidDispositionDoNotRebill, result.Invoice.VoidDisposition)
	assert.Equal(t, []pulid.ID{f.QueueItemID}, result.ReleasedQueueItemIDs)
}

func TestVoidInvoiceDraftWithAppliedAmountIsRefused(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	draft := f.Invoice(invoice.StatusDraft)
	draft.AppliedAmountMinor = 2500

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(draft, nil).Once()
	repo.EXPECT().LockForUpdate(mock.Anything, mock.Anything).Return(draft, nil).Once()

	_, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unapply the customer payments and credit memos")
}

func TestVoidInvoicePostedWithPaymentsIsRefusedBeforeAnyReversal(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	posted := f.Invoice(invoice.StatusPosted)
	posted.AppliedAmountMinor = 10000

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Once()

	adjustments := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentService(t)
	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.adjustmentService = adjustments

	_, err := svc.VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionDoNotRebill), f.Actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unapply the customer payments and credit memos")
	adjustments.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything, mock.Anything)
}

func TestVoidInvoicePostedSubmitsFullReversalAndReportsPendingApproval(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	posted := f.Invoice(invoice.StatusPosted)
	adjustmentID := pulid.MustNew("iadj_")

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.InvoiceID, TenantInfo: f.TenantInfo}).
		Return(posted, nil).
		Twice()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			// The reason and disposition are persisted before the reversal runs so
			// the engine has them when it executes, but the status is untouched.
			return updated.Status == invoice.StatusPosted &&
				updated.VoidReason == "Billed the wrong customer" &&
				updated.VoidDisposition == invoice.VoidDispositionRebill
		})).
		Return(posted, nil).
		Once()

	adjustments := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentService(t)
	adjustments.EXPECT().
		Submit(mock.Anything, mock.MatchedBy(func(req *invoiceadjustmentports.InvoiceAdjustmentRequest) bool {
			return req.InvoiceID == f.InvoiceID &&
				req.Kind == invoiceadjustment.KindFullReversal &&
				req.IdempotencyKey == "invoice-void:"+f.InvoiceID.String() &&
				req.Reason == "Billed the wrong customer" &&
				req.TenantInfo == f.TenantInfo
		}), f.Actor).
		Return(&invoiceadjustment.InvoiceAdjustment{
			ID:     adjustmentID,
			Status: invoiceadjustment.StatusPendingApproval,
		}, nil).
		Once()

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.adjustmentService = adjustments

	result, err := svc.VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.NoError(t, err)
	assert.Equal(t, adjustmentID, result.AdjustmentID)
	assert.True(t, result.PendingApproval)
	assert.Empty(t, result.ReleasedQueueItemIDs, "release happens when the reversal executes")
}

func TestVoidInvoicePostedExecutedReversalIsNotPending(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	posted := f.Invoice(invoice.StatusPosted)

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Twice()
	repo.EXPECT().Update(mock.Anything, mock.Anything).Return(posted, nil).Once()

	adjustments := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentService(t)
	adjustments.EXPECT().
		Submit(mock.Anything, mock.Anything, mock.Anything).
		Return(&invoiceadjustment.InvoiceAdjustment{
			ID:     pulid.MustNew("iadj_"),
			Status: invoiceadjustment.StatusExecuted,
		}, nil).
		Once()

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.adjustmentService = adjustments

	result, err := svc.VoidInvoice(
		t.Context(),
		f.Request(invoice.VoidDispositionDoNotRebill),
		f.Actor,
	)

	require.NoError(t, err)
	assert.False(t, result.PendingApproval)
}

func TestVoidInvoicePostedWithoutAdjustmentEngineIsRefused(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.Invoice(invoice.StatusPosted), nil).
		Once()

	_, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionDoNotRebill), f.Actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adjustments are unavailable")
}

func TestVoidInvoiceAlreadyVoidedIsRefused(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.Invoice(invoice.StatusVoided), nil).
		Once()

	_, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already voided")
}

func TestVoidInvoiceRefusedWhileALateChargeMemoStands(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	memoID := pulid.MustNew("inv_")

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.Invoice(invoice.StatusDraft), nil).
		Once()
	repo.EXPECT().
		GetByIDs(mock.Anything, invoiceports.GetInvoicesByIDsRequest{TenantInfo: f.TenantInfo, InvoiceIDs: []pulid.ID{memoID}}).
		Return([]*invoice.Invoice{{ID: memoID, Number: "DM-77", Status: invoice.StatusPosted}}, nil).
		Once()

	lateChargeRepo := latechargeportsmock.NewMockLateChargeRepository(t)
	lateChargeRepo.EXPECT().
		ListBySourceInvoiceIDs(mock.Anything, &latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest{
			TenantInfo: f.TenantInfo,
			InvoiceIDs: []pulid.ID{f.InvoiceID},
		}).
		Return(map[pulid.ID][]*latecharge.LateChargeAssessment{
			f.InvoiceID: {
				{SourceInvoiceID: f.InvoiceID, PeriodIndex: 1, DebitMemoInvoiceID: memoID},
				{SourceInvoiceID: f.InvoiceID, PeriodIndex: 2, DebitMemoInvoiceID: memoID},
			},
		}, nil).
		Once()

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.lateChargeRepo = lateChargeRepo

	_, err := svc.VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DM-77")
	assert.Contains(t, err.Error(), "late-charge memo")
}

func TestVoidInvoiceProceedsWhenLateChargeMemosAreVoided(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	memoID := pulid.MustNew("inv_")
	draft := f.Invoice(invoice.StatusDraft)

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(draft, nil).Once()
	repo.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return([]*invoice.Invoice{{ID: memoID, Number: "DM-77", Status: invoice.StatusVoided}}, nil).
		Once()
	repo.EXPECT().LockForUpdate(mock.Anything, mock.Anything).Return(draft, nil).Once()
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()

	lateChargeRepo := latechargeportsmock.NewMockLateChargeRepository(t)
	lateChargeRepo.EXPECT().
		ListBySourceInvoiceIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*latecharge.LateChargeAssessment{
			f.InvoiceID: {
				{SourceInvoiceID: f.InvoiceID, PeriodIndex: 1, DebitMemoInvoiceID: memoID},
			},
		}, nil).
		Once()

	queueRepo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	repo.EXPECT().
		RevokeDocumentShareTokens(mock.Anything, mock.Anything).
		Return(int64(0), nil).
		Once()
	queueRepo.EXPECT().ReleaseForInvoice(mock.Anything, mock.Anything).
		Return([]*billingqueue.BillingQueueItem{}, nil).
		Once()

	svc := newVoidService(repo, queueRepo)
	svc.lateChargeRepo = lateChargeRepo

	result, err := svc.VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.NoError(t, err)
	assert.Equal(t, invoice.StatusVoided, result.Invoice.Status)
}

type txMarker struct{}

// markingInvoiceDB tags the context a transaction body runs in, so a test can
// tell which writes shared it.
type markingInvoiceDB struct{ invoiceservicetest.FakeTx }

func (markingInvoiceDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(context.WithValue(ctx, txMarker{}, true))
}

func inTx(ctx context.Context) bool {
	marked, _ := ctx.Value(txMarker{}).(bool)
	return marked
}

func TestVoidInvoicePostedRecordsTheReasonInTheReversalsTransaction(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	posted := f.Invoice(invoice.StatusPosted)
	refused := errortypes.NewBusinessError("The accounting period is closed")

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Once()
	repo.EXPECT().
		Update(mock.MatchedBy(inTx), mock.Anything).
		Return(posted, nil).
		Once()

	adjustments := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentService(t)
	adjustments.EXPECT().
		Submit(mock.MatchedBy(inTx), mock.Anything, f.Actor).
		Return(nil, refused).
		Once()

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.db = markingInvoiceDB{}
	svc.adjustmentService = adjustments

	result, err := svc.VoidInvoice(t.Context(), f.Request(invoice.VoidDispositionRebill), f.Actor)

	require.ErrorIs(t, err, refused)
	assert.Nil(t, result)
}
