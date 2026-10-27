package triageoverlaprepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func load() *triageports.TenderLoad {
	return &triageports.TenderLoad{
		Index:            0,
		OriginCity:       " Dallas ",
		OriginState:      "tx",
		DestinationCity:  "Memphis",
		DestinationState: "TN",
		PickupFrom:       1_760_245_200,
		PickupTo:         1_760_331_600,
	}
}

func TestOverlapQueryLooksForTheSameLaneAndPickupDay(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, overlapQuery("cus_1", load()))

	assert.Contains(t, sqlText, `"organization_id" = $`, sqlText)
	assert.Contains(t, sqlText, "EXISTS (", sqlText)
	assert.Contains(t, sqlText, "lower(btrim(", sqlText)
	assert.Contains(t, sqlText, "upper(", sqlText)
	assert.Contains(t, sqlText, "LIMIT 1", sqlText)
	assert.Contains(t, args, keeltest.TestOrganizationID.String())
	assert.Contains(t, args, "cus_1")
	assert.Contains(t, sqlText, `s."status" <> 'Canceled'`, sqlText)
	assert.Contains(t, sqlText, `pick_stp."type" IN ('Pickup', 'SplitPickup')`, sqlText)
	assert.Contains(t, sqlText, `drop_stp."type" IN ('Delivery', 'SplitDelivery')`, sqlText)
	assert.Contains(t, args, "dallas")
	assert.Contains(t, args, "TX")
	assert.Contains(t, args, "memphis")
	assert.Contains(t, args, int64(1_760_245_200))
	assert.Contains(t, args, int64(1_760_331_600))
}

func TestCheckableNeedsTheWholeLaneAndAWindow(t *testing.T) {
	t.Parallel()

	assert.True(t, checkable(load()))

	noCity := load()
	noCity.DestinationCity = ""
	assert.False(t, checkable(noCity))

	noWindow := load()
	noWindow.PickupTo = noWindow.PickupFrom
	assert.False(t, checkable(noWindow))
}
