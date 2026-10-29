package customertrackingrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestCandidateShipmentsQuery_CapsTheUnionOfIndexServedCandidates(
	t *testing.T,
) {
	t.Parallel()

	sql, args := keeltest.Render(t, candidateShipmentsQuery(1_800_000_000, 2000))

	assert.Equal(t, 3, strings.Count(sql, " UNION "), "one branch per way to owe an email")
	assert.NotContains(t, sql, " OR ", "no branch is an OR no single index serves")
	assert.Contains(t, sql, `ORDER BY sec."updated_at" DESC, sec."id" DESC LIMIT 2000`)
	assert.Contains(t, sql, `"status" IN ('Assigned', 'PartiallyAssigned', 'InTransit', 'Delayed', 'PartiallyCompleted')`)
	assert.Contains(t, sql, `"updated_at" >= `)
	assert.Contains(t, sql, `"actual_delivery_date" >= `)
	assert.Contains(t, sql, `"actual_delivery_date" <= `)
	assert.Contains(t, sql, `"customer_tracking_profiles"`)
	assert.Contains(t, sql, `"notifications_enabled_at" IS NOT NULL`)
	assert.Contains(t, sql, `"status" <> 'Canceled'`)
	assert.Contains(t, sql, `"documents"`)
	assert.Contains(t, args, int64(1_800_000_000))
	assert.Contains(t, args, openEnded)
}
