package triageoverlaprepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func load() triageports.TenderLoad {
	return triageports.TenderLoad{
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

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	sqlText := overlapQuery(
		db,
		pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		"cus_1",
		load(),
	).String()

	assert.Contains(t, sqlText, "organization_id = 'org_1'", sqlText)
	assert.Contains(t, sqlText, "business_unit_id = 'bu_1'", sqlText)
	assert.Contains(t, sqlText, "'cus_1'", sqlText)
	assert.Contains(t, sqlText, "'Canceled'", sqlText)
	assert.Contains(t, sqlText, "= 'dallas'", sqlText)
	assert.Contains(t, sqlText, "= 'TX'", sqlText)
	assert.Contains(t, sqlText, "= 'memphis'", sqlText)
	assert.Contains(t, sqlText, ">= 1760245200", sqlText)
	assert.Contains(t, sqlText, "< 1760331600", sqlText)
	assert.Contains(t, sqlText, "LIMIT 1", sqlText)
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
