//go:build integration

package shipmentlistreadrepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listReadShipment struct {
	ShipmentID     pulid.ID
	CustomerID     pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func TestListReads_FindTheOrdersPOAndTheHoldsInForce(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sys := dbscope.WithSystem(ctx, "test")
	shipments, err := keel.QueryAll(sys, db, pgx.RowToStructByPos[listReadShipment], `
		SELECT s.id, s.customer_id, s.organization_id, s.business_unit_id
		FROM shipments AS s
		WHERE s.status <> 'Canceled'
		ORDER BY s.organization_id, s.business_unit_id, s.created_at, s.id
		LIMIT 2`)
	require.NoError(t, err)
	require.Len(t, shipments, 2)
	require.Equal(t, shipments[0].OrganizationID, shipments[1].OrganizationID)

	held, free := shipments[0], shipments[1]
	orderID := pulid.MustNew("ord_")
	_, err = keel.Exec(sys, db, `
		INSERT INTO orders (id, business_unit_id, organization_id, customer_id, order_number,
			po_number)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		orderID.String(), held.BusinessUnitID.String(), held.OrganizationID.String(),
		held.CustomerID.String(), "ORD-LISTREADS-1", "77Q-4182",
	)
	require.NoError(t, err)
	_, err = keel.Exec(sys, db,
		"UPDATE shipments SET order_id = $1, external_reference = $2 WHERE id = $3",
		orderID.String(), "LISTREADS-662193", held.ShipmentID.String(),
	)
	require.NoError(t, err)

	holdID := pulid.MustNew("shh_")
	_, err = keel.Exec(sys, db, `
		INSERT INTO shipment_holds (id, shipment_id, organization_id, business_unit_id, type,
			severity, reason_code, blocks_billing, started_at)
		VALUES ($1, $2, $3, $4, 'FinanceHold', 'Blocking', 'CREDIT_HOLD', true, 1800000000)`,
		holdID.String(), held.ShipmentID.String(), held.OrganizationID.String(),
		held.BusinessUnitID.String(),
	)
	require.NoError(t, err)

	tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: held.OrganizationID,
		BusinessUnitID: held.BusinessUnitID,
	})

	orders, err := keel.Into[shipmentports.OrderReferenceMatch](tenantCtx, db,
		orderReferenceQuery("77Q-4182", orderReferenceLimit(0)))
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, orderID, orders[0].OrderID)
	assert.Equal(t, "77Q-4182", orders[0].PONumber)

	sp := shipmentdb.Shipments
	byOrder, err := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
		Where(shipmentpredicate.OptionPredicates(&shipmentports.ShipmentOptions{
			PartyGroups: []shipmentports.ShipmentPartyGroup{{OrderIDs: []pulid.ID{orderID}}},
		})...))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{held.ShipmentID}, byOrder)

	byReference, err := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
		Where(listquery.MatchVector(sp.SearchVector, "LISTREADS-662193")))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{held.ShipmentID}, byReference,
		"the shipment's own words match its external reference")

	onHold := true
	heldIDs, err := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
		Where(shipmentpredicate.OptionPredicates(&shipmentports.ShipmentOptions{
			IDs:             []pulid.ID{held.ShipmentID, free.ShipmentID},
			ActiveHold:      &onHold,
			HoldReasonCodes: []string{"credit_hold"},
		})...))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{held.ShipmentID}, heldIDs)

	notHeld := false
	freeIDs, err := keel.Into[pulid.ID](tenantCtx, db, sp.Pick(sp.ID).
		Where(shipmentpredicate.OptionPredicates(&shipmentports.ShipmentOptions{
			IDs:        []pulid.ID{held.ShipmentID, free.ShipmentID},
			ActiveHold: &notHeld,
		})...))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{free.ShipmentID}, freeIDs)

	holds, err := activeHoldsQuery([]pulid.ID{held.ShipmentID, free.ShipmentID}).
		All(tenantCtx, db)
	require.NoError(t, err)
	require.Len(t, holds, 1)
	assert.Equal(t, holdID, holds[0].ID)

	older, newer := pulid.MustNew("shc_"), pulid.MustNew("shc_")
	_, err = keel.Exec(sys, db, `
		INSERT INTO shipment_comments (id, shipment_id, organization_id, business_unit_id,
			comment, source, created_at)
		VALUES ($1, $3, $4, $5, 'older note', 'System', 1800000000),
			($2, $3, $4, $5, 'newer note', 'System', 1800000600)`,
		older.String(), newer.String(), held.ShipmentID.String(),
		held.OrganizationID.String(), held.BusinessUnitID.String(),
	)
	require.NoError(t, err)

	newest, err := newestCommentsQuery([]pulid.ID{held.ShipmentID, free.ShipmentID}).
		All(tenantCtx, db)
	require.NoError(t, err)
	byShipment := make(map[pulid.ID]string, len(newest))
	for _, comment := range newest {
		byShipment[comment.ShipmentID] = comment.Comment
	}
	assert.Equal(t, "newer note", byShipment[held.ShipmentID],
		"each shipment's newest comment, in one read")

	otherTenant := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	foreign, err := keel.Into[shipmentports.OrderReferenceMatch](otherTenant, db,
		orderReferenceQuery("77Q-4182", orderReferenceLimit(0)))
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant never reads this order")
}
