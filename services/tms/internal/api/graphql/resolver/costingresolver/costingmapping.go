package costingresolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/fuelsurchargemapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func costingControlToModel(entity *costingcontrol.CostingControl) *gqlmodel.CostingControl {
	if entity == nil {
		return nil
	}

	model := &gqlmodel.CostingControl{
		ID:                   entity.ID.String(),
		BusinessUnitID:       entity.BusinessUnitID.String(),
		OrganizationID:       entity.OrganizationID.String(),
		FuelIndexID:          gqlid.IDPtrFromPulidPtr(entity.FuelIndexID),
		FuelIndex:            fuelsurchargemapping.FuelIndexToModel(entity.FuelIndex),
		UseLiveFuelPrice:     entity.UseLiveFuelPrice,
		MilesPerGallon:       entity.MilesPerGallon.String(),
		IncludeDeadheadMiles: entity.IncludeDeadheadMiles,
		GlActualsEnabled:     entity.GLActualsEnabled,
		GlRollingMonths:      int(entity.GLRollingMonths),
		PlannedMonthlyMiles:  gqlscalar.Int64PtrToIntPtr(entity.PlannedMonthlyMiles),
		TargetMarginPercent:  gqldecimal.NullDecimalToStringPtr(entity.TargetMarginPercent),
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
		OverrideRatePerMile:  gqldecimal.NullDecimalToStringPtr(entity.OverrideRatePerMile),
		IsActive:             entity.IsActive,
		SortOrder:            int(entity.SortOrder),
		Version:              int(entity.Version),
		GlAccounts: make(
			[]*gqlmodel.CostCategoryGLAccountLink,
			0,
			len(entity.GLAccounts),
		),
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
		TargetMarginPercent:  gqldecimal.NullDecimalToStringPtr(profile.TargetMarginPercent),
		IncludeDeadheadMiles: profile.IncludeDeadheadMiles,
		AsOfDate:             profile.AsOfDate,
		Categories:           make([]*gqlmodel.ResolvedCategoryRate, 0, len(profile.Categories)),
	}

	if profile.Fuel != nil {
		model.Fuel = &gqlmodel.FuelCostResolution{
			PricePerGallon: gqldecimal.NullDecimalToStringPtr(profile.Fuel.PricePerGallon),
			PriceDate:      profile.Fuel.PriceDate,
			FuelIndexID:    gqlid.IDPtrFromPulidPtr(profile.Fuel.FuelIndexID),
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
		MarginPercent:        gqldecimal.NullDecimalToStringPtr(estimate.MarginPercent),
		RevenuePerLoadedMile: gqldecimal.NullDecimalToStringPtr(estimate.RevenuePerLoadedMile),
		BreakEvenRpm:         gqldecimal.NullDecimalToStringPtr(estimate.BreakEvenRPM),
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

func fleetCostSummaryToModel(summary *costingservice.FleetCostSummary) *gqlmodel.FleetCostSummary {
	if summary == nil {
		return nil
	}

	return &gqlmodel.FleetCostSummary{
		AvgCpm:            summary.AvgCPM.String(),
		AvgMarginPercent:  gqldecimal.NullDecimalToStringPtr(summary.AvgMarginPercent),
		ShipmentCount:     summary.ShipmentCount,
		UnprofitableCount: &summary.UnprofitableCount,
		TotalRevenue: gqldecimal.NullDecimalStringPtr(
			decimal.NewNullDecimal(summary.TotalRevenue),
		),
		TotalEstimatedCost: summary.TotalEstimatedCost.String(),
		TotalMiles:         summary.TotalMiles,
		EmptyMiles:         summary.EmptyMiles,
	}
}

func costingSaveRequestFromInput(
	input *gqlmodel.CostingControlSaveInput,
) (*costingservice.SaveRequest, error) {
	req := &costingservice.SaveRequest{
		Categories: make([]*costingservice.CategoryChange, 0, len(input.Categories)),
	}

	if input.Control != nil {
		control, err := costingControlFromInput(input.Control)
		if err != nil {
			return nil, err
		}
		req.Control = control
	}

	for i, category := range input.Categories {
		change, err := costCategoryChangeFromInput(category, fmt.Sprintf("categories[%d]", i))
		if err != nil {
			return nil, err
		}
		req.Categories = append(req.Categories, change)
	}

	return req, nil
}

func costingControlFromInput(
	input *gqlmodel.CostingControlInput,
) (*costingcontrol.CostingControl, error) {
	glRollingMonths, err := gqlscalar.NarrowInt[int16](
		"control.glRollingMonths",
		input.GlRollingMonths,
	)
	if err != nil {
		return nil, err
	}

	entity := &costingcontrol.CostingControl{
		UseLiveFuelPrice:     input.UseLiveFuelPrice,
		IncludeDeadheadMiles: input.IncludeDeadheadMiles,
		GLActualsEnabled:     input.GlActualsEnabled,
		GLRollingMonths:      glRollingMonths,
		Version:              int64(input.Version),
	}

	fuelIndexID, err := gqlid.PulidPtrFromOptionalString(input.FuelIndexID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"control.fuelIndexId", errortypes.ErrInvalid, "Fuel index ID must be a valid ID")
	}
	entity.FuelIndexID = fuelIndexID

	if entity.MilesPerGallon, err = gqldecimal.DecimalFromString(
		input.MilesPerGallon,
		"control.milesPerGallon",
	); err != nil {
		return nil, err
	}

	if input.PlannedMonthlyMiles != nil {
		planned := int64(*input.PlannedMonthlyMiles)
		entity.PlannedMonthlyMiles = &planned
	}

	if entity.TargetMarginPercent, err = gqldecimal.NullDecimalFromStringPtr(
		input.TargetMarginPercent, "control.targetMarginPercent"); err != nil {
		return nil, err
	}

	return entity, nil
}

func costCategoryChangeFromInput(
	input *gqlmodel.CostCategoryUpdateInput,
	path string,
) (*costingservice.CategoryChange, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			path+".id", errortypes.ErrInvalid, "Invalid cost category")
	}

	rateSource, err := costingcontrol.RateSourceFromString(string(input.RateSource))
	if err != nil {
		return nil, errortypes.NewValidationError(
			path+".rateSource", errortypes.ErrInvalid, "Rate source is invalid")
	}

	overrideRate, err := gqldecimal.NullDecimalFromStringPtr(
		input.OverrideRatePerMile,
		path+".overrideRatePerMile",
	)
	if err != nil {
		return nil, err
	}

	glAccountIDs, err := gqlid.PulidsFromStrings(input.GlAccountIds, path+".glAccountIds")
	if err != nil {
		return nil, err
	}

	return &costingservice.CategoryChange{
		ID:                  id,
		RateSource:          rateSource,
		OverrideRatePerMile: overrideRate,
		IsActive:            input.IsActive,
		GLAccountIDs:        glAccountIDs,
		Version:             int64(input.Version),
	}, nil
}
