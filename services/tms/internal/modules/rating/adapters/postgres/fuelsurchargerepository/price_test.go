package fuelsurchargerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestLatestPerIndexQuery_KeepsTheNewestPricesOfEachIndex(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, latestPerIndexQuery(2, "2026-10-19").Limit(100))

	assert.Contains(t, sql, `FROM "fuel_index_prices" newer`)
	assert.Contains(t, sql, `newer."price_date" > fip."price_date"`)
	assert.Contains(t, args, "2026-10-19")
	assert.Contains(t, args, 2)
}

func TestLastPricePerKey_KeepsTheLastPriceOfARepeatedWeek(t *testing.T) {
	t.Parallel()

	first := &fuelsurcharge.FuelIndexPrice{FuelIndexID: "fidx_1", PriceDate: "2026-10-19", SourceRaw: "1"}
	second := &fuelsurcharge.FuelIndexPrice{FuelIndexID: "fidx_1", PriceDate: "2026-10-19", SourceRaw: "2"}
	other := &fuelsurcharge.FuelIndexPrice{FuelIndexID: "fidx_2", PriceDate: "2026-10-19"}

	rows := lastPricePerKey([]*fuelsurcharge.FuelIndexPrice{first, nil, other, second})

	assert.Equal(t, []*fuelsurcharge.FuelIndexPrice{second, other}, rows)
}
