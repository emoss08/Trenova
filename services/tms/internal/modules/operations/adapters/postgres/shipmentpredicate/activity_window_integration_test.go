//go:build integration

package shipmentpredicate_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type activityWindowShipment struct {
	ShipmentID     pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

// A day far past any seeded appointment, so no seeded load falls inside it by accident.
const (
	awWindowStart = int64(4_102_444_800)
	awWindowEnd   = awWindowStart + 86_400
	awHour        = int64(3_600)
	awDay         = int64(86_400)
)

func TestActivityWindow_KeepsLoadsWhoseLiveStopsSpanTheWindow(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	sys := dbscope.WithSystem(ctx, "test")

	shipments, err := keel.QueryAll(sys, db, pgx.RowToStructByPos[activityWindowShipment], `
		WITH tenant AS (
			SELECT organization_id, business_unit_id FROM shipments
			GROUP BY organization_id, business_unit_id
			ORDER BY count(*) DESC, organization_id
			LIMIT 1
		)
		SELECT s.id, s.organization_id, s.business_unit_id
		FROM shipments AS s
		JOIN tenant AS t
			ON t.organization_id = s.organization_id AND t.business_unit_id = s.business_unit_id
		WHERE (
				SELECT count(*) FROM shipment_moves AS sm
				WHERE sm.shipment_id = s.id
					AND sm.organization_id = s.organization_id
					AND sm.business_unit_id = s.business_unit_id
			) = 1
			AND (
				SELECT count(*) FROM stops AS st
				JOIN shipment_moves AS sm
					ON sm.id = st.shipment_move_id
					AND sm.organization_id = st.organization_id
					AND sm.business_unit_id = st.business_unit_id
				WHERE sm.shipment_id = s.id AND sm.status <> 'Canceled'
			) >= 2
		ORDER BY s.id
		LIMIT 5`)
	require.NoError(t, err)
	require.Len(t, shipments, 5, "the seed needs five single-move shipments with two stops in one tenant")

	// Every stop of the shipment takes the first stop's appointment; then the last stop takes its own.
	schedule := func(s activityWindowShipment, first, last int64, lastStatus string) {
		_, uErr := keel.Exec(sys, db, `
			UPDATE stops AS st
			SET scheduled_window_start = $2, scheduled_window_end = $2 + 3600, status = 'New'
			FROM shipment_moves AS sm
			WHERE sm.id = st.shipment_move_id
				AND sm.organization_id = st.organization_id
				AND sm.business_unit_id = st.business_unit_id
				AND sm.shipment_id = $1`,
			s.ShipmentID.String(), first,
		)
		require.NoError(t, uErr)
		_, uErr = keel.Exec(sys, db, `
			UPDATE stops SET scheduled_window_start = $2, scheduled_window_end = $2 + 3600, status = $3
			WHERE id = (
				SELECT st.id FROM stops AS st
				JOIN shipment_moves AS sm
					ON sm.id = st.shipment_move_id
					AND sm.organization_id = st.organization_id
					AND sm.business_unit_id = st.business_unit_id
				WHERE sm.shipment_id = $1
				ORDER BY st.sequence DESC, st.id DESC
				LIMIT 1
			)`,
			s.ShipmentID.String(), last, lastStatus,
		)
		require.NoError(t, uErr)
	}

	inside, spanning, before, after, canceledInside := shipments[0], shipments[1], shipments[2], shipments[3], shipments[4]
	schedule(inside, awWindowStart+awHour, awWindowStart+2*awHour, "New")
	schedule(spanning, awWindowStart-awDay, awWindowEnd+awDay, "New")
	schedule(before, awWindowStart-3*awDay, awWindowStart-2*awDay, "New")
	schedule(after, awWindowEnd+2*awDay, awWindowEnd+3*awDay, "New")
	schedule(canceledInside, awWindowStart-5*awDay, awWindowStart+awHour, "Canceled")

	tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: inside.OrganizationID,
		BusinessUnitID: inside.BusinessUnitID,
	})
	sp := shipmentdb.Shipments
	matched, err := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
		Where(shipmentpredicate.OptionPredicates(&shipmentports.ShipmentOptions{
			ActivityWindowStart: awWindowStart,
			ActivityWindowEnd:   awWindowEnd,
		})...))
	require.NoError(t, err)

	assert.Contains(t, matched, inside.ShipmentID, "a stop inside the window")
	assert.Contains(t, matched, spanning.ShipmentID, "picks up before the window and delivers after it")
	assert.NotContains(t, matched, before.ShipmentID, "finished before the window")
	assert.NotContains(t, matched, after.ShipmentID, "starts after the window")
	assert.NotContains(t, matched, canceledInside.ShipmentID, "only a canceled stop reaches the window")
}
