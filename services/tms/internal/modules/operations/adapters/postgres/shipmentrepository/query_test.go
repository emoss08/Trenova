package shipmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentOptionPredicates_ActivityWindowAddsOverlapPredicate(t *testing.T) {
	t.Parallel()

	sql, args := renderShipments(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{
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

	assert.Empty(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{}))
	require.False(t, (&shipmentports.ShipmentOptions{ActivityWindowStart: 100}).HasActivityWindow())
	require.False(t, (&shipmentports.ShipmentOptions{ActivityWindowEnd: 100}).HasActivityWindow())
}

func TestShipmentOptionPredicates_AssetFiltersMatchLiveAssignments(t *testing.T) {
	t.Parallel()

	tractorID := pulid.MustNew("tr_")
	workerID := pulid.MustNew("wrk_")
	trailerID := pulid.MustNew("trl_")
	carrierID := pulid.MustNew("car_")

	sql, args := renderShipments(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{
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
	sql, args := renderShipments(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{
		IDs:      []pulid.ID{id},
		Statuses: []shipment.Status{shipment.StatusNew},
	})...)

	assert.Contains(t, sql, `s."id" = ANY(`)
	assert.Contains(t, sql, `s."status" IN ('New')`)
	assert.Contains(t, args, []string{id.String()})
}

func TestShipmentOptionPredicates_BillingTransferEligibleKeepsQueuedShipmentsOut(t *testing.T) {
	t.Parallel()

	sql, _ := renderShipments(t, shipmentOptionPredicates(&shipmentports.ShipmentOptions{
		BillingTransferEligible: true,
	})...)

	assert.Contains(t, sql, `s."billing_transfer_status"`)
	assert.Contains(t, sql, `s."status" IN ('Completed', 'ReadyToInvoice')`)
}

func TestShipmentLoads_IncludeCustomerJoinsOnlyTheCustomer(t *testing.T) {
	t.Parallel()

	sp := shipmentdb.Shipments
	sql, _ := keeltest.Render(t, sp.Where(sp.ID.Eq(pulid.MustNew("shp_"))).
		Load(shipmentLoads(&shipmentports.ShipmentOptions{IncludeCustomer: true})...).
		Limit(1))

	assert.Contains(t, sql, `LEFT JOIN "customers"`)
	assert.NotContains(t, sql, `"service_types"`)
}

func TestShipmentLoads_ExpandLoadsEveryDetail(t *testing.T) {
	t.Parallel()

	sp := shipmentdb.Shipments
	sql, _ := keeltest.Render(t, sp.Where(sp.ID.Eq(pulid.MustNew("shp_"))).
		Load(shipmentLoads(&shipmentports.ShipmentOptions{ExpandShipmentDetails: true})...).
		Limit(1))

	for _, table := range []string{
		`"customers"`, `"service_types"`, `"shipment_types"`, `"formula_templates"`,
		`"equipment_types"`, `"users"`,
	} {
		assert.Contains(t, sql, table)
	}
}

func TestDelayedShipments_UsesThresholdAndExcludedStatuses(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, delayedShipments(30, 1_800_000_000, 0))

	assert.Contains(t, sql, `"stops" late_stp`)
	assert.Contains(t, sql, `"shipment_moves" late_sm`)
	assert.Contains(t, sql, `late_stp."actual_departure" IS NULL`)
	assert.Contains(t, sql, `ORDER BY s."id" ASC LIMIT 100`)
	assert.Contains(t, args, int64(30*60))
	assert.Contains(t, args, int64(1_800_000_000))
}

func TestAutoDelayedShipmentIDs_JoinsTheShipmentControl(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, autoDelayedShipmentIDs(1_800_000_000, scopedTenant).Limit(10))

	assert.Contains(t, sql, `"shipment_controls" auto_sc`)
	assert.Contains(t, sql, `auto_sc."auto_delay_shipments" IS TRUE`)
	assert.Contains(
		t,
		sql,
		`(CAST(coalesce(auto_sc."auto_delay_shipments_threshold", $4) AS bigint) * $5::bigint)`,
	)
	assert.Contains(t, args, int64(1_800_000_000))
}

func TestAutoCancelableShipments_UsesThreshold(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, autoCancelableShipments(2, 1_800_000_000, 25))

	assert.Contains(t, sql, `s."status" = 'New'`)
	assert.Contains(t, sql, `s."created_at" <= $3`)
	assert.Contains(t, sql, `LIMIT 25`)
	assert.Len(t, args, 3)
	assert.Equal(t, int64(1_800_000_000-2*secondsPerDay), args[2])
}

func TestControlledAutoCancelIDs_JoinsTheShipmentControl(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, controlledAutoCancelIDs(1_800_000_000, scopedTenant).Limit(10))

	assert.Contains(t, sql, `"shipment_controls" cancel_sc`)
	assert.Contains(t, sql, `cancel_sc."auto_cancel_shipments" IS TRUE`)
	assert.Contains(
		t,
		sql,
		`cancel_sp."created_at" <= ($3::bigint - (CAST(coalesce(`+
			`cancel_sc."auto_cancel_shipments_threshold", $4) AS bigint) * $5::bigint))`,
		"the day length is bigint so the threshold and the clock are not read as smallint",
	)
	assert.Contains(t, sql, `cancel_sp."status" = 'New'`)
	assert.Equal(t, int64(1_800_000_000), args[2])
	assert.Equal(t, secondsPerDay, args[4])
}

func TestPreviousRateOrigins_PinsTheFirstPickup(t *testing.T) {
	t.Parallel()

	origin := pulid.MustNew("loc_")
	sql, args := keeltest.Render(t, previousRateOrigins(origin))

	assert.Contains(t, sql, `first_move."sequence" = $`)
	assert.Contains(t, sql, `origin_stop."sequence" = $`)
	assert.Contains(t, args, origin.String())
}

func TestPreviousRateDestinations_PinsTheLastDelivery(t *testing.T) {
	t.Parallel()

	destination := pulid.MustNew("loc_")
	sql, args := keeltest.Render(t, previousRateDestinations(destination))

	assert.Contains(t, sql, `max(sm3."sequence")`)
	assert.Contains(t, sql, `max(stp3."sequence")`)
	assert.Contains(t, args, destination.String())
}

func TestUnassignedShipmentPredicate_ExcludesLiveAssignments(t *testing.T) {
	t.Parallel()

	sql, _ := renderShipments(t, unassignedShipmentPredicate())

	assert.Contains(t, sql, "NOT EXISTS")
	assert.Contains(t, sql, `"assignments"`)
	assert.Contains(t, sql, `"archived_at" IS NULL`)
}
