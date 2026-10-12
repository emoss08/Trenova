package invoiceservice

import (
	"context"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/latechargeportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// fakeInvoiceDB runs the transaction body directly; repositories are mocked so
// no connection is ever touched.
type fakeInvoiceDB struct{}

func (fakeInvoiceDB) DB() *bun.DB                          { return nil }
func (fakeInvoiceDB) DBForContext(context.Context) bun.IDB { return nil }

func (fakeInvoiceDB) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}
func (fakeInvoiceDB) HealthCheck(context.Context) error { return nil }
func (fakeInvoiceDB) IsHealthy(context.Context) bool    { return true }
func (fakeInvoiceDB) Close() error                      { return nil }

type voidFixture struct {
	orgID, buID, userID, invoiceID, queueItemID pulid.ID
	tenantInfo                                  pagination.TenantInfo
	actor                                       *permissionports.RequestActor
}

func newVoidFixture() voidFixture {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	return voidFixture{
		orgID:       orgID,
		buID:        buID,
		userID:      userID,
		invoiceID:   pulid.MustNew("inv_"),
		queueItemID: pulid.MustNew("bqi_"),
		tenantInfo:  pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		actor:       testutil.NewSessionActor(userID, orgID, buID),
	}
}

func (f voidFixture) invoice(status invoice.Status) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 f.invoiceID,
		OrganizationID:     f.orgID,
		BusinessUnitID:     f.buID,
		BillingQueueItemID: f.queueItemID,
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         pulid.MustNew("cus_"),
		Scope:              invoice.ScopeShipment,
		Number:             "INV-2001",
		BillType:           billingqueue.BillTypeInvoice,
		Status:             status,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        1_700_000_000,
		BillToName:         "Acme Logistics",
		SubtotalAmount:     decimal.NewFromInt(100),
		TotalAmount:        decimal.NewFromInt(100),
		TotalAmountMinor:   10000,
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Linehaul",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(100),
			Amount:      decimal.NewFromInt(100),
		}},
	}
}

func (f voidFixture) request(
	disposition invoice.VoidDisposition,
) *invoiceports.VoidInvoiceRequest {
	return &invoiceports.VoidInvoiceRequest{
		InvoiceID:   f.invoiceID,
		TenantInfo:  f.tenantInfo,
		Reason:      "Billed the wrong customer",
		Disposition: disposition,
	}
}

func newVoidService(
	repo *invoiceportsmock.MockInvoiceRepository,
	queueRepo *billingqueueportsmock.MockBillingQueueRepository,
) *Service {
	return &Service{
		l:                zap.NewNop(),
		db:               fakeInvoiceDB{},
		repo:             repo,
		billingQueueRepo: queueRepo,
		validator: &Validator{
			validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().Build(),
		},
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "INV-2002"},
	}
}

func TestVoidInvoiceRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	svc := newVoidService(invoiceportsmock.NewMockInvoiceRepository(t), billingqueueportsmock.NewMockBillingQueueRepository(t))

	_, err := svc.VoidInvoice(t.Context(), nil, f.actor)
	require.Error(t, err)

	req := f.request(invoice.VoidDispositionRebill)
	req.Reason = "   "
	_, err = svc.VoidInvoice(t.Context(), req, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "why the invoice is being voided")

	req = f.request(invoice.VoidDisposition("Later"))
	_, err = svc.VoidInvoice(t.Context(), req, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rebill or DoNotRebill")
}

func TestVoidInvoiceDraftRebillReleasesQueueItemsForRebilling(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	draft := f.invoice(invoice.StatusDraft)

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.invoiceID, TenantInfo: f.tenantInfo}).
		Return(draft, nil).
		Once()
	repo.EXPECT().
		LockForUpdate(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.invoiceID, TenantInfo: f.tenantInfo}).
		Return(draft, nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.Status == invoice.StatusVoided &&
				updated.VoidReason == "Billed the wrong customer" &&
				updated.VoidDisposition == invoice.VoidDispositionRebill &&
				updated.VoidedByID == f.userID &&
				updated.VoidedAt != nil
		})).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()

	released := &billingqueue.BillingQueueItem{
		ID:     f.queueItemID,
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
				req.InvoiceID == f.invoiceID &&
				req.AnchorItemID == f.queueItemID &&
				req.CanceledByID != nil && *req.CanceledByID == f.userID &&
				req.RenumberFn != nil &&
				req.CancelReason == "Invoice voided: Billed the wrong customer"
		})).
		Return([]*billingqueue.BillingQueueItem{released}, nil).
		Once()

	result, err := newVoidService(
		repo,
		queueRepo,
	).VoidInvoice(t.Context(), f.request(invoice.VoidDispositionRebill), f.actor)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, invoice.StatusVoided, result.Invoice.Status)
	assert.False(t, result.PendingApproval)
	assert.True(t, result.AdjustmentID.IsNil(), "a draft never needs a reversal")
	assert.Equal(t, []pulid.ID{f.queueItemID}, result.ReleasedQueueItemIDs)
}

func TestVoidInvoiceDraftDoNotRebillCancelsQueueItems(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	draft := f.invoice(invoice.StatusDraft)

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
			return !req.Rebill && req.InvoiceID == f.invoiceID
		})).
		Return([]*billingqueue.BillingQueueItem{{ID: f.queueItemID, Status: billingqueue.StatusCanceled}}, nil).
		Once()

	result, err := newVoidService(
		repo,
		queueRepo,
	).VoidInvoice(t.Context(), f.request(invoice.VoidDispositionDoNotRebill), f.actor)

	require.NoError(t, err)
	assert.Equal(t, invoice.VoidDispositionDoNotRebill, result.Invoice.VoidDisposition)
	assert.Equal(t, []pulid.ID{f.queueItemID}, result.ReleasedQueueItemIDs)
}

