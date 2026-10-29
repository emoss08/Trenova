//go:build integration

package shipmentpredicate

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type carrierFilterMove struct {
	MoveID         pulid.ID
	ShipmentID     pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func TestCarrierIDsFilter_MatchesShipmentsTenderedToTheCarrier(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sys := dbscope.WithSystem(ctx, "test")
	moves, err := keel.QueryAll(sys, db, pgx.RowToStructByPos[carrierFilterMove], `
		SELECT sm.id, sm.shipment_id, sm.organization_id, sm.business_unit_id
		FROM shipment_moves AS sm
		WHERE sm.status != 'Canceled'
			AND NOT EXISTS (
				SELECT 1 FROM carrier_assignments AS ca
				WHERE ca.shipment_move_id = sm.id
					AND ca.organization_id = sm.organization_id
					AND ca.business_unit_id = sm.business_unit_id
			)
		ORDER BY sm.organization_id, sm.business_unit_id, sm.shipment_id
		LIMIT 2`)
	require.NoError(t, err)
	require.Len(t, moves, 2)
	require.NotEqual(t, moves[0].ShipmentID, moves[1].ShipmentID)

	carrierID, err := keel.QueryValue[string](sys, db,
		"SELECT id FROM carriers WHERE organization_id = $1 ORDER BY id LIMIT 1",
		moves[0].OrganizationID.String(),
	)
	require.NoError(t, err)
	require.NotEmpty(t, carrierID)

	tender := func(moveIndex int, status shipment.CarrierAssignmentStatus) {
		_, tErr := keel.Exec(sys, db, `
			INSERT INTO carrier_assignments (
				id, business_unit_id, organization_id, shipment_move_id, carrier_id, status,
				base_rate, base_amount, total_cost
			) VALUES ($1, $2, $3, $4, $5, $6, 1000, 1000, 1000)`,
			pulid.MustNew("casn_").String(),
			moves[moveIndex].BusinessUnitID.String(),
			moves[moveIndex].OrganizationID.String(),
			moves[moveIndex].MoveID.String(),
			carrierID,
			string(status),
		)
		require.NoError(t, tErr)
	}
	tender(0, shipment.CarrierAssignmentStatusConfirmed)
	tender(1, shipment.CarrierAssignmentStatusCanceled)

	tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: moves[0].OrganizationID,
		BusinessUnitID: moves[0].BusinessUnitID,
	})
	sp := shipmentdb.Shipments
	matched, err := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
		Where(OptionPredicates(&shipmentports.ShipmentOptions{
			CarrierIDs: []pulid.ID{pulid.ID(carrierID)},
		})...))
	require.NoError(t, err)

	assert.Contains(t, matched, moves[0].ShipmentID)
	assert.NotContains(
		t,
		matched,
		moves[1].ShipmentID,
		"a canceled tender is not the carrier's load",
	)
}
