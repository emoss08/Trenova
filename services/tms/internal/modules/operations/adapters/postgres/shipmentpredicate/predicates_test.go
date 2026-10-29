package shipmentpredicate

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentOptionPredicates_ActivityWindowAddsOverlapPredicate(t *testing.T) {
	t.Parallel()

	sql, args := renderShipments(t, OptionPredicates(&shipmentports.ShipmentOptions{
		ActivityWindowStart: 1_700_000_000,
		ActivityWindowEnd:   1_700_086_400,
	})...)

	assert.Contains(t, sql, "EXISTS (SELECT 1 FROM \"shipment_moves\"")
	assert.Contains(t, sql, `"scheduled_window_start" > $`)
	assert.Contains(t, sql, `coalesce(s2."scheduled_window_end", s2."scheduled_window_start") >= $`)
	assert.Contains(t, args, int64(1_700_086_400))
	assert.Contains(t, args, int64(1_700_000_000))
	assert.Contains(t, sql, `"status" <> 'Canceled'`)
	assert.NotContains(t, args, "Canceled")
}

func TestShipmentOptionPredicates_NoOptionsAddNothing(t *testing.T) {
	t.Parallel()

	assert.Empty(t, OptionPredicates(&shipmentports.ShipmentOptions{}))
	require.False(t, (&shipmentports.ShipmentOptions{ActivityWindowStart: 100}).HasActivityWindow())
	require.False(t, (&shipmentports.ShipmentOptions{ActivityWindowEnd: 100}).HasActivityWindow())
}

func TestShipmentOptionPredicates_AssetFiltersMatchLiveAssignments(t *testing.T) {
	t.Parallel()

	tractorID := pulid.MustNew("tr_")
	workerID := pulid.MustNew("wrk_")
	trailerID := pulid.MustNew("trl_")
	carrierID := pulid.MustNew("car_")

	sql, args := renderShipments(t, OptionPredicates(&shipmentports.ShipmentOptions{
		TractorIDs: []pulid.ID{tractorID},
		WorkerIDs:  []pulid.ID{workerID},
		TrailerIDs: []pulid.ID{trailerID},
		CarrierIDs: []pulid.ID{carrierID},
	})...)

	assert.Contains(t, sql, `"assignments"`)
	assert.Contains(t, sql, `"carrier_assignments"`)
	assert.Contains(t, sql, `"archived_at" IS NULL`)
	assert.Contains(t, sql, `"primary_worker_id" = ANY(`)
	assert.Contains(t, sql, `"secondary_worker_id" = ANY(`)
	assert.Contains(t, sql, `"tractor_id" = ANY(`)
	assert.Contains(t, sql, `"trailer_id" = ANY(`)
	assert.Contains(t, sql, `"carrier_id" = ANY(`)
	assert.Contains(t, args, []string{tractorID.String()})
	assert.Contains(t, args, []string{carrierID.String()})
}

func TestShipmentOptionPredicates_IDsAndStatusesMatchThoseShipments(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	sql, args := renderShipments(t, OptionPredicates(&shipmentports.ShipmentOptions{
		IDs:      []pulid.ID{id},
		Statuses: []shipment.Status{shipment.StatusNew},
	})...)

	assert.Contains(t, sql, `s."id" = ANY(`)
	assert.Contains(t, sql, `s."status" IN ('New')`)
	assert.Contains(t, args, []string{id.String()})
}

func TestShipmentOptionPredicates_BillingTransferEligibleKeepsQueuedShipmentsOut(t *testing.T) {
	t.Parallel()

	sql, _ := renderShipments(t, OptionPredicates(&shipmentports.ShipmentOptions{
		BillingTransferEligible: true,
	})...)

	assert.Contains(t, sql, `s."billing_transfer_status"`)
	assert.Contains(t, sql, `s."status" IN ('Completed', 'ReadyToInvoice')`)
}

func TestUnassignedShipmentPredicate_ExcludesLiveAssignments(t *testing.T) {
	t.Parallel()

	sql, _ := renderShipments(t, UnassignedPredicate())

	assert.Contains(t, sql, "NOT EXISTS")
	assert.Contains(t, sql, `"assignments"`)
	assert.Contains(t, sql, `"archived_at" IS NULL`)
}

func TestShipmentOptionPredicates_EachPartyGroupMatchesAnyOfItsReadings(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	commodityID := pulid.MustNew("com_")
	orderID := pulid.MustNew("ord_")
	preds := OptionPredicates(&shipmentports.ShipmentOptions{
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
	sql, args := renderShipments(t, OptionPredicates(&shipmentports.ShipmentOptions{
		ActiveHold:      &held,
		HoldReasonCodes: []string{" CREDIT_HOLD ", "credit_hold", ""},
	})...)
	assert.Contains(t, sql, `"shipment_holds"`)
	assert.Contains(t, sql, `"released_at" IS NULL`)
	assert.Contains(t, sql, " IN (SELECT ")
	assert.Contains(t, sql, "lower(")
	assert.Contains(t, args, []string{"credit_hold"})

	unheld := false
	sql, _ = renderShipments(t, OptionPredicates(&shipmentports.ShipmentOptions{
		ActiveHold: &unheld,
	})...)
	assert.Contains(t, sql, " NOT IN (SELECT ")
	assert.NotContains(t, sql, "lower(")

	assert.Empty(t, OptionPredicates(&shipmentports.ShipmentOptions{}))
}
