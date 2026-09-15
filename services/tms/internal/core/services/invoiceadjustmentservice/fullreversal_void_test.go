package invoiceadjustmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func reversalFixture(t *testing.T) (*Service, *mocks.MockInvoiceRepository, *mocks.MockBillingQueueRepository, *invoiceadjustment.InvoiceAdjustment, *invoice.Invoice) {
	t.Helper()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	original := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		Number:             "INV-1",
		Status:             invoice.StatusPosted,
	}
	adjustment := &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		OriginalInvoiceID: original.ID,
		Kind:              invoiceadjustment.KindFullReversal,
		Reason:            "Duplicate billing",
	}
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	queueRepo := mocks.NewMockBillingQueueRepository(t)
	svc := &Service{
		l:                 zap.NewNop(),
		invoiceRepo:       invoiceRepo,
		billingQueueRepo:  queueRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "INV-2"},
	}

	return svc, invoiceRepo, queueRepo, adjustment, original
}

func TestVoidReversedInvoiceDefaultsToDoNotRebillWithTheAdjustmentReason(t *testing.T) {
	t.Parallel()

	svc, invoiceRepo, queueRepo, adjustment, original := reversalFixture(t)
	actor := testutil.NewSessionActor(pulid.MustNew("usr_"), original.OrganizationID, original.BusinessUnitID)

	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.ID == original.ID &&
				updated.Status == invoice.StatusVoided &&
				updated.VoidedByAdjustmentID == adjustment.ID &&
				updated.VoidDisposition == invoice.VoidDispositionDoNotRebill &&
				updated.VoidReason == "Duplicate billing" &&
				updated.VoidedByID == actor.UserID &&
				updated.VoidedAt != nil && *updated.VoidedAt == 1_700_000_000
		})).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()
	queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.MatchedBy(func(req *repositories.ReleaseForInvoiceRequest) bool {
			return !req.Rebill &&
				req.InvoiceID == original.ID &&
				req.AnchorItemID == original.BillingQueueItemID &&
				req.CanceledAt == 1_700_000_000
		})).
		Return([]*billingqueue.BillingQueueItem{}, nil).
		Once()

	err := svc.voidReversedInvoice(t.Context(), adjustment, original, actor, 1_700_000_000)

	require.NoError(t, err)
	assert.Equal(t, invoice.StatusVoided, original.Status, "the caller's copy follows the write")
	assert.Equal(t, adjustment.ID, original.VoidedByAdjustmentID)
}

func TestVoidReversedInvoiceHonoursTheRequestedDispositionAndReason(t *testing.T) {
	t.Parallel()

	svc, invoiceRepo, queueRepo, adjustment, original := reversalFixture(t)
	original.VoidDisposition = invoice.VoidDispositionRebill
	original.VoidReason = "Rebill to the right customer"
	actor := testutil.NewSessionActor(pulid.MustNew("usr_"), original.OrganizationID, original.BusinessUnitID)

	invoiceRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.VoidDisposition == invoice.VoidDispositionRebill &&
				updated.VoidReason == "Rebill to the right customer"
		})).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()
	queueRepo.EXPECT().
		ReleaseForInvoice(mock.Anything, mock.MatchedBy(func(req *repositories.ReleaseForInvoiceRequest) bool {
			if !req.Rebill || req.RenumberFn == nil {
				return false
			}
			number, err := req.RenumberFn(t.Context(), billingqueue.BillTypeInvoice)
			return err == nil && number == "INV-2"
		})).
		Return([]*billingqueue.BillingQueueItem{}, nil).
		Once()

	require.NoError(t, svc.voidReversedInvoice(t.Context(), adjustment, original, actor, 1_700_000_000))
}

func TestVoidReversedInvoiceIsIdempotentAndGuardsTheTransition(t *testing.T) {
	t.Parallel()

	svc, _, _, adjustment, original := reversalFixture(t)
	actor := testutil.NewSessionActor(pulid.MustNew("usr_"), original.OrganizationID, original.BusinessUnitID)

	original.Status = invoice.StatusVoided
	require.NoError(t, svc.voidReversedInvoice(t.Context(), adjustment, original, actor, 1), "already voided is a no-op")

	original.Status = invoice.Status("Archived")
	err := svc.voidReversedInvoice(t.Context(), adjustment, original, actor, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be voided")
}
