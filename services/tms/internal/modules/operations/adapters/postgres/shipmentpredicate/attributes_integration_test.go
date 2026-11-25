//go:build integration

package shipmentpredicate_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type attributeShipment struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type attributeStop struct {
	ShipmentID     pulid.ID
	MoveID         pulid.ID
	StopType       string
	WindowStart    int64
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func TestStatusChangedAt_TheDatabaseKeepsWhenTheStatusLastChanged(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sys := dbscope.WithSystem(ctx, "test")
	rows, err := keel.QueryAll(sys, db, pgx.RowToStructByPos[attributeShipment], `
		SELECT id, organization_id, business_unit_id FROM shipments
		WHERE status = 'New' ORDER BY id LIMIT 1`)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	shipmentID := rows[0].ID.String()

	_, err = keel.Exec(sys, db,
		"UPDATE shipments SET status_changed_at = 1 WHERE id = $1", shipmentID)
	require.NoError(t, err)
	kept, err := keel.QueryValue[int64](sys, db,
		"SELECT status_changed_at FROM shipments WHERE id = $1", shipmentID)
	require.NoError(t, err)
	assert.NotEqual(t, int64(1), kept, "a write to the column itself is ignored")

	_, err = keel.Exec(sys, db,
		"UPDATE shipments SET bol = 'BOL-AGE-1' WHERE id = $1", shipmentID)
	require.NoError(t, err)
	unchanged, err := keel.QueryValue[int64](sys, db,
		"SELECT status_changed_at FROM shipments WHERE id = $1", shipmentID)
	require.NoError(t, err)
	assert.Equal(t, kept, unchanged, "a write that keeps the status keeps its age")

	_, err = keel.Exec(sys, db,
		"UPDATE shipments SET status = 'Assigned' WHERE id = $1", shipmentID)
	require.NoError(t, err)
	changed, err := keel.QueryValue[int64](sys, db,
		"SELECT status_changed_at FROM shipments WHERE id = $1", shipmentID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, changed, kept, "a new status starts its age now")

	tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: rows[0].OrganizationID,
		BusinessUnitID: rows[0].BusinessUnitID,
	})
	sp := shipmentdb.Shipments
	recent, err := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
		Where(append(shipmentpredicate.OptionPredicates(&shipmentports.ShipmentOptions{
			InStatusSince: changed - 86400,
		}), sp.ID.Eq(pulid.ID(shipmentID)))...))
	require.NoError(t, err)
	assert.Empty(t, recent, "a load that changed status today has not sat a day")
}

func TestCoverageAndStopWindow_ReadTheLiveRecords(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sys := dbscope.WithSystem(ctx, "test")
	stops, err := keel.QueryAll(sys, db, pgx.RowToStructByPos[attributeStop], `
		SELECT sm.shipment_id, sm.id, st.type, st.scheduled_window_start,
			sm.organization_id, sm.business_unit_id
		FROM stops AS st
		JOIN shipment_moves AS sm
			ON sm.id = st.shipment_move_id
			AND sm.organization_id = st.organization_id
			AND sm.business_unit_id = st.business_unit_id
		JOIN shipments AS sp
			ON sp.id = sm.shipment_id
			AND sp.organization_id = sm.organization_id
			AND sp.business_unit_id = sm.business_unit_id
		WHERE st.type = 'Pickup'
			AND st.status <> 'Canceled'
			AND sm.status <> 'Canceled'
			AND st.scheduled_window_start > 0
			AND NOT sp.has_active_assignment
			AND NOT EXISTS (
				SELECT 1 FROM carrier_assignments AS ca
				WHERE ca.shipment_move_id = sm.id
					AND ca.organization_id = sm.organization_id
					AND ca.business_unit_id = sm.business_unit_id
			)
		ORDER BY sm.organization_id, sm.business_unit_id, sm.shipment_id
		LIMIT 1`)
	require.NoError(t, err)
	require.Len(t, stops, 1)
	stop := stops[0]

	tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: stop.OrganizationID,
		BusinessUnitID: stop.BusinessUnitID,
	})
	sp := shipmentdb.Shipments
	read := func(opts *shipmentports.ShipmentOptions) []pulid.ID {
		t.Helper()
		opts.IDs = []pulid.ID{stop.ShipmentID}
		ids, readErr := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
			Where(shipmentpredicate.OptionPredicates(opts)...))
		require.NoError(t, readErr)

		return ids
	}

	unassigned, assigned := true, false
	assert.Equal(t, []pulid.ID{stop.ShipmentID}, read(&shipmentports.ShipmentOptions{
		Unassigned: &unassigned,
	}))
	assert.Empty(t, read(&shipmentports.ShipmentOptions{Unassigned: &assigned}))

	inWindow := shipmentports.ShipmentStopPlace{
		Types:      []shipmentvocab.StopType{shipmentvocab.StopTypePickup},
		WindowFrom: stop.WindowStart - 3600,
		WindowTo:   stop.WindowStart + 3600,
	}
	assert.Equal(t, []pulid.ID{stop.ShipmentID}, read(&shipmentports.ShipmentOptions{
		StopPlaces: []shipmentports.ShipmentStopPlace{inWindow},
	}))
	later := inWindow
	later.WindowFrom = stop.WindowStart + 30*86400
	later.WindowTo = stop.WindowStart + 31*86400
	assert.NotContains(t, read(&shipmentports.ShipmentOptions{
		StopPlaces: []shipmentports.ShipmentStopPlace{later},
	}), stop.ShipmentID)

	carrierID, err := keel.QueryValue[string](sys, db,
		"SELECT id FROM carriers WHERE organization_id = $1 ORDER BY id LIMIT 1",
		stop.OrganizationID.String())
	require.NoError(t, err)
	_, err = keel.Exec(sys, db, `
		INSERT INTO carrier_assignments (
			id, business_unit_id, organization_id, shipment_move_id, carrier_id, status,
			base_rate, base_amount, total_cost
		) VALUES ($1, $2, $3, $4, $5, 'Confirmed', 1000, 1000, 1000)`,
		pulid.MustNew("casn_").String(), stop.BusinessUnitID.String(),
		stop.OrganizationID.String(), stop.MoveID.String(), carrierID,
	)
	require.NoError(t, err)

	assert.Empty(t, read(&shipmentports.ShipmentOptions{Unassigned: &unassigned}),
		"a load a carrier covers is not unassigned, though no driver is on it")
	assert.Equal(t, []pulid.ID{stop.ShipmentID}, read(&shipmentports.ShipmentOptions{
		Unassigned: &assigned,
	}))
}
