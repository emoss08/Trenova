package stopetarepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAttentionQueryWalksTheAttentionIndexAlone(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, attentionQuery(&stopetaports.ListAttentionRequest{}, nil, 5000))

	assert.NotContains(t, sql, `"shipments"`)
	assert.Contains(t, sql, `se."verdict" IN ('AtRisk', 'Late') AND se."arrived" IS FALSE`)
	assert.Contains(
		t,
		sql,
		`ORDER BY se."shipment_id" ASC, se."move_sequence" ASC, se."stop_sequence" ASC LIMIT 5000`,
	)
	assert.NotContains(t, sql, "ROW(")
}

func TestAttentionQueryContinuesAfterTheLastRowRead(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, attentionQuery(
		&stopetaports.ListAttentionRequest{ShipmentIDs: []pulid.ID{"shp_1"}},
		&stopeta.StopETA{ShipmentID: "shp_2", MoveSequence: 1, StopSequence: 2},
		10,
	))

	assert.Contains(
		t,
		sql,
		`ROW(se."shipment_id", se."move_sequence", se."stop_sequence") > ROW($`,
	)
	assert.Contains(t, sql, "LIMIT 10")
	assert.Contains(t, args, "shp_2")
	assert.Contains(t, args, int64(2))
}

func TestTrackedShipmentsQueryReadsTheShipmentsByID(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, trackedShipmentsQuery(
		[]pulid.ID{"shp_1", "shp_2"},
		stopeta.TrackedShipmentStatuses(),
	))

	assert.Contains(t, sql, `s."id" = ANY($`)
	assert.Contains(t, sql, `s."status" IN (`)
	assert.NotContains(t, sql, "ORDER BY")
	assert.Contains(t, sql, "LIMIT 2")
}
