package costingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var decimalOneHundred = decimal.NewFromInt(100)

func (s *Service) ResolveCostProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	asOf time.Time,
) (*ResolvedCostProfile, error) {
	control, err := s.repo.GetByOrgID(ctx, &repositories.GetCostingControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return s.resolveProfileFromControl(ctx, tenantInfo, control, asOf)
}

func (s *Service) resolveProfileFromControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *costingcontrol.CostingControl,
	asOf time.Time,
) (*ResolvedCostProfile, error) {
	profile := &ResolvedCostProfile{
		TotalCPM:             decimal.Zero,
		VariableCPM:          decimal.Zero,
		FixedCPM:             decimal.Zero,
		Categories:           make([]*ResolvedCategoryRate, 0, len(control.Categories)),
		TargetMarginPercent:  control.TargetMarginPercent,
		IncludeDeadheadMiles: control.IncludeDeadheadMiles,
		AsOfDate:             asOf.Format(fuelsurcharge.PriceDateLayout),
	}

	var glRates *glActualRates
	if control.GLActualsEnabled {
		rates, err := s.resolveGLActualRates(ctx, tenantInfo, control, asOf)
		if err != nil {
			return nil, err
		}
		glRates = rates
		profile.GLWindow = rates.window
	}

	fuel := s.resolveFuel(ctx, tenantInfo, control, asOf)
	profile.Fuel = fuel

	for _, category := range control.Categories {
		if category == nil || !category.IsActive {
			continue
		}

		rate, source := s.resolveCategoryRate(category, control, glRates, fuel)

		profile.Categories = append(profile.Categories, &ResolvedCategoryRate{
			Category:        category.Category,
			Name:            category.Name,
			CostBehavior:    category.CostBehavior,
			RatePerMile:     rate,
			EffectiveSource: source,
		})

		profile.TotalCPM = profile.TotalCPM.Add(rate)
		if category.CostBehavior == costingcontrol.CostBehaviorFixed {
			profile.FixedCPM = profile.FixedCPM.Add(rate)
		} else {
			profile.VariableCPM = profile.VariableCPM.Add(rate)
		}
	}

	return profile, nil
}

func (s *Service) resolveCategoryRate(
	category *costingcontrol.CostCategory,
	control *costingcontrol.CostingControl,
	glRates *glActualRates,
	fuel *FuelResolution,
) (decimal.Decimal, costingcontrol.EffectiveRateSource) {
	if category.RateSource == costingcontrol.RateSourceOverride &&
		category.OverrideRatePerMile.Valid {
		return category.OverrideRatePerMile.Decimal, costingcontrol.EffectiveRateSourceOverride
	}

	if category.RateSource == costingcontrol.RateSourceGLActual && glRates != nil {
		if rate, ok := glRates.rates[category.Category]; ok {
			return rate, costingcontrol.EffectiveRateSourceGLActual
		}
	}

	if category.Category == costingcontrol.CategoryTypeFuel &&
		fuel != nil && fuel.Source == costingcontrol.EffectiveRateSourceLiveIndex &&
		fuel.PricePerGallon.Valid && control.MilesPerGallon.GreaterThan(decimal.Zero) {
		return fuel.PricePerGallon.Decimal.Div(control.MilesPerGallon),
			costingcontrol.EffectiveRateSourceLiveIndex
	}

	return category.BenchmarkRatePerMile, costingcontrol.EffectiveRateSourceBenchmark
}

func (s *Service) resolveFuel(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *costingcontrol.CostingControl,
	asOf time.Time,
) *FuelResolution {
	fuel := &FuelResolution{
		MilesPerGallon: control.MilesPerGallon,
		FuelIndexID:    control.FuelIndexID,
		Source:         costingcontrol.EffectiveRateSourceBenchmark,
	}

	if !control.UseLiveFuelPrice || control.FuelIndexID == nil {
		return fuel
	}

	prices, err := s.priceRepo.GetLatestOnOrBefore(ctx, &repositories.GetLatestFuelPricesRequest{
		FuelIndexID: *control.FuelIndexID,
		TenantInfo:  tenantInfo,
		Date:        asOf.Format(fuelsurcharge.PriceDateLayout),
		Limit:       1,
	})
	if err != nil {
		s.l.Warn("failed to load fuel index price for cost profile", zap.Error(err))
		return fuel
	}

	if len(prices) == 0 || prices[0] == nil {
		return fuel
	}

	fuel.PricePerGallon = decimal.NewNullDecimal(prices[0].Price)
	fuel.PriceDate = prices[0].PriceDate
	fuel.Source = costingcontrol.EffectiveRateSourceLiveIndex

	return fuel
}

func (s *Service) EstimateShipment(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*ShipmentProfitabilityEstimate, error) {
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	profile, err := s.ResolveCostProfile(ctx, tenantInfo, s.now())
	if err != nil {
		return nil, err
	}

	return s.estimateFromShipment(entity, profile), nil
}

func (s *Service) EstimateShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*ShipmentProfitabilityEstimate, error) {
	if len(shipmentIDs) == 0 {
		return map[pulid.ID]*ShipmentProfitabilityEstimate{}, nil
	}

	profile, err := s.ResolveCostProfile(ctx, tenantInfo, s.now())
	if err != nil {
		return nil, err
	}

	rows, err := s.actualsRepo.ShipmentMilesByIDs(ctx, &repositories.ShipmentMilesByIDsRequest{
		TenantInfo:  tenantInfo,
		ShipmentIDs: shipmentIDs,
	})
	if err != nil {
		return nil, err
	}

	estimates := make(map[pulid.ID]*ShipmentProfitabilityEstimate, len(rows))
	for _, row := range rows {
		estimates[row.ShipmentID] = s.buildEstimate(profile, &shipmentMilesInput{
			shipmentID:      row.ShipmentID,
			loadedMiles:     row.LoadedMiles,
			deadheadMiles:   row.DeadheadMiles,
			missingDistance: row.MissingDistance,
			revenue:         row.Revenue,
		})
	}

	return estimates, nil
}

