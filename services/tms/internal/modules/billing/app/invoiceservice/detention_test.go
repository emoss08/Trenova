package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/detentionportsmock"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreatingAnInvoiceIsRefusedWhileADetentionChargeAwaitsApproval(t *testing.T) {
	t.Parallel()

	h := newApprovalHarness(t, perShipmentProfile(), false)
	billing := detentionportsmock.NewMockDetentionBillingService(t)
	billing.EXPECT().
		GuardShipments(mock.Anything, &detentionports.DetentionBillingHoldsRequest{
			TenantInfo:  h.tenantInfo,
			ShipmentIDs: []pulid.ID{h.shipmentID},
		}).
		Return(detention.BillingHoldError([]*detention.DetentionOccurrence{{
			ID:               pulid.MustNew("dto_"),
			ShipmentID:       h.shipmentID,
			Status:           detention.OccurrenceStatusPending,
			RequiresApproval: true,
		}})).
		Once()
	h.svc.detentionBilling = billing

	result, err := h.approve(t, &invoiceports.CreateInvoiceFromBillingQueueRequest{})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errortypes.IsBusinessError(err))
	billing.AssertNotCalled(t, "SyncInvoiceBilling", mock.Anything, mock.Anything)
}

func TestCreatingAnInvoiceMarksItsDetentionChargesBilled(t *testing.T) {
	t.Parallel()

	h := newApprovalHarness(t, perShipmentProfile(), true)
	billing := detentionportsmock.NewMockDetentionBillingService(t)
	billing.EXPECT().
		GuardShipments(mock.Anything, mock.Anything).
		Return(nil).
		Once()
	var synced *detentionports.SyncDetentionInvoiceBillingRequest
	billing.EXPECT().
		SyncInvoiceBilling(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *detentionports.SyncDetentionInvoiceBillingRequest) error {
			synced = req
			return nil
		}).
		Once()
	h.svc.detentionBilling = billing

	result, err := h.approve(t, &invoiceports.CreateInvoiceFromBillingQueueRequest{})

	require.NoError(t, err)
	require.NotNil(t, result.Invoice)
	require.NotNil(t, synced)
	assert.Equal(t, []pulid.ID{result.Invoice.ID}, synced.InvoiceIDs)
	assert.Equal(t, "INV-1001", synced.InvoiceNumber)
	assert.Equal(t, detentionports.DetentionBillingInvoiceCreated, synced.Event)
	assert.Equal(t, h.userID, synced.ActorUserID)
	assert.Equal(t, h.orgID, synced.TenantInfo.OrgID)
	assert.Equal(t, h.buID, synced.TenantInfo.BuID)
	assert.Equal(t, invoice.StatusDraft, result.Invoice.Status)
}
