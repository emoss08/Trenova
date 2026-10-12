package invoiceadjustmentservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type rejectFixture struct {
	tenantInfo pagination.TenantInfo
	adjustment *invoiceadjustment.InvoiceAdjustment
	invoice    *invoice.Invoice
	repo       *invoiceadjustmentportsmock.MockInvoiceAdjustmentRepository
	invoices   *invoiceportsmock.MockInvoiceRepository
	db         *fakeAdjustmentDB
	svc        *Service
}

func newRejectFixture(t *testing.T, status invoiceadjustment.Status, kind invoiceadjustment.Kind) *rejectFixture {
	t.Helper()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	invoiceID := pulid.MustNew("inv_")
	f := &rejectFixture{
		tenantInfo: tenantInfo,
		adjustment: &invoiceadjustment.InvoiceAdjustment{
			ID:                pulid.MustNew("iadj_"),
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			OriginalInvoiceID: invoiceID,
			Kind:              kind,
			Status:            status,
			IdempotencyKey:    "invoice-void:" + invoiceID.String(),
		},
		invoice: &invoice.Invoice{
			ID:              invoiceID,
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			Status:          invoice.StatusPosted,
			VoidReason:      "Billed the wrong customer",
			VoidDisposition: invoice.VoidDispositionRebill,
		},
		repo:     invoiceadjustmentportsmock.NewMockInvoiceAdjustmentRepository(t),
		invoices: invoiceportsmock.NewMockInvoiceRepository(t),
		db:       &fakeAdjustmentDB{},
	}
	f.svc = &Service{
		l:            zap.NewNop(),
		db:           f.db,
		repo:         f.repo,
		invoiceRepo:  f.invoices,
		auditService: &mocks.NoopAuditService{},
		realtime:     &mocks.NoopRealtimeService{},
	}

	return f
}

func (f *rejectFixture) expectLockedRead() {
	request := invoiceadjustmentports.GetInvoiceAdjustmentRequest{ID: f.adjustment.ID, TenantInfo: f.tenantInfo}
	f.repo.EXPECT().
		GetByID(mock.Anything, request).
		RunAndReturn(func(context.Context, invoiceadjustmentports.GetInvoiceAdjustmentRequest) (*invoiceadjustment.InvoiceAdjustment, error) {
			clone := *f.adjustment
			return &clone, nil
		}).
		Twice()
	f.repo.EXPECT().
		LockInvoiceForUpdate(mock.Anything, invoiceadjustmentports.LockInvoiceAdjustmentRequest{
			InvoiceID:  f.invoice.ID,
			TenantInfo: f.tenantInfo,
		}).
		Return(f.invoice, nil).
		Once()
}

func (f *rejectFixture) reject(t *testing.T) (*invoiceadjustment.InvoiceAdjustment, error) {
	t.Helper()

	return f.svc.Reject(t.Context(), &invoiceadjustmentports.RejectInvoiceAdjustmentRequest{
		AdjustmentID: f.adjustment.ID,
		Reason:       "Customer accepted the invoice",
		TenantInfo:   f.tenantInfo,
	}, testutil.NewSessionActor(pulid.MustNew("usr_"), f.tenantInfo.OrgID, f.tenantInfo.BuID))
}

func TestRejectClosesAFailedExecutionAndWithdrawsTheVoid(t *testing.T) {
	t.Parallel()

	f := newRejectFixture(t, invoiceadjustment.StatusExecutionFailed, invoiceadjustment.KindFullReversal)
	f.expectLockedRead()
	f.repo.EXPECT().
		UpdateAdjustment(mock.Anything, mock.MatchedBy(func(entity *invoiceadjustment.InvoiceAdjustment) bool {
			return entity.Status == invoiceadjustment.StatusRejected &&
				entity.ApprovalStatus == invoiceadjustment.ApprovalStatusRejected &&
				entity.RejectionReason == "Customer accepted the invoice" &&
				strings.HasPrefix(entity.IdempotencyKey, "rejected:"+f.adjustment.ID.String())
		})).
		RunAndReturn(func(_ context.Context, entity *invoiceadjustment.InvoiceAdjustment) (*invoiceadjustment.InvoiceAdjustment, error) {
			return entity, nil
		}).
		Once()
	f.invoices.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *invoice.Invoice) bool {
			return entity.Status == invoice.StatusPosted &&
				entity.VoidReason == "" &&
				entity.VoidDisposition == ""
		})).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			return entity, nil
		}).
		Once()

	rejected, err := f.reject(t)

	require.NoError(t, err)
	assert.Equal(t, invoiceadjustment.StatusRejected, rejected.Status)
	assert.Equal(t, 1, f.db.transactions)
}

func TestRejectLeavesTheInvoiceAloneForOtherKinds(t *testing.T) {
	t.Parallel()

	f := newRejectFixture(t, invoiceadjustment.StatusPendingApproval, invoiceadjustment.KindCreditOnly)
	f.expectLockedRead()
	f.repo.EXPECT().
		UpdateAdjustment(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoiceadjustment.InvoiceAdjustment) (*invoiceadjustment.InvoiceAdjustment, error) {
			return entity, nil
		}).
		Once()

	_, err := f.reject(t)

	require.NoError(t, err)
	f.invoices.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestRejectRefusesWhatAlreadyRanOrClosed(t *testing.T) {
	t.Parallel()

	for _, status := range []invoiceadjustment.Status{
		invoiceadjustment.StatusExecuted,
		invoiceadjustment.StatusRejected,
		invoiceadjustment.StatusDraft,
		invoiceadjustment.StatusExecuting,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			f := newRejectFixture(t, status, invoiceadjustment.KindFullReversal)
			f.expectLockedRead()

			_, err := f.reject(t)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "may be rejected")
			f.repo.AssertNotCalled(t, "UpdateAdjustment", mock.Anything, mock.Anything)
		})
	}
}

func TestRetiredIdempotencyKeyStaysWithinTheColumn(t *testing.T) {
	t.Parallel()

	adjustment := &invoiceadjustment.InvoiceAdjustment{
		ID:             pulid.MustNew("iadj_"),
		IdempotencyKey: strings.Repeat("k", 200),
	}
	adjustment.Reject(pulid.MustNew("usr_"), 1_700_000_000, "no")

	assert.Len(t, adjustment.IdempotencyKey, 200)
	assert.True(t, strings.HasPrefix(adjustment.IdempotencyKey, "rejected:"+adjustment.ID.String()+":"))
}
