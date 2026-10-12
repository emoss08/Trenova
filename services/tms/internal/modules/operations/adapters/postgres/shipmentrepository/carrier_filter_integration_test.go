//go:build integration

package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCarrierIDsFilter_MatchesShipmentsTenderedToTheCarrier(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment, Force: true})
	require.NoError(t, err)

	var moves []struct {
		MoveID         pulid.ID `bun:"move_id"`
		ShipmentID     pulid.ID `bun:"shipment_id"`
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewRaw(`
		SELECT sm.id AS move_id, sm.shipment_id, sm.organization_id, sm.business_unit_id
		FROM shipment_moves AS sm
		WHERE sm.status != 'Canceled'
			AND NOT EXISTS (
				SELECT 1 FROM carrier_assignments AS ca
				WHERE ca.shipment_move_id = sm.id
					AND ca.organization_id = sm.organization_id
					AND ca.business_unit_id = sm.business_unit_id
			)
		ORDER BY sm.organization_id, sm.business_unit_id, sm.shipment_id
		LIMIT 2`).Scan(ctx, &moves))
	require.Len(t, moves, 2)
	require.NotEqual(t, moves[0].ShipmentID, moves[1].ShipmentID)

	var carrierID pulid.ID
	require.NoError(t, db.NewRaw(
		"SELECT id FROM carriers WHERE organization_id = ? ORDER BY id LIMIT 1",
		moves[0].OrganizationID,
	).Scan(ctx, &carrierID))
	require.False(t, carrierID.IsNil())

	tender := func(moveIndex int, status shipment.CarrierAssignmentStatus) {
		_, err := db.NewRaw(`
			INSERT INTO carrier_assignments (
				id, business_unit_id, organization_id, shipment_move_id, carrier_id, status,
				base_rate, base_amount, total_cost
			) VALUES (?, ?, ?, ?, ?, ?, 1000, 1000, 1000)`,
			pulid.MustNew("casn_"),
			moves[moveIndex].BusinessUnitID,
			moves[moveIndex].OrganizationID,
			moves[moveIndex].MoveID,
			carrierID,
			status,
		).Exec(ctx)
		require.NoError(t, err)
	}
	tender(0, shipment.CarrierAssignmentStatusConfirmed)
	tender(1, shipment.CarrierAssignmentStatusCanceled)

	var matched []pulid.ID
	q := db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Column("sp.id").
		Where("sp.organization_id = ?", moves[0].OrganizationID).
		Where("sp.business_unit_id = ?", moves[0].BusinessUnitID)
	q = applyShipmentOptionFilters(q, db, shipmentports.ShipmentOptions{
		CarrierIDs: []pulid.ID{carrierID},
	})
	require.NoError(t, q.Scan(ctx, &matched))

	assert.Contains(t, matched, moves[0].ShipmentID)
	assert.NotContains(t, matched, moves[1].ShipmentID, "a canceled tender is not the carrier's load")
}
