package lanequery

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestLanes_CountsShipmentsBetweenTheirFirstPickupAndLastDeliveryState(t *testing.T) {
	t.Parallel()

	s := shipmentdb.Shipments
	sql, args := keeltest.Render(t, lanes(Options{
		Shipments: []keel.Pred{
			s.CreatedAt.Gte(100),
			s.CreatedAt.Lte(200),
			s.Status.Ne(shipment.StatusCanceled),
		},
		Limit: 5,
	}))
	t.Log(sql)

	assert.Contains(t, sql, `WITH "lane_endpoints" ("origin_location_id", "destination_location_id") AS (`)
	assert.Contains(t, sql, `(array_agg(s2."location_id" ORDER BY sm."sequence" ASC, s2."sequence" ASC) FILTER (WHERE s2."type" = ANY(`)
	assert.Contains(t, sql, `(array_agg(s2."location_id" ORDER BY sm."sequence" DESC, s2."sequence" DESC) FILTER (WHERE s2."type" = ANY(`)
	assert.Contains(t, sql, `GROUP BY s."id"`)
	assert.Contains(t, sql, `JOIN "locations" loc_orig ON`)
	assert.Contains(t, sql, `JOIN "us_states" ust_orig ON ust_orig."id" = loc_orig."state_id"`)
	assert.Contains(t, sql, `JOIN "locations" loc_dest ON`)
	assert.Contains(t, sql, `JOIN "us_states" ust_dest ON ust_dest."id" = loc_dest."state_id"`)
	assert.Contains(t, sql, `GROUP BY ust_orig."abbreviation", ust_dest."abbreviation"`)
	assert.Contains(t, sql, `ORDER BY count(*) DESC, ust_orig."abbreviation" ASC, ust_dest."abbreviation" ASC LIMIT 5`)
	assert.Contains(t, args, keeltest.TestOrganizationID.String())
	assert.Contains(t, args, string(shipment.StatusCanceled))
}

func TestLanes_WithoutALimitReadsEveryLane(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, lanes(Options{}))

	assert.NotContains(t, sql, "LIMIT")
}
