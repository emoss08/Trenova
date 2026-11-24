package detentionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestStayWhere_AStayFollowsItsShipmentsRowScope(t *testing.T) {
	t.Parallel()

	o := detentiondb.DetentionOccurrences
	sql, _ := keeltest.Render(t, o.Where(stayWhere(&detentionports.ListDetentionStaysRequest{
		From:  1,
		To:    2,
		Stage: detentionports.StayStageToBill,
	})...))

	assert.Contains(t, sql, `EXISTS (SELECT`, "a stay is read only when its shipment is")
	assert.Contains(t, sql, `"shipments"`)
	assert.Contains(t, sql, `"status" IN`)
	assert.Contains(t, sql, `"is_open" IS FALSE`)
}
