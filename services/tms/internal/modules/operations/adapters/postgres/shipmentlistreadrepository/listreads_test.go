package shipmentlistreadrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestOrderReferenceCandidatesQuery_ReadsABoundedMatchWithoutSorting(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, orderReferenceCandidatesQuery("77Q-4182"))

	assert.Contains(t, sql, `"orders"`)
	assert.Contains(t, sql, "@@")
	assert.Contains(t, sql, `"created_at"`)
	assert.NotContains(t, sql, "ORDER BY")
	assert.Contains(t, sql, "LIMIT 501")
	assert.Contains(t, args, "77Q-4182")
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

func TestNewestOrderReferences_OrdersTheMatchesNewestFirstAndKeepsTheLimit(t *testing.T) {
	t.Parallel()

	candidate := func(id string, createdAt int64) orderReferenceCandidate {
		return orderReferenceCandidate{
			OrderID:     pulid.ID(id),
			OrderNumber: "ORD-" + id,
			PONumber:    "PO-" + id,
			CreatedAt:   createdAt,
		}
	}
	matches := newestOrderReferences([]orderReferenceCandidate{
		candidate("ord_A", 100),
		candidate("ord_C", 300),
		candidate("ord_B", 300),
		candidate("ord_D", 200),
	}, 3)

	ids := make([]pulid.ID, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match.OrderID)
	}
	assert.Equal(t, []pulid.ID{"ord_C", "ord_B", "ord_D"}, ids,
		"created_at DESC, then id DESC, as the newest-first read orders them")
	assert.Equal(t, shipmentports.OrderReferenceMatch{
		OrderID:     "ord_C",
		OrderNumber: "ORD-ord_C",
		PONumber:    "PO-ord_C",
	}, matches[0])
	assert.Empty(t, newestOrderReferences(nil, 3))
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

func TestNewestCommentsQuery_RanksEachShipmentsTopLevelCommentsNewestFirst(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	sql, args := keeltest.Render(t, newestCommentsQuery([]pulid.ID{first, second}, 3))

	assert.Contains(t, sql, "row_number() OVER (PARTITION BY")
	assert.Contains(t, sql, `"parent_comment_id" IS NULL`)
	assert.Contains(t, sql, `"deleted_at" IS NULL`)
	assert.Contains(t, sql, `"created_at" DESC`)
	assert.Contains(t, sql, `"comment_rank" <= $`)
	assert.Contains(t, sql, "LIMIT 6")
	assert.Contains(t, args, []string{first.String(), second.String()})
}

func TestCommentCountsQuery_CountsTopLevelCommentsByShipment(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	sql, args := keeltest.Render(t, commentCountsQuery([]pulid.ID{id}))

	assert.Contains(t, sql, "count(*)")
	assert.Contains(t, sql, "GROUP BY")
	assert.Contains(t, sql, `"parent_comment_id" IS NULL`)
	assert.Contains(t, args, []string{id.String()})
}

func TestCommentsPerShipment_IsBounded(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, commentsPerShipment(0))
	assert.Equal(t, 5, commentsPerShipment(5))
	assert.Equal(t, shipmentports.MaxNewestCommentsPerShipment, commentsPerShipment(500))
}
