package assistantartifact

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestLineageKeyFor(t *testing.T) {
	t.Parallel()

	first := LineageKeyFor(KindTableView, "list_billing_queue", map[string]any{"status": "Ready", "limit": 50})
	again := LineageKeyFor(KindTableView, "list_billing_queue", map[string]any{"limit": 50, "status": "Ready"})
	// Read again with other filters, it is the same table: a new version.
	refiltered := LineageKeyFor(KindTableView, "list_billing_queue", map[string]any{"status": "Posted", "limit": 50})

	assert.NotEmpty(t, first)
	assert.Equal(t, first, again)
	assert.Equal(t, first, refiltered)
	assert.Equal(t,
		LineageKeyFor(KindTableView, "compose_table_view", map[string]any{"entity": "shipment", "description": "late"}),
		LineageKeyFor(KindTableView, "compose_table_view", map[string]any{"entity": "shipment", "description": "in Iowa"}),
	)
	assert.NotEqual(t,
		LineageKeyFor(KindEntityCard, "get_shipment", map[string]any{"shipmentId": "shp_a", "detail": "full"}),
		LineageKeyFor(KindEntityCard, "get_shipment", map[string]any{"shipmentId": "shp_b"}),
	)
	assert.Equal(t,
		LineageKeyFor(KindEntityCard, "get_shipment", map[string]any{"shipmentId": "shp_a", "detail": "full"}),
		LineageKeyFor(KindEntityCard, "get_shipment", map[string]any{"shipmentId": "shp_a"}),
	)
	assert.NotEqual(t, first, LineageKeyFor(KindTableView, "list_shipments", map[string]any{"status": "Ready", "limit": 50}))
	assert.Empty(t, LineageKeyFor(KindDocument, "publish_document", nil))
	assert.Empty(t, LineageKeyFor(KindTableView, "", nil))
}

func TestFollowLineage(t *testing.T) {
	t.Parallel()

	root := &Artifact{ID: pulid.MustNew("art_"), LineageSeq: 1}
	second := &Artifact{ID: pulid.MustNew("art_")}
	second.FollowLineage(root)
	third := &Artifact{ID: pulid.MustNew("art_")}
	third.FollowLineage(second)

	assert.Equal(t, root.ID, second.LineageID)
	assert.Equal(t, 2, second.LineageSeq)
	assert.Equal(t, root.ID, third.LineageID)
	assert.Equal(t, 3, third.LineageSeq)

	untouched := &Artifact{LineageSeq: 1}
	untouched.FollowLineage(nil)
	assert.True(t, untouched.LineageID.IsNil())
}
