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

	assert.Contains(t, sql, "LATERAL")
	assert.Contains(t, sql, "NOT (EXISTS")
	assert.Contains(t, sql, `"document_types"`)
	assert.Contains(t, sql, "LIMIT 2000")
}

func TestExhaustedTendersQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(
		t,
		exhaustedTendersQuery(&watchtowerports.ListExhaustedTendersRequest{}),
	)

	assert.Contains(t, sql, "NOT EXISTS")
	assert.Contains(t, sql, `"carrier_assignments"`)
	assert.Contains(t, sql, "newer_tnd")
}