type shipmentMilesInput struct {
	shipmentID      pulid.ID
	loadedMiles     float64
	deadheadMiles   float64
	missingDistance bool
	revenue         decimal.Decimal
}

func (s *Service) estimateFromShipment(
	entity *shipment.Shipment,
	profile *ResolvedCostProfile,
) *ShipmentProfitabilityEstimate {
	input := &shipmentMilesInput{shipmentID: entity.ID}

	for _, move := range entity.Moves {
		if move == nil || move.Status == shipment.MoveStatusCanceled {
			continue
		}
		if move.Distance == nil {
			input.missingDistance = true
			continue
		}
		if move.Loaded {
			input.loadedMiles += *move.Distance
		} else {
			input.deadheadMiles += *move.Distance
		}
	}

	if entity.TotalChargeAmount.Valid {
		input.revenue = entity.TotalChargeAmount.Decimal
	}

	return s.buildEstimate(profile, input)
}

func (s *Service) buildEstimate(
	profile *ResolvedCostProfile,
	input *shipmentMilesInput,
) *ShipmentProfitabilityEstimate {
	estimate := &ShipmentProfitabilityEstimate{
		ShipmentID:      input.shipmentID,
		LoadedMiles:     input.loadedMiles,
		DeadheadMiles:   input.deadheadMiles,
		TotalMiles:      input.loadedMiles + input.deadheadMiles,
		Revenue:         input.revenue,
		MissingDistance: input.missingDistance,
		Profile:         profile,
	}

	costMiles := estimate.TotalMiles
	if !profile.IncludeDeadheadMiles {
		costMiles = estimate.LoadedMiles
	}

	costMilesDecimal := decimal.NewFromFloat(costMiles)
	estimate.EstimatedCost = profile.TotalCPM.Mul(costMilesDecimal).Round(2)
	estimate.Profit = estimate.Revenue.Sub(estimate.EstimatedCost)

	if estimate.Revenue.GreaterThan(decimal.Zero) {
		estimate.MarginPercent = decimal.NewNullDecimal(
			estimate.Profit.Div(estimate.Revenue).Mul(decimalOneHundred).Round(2),
		)
	}

	if estimate.LoadedMiles > 0 {
		loadedMilesDecimal := decimal.NewFromFloat(estimate.LoadedMiles)
		estimate.RevenuePerLoadedMile = decimal.NewNullDecimal(
			estimate.Revenue.Div(loadedMilesDecimal).Round(4),
		)
		estimate.BreakEvenRPM = decimal.NewNullDecimal(
			estimate.EstimatedCost.Div(loadedMilesDecimal).Round(4),
		)
	}

	estimate.Breakdown = make([]*CategoryCostLine, 0, len(profile.Categories))
	for _, category := range profile.Categories {
		estimate.Breakdown = append(estimate.Breakdown, &CategoryCostLine{
			Category:        category.Category,
			Name:            category.Name,
			CostBehavior:    category.CostBehavior,
			RatePerMile:     category.RatePerMile,
			Amount:          category.RatePerMile.Mul(costMilesDecimal).Round(2),
			EffectiveSource: category.EffectiveSource,
		})
	}

	return estimate
}

func (s *Service) FleetSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	from, to int64,
) (*FleetCostSummary, error) {
	control, err := s.repo.GetByOrgID(ctx, &repositories.GetCostingControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	profile, err := s.resolveProfileFromControl(ctx, tenantInfo, control, s.now())
	if err != nil {
		return nil, err
	}

	aggregates, err := s.actualsRepo.FleetCostAggregates(
		ctx,
		&repositories.FleetCostAggregatesRequest{
			TenantInfo:           tenantInfo,
			FromDate:             from,
			ToDate:               to,
			CostPerMile:          profile.TotalCPM,
			IncludeDeadheadMiles: control.IncludeDeadheadMiles,
		},
	)
	if err != nil {
		return nil, err
	}

	summary := &FleetCostSummary{
		AvgCPM:            profile.TotalCPM,
		ShipmentCount:     aggregates.ShipmentCount,
		UnprofitableCount: aggregates.UnprofitableCount,
		TotalRevenue:      aggregates.TotalRevenue,
		TotalMiles:        aggregates.TotalMiles,
		EmptyMiles:        aggregates.DeadheadMiles,
	}

	costMiles := aggregates.TotalMiles
	if !control.IncludeDeadheadMiles {
		costMiles = aggregates.LoadedMiles
	}
	summary.TotalEstimatedCost = profile.TotalCPM.
		Mul(decimal.NewFromFloat(costMiles)).
		Round(2)

	if summary.TotalRevenue.GreaterThan(decimal.Zero) {
		summary.AvgMarginPercent = decimal.NewNullDecimal(
			summary.TotalRevenue.Sub(summary.TotalEstimatedCost).
				Div(summary.TotalRevenue).
				Mul(decimalOneHundred).
				Round(2),
		)
	}

	return summary, nil
}
