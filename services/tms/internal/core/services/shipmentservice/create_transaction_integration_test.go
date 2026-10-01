//go:build integration

package shipmentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingShipmentEventService struct {
	noopShipmentEventService
}

func (failingShipmentEventService) Record(
	_ context.Context,
	_ *services.RecordShipmentEventParams,
) error {
	return errors.New("event store unavailable")
}

func TestCreateIntegration_RollsBackEverythingWhenALaterStepFails(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentServiceWith(
		t,
		ctx,
		db,
		func(p *Params) { p.EventService = failingShipmentEventService{} },
	)
	configureShipmentControl(t, ctx, controlRepo, tenantInfo, func(sc *tenant.ShipmentControl) {
		sc.CheckHazmatSegregation = false
	})

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-ROLLBACK-ON-FAILURE"

	_, err := svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event store unavailable")

	moveCount, err := db.NewSelect().
		TableExpr("shipment_moves").
		Where("shipment_id = ?", entity.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, moveCount, "the moves are rolled back with the shipment")

	shipmentCount, err := db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where("id = ?", entity.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, shipmentCount, "the shipment row is rolled back with the rest")

	var orphanOrders int
	orphanOrders, err = db.NewSelect().
		TableExpr("orders AS o").
		Where("o.organization_id = ?", tenantInfo.OrgID).
		Where("NOT EXISTS (SELECT 1 FROM shipments AS s WHERE s.order_id = o.id)").
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, orphanOrders, "the auto-created order is rolled back too")
}
