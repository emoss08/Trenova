package billingqueueservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// The preview runs against mocks with no UpdateDerivedState expectation, so
// a save during it fails the test: that is what holds it to reading.
func TestPreviewUpdateCharges_IsWhatUpdateChargesSaves(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	h.expectItem(f.acmeItem)
	actor := testActor(f.tenantInfo)

	request := func() *services.UpdateChargesRequest {
		return &services.UpdateChargesRequest{
			ItemID:            f.acmeItem.ID,
			TenantInfo:        f.tenantInfo,
			AdditionalCharges: editedDetention(f, "1200"),
		}
	}

	preview, err := h.svc.PreviewUpdateCharges(t.Context(), request(), actor)
	require.NoError(t, err)

	var saved *shipment.Shipment
	h.shipmentRepo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, shp *shipment.Shipment) (*shipment.Shipment, error) {
			saved = shp
			return shp, nil
		}).
		Once()

	_, err = h.svc.UpdateCharges(t.Context(), request(), actor)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.True(t, preview.After.TotalChargeAmount.Decimal.Equal(saved.TotalChargeAmount.Decimal))
	assert.True(t, preview.After.OtherChargeAmount.Decimal.Equal(saved.OtherChargeAmount.Decimal))
	require.Len(t, preview.After.AdditionalCharges, len(saved.AdditionalCharges))
	for i := range saved.AdditionalCharges {
		assert.True(t,
			preview.After.AdditionalCharges[i].Amount.Equal(saved.AdditionalCharges[i].Amount))
		assert.Equal(t,
			saved.AdditionalCharges[i].IsDetention, preview.After.AdditionalCharges[i].IsDetention)
	}
	assert.False(t,
		preview.Before.TotalChargeAmount.Decimal.Equal(preview.After.TotalChargeAmount.Decimal),
		"the preview keeps the totals the edit replaces")
}

func TestPreviewUpdateCharges_RefusesARepricing(t *testing.T) {
	t.Parallel()

	h := newReassignHarness(t)
	f := h.f
	rate := decimal.NewFromInt(2000)

	_, err := h.svc.PreviewUpdateCharges(t.Context(), &services.UpdateChargesRequest{
		ItemID:     f.acmeItem.ID,
		TenantInfo: f.tenantInfo,
		BaseRate:   &rate,
	}, testActor(f.tenantInfo))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be previewed")
}
