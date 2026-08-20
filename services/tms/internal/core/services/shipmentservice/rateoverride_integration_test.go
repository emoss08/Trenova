//go:build integration

package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A hand-set rate must survive the trip through the calculator — that is the
// entire point of an override — and clearing it must hand pricing back to the
// engine.
func TestSetRateOverrideIntegration_AppliesAndClears(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, _, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-RATE-OVERRIDE"

	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	created, err := svc.Create(ctx, entity, actor)
	require.NoError(t, err)

	engineRated := created.FreightChargeAmount

	override := decimal.NewFromFloat(999.99)
	overridden, err := svc.SetRateOverride(ctx, &services.SetRateOverrideRequest{
		TenantInfo: tenantInfo,
		ShipmentID: created.ID,
		Amount:     decimal.NewNullDecimal(override),
		Reason:     "Negotiated spot rate",
	}, actor)
	require.NoError(t, err)

	assert.True(t, override.Equal(overridden.FreightChargeAmount.Decimal),
		"the hand-set rate must become the linehaul")
	require.True(t, overridden.RateOverrideAmount.Valid)
	require.NotNil(t, overridden.RateOverrideByID)
	assert.Equal(t, data.User.ID, *overridden.RateOverrideByID)
	require.NotNil(t, overridden.RateOverrideAt)

	persisted, err := shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	assert.True(t, override.Equal(persisted.FreightChargeAmount.Decimal))
	assert.True(t, persisted.RateOverrideAmount.Valid)

	cleared, err := svc.SetRateOverride(ctx, &services.SetRateOverrideRequest{
		TenantInfo: tenantInfo,
		ShipmentID: created.ID,
		Clear:      true,
	}, actor)
	require.NoError(t, err)

	assert.False(t, cleared.RateOverrideAmount.Valid, "clearing removes the override")
	assert.True(t, engineRated.Decimal.Equal(cleared.FreightChargeAmount.Decimal),
		"the engine's own answer returns once the override is gone")
}

// A locked override keeps its number through re-rating paths: the lock is what
// protects an invoiced shipment from a stop-time edit repricing it.
func TestSetRateOverrideIntegration_LockHoldsThroughAPlainSave(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, _, _, tenantInfo, fixture, data := newIntegrationShipmentService(t, ctx, db)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-RATE-OVERRIDE-LOCK"

	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	created, err := svc.Create(ctx, entity, actor)
	require.NoError(t, err)

	override := decimal.NewFromFloat(1234.56)
	overridden, err := svc.SetRateOverride(ctx, &services.SetRateOverrideRequest{
		TenantInfo: tenantInfo,
		ShipmentID: created.ID,
		Amount:     decimal.NewNullDecimal(override),
		Reason:     "Invoiced at the agreed spot rate",
		RateLocked: true,
	}, actor)
	require.NoError(t, err)
	assert.True(t, overridden.RateLocked)
	assert.True(t, override.Equal(overridden.FreightChargeAmount.Decimal))

	// A plain save carries neither the override nor the lock — an older client
	// round-tripping the shipment — and must not disturb either.
	resaved, err := svc.Update(ctx, overridden, actor)
	require.NoError(t, err)

	assert.True(t, resaved.RateLocked, "a plain save cannot unlock the rate")
	assert.True(t, override.Equal(resaved.FreightChargeAmount.Decimal),
		"a plain save cannot reprice a locked shipment")
	assert.True(t, resaved.RateOverrideAmount.Valid)
}
