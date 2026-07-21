package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func costingControlToModel(entity *costingcontrol.CostingControl) *gqlmodel.CostingControl {
	if entity == nil {
		return nil
	}

	model := &gqlmodel.CostingControl{
		ID:                   entity.ID.String(),
		BusinessUnitID:       entity.BusinessUnitID.String(),
		OrganizationID:       entity.OrganizationID.String(),
		FuelIndexID:          idPtrFromPulidPtr(entity.FuelIndexID),
		FuelIndex:            fuelIndexToModel(entity.FuelIndex),
		UseLiveFuelPrice:     entity.UseLiveFuelPrice,
		MilesPerGallon:       entity.MilesPerGallon.String(),
		IncludeDeadheadMiles: entity.IncludeDeadheadMiles,
		GlActualsEnabled:     entity.GLActualsEnabled,
		GlRollingMonths:      int(entity.GLRollingMonths),
		PlannedMonthlyMiles:  int64PtrToIntPtr(entity.PlannedMonthlyMiles),
		TargetMarginPercent:  nullDecimalToStringPtr(entity.TargetMarginPercent),
		Version:              int(entity.Version),
		CreatedAt:            int(entity.CreatedAt),
		UpdatedAt:            int(entity.UpdatedAt),
		Categories:           make([]*gqlmodel.CostCategory, 0, len(entity.Categories)),
	}

	for _, category := range entity.Categories {
		if category == nil {
			continue
		}
		model.Categories = append(model.Categories, costCategoryToModel(category))
	}

	return model
}

func costCategoryToModel(entity *costingcontrol.CostCategory) *gqlmodel.CostCategory {
	model := &gqlmodel.CostCategory{
		ID:                   entity.ID.String(),
		Category:             gqlmodel.CostCategoryType(entity.Category.String()),
		Name:                 entity.Name,
		CostBehavior:         gqlmodel.CostBehavior(entity.CostBehavior.String()),
		RateSource:           gqlmodel.CostRateSource(entity.RateSource.String()),
		BenchmarkRatePerMile: entity.BenchmarkRatePerMile.String(),
		OverrideRatePerMile:  nullDecimalToStringPtr(entity.OverrideRatePerMile),
		IsActive:             entity.IsActive,
		SortOrder:            int(entity.SortOrder),
		Version:              int(entity.Version),
		GlAccounts:           make([]*gqlmodel.CostCategoryGLAccountLink, 0, len(entity.GLAccounts)),
	}

	for _, link := range entity.GLAccounts {
		if link == nil {
			continue
		}
		linkModel := &gqlmodel.CostCategoryGLAccountLink{
			ID:          link.ID.String(),
			GlAccountID: link.GLAccountID.String(),
		}
		if link.GLAccount != nil {
			linkModel.AccountCode = link.GLAccount.AccountCode
			linkModel.AccountName = link.GLAccount.Name
		}
		model.GlAccounts = append(model.GlAccounts, linkModel)
	}

	return model
}

func resolvedCostProfileToModel(
	profile *costingservice.ResolvedCostProfile,
) *gqlmodel.ResolvedCostProfile {
	if profile == nil {
		return nil
	}

	model := &gqlmodel.ResolvedCostProfile{
		TotalCpm:             profile.TotalCPM.String(),
		VariableCpm:          profile.VariableCPM.String(),
		FixedCpm:             profile.FixedCPM.String(),
		TargetMarginPercent:  nullDecimalToStringPtr(profile.TargetMarginPercent),
		IncludeDeadheadMiles: profile.IncludeDeadheadMiles,
		AsOfDate:             profile.AsOfDate,
		Categories:           make([]*gqlmodel.ResolvedCategoryRate, 0, len(profile.Categories)),
	}

	if profile.Fuel != nil {
		model.Fuel = &gqlmodel.FuelCostResolution{
			PricePerGallon: nullDecimalToStringPtr(profile.Fuel.PricePerGallon),
			PriceDate:      profile.Fuel.PriceDate,
			FuelIndexID:    idPtrFromPulidPtr(profile.Fuel.FuelIndexID),
			MilesPerGallon: profile.Fuel.MilesPerGallon.String(),
			Source:         gqlmodel.EffectiveRateSource(profile.Fuel.Source.String()),
		}
	}

	if profile.GLWindow != nil {
		model.GlWindow = &gqlmodel.GLActualsWindow{
			FromDate:    int(profile.GLWindow.FromDate),
			ToDate:      int(profile.GLWindow.ToDate),
			FleetMiles:  profile.GLWindow.FleetMiles,
			HasPostings: profile.GLWindow.HasPostings,
		}
	}

	for _, category := range profile.Categories {
		model.Categories = append(model.Categories, &gqlmodel.ResolvedCategoryRate{
			Category:        gqlmodel.CostCategoryType(category.Category.String()),
			Name:            category.Name,
			CostBehavior:    gqlmodel.CostBehavior(category.CostBehavior.String()),
			RatePerMile:     category.RatePerMile.String(),
			EffectiveSource: gqlmodel.EffectiveRateSource(category.EffectiveSource.String()),
		})
	}

	return model
}

