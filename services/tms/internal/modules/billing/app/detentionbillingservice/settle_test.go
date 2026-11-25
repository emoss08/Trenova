package detentionbillingservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncharge"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSettler struct {
	requests []*detentionports.SettleDetentionChargesRequest
}

func (r *recordingSettler) SettleShipmentCharges(
	_ context.Context,
	req *detentionports.SettleDetentionChargesRequest,
) ([]detentioncharge.Change, error) {
	r.requests = append(r.requests, req)
	return nil, nil
}

func TestSettleCharges_HandsTheShipmentToTheSettler(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	settler := &recordingSettler{}
	f.svc.charges = settler
	req := &detentionports.SettleDetentionChargesRequest{
		TenantInfo: f.tenant,
		Shipment:   &shipment.Shipment{ID: f.shipmentID},
		Persist:    true,
	}

	require.NoError(t, f.svc.SettleCharges(t.Context(), req))

	assert.Equal(t, []*detentionports.SettleDetentionChargesRequest{req}, settler.requests)
}

func TestSettleCharges_DoesNothingWithoutDetentionOnThePlan(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	settler := &recordingSettler{}
	f.svc.charges = settler
	f.svc.features = plantest.Without(platformcatalog.FeatureDetention)

	require.NoError(t, f.svc.SettleCharges(t.Context(), &detentionports.SettleDetentionChargesRequest{
		TenantInfo: f.tenant,
		Shipment:   &shipment.Shipment{ID: f.shipmentID},
	}))

	assert.Empty(t, settler.requests)
}
