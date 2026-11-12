package shipmentprojection

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestRefreshQuery_ReadsTheLastDeliveredArrivalByMoveAndStopSequence(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		refreshQuery(t.Context(), shipmentdb.Shipments.ID.In(pulid.ID("shp_1"))),
	)

	assert.Contains(t, sql, `s."last_delivery_arrival_at" IS DISTINCT FROM (SELECT prj_stp."actual_arrival"`)
	assert.Contains(t, sql, `prj_stp."business_unit_id" = $1 AND prj_stp."organization_id" = $2`)
	assert.Contains(t, sql, `JOIN "shipment_moves" prj_dsm`)
	assert.Contains(t, sql, `prj_dsm."shipment_id" = s."id"`)
	assert.Contains(t, sql, `prj_dsm."status" <> 'Canceled'`)
	assert.Contains(t, sql, `prj_stp."type" IN ('Delivery', 'SplitDelivery')`)
	assert.Contains(t, sql, `prj_stp."status" <> 'Canceled'`)
	assert.Contains(t, sql, `prj_stp."actual_arrival" IS NOT NULL`)
	assert.Contains(t, sql, `ORDER BY prj_dsm."sequence" DESC, prj_stp."sequence" DESC LIMIT 1`)
}

func TestRefreshQuery_ReadsWhenTheLastDeliveryLandsByMoveAndStopSequence(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		refreshQuery(t.Context(), shipmentdb.Shipments.ID.In(pulid.ID("shp_1"))),
	)

	assert.Contains(
		t,
		sql,
		`s."delivery_at" IS DISTINCT FROM (SELECT coalesce(prj_dts."actual_arrival", prj_dts."scheduled_window_start")`,
	)
	assert.Contains(t, sql, `JOIN "shipment_moves" prj_dtm`)
	assert.Contains(t, sql, `prj_dtm."shipment_id" = s."id"`)
	assert.Contains(t, sql, `prj_dtm."status" <> 'Canceled'`)
	assert.Contains(t, sql, `prj_dts."status" <> 'Canceled'`)
	assert.Contains(t, sql, `prj_dts."type" IN ('Delivery', 'SplitDelivery')`)
	assert.Contains(t, sql, `ORDER BY prj_dtm."sequence" DESC, prj_dts."sequence" DESC LIMIT 1`)
	assert.NotContains(t, sql, `prj_dts."actual_arrival" IS NOT NULL`)
}

func TestRefreshQuery_ReadsTheFirstPickupWindowByMoveAndStopSequence(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		refreshQuery(t.Context(), shipmentdb.Shipments.ID.In(pulid.ID("shp_1"))),
	)

	assert.Contains(
		t,
		sql,
		`s."pickup_at" IS DISTINCT FROM (SELECT prj_pts."scheduled_window_start"`,
	)
	assert.Contains(t, sql, `JOIN "shipment_moves" prj_ptm`)
	assert.Contains(t, sql, `prj_ptm."shipment_id" = s."id"`)
	assert.Contains(t, sql, `prj_ptm."status" <> 'Canceled'`)
	assert.Contains(t, sql, `prj_pts."status" <> 'Canceled'`)
	assert.Contains(t, sql, `prj_pts."type" IN ('Pickup', 'SplitPickup')`)
	assert.Contains(t, sql, `ORDER BY prj_ptm."sequence" ASC, prj_pts."sequence" ASC LIMIT 1`)
}

func TestRefreshQuery_FlagsALiveAssignmentOnAMoveStillRunning(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		refreshQuery(t.Context(), shipmentdb.Shipments.ID.In(pulid.ID("shp_1"))),
	)

	assert.Contains(t, sql, `s."has_active_assignment" IS DISTINCT FROM EXISTS (`)
	assert.Contains(t, sql, `JOIN "assignments" prj_a`)
	assert.Contains(t, sql, `prj_a."shipment_move_id" = prj_asm."id"`)
	assert.Contains(t, sql, `prj_a."archived_at" IS NULL`)
	assert.Contains(t, sql, `prj_a."status" <> 'Canceled'`)
	assert.Contains(t, sql, `prj_asm."shipment_id" = s."id"`)
	assert.Contains(t, sql, `prj_asm."status" <> 'Canceled'`)
	assert.NotContains(t, sql, `"carrier_assignments"`)
}

func TestRefreshQuery_TargetsTheShipmentsOfTheMovesWritten(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		refreshQuery(
			t.Context(),
			shipmentdb.Shipments.ID.InQuery(shipmentsOfMoves(t.Context(), []pulid.ID{"sm_1", "sm_2"})),
		),
	)

	assert.Contains(t, sql, `s."id" IN (SELECT prj_tsm."shipment_id" FROM "shipment_moves" prj_tsm`)
	assert.Contains(t, sql, `prj_tsm."id" = ANY(`)
}

func TestRefreshQuery_AcrossTenantsCorrelatesEveryTableByTenant(t *testing.T) {
	t.Parallel()

	system := dbscope.WithSystem(t.Context(), "render the projection refresh across tenants")
	sql, _ := keeltest.RenderIn(
		t,
		system,
		refreshQuery(system, shipmentdb.Shipments.ID.In(pulid.ID("shp_1"))),
	)

	assert.Contains(t, sql, `prj_stp."organization_id" = s."organization_id"`)
	assert.Contains(t, sql, `prj_stp."business_unit_id" = s."business_unit_id"`)
	assert.Contains(t, sql, `prj_asm."organization_id" = s."organization_id"`)
	assert.Contains(t, sql, `prj_a."business_unit_id" = s."business_unit_id"`)
	assert.Contains(t, sql, `prj_dts."organization_id" = s."organization_id"`)
	assert.Contains(t, sql, `prj_dtm."business_unit_id" = s."business_unit_id"`)
	assert.Contains(t, sql, `prj_pts."business_unit_id" = s."business_unit_id"`)
	assert.Contains(t, sql, `prj_ptm."organization_id" = s."organization_id"`)
	assert.NotContains(t, sql, `s."organization_id" = $`)
}

func TestRefreshQuery_ChangesOnlyRowsThatDiffer(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		refreshQuery(t.Context(), shipmentdb.Shipments.ID.In(pulid.ID("shp_1"))),
	)

	assert.Contains(t, sql, ` OR s."has_active_assignment" IS DISTINCT FROM EXISTS (`)
}

func TestTenantPage_WalksTheTenantsShipmentsByIDInBatches(t *testing.T) {
	t.Parallel()

	first, _ := keeltest.Render(t, tenantPage(""))
	assert.Contains(t, first, `s."business_unit_id" = $1 AND s."organization_id" = $2`)
	assert.Contains(t, first, `ORDER BY s."id" ASC LIMIT 1000`)
	assert.NotContains(t, first, `s."id" >`)

	next, args := keeltest.Render(t, tenantPage(pulid.ID("shp_9")))
	assert.Contains(t, next, `s."id" > $`)
	assert.Contains(t, args, "shp_9")
}

func TestRefresh_SkipsAnEmptyTarget(t *testing.T) {
	t.Parallel()

	assert.NoError(t, Refresh(t.Context(), nil))
	assert.NoError(t, RefreshMoves(t.Context(), nil))
}
