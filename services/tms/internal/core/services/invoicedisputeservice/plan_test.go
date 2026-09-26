package invoicedisputeservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (f *disputeFixture) readInvoice() {
	f.invoiceRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{ID: f.inv.ID, TenantInfo: f.tenantInfo}).
		Return(f.inv, nil).
		Once()
}

func TestPreviewOpenPlansTheCaseAndTheFlagWithoutWriting(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.readInvoice()
	f.repo.EXPECT().
		GetOpenByInvoiceID(mock.Anything, repositories.GetOpenInvoiceDisputeRequest{InvoiceID: f.inv.ID, TenantInfo: f.tenantInfo}).
		Return(nil, errortypes.NewNotFoundError("none")).
		Once()

	preview, err := f.svc.PreviewOpen(t.Context(), f.openRequest("40.50"), f.actor)
	require.NoError(t, err)

	assert.Nil(t, preview.Before)
	require.NotNil(t, preview.After)
	assert.Equal(t, invoice.DisputeCaseStatusOpen, preview.After.Status)
	assert.Equal(t, int64(4050), preview.After.DisputedAmountMinor)
	assert.Equal(t, "Rate on lane differs from contract", preview.After.Notes)
	assert.Equal(t, f.userID, preview.After.OpenedByID)
	assert.Equal(t, invoice.DisputeStatusNone, preview.InvoiceBefore.DisputeStatus)
	assert.Equal(t, invoice.DisputeStatusDisputed, preview.InvoiceAfter.DisputeStatus)
}

func TestPreviewOpenRefusesWhatOpenRefuses(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.readInvoice()
	f.repo.EXPECT().
		GetOpenByInvoiceID(mock.Anything, repositories.GetOpenInvoiceDisputeRequest{InvoiceID: f.inv.ID, TenantInfo: f.tenantInfo}).
		Return(f.openCase(), nil).
		Once()

	_, err := f.svc.PreviewOpen(t.Context(), f.openRequest("10"), f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has an open dispute")

	over := newDisputeFixture(t)
	over.readInvoice()
	_, err = over.svc.PreviewOpen(t.Context(), over.openRequest("500"), over.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the open balance")
}

func TestPreviewResolveClosesTheCaseAndClearsTheFlag(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.inv.DisputeStatus = invoice.DisputeStatusDisputed
	existing := f.openCase()
	adjustmentID := pulid.MustNew("iadj_")
	f.repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceDisputeByIDRequest{ID: existing.ID, TenantInfo: f.tenantInfo}).
		Return(existing, nil).
		Once()
	f.readInvoice()
	f.adjustmentRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceAdjustmentRequest{ID: adjustmentID, TenantInfo: f.tenantInfo}).
		Return(&invoiceadjustment.InvoiceAdjustment{
			ID:                adjustmentID,
			OriginalInvoiceID: f.inv.ID,
			Status:            invoiceadjustment.StatusExecuted,
		}, nil).
		Once()

	preview, err := f.svc.PreviewResolve(t.Context(), &servicesports.ResolveInvoiceDisputeRequest{
		DisputeID:              existing.ID,
		TenantInfo:             f.tenantInfo,
		Resolution:             invoice.DisputeResolutionCreditIssued,
		ResolutionAdjustmentID: adjustmentID,
		ResolutionNotes:        " Credited the fuel line ",
	}, f.actor)
	require.NoError(t, err)

	assert.Equal(t, invoice.DisputeCaseStatusOpen, preview.Before.Status)
	assert.Equal(t, invoice.DisputeCaseStatusResolved, preview.After.Status)
	assert.Equal(t, invoice.DisputeResolutionCreditIssued, preview.After.Resolution)
	assert.Equal(t, "Credited the fuel line", preview.After.ResolutionNotes)
	assert.Equal(t, invoice.DisputeStatusNone, preview.InvoiceAfter.DisputeStatus)
	assert.Equal(t, invoice.DisputeStatusDisputed, preview.InvoiceBefore.DisputeStatus)
}

func TestPreviewResolveRefusesACreditWithoutItsAdjustment(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	existing := f.openCase()
	f.repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceDisputeByIDRequest{ID: existing.ID, TenantInfo: f.tenantInfo}).
		Return(existing, nil).
		Once()
	f.readInvoice()

	_, err := f.svc.PreviewResolve(t.Context(), &servicesports.ResolveInvoiceDisputeRequest{
		DisputeID:  existing.ID,
		TenantInfo: f.tenantInfo,
		Resolution: invoice.DisputeResolutionWrittenOff,
	}, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "executed adjustment")
}

func TestPreviewWithdrawRefusesAClosedCaseBeforeReadingTheInvoice(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	closed := f.openCase()
	closed.Status = invoice.DisputeCaseStatusWithdrawn
	f.repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceDisputeByIDRequest{ID: closed.ID, TenantInfo: f.tenantInfo}).
		Return(closed, nil).
		Once()

	_, err := f.svc.PreviewWithdraw(t.Context(), &servicesports.WithdrawInvoiceDisputeRequest{
		DisputeID:  closed.ID,
		TenantInfo: f.tenantInfo,
	}, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only an open dispute can be withdrawn")
}

func TestPreviewWithdrawKeepsTheNotes(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	existing := f.openCase()
	f.repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceDisputeByIDRequest{ID: existing.ID, TenantInfo: f.tenantInfo}).
		Return(existing, nil).
		Once()
	f.readInvoice()

	preview, err := f.svc.PreviewWithdraw(t.Context(), &servicesports.WithdrawInvoiceDisputeRequest{
		DisputeID:  existing.ID,
		TenantInfo: f.tenantInfo,
		Notes:      "Customer paid in full",
	}, f.actor)
	require.NoError(t, err)
	assert.Equal(t, invoice.DisputeCaseStatusWithdrawn, preview.After.Status)
	assert.Equal(t, "Customer paid in full", preview.After.ResolutionNotes)
	assert.Equal(t, f.userID, preview.After.ResolvedByID)
}