func shipmentProfitabilityToModel(
	estimate *costingservice.ShipmentProfitabilityEstimate,
) *gqlmodel.ShipmentProfitability {
	if estimate == nil {
		return nil
	}

	model := &gqlmodel.ShipmentProfitability{
		ShipmentID:           estimate.ShipmentID.String(),
		LoadedMiles:          estimate.LoadedMiles,
		DeadheadMiles:        estimate.DeadheadMiles,
		TotalMiles:           estimate.TotalMiles,
		Revenue:              estimate.Revenue.String(),
		EstimatedCost:        estimate.EstimatedCost.String(),
		Profit:               estimate.Profit.String(),
		MarginPercent:        nullDecimalToStringPtr(estimate.MarginPercent),
		RevenuePerLoadedMile: nullDecimalToStringPtr(estimate.RevenuePerLoadedMile),
		BreakEvenRpm:         nullDecimalToStringPtr(estimate.BreakEvenRPM),
		MissingDistance:      estimate.MissingDistance,
		Breakdown:            make([]*gqlmodel.CategoryCostLine, 0, len(estimate.Breakdown)),
		Profile:              resolvedCostProfileToModel(estimate.Profile),
	}

	for _, line := range estimate.Breakdown {
		model.Breakdown = append(model.Breakdown, &gqlmodel.CategoryCostLine{
			Category:        gqlmodel.CostCategoryType(line.Category.String()),
			Name:            line.Name,
			CostBehavior:    gqlmodel.CostBehavior(line.CostBehavior.String()),
			RatePerMile:     line.RatePerMile.String(),
			Amount:          line.Amount.String(),
			EffectiveSource: gqlmodel.EffectiveRateSource(line.EffectiveSource.String()),
		})
	}

	return model
}

func shipmentProfitabilityEstimateToModel(
	estimate *costingservice.ShipmentProfitabilityEstimate,
) *gqlmodel.ShipmentProfitabilityEstimate {
	if estimate == nil {
		return nil
	}

	model := &gqlmodel.ShipmentProfitabilityEstimate{
		ShipmentID:      estimate.ShipmentID.String(),
		LoadedMiles:     estimate.LoadedMiles,
		DeadheadMiles:   estimate.DeadheadMiles,
		TotalMiles:      estimate.TotalMiles,
		EstimatedCost:   estimate.EstimatedCost.String(),
		Profit:          estimate.Profit.String(),
		MarginPercent:   nullDecimalToStringPtr(estimate.MarginPercent),
		BreakEvenRpm:    nullDecimalToStringPtr(estimate.BreakEvenRPM),
		MissingDistance: estimate.MissingDistance,
	}

	if estimate.Profile != nil {
		model.CostPerMile = estimate.Profile.TotalCPM.String()
		model.TargetMarginPercent = nullDecimalToStringPtr(estimate.Profile.TargetMarginPercent)
	}

	return model
}

func fleetCostSummaryToModel(summary *costingservice.FleetCostSummary) *gqlmodel.FleetCostSummary {
	if summary == nil {
		return nil
	}

	return &gqlmodel.FleetCostSummary{
		AvgCpm:             summary.AvgCPM.String(),
		AvgMarginPercent:   nullDecimalToStringPtr(summary.AvgMarginPercent),
		ShipmentCount:      summary.ShipmentCount,
		UnprofitableCount:  summary.UnprofitableCount,
		TotalRevenue:       summary.TotalRevenue.String(),
		TotalEstimatedCost: summary.TotalEstimatedCost.String(),
		TotalMiles:         summary.TotalMiles,
		EmptyMiles:         summary.EmptyMiles,
	}
}

