package exceptionsourcerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestMissedStopsQuery(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, missedStopsQuery(&watchtowerports.ListMissedStopsRequest{
		StopTypes:    []shipmentvocab.StopType{shipmentvocab.StopTypePickup},
		ClosedBefore: 100,
		ShipmentIDs:  []pulid.ID{"shp_1"},
	}))

	assert.Contains(t, sql, `JOIN "shipment_moves"`)
	assert.Contains(t, sql, `JOIN "shipments"`)
	assert.Contains(t, sql, `LEFT JOIN "locations"`)
	assert.Contains(t, sql, `"us_states"`)
	assert.NotContains(t, sql, "LIMIT")
	assert.Contains(t, args, int64(100))
}

func TestMissedStopsQuery_LimitsTheTenantWideRead(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, missedStopsQuery(&watchtowerports.ListMissedStopsRequest{
		StopTypes:    []shipmentvocab.StopType{shipmentvocab.StopTypePickup},
		ClosedBefore: 100,
	}))

	assert.Contains(t, sql, "LIMIT 500")
}

func TestMissingPODsQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, missingPODsQuery(&watchtowerports.ListMissingPODsRequest{
		DeliveredBefore: 200,
		DeliveredAfter:  100,
		Limit:           5000,
	}))

	assert.NotContains(t, sql, "LATERAL")
	assert.NotContains(t, sql, `"stops"`)
	assert.Contains(t, sql, `s."last_delivery_arrival_at" < $`)
	assert.Contains(t, sql, `s."last_delivery_arrival_at" >= $`)
	assert.Contains(t, sql, `ORDER BY s."last_delivery_arrival_at" ASC, s."id" ASC`)
	assert.Contains(t, sql, "NOT (EXISTS")
	assert.Contains(t, sql, `"document_types"`)
	assert.Contains(t, sql, "LIMIT 2000")
}

func TestExhaustedTendersQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		exhaustedTendersQuery(&watchtowerports.ListExhaustedTendersRequest{}, nil, 500),
	)

	assert.Contains(t, sql, "NOT EXISTS")
	assert.Contains(t, sql, `"carrier_assignments"`)
	assert.Contains(t, sql, `ca."shipment_move_id" = t."shipment_move_id"`)
	assert.Contains(t, sql, "newer_tnd")
	assert.Contains(t, sql, `t."status" = 'Exhausted'`)
	assert.NotContains(t, sql, `"shipment_moves"`)
	assert.NotContains(t, sql, `"shipments"`)
	assert.Contains(t, sql, `ORDER BY t."created_at" ASC, t."id" ASC LIMIT 500`)
	assert.NotContains(t, sql, `ROW(t."created_at", t."id") > ROW($`)
}

func TestExhaustedTendersQuery_ContinuesAfterTheLastTenderRead(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, exhaustedTendersQuery(
		&watchtowerports.ListExhaustedTendersRequest{ShipmentIDs: []pulid.ID{"shp_1"}},
		&exhaustedTenderRow{TenderID: "tnd_1", CreatedAt: 42},
		10,
	))

	assert.Contains(t, sql, `ROW(t."created_at", t."id") > ROW($`)
	assert.Contains(t, sql, `t."shipment_id" = ANY($`)
	assert.Contains(t, sql, "LIMIT 10")
	assert.Contains(t, args, int64(42))
	assert.Contains(t, args, "tnd_1")
}

func TestUncoveredMovesQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, uncoveredMovesQuery(
		&watchtowerports.ListExhaustedTendersRequest{},
		[]pulid.ID{"smv_1", "smv_2"},
	))

	assert.Contains(t, sql, `sm."id" = ANY($`)
	assert.Contains(t, sql, `sm."status" = 'New'`)
	assert.Contains(t, sql, `JOIN "shipments"`)
	assert.NotContains(t, sql, "ORDER BY")
	assert.Contains(t, sql, "LIMIT 2")
}
