package fuelsurchargeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

var _ formulatemplatetypes.ContextVariableProvider = (*Service)(nil)

const (
	fuelPriceVariable     = "fuelPrice"
	fuelPriceDateVariable = "fuelPriceDate"
	fuelIndexCodeVariable = "fuelIndexCode"
)

// ContextVariables feeds the tenant's latest fuel price into formulas as
// fuelPrice, with the date and index code beside it. Among active indices the
// freshest price wins; on the same date the lowest index code does, so the
// answer is stable across runs rather than following map order. A tenant with
// no priced index gets nothing, and the schema's nullable placeholder stands.
func (s *Service) ContextVariables(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]any, error) {
	indices, err := s.indexRepo.ListActive(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if len(indices) == 0 {
		return map[string]any{}, nil
	}

	pricesByIndex, err := s.priceRepo.LatestPerIndex(ctx, &repositories.LatestPricesPerIndexRequest{
		TenantInfo: tenantInfo,
		PerIndex:   1,
	})
	if err != nil {
		return nil, err
	}

	var (
		chosenIndex *fuelsurcharge.FuelIndex
		chosenPrice *fuelsurcharge.FuelIndexPrice
	)
	for _, index := range indices {
		if index == nil {
			continue
		}
		prices := pricesByIndex[index.ID]
		if len(prices) == 0 || prices[0] == nil {
			continue
		}
		candidate := prices[0]
		if chosenPrice == nil ||
			candidate.PriceDate > chosenPrice.PriceDate ||
			(candidate.PriceDate == chosenPrice.PriceDate && index.Code < chosenIndex.Code) {
			chosenIndex, chosenPrice = index, candidate
		}
	}

	if chosenPrice == nil {
		return map[string]any{}, nil
	}

	return map[string]any{
		fuelPriceVariable:     chosenPrice.Price.InexactFloat64(),
		fuelPriceDateVariable: chosenPrice.PriceDate,
		fuelIndexCodeVariable: chosenIndex.Code,
	}, nil
}
