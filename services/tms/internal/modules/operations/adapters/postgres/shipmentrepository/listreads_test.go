package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentOptionPredicates_EachPartyGroupMatchesAnyOfItsReadings(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	commodityID := pulid.MustNew("com_")
	orderID := pulid.MustNew("ord_")
	preds := shipmentOptionPredicates(&shipmentports.ShipmentOptions{
		PartyGroups: []shipmentports.ShipmentPartyGroup{
			{CustomerIDs: []pulid.ID{customerID}, CommodityIDs: []pulid.ID{commodityID}},
			{OrderIDs: []pulid.ID{orderID}},
			{},
		},
	})
	require.Len(t, preds, 2, "an empty group narrows nothing")

	sql, args := renderShipments(t, preds...)
	assert.Contains(t, sql, `"customer_id" = ANY(`)
	assert.Contains(t, sql, `"shipment_commodities"`)
	assert.Contains(t, sql, " OR ")
	assert.Contains(t, sql, `"order_id" = ANY(`)
	assert.Contains(t, args, []string{customerID.String()})
	assert.Contains(t, args, []string{orderID.String()})
}

func TestShipmentOptionPredicates_ActiveHoldsNarrowToHeldOrUnheldLoads(t *testing.T) {
	t.Parallel()

	held := true
	sql, args := renderShipments(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{
		ActiveHold:      &held,
		HoldReasonCodes: []string{" CREDIT_HOLD ", "credit_hold", ""},
	})...)
	assert.Contains(t, sql, `"shipment_holds"`)
	assert.Contains(t, sql, `"released_at" IS NULL`)
	assert.Contains(t, sql, " IN (SELECT ")
	assert.Contains(t, sql, "lower(")
	assert.Contains(t, args, []string{"credit_hold"})

	unheld := false
	sql, _ = renderShipments(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{
		ActiveHold: &unheld,
	})...)
	assert.Contains(t, sql, " NOT IN (SELECT ")
	assert.NotContains(t, sql, "lower(")

	assert.Empty(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{}))
}

func TestShipmentLoads_IncludeCommoditiesLoadsWhatEachLoadCarries(t *testing.T) {
	t.Parallel()

	listed := shipmentports.ShipmentOptions{IncludeCustomer: true, IncludeRoute: true}
	assert.Len(t, shipmentLoads(&listed), 3)

	listed.IncludeCommodities = true
	assert.Len(t, shipmentLoads(&listed), 4)
	assert.Len(t, shipmentLoads(&shipmentports.ShipmentOptions{IncludeCommodities: true}), 1)
	assert.Empty(t, shipmentLoads(&shipmentports.ShipmentOptions{}))
}

func TestOrderReferenceQuery_SearchesTheOrdersVectorNewestFirst(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, orderReferenceQuery("77Q-4182", orderReferenceLimit(0)))

	assert.Contains(t, sql, `"orders"`)
	assert.Contains(t, sql, "@@")
	assert.Contains(t, sql, "coalesce(")
	assert.Contains(t, sql, "ORDER BY")
	assert.Contains(t, args, "77Q-4182")
	assert.Equal(t, maxOrderReferenceMatches, orderReferenceLimit(1_000))
	assert.Equal(t, 3, orderReferenceLimit(3))
}

func TestActiveHoldsQuery_ReadsTheHoldsInForceForThePage(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("shp_")
	second := pulid.MustNew("shp_")
	ids := holdShipmentIDs([]pulid.ID{first, pulid.Nil, first, second})
	assert.Equal(t, []pulid.ID{first, second}, ids)

	sql, args := keeltest.Render(t, activeHoldsQuery(ids))
	assert.Contains(t, sql, `"shipment_holds"`)
	assert.Contains(t, sql, `"released_at" IS NULL`)
	assert.Contains(t, sql, `"hold_reasons"`)
	assert.Contains(t, args, []string{first.String(), second.String()})

	many := make([]pulid.ID, 0, maxActiveHoldShipments+5)
	for range maxActiveHoldShipments + 5 {
		many = append(many, pulid.MustNew("shp_"))
	}
	assert.Len(t, holdShipmentIDs(many), maxActiveHoldShipments)
}