func TestVoidInvoiceDraftWithAppliedAmountIsRefused(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	draft := f.invoice(invoice.StatusDraft)
	draft.AppliedAmountMinor = 2500

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(draft, nil).Once()
	repo.EXPECT().LockForUpdate(mock.Anything, mock.Anything).Return(draft, nil).Once()

	_, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		VoidInvoice(t.Context(), f.request(invoice.VoidDispositionRebill), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unapply the customer payments and credit memos")
}

func TestVoidInvoicePostedWithPaymentsIsRefusedBeforeAnyReversal(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	posted := f.invoice(invoice.StatusPosted)
	posted.AppliedAmountMinor = 10000

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil).Once()

	adjustments := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentService(t)
	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.adjustmentService = adjustments

	_, err := svc.VoidInvoice(t.Context(), f.request(invoice.VoidDispositionDoNotRebill), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unapply the customer payments and credit memos")
	adjustments.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything, mock.Anything)
}

func TestVoidInvoicePostedSubmitsFullReversalAndReportsPendingApproval(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	posted := f.invoice(invoice.StatusPosted)
	adjustmentID := pulid.MustNew("iadj_")

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.invoiceID, TenantInfo: f.tenantInfo}).
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
			return req.InvoiceID == f.invoiceID &&
				req.Kind == invoiceadjustment.KindFullReversal &&
				req.IdempotencyKey == "invoice-void:"+f.invoiceID.String() &&
				req.Reason == "Billed the wrong customer" &&
				req.TenantInfo == f.tenantInfo
		}), f.actor).
		Return(&invoiceadjustment.InvoiceAdjustment{
			ID:     adjustmentID,
			Status: invoiceadjustment.StatusPendingApproval,
		}, nil).
		Once()

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.adjustmentService = adjustments

	result, err := svc.VoidInvoice(t.Context(), f.request(invoice.VoidDispositionRebill), f.actor)

	require.NoError(t, err)
	assert.Equal(t, adjustmentID, result.AdjustmentID)
	assert.True(t, result.PendingApproval)
	assert.Empty(t, result.ReleasedQueueItemIDs, "release happens when the reversal executes")
}

func TestVoidInvoicePostedExecutedReversalIsNotPending(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	posted := f.invoice(invoice.StatusPosted)

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
		f.request(invoice.VoidDispositionDoNotRebill),
		f.actor,
	)

	require.NoError(t, err)
	assert.False(t, result.PendingApproval)
}

func TestVoidInvoicePostedWithoutAdjustmentEngineIsRefused(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.invoice(invoice.StatusPosted), nil).
		Once()

	_, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		VoidInvoice(t.Context(), f.request(invoice.VoidDispositionDoNotRebill), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adjustments are unavailable")
}

func TestVoidInvoiceAlreadyVoidedIsRefused(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.invoice(invoice.StatusVoided), nil).
		Once()

	_, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).
		VoidInvoice(t.Context(), f.request(invoice.VoidDispositionRebill), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already voided")
}

func TestPostRefusesAVoidedInvoice(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.invoice(invoice.StatusVoided), nil).
		Once()

	_, err := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t)).Post(
		t.Context(),
		&invoiceports.PostInvoiceRequest{
			InvoiceID:   f.invoiceID,
			TenantInfo:  f.tenantInfo,
			TriggeredBy: "manual",
		},
		f.actor,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Voided invoices cannot be posted")
}

func TestVoidInvoiceRefusedWhileALateChargeMemoStands(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	memoID := pulid.MustNew("inv_")

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.invoice(invoice.StatusDraft), nil).
		Once()
	repo.EXPECT().
		GetByIDs(mock.Anything, invoiceports.GetInvoicesByIDsRequest{TenantInfo: f.tenantInfo, InvoiceIDs: []pulid.ID{memoID}}).
		Return([]*invoice.Invoice{{ID: memoID, Number: "DM-77", Status: invoice.StatusPosted}}, nil).
		Once()

	lateChargeRepo := latechargeportsmock.NewMockLateChargeRepository(t)
	lateChargeRepo.EXPECT().
		ListBySourceInvoiceIDs(mock.Anything, &latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest{
			TenantInfo: f.tenantInfo,
			InvoiceIDs: []pulid.ID{f.invoiceID},
		}).
		Return(map[pulid.ID][]*latecharge.LateChargeAssessment{
			f.invoiceID: {
				{SourceInvoiceID: f.invoiceID, PeriodIndex: 1, DebitMemoInvoiceID: memoID},
				{SourceInvoiceID: f.invoiceID, PeriodIndex: 2, DebitMemoInvoiceID: memoID},
			},
		}, nil).
		Once()

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.lateChargeRepo = lateChargeRepo

	_, err := svc.VoidInvoice(t.Context(), f.request(invoice.VoidDispositionRebill), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DM-77")
	assert.Contains(t, err.Error(), "late-charge memo")
}

func TestVoidInvoiceProceedsWhenLateChargeMemosAreVoided(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	memoID := pulid.MustNew("inv_")
	draft := f.invoice(invoice.StatusDraft)

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
			f.invoiceID: {
				{SourceInvoiceID: f.invoiceID, PeriodIndex: 1, DebitMemoInvoiceID: memoID},
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

	result, err := svc.VoidInvoice(t.Context(), f.request(invoice.VoidDispositionRebill), f.actor)

	require.NoError(t, err)
	assert.Equal(t, invoice.StatusVoided, result.Invoice.Status)
}
