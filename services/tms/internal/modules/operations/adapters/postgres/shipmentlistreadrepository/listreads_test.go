package shipmentlistreadrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

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

func TestNewestCommentsQuery_TakesEachShipmentsNewestTopLevelComment(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	sql, args := keeltest.Render(t, newestCommentsQuery([]pulid.ID{first, second}))

	assert.Contains(t, sql, "DISTINCT ON (")
	assert.Contains(t, sql, `"shipment_id") `)
	assert.Contains(t, sql, `"parent_comment_id" IS NULL`)
	assert.Contains(t, sql, `"deleted_at" IS NULL`)
	assert.Contains(t, sql, `"created_at" DESC`)
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, args, []string{first.String(), second.String()})
}