func costingControlFromInput(
	input *gqlmodel.CostingControlInput,
	existing *costingcontrol.CostingControl,
) (*costingcontrol.CostingControl, error) {
	entity := &costingcontrol.CostingControl{
		ID:                   existing.ID,
		BusinessUnitID:       existing.BusinessUnitID,
		OrganizationID:       existing.OrganizationID,
		UseLiveFuelPrice:     input.UseLiveFuelPrice,
		IncludeDeadheadMiles: input.IncludeDeadheadMiles,
		GLActualsEnabled:     input.GlActualsEnabled,
		CreatedAt:            existing.CreatedAt,
		Version:              int64(input.Version),
	}

	fuelIndexID, err := pulidPtrFromOptionalString(input.FuelIndexID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"fuelIndexId", errortypes.ErrInvalid, "Fuel index ID must be a valid ID")
	}
	entity.FuelIndexID = fuelIndexID

	if entity.MilesPerGallon, err = decimalFromString(input.MilesPerGallon, "milesPerGallon"); err != nil {
		return nil, err
	}

	if input.GlRollingMonths < 1 || input.GlRollingMonths > 12 {
		return nil, errortypes.NewValidationError(
			"glRollingMonths", errortypes.ErrInvalid, "GL rolling months must be between 1 and 12")
	}
	entity.GLRollingMonths = int16(input.GlRollingMonths)

	if input.PlannedMonthlyMiles != nil {
		planned := int64(*input.PlannedMonthlyMiles)
		entity.PlannedMonthlyMiles = &planned
	}

	if entity.TargetMarginPercent, err = nullDecimalFromStringPtr(
		input.TargetMarginPercent, "targetMarginPercent"); err != nil {
		return nil, err
	}

	return entity, nil
}

func costCategoryFromUpdateInput(
	input *gqlmodel.CostCategoryUpdateInput,
	existing *costingcontrol.CostCategory,
) (*costingcontrol.CostCategory, error) {
	rateSource, err := costingcontrol.RateSourceFromString(string(input.RateSource))
	if err != nil {
		return nil, errortypes.NewValidationError(
			"rateSource", errortypes.ErrInvalid, "Rate source is invalid")
	}

	entity := &costingcontrol.CostCategory{
		ID:                   existing.ID,
		BusinessUnitID:       existing.BusinessUnitID,
		OrganizationID:       existing.OrganizationID,
		CostingControlID:     existing.CostingControlID,
		Category:             existing.Category,
		Name:                 existing.Name,
		CostBehavior:         existing.CostBehavior,
		RateSource:           rateSource,
		BenchmarkRatePerMile: existing.BenchmarkRatePerMile,
		IsActive:             input.IsActive,
		SortOrder:            existing.SortOrder,
		CreatedAt:            existing.CreatedAt,
		Version:              int64(input.Version),
	}

	if entity.OverrideRatePerMile, err = nullDecimalFromStringPtr(
		input.OverrideRatePerMile, "overrideRatePerMile"); err != nil {
		return nil, err
	}

	return entity, nil
}

func findCostCategory(
	control *costingcontrol.CostingControl,
	categoryID pulid.ID,
) *costingcontrol.CostCategory {
	for _, category := range control.Categories {
		if category != nil && category.ID == categoryID {
			return category
		}
	}
	return nil
}

func shipmentProfitabilityAnalyticsToModel(
	summary *costingservice.FleetCostSummary,
) *gqlmodel.ShipmentProfitabilityAnalytics {
	if summary == nil {
		return nil
	}

	model := &gqlmodel.ShipmentProfitabilityAnalytics{
		UnprofitableCount: summary.UnprofitableCount,
		ShipmentCount:     summary.ShipmentCount,
		TotalMiles:        summary.TotalMiles,
	}
	model.AvgCpm, _ = summary.AvgCPM.Float64()
	if summary.AvgMarginPercent.Valid {
		model.HasMargin = true
		model.AvgMarginPct, _ = summary.AvgMarginPercent.Decimal.Float64()
	}

	return model
}

func costingTenantInfo(orgID, buID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: orgID, BuID: buID}
}
