package capturerepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestSearchBatchesReadsShipmentsByID(t *testing.T) {
	t.Parallel()

	b := capturedb.CaptureBatches
	sql, args := keeltest.Render(t, b.Where(searchBatches("PRO-1")))

	assert.NotContains(t, sql, "UNION")
	assert.NotContains(t, sql, " IN ((SELECT")
	assert.Equal(t, 3, strings.Count(sql, `FROM "shipments"`)+strings.Count(sql, `JOIN "shipments"`))
	assert.Contains(t, sql, `."id" = cb."target_id"`)
	assert.Contains(t, sql, `"filed_id"`)
	assert.Contains(t, sql, `"suggested_id"`)
	assert.Contains(t, sql, "count(*)")
	assert.Contains(t, sql, "IS TRUE")
	assert.Contains(t, args, "%PRO-1%")
}
