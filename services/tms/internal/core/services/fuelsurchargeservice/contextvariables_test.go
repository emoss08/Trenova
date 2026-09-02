package fuelsurchargeservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubIndexRepo struct {
	repositories.FuelIndexRepository
	indices []*fuelsurcharge.FuelIndex
}

func (s *stubIndexRepo) ListActive(
	context.Context,
	pagination.TenantInfo,
) ([]*fuelsurcharge.FuelIndex, error) {
	return s.indices, nil
}

type stubLatestPriceRepo struct {
	repositories.FuelIndexPriceRepository
	latest map[pulid.ID][]*fuelsurcharge.FuelIndexPrice
}

func (s *stubLatestPriceRepo) LatestPerIndex(
	context.Context,
	*repositories.LatestPricesPerIndexRequest,
) (map[pulid.ID][]*fuelsurcharge.FuelIndexPrice, error) {
	return s.latest, nil
}

func TestContextVariables_UsesTheFreshestIndexPrice(t *testing.T) {
	t.Parallel()

	national := &fuelsurcharge.FuelIndex{ID: pulid.MustNew("fidx_"), Code: "DOE_US"}
	regional := &fuelsurcharge.FuelIndex{ID: pulid.MustNew("fidx_"), Code: "DOE_GULF"}
	stale := &fuelsurcharge.FuelIndex{ID: pulid.MustNew("fidx_"), Code: "AAA"}

	svc := &Service{
		l:         zap.NewNop(),
		indexRepo: &stubIndexRepo{indices: []*fuelsurcharge.FuelIndex{national, regional, stale}},
		priceRepo: &stubLatestPriceRepo{latest: map[pulid.ID][]*fuelsurcharge.FuelIndexPrice{
			national.ID: {{PriceDate: "2026-08-31", Price: decimal.RequireFromString("3.85")}},
			regional.ID: {{PriceDate: "2026-08-31", Price: decimal.RequireFromString("3.61")}},
			stale.ID:    {{PriceDate: "2026-06-01", Price: decimal.RequireFromString("3.20")}},
		}},
	}

	variables, err := svc.ContextVariables(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.InDelta(t, 3.61, variables["fuelPrice"], 0.0001,
		"the freshest price wins; on a tie the lowest index code does, so the answer never flips")
	assert.Equal(t, "2026-08-31", variables["fuelPriceDate"])
	assert.Equal(t, "DOE_GULF", variables["fuelIndexCode"])
}

func TestContextVariables_EmptyWithoutPrices(t *testing.T) {
	t.Parallel()

	index := &fuelsurcharge.FuelIndex{ID: pulid.MustNew("fidx_"), Code: "DOE_US"}
	svc := &Service{
		l:         zap.NewNop(),
		indexRepo: &stubIndexRepo{indices: []*fuelsurcharge.FuelIndex{index}},
		priceRepo: &stubLatestPriceRepo{latest: map[pulid.ID][]*fuelsurcharge.FuelIndexPrice{}},
	}

	variables, err := svc.ContextVariables(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Empty(t, variables, "no price means the schema's nullable placeholder stays empty")
}
