package fuelsurchargeresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/fuelsurchargemapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func fuelIndexPriceToModel(entity *fuelsurcharge.FuelIndexPrice) *gqlmodel.FuelIndexPrice {
	if entity == nil {
		return nil
	}

	return &gqlmodel.FuelIndexPrice{
		ID:             entity.ID.String(),
		BusinessUnitID: entity.BusinessUnitID.String(),
		OrganizationID: entity.OrganizationID.String(),
		FuelIndexID:    entity.FuelIndexID.String(),
		PriceDate:      entity.PriceDate,
		Price:          entity.Price.String(),
		Currency:       entity.Currency,
		IsManual:       entity.IsManual,
		EnteredByID:    gqlid.IDPtrFromPulidPtr(entity.EnteredByID),
		SourceRaw:      entity.SourceRaw,
		FetchedAt:      entity.FetchedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func fuelIndexPricesToModel(
	entities []*fuelsurcharge.FuelIndexPrice,
) []*gqlmodel.FuelIndexPrice {
	result := make([]*gqlmodel.FuelIndexPrice, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		result = append(result, fuelIndexPriceToModel(entity))
	}
	return result
}

func fuelTableRowToModel(
	entity *fuelsurcharge.FuelSurchargeTableRow,
) *gqlmodel.FuelSurchargeTableRow {
	if entity == nil {
		return nil
	}

	return &gqlmodel.FuelSurchargeTableRow{
		ID:                     entity.ID.String(),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		FuelSurchargeProgramID: entity.FuelSurchargeProgramID.String(),
		PriceMin:               gqldecimal.NullDecimalToStringPtr(entity.PriceMin),
		PriceMax:               gqldecimal.NullDecimalToStringPtr(entity.PriceMax),
		Value:                  entity.Value.String(),
		SortOrder:              int(entity.SortOrder),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
	}
}

func fuelSurchargeProgramToModel(
	entity *fuelsurcharge.FuelSurchargeProgram,
) *gqlmodel.FuelSurchargeProgram {
	if entity == nil {
		return nil
	}

	model := &gqlmodel.FuelSurchargeProgram{
		ID:                   entity.ID.String(),
		BusinessUnitID:       entity.BusinessUnitID.String(),
		OrganizationID:       entity.OrganizationID.String(),
		Name:                 entity.Name,
		Code:                 entity.Code,
		Description:          entity.Description,
		Status:               entity.Status,
		FuelIndexID:          entity.FuelIndexID.String(),
		AccessorialChargeID:  entity.AccessorialChargeID.String(),
		Method:               entity.Method,
		PegPrice:             gqldecimal.NullDecimalToStringPtr(entity.PegPrice),
		Increment:            gqldecimal.NullDecimalToStringPtr(entity.Increment),
		IncrementRate:        gqldecimal.NullDecimalToStringPtr(entity.IncrementRate),
		MilesPerGallon:       gqldecimal.NullDecimalToStringPtr(entity.MilesPerGallon),
		PercentBasis:         entity.PercentBasis,
		StepRounding:         entity.StepRounding,
		RateRounding:         entity.RateRounding,
		RatePrecision:        int(entity.RatePrecision),
		MinAmount:            gqldecimal.NullDecimalToStringPtr(entity.MinAmount),
		MaxAmount:            gqldecimal.NullDecimalToStringPtr(entity.MaxAmount),
		DateBasis:            entity.DateBasis,
		PriceEffectiveDay:    int(entity.PriceEffectiveDay),
		MissingPriceFallback: entity.MissingPriceFallback,
		EffectiveStartDate:   gqlscalar.Int64PtrToIntPtr(entity.EffectiveStartDate),
		EffectiveEndDate:     gqlscalar.Int64PtrToIntPtr(entity.EffectiveEndDate),
		ShipmentTypeIds:      gqlid.PulidsToStrings(entity.ShipmentTypeIDs),
		ServiceTypeIds:       gqlid.PulidsToStrings(entity.ServiceTypeIDs),
		TractorTypeIds:       gqlid.PulidsToStrings(entity.TractorTypeIDs),
		TrailerTypeIds:       gqlid.PulidsToStrings(entity.TrailerTypeIDs),
		Version:              int(entity.Version),
		CreatedAt:            int(entity.CreatedAt),
		UpdatedAt:            int(entity.UpdatedAt),
		FuelIndex:            fuelsurchargemapping.FuelIndexToModel(entity.FuelIndex),
		AccessorialCharge: shipmentmapping.ShipmentAccessorialChargeToModel(
			entity.AccessorialCharge,
		),
	}

	if len(entity.TableRows) > 0 {
		model.TableRows = make([]*gqlmodel.FuelSurchargeTableRow, 0, len(entity.TableRows))
		for _, row := range entity.TableRows {
			if row == nil {
				continue
			}
			model.TableRows = append(model.TableRows, fuelTableRowToModel(row))
		}
	}

	return model
}

func fuelIndexConnectionToModel(
	result *pagination.CursorListResult[*fuelsurcharge.FuelIndex],
) (*gqlmodel.FuelIndexConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *fuelsurcharge.FuelIndex, cursor string) *gqlmodel.FuelIndexEdge {
			return &gqlmodel.FuelIndexEdge{
				Node:   fuelsurchargemapping.FuelIndexToModel(node),
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FuelIndexEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FuelIndexConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func fuelSurchargeProgramConnectionToModel(
	result *pagination.CursorListResult[*fuelsurcharge.FuelSurchargeProgram],
) (*gqlmodel.FuelSurchargeProgramConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(
			node *fuelsurcharge.FuelSurchargeProgram,
			cursor string,
		) *gqlmodel.FuelSurchargeProgramEdge {
			return &gqlmodel.FuelSurchargeProgramEdge{
				Node:   fuelSurchargeProgramToModel(node),
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FuelSurchargeProgramEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FuelSurchargeProgramConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func fuelIndexFromInput(
	input *gqlmodel.FuelIndexInput,
	tenantInfo pagination.TenantInfo,
) *fuelsurcharge.FuelIndex {
	entity := &fuelsurcharge.FuelIndex{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           input.Name,
		Code:           input.Code,
		Source:         input.Source,
		FuelType:       fuelsurcharge.FuelTypeDiesel,
		Currency:       "USD",
		IsActive:       true,
	}

	if input.Description != nil {
		entity.Description = *input.Description
	}
	if input.FuelType != nil {
		entity.FuelType = *input.FuelType
	}
	if input.Region != nil {
		entity.Region = *input.Region
	}
	if input.EiaSeriesID != nil {
		entity.EIASeriesID = *input.EiaSeriesID
	}

	if entity.Source == fuelsurcharge.IndexSourceEIA && entity.EIASeriesID != "" {
		if def, ok := fuelsurcharge.EIASeriesByID(entity.EIASeriesID); ok {
			entity.FuelType = def.FuelType
			if entity.Region == "" {
				entity.Region = def.Region
			}
		}
	}
	if input.Currency != nil && *input.Currency != "" {
		entity.Currency = *input.Currency
	}
	if input.IsActive != nil {
		entity.IsActive = *input.IsActive
	}

	return entity
}

func fuelSurchargeProgramFromInput(
	input *gqlmodel.FuelSurchargeProgramInput,
	tenantInfo pagination.TenantInfo,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	fuelIndexID, err := pulid.MustParse(input.FuelIndexID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"fuelIndexId", errortypes.ErrInvalid, "Invalid fuel index")
	}

	accessorialChargeID, err := pulid.MustParse(input.AccessorialChargeID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"accessorialChargeId", errortypes.ErrInvalid, "Invalid accessorial charge")
	}

	entity := &fuelsurcharge.FuelSurchargeProgram{
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		Name:                 input.Name,
		Code:                 input.Code,
		Status:               fuelsurcharge.ProgramStatusActive,
		FuelIndexID:          fuelIndexID,
		AccessorialChargeID:  accessorialChargeID,
		Method:               input.Method,
		PercentBasis:         fuelsurcharge.PercentBasisLinehaul,
		StepRounding:         fuelsurcharge.StepRoundingUp,
		RateRounding:         fuelsurcharge.RateRoundingHalfUp,
		RatePrecision:        4,
		DateBasis:            fuelsurcharge.DateBasisPickupDate,
		PriceEffectiveDay:    3,
		MissingPriceFallback: fuelsurcharge.FallbackUseLatestAvailable,
	}

	if input.Description != nil {
		entity.Description = *input.Description
	}
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if input.PercentBasis != nil {
		entity.PercentBasis = *input.PercentBasis
	}
	if input.StepRounding != nil {
		entity.StepRounding = *input.StepRounding
	}
	if input.RateRounding != nil {
		entity.RateRounding = *input.RateRounding
	}
	if input.RatePrecision != nil {
		entity.RatePrecision = intutils.SafeToInt16(*input.RatePrecision)
	}
	if input.DateBasis != nil {
		entity.DateBasis = *input.DateBasis
	}
	if input.PriceEffectiveDay != nil {
		entity.PriceEffectiveDay = intutils.SafeToInt16(*input.PriceEffectiveDay)
	}
	if input.MissingPriceFallback != nil {
		entity.MissingPriceFallback = *input.MissingPriceFallback
	}
	if input.EffectiveStartDate != nil {
		start := int64(*input.EffectiveStartDate)
		entity.EffectiveStartDate = &start
	}
	if input.EffectiveEndDate != nil {
		end := int64(*input.EffectiveEndDate)
		entity.EffectiveEndDate = &end
	}

	if entity.PegPrice, err = gqldecimal.NullDecimalFromStringPtr(
		input.PegPrice,
		"pegPrice",
	); err != nil {
		return nil, err
	}
	if entity.Increment, err = gqldecimal.NullDecimalFromStringPtr(
		input.Increment,
		"increment",
	); err != nil {
		return nil, err
	}
	if entity.IncrementRate, err = gqldecimal.NullDecimalFromStringPtr(
		input.IncrementRate,
		"incrementRate",
	); err != nil {
		return nil, err
	}
	if entity.MilesPerGallon, err = gqldecimal.NullDecimalFromStringPtr(
		input.MilesPerGallon,
		"milesPerGallon",
	); err != nil {
		return nil, err
	}
	if entity.MinAmount, err = gqldecimal.NullDecimalFromStringPtr(
		input.MinAmount,
		"minAmount",
	); err != nil {
		return nil, err
	}
	if entity.MaxAmount, err = gqldecimal.NullDecimalFromStringPtr(
		input.MaxAmount,
		"maxAmount",
	); err != nil {
		return nil, err
	}

	if entity.ShipmentTypeIDs, err = gqlid.PulidsFromStrings(
		input.ShipmentTypeIds,
		"shipmentTypeIds",
	); err != nil {
		return nil, err
	}
	if entity.ServiceTypeIDs, err = gqlid.PulidsFromStrings(
		input.ServiceTypeIds,
		"serviceTypeIds",
	); err != nil {
		return nil, err
	}
	if entity.TractorTypeIDs, err = gqlid.PulidsFromStrings(
		input.TractorTypeIds,
		"tractorTypeIds",
	); err != nil {
		return nil, err
	}
	if entity.TrailerTypeIDs, err = gqlid.PulidsFromStrings(
		input.TrailerTypeIds,
		"trailerTypeIds",
	); err != nil {
		return nil, err
	}

	if len(input.TableRows) > 0 {
		entity.TableRows = make([]*fuelsurcharge.FuelSurchargeTableRow, 0, len(input.TableRows))
		for idx, rowInput := range input.TableRows {
			if rowInput == nil {
				continue
			}

			row := &fuelsurcharge.FuelSurchargeTableRow{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				SortOrder:      int32(idx),
			}
			if rowInput.SortOrder != nil {
				row.SortOrder = intutils.SafeToInt32(*rowInput.SortOrder)
			}
			if row.PriceMin, err = gqldecimal.NullDecimalFromStringPtr(
				rowInput.PriceMin,
				"tableRows.priceMin",
			); err != nil {
				return nil, err
			}
			if row.PriceMax, err = gqldecimal.NullDecimalFromStringPtr(
				rowInput.PriceMax,
				"tableRows.priceMax",
			); err != nil {
				return nil, err
			}
			if row.Value, err = gqldecimal.DecimalFromString(
				rowInput.Value,
				"tableRows.value",
			); err != nil {
				return nil, err
			}

			entity.TableRows = append(entity.TableRows, row)
		}
	}

	return entity, nil
}

func fuelDashboardToModel(
	entries []*fuelsurchargeservice.IndexLatestPrice,
) []*gqlmodel.FuelIndexLatestPrice {
	result := make([]*gqlmodel.FuelIndexLatestPrice, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.Index == nil {
			continue
		}

		model := &gqlmodel.FuelIndexLatestPrice{
			Index:    fuelsurchargemapping.FuelIndexToModel(entry.Index),
			Latest:   fuelIndexPriceToModel(entry.Latest),
			Previous: fuelIndexPriceToModel(entry.Previous),
		}

		if entry.Latest != nil && entry.Previous != nil {
			delta := entry.Latest.Price.Sub(entry.Previous.Price).String()
			model.Delta = &delta
		}

		result = append(result, model)
	}
	return result
}

func fuelProgramCurrentRatesToModel(
	entries []*fuelsurchargeservice.ProgramCurrentRate,
) []*gqlmodel.FuelProgramCurrentRate {
	result := make([]*gqlmodel.FuelProgramCurrentRate, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.Program == nil {
			continue
		}

		result = append(result, &gqlmodel.FuelProgramCurrentRate{
			ID:           entry.Program.ID.String(),
			Program:      fuelSurchargeProgramToModel(entry.Program),
			Price:        fuelIndexPriceToModel(entry.Price),
			RatePerMile:  gqldecimal.DecimalPtrToStringPtr(entry.RatePerMile),
			Percent:      gqldecimal.DecimalPtrToStringPtr(entry.Percent),
			FlatAmount:   gqldecimal.DecimalPtrToStringPtr(entry.FlatAmount),
			UsedFallback: entry.UsedFallback,
			MatchedRow:   fuelTableRowToModel(entry.MatchedRow),
		})
	}
	return result
}

func generatedFuelRowsToModel(
	rows []fuelsurchargeservice.GeneratedRow,
) []*gqlmodel.GeneratedFuelTableRow {
	result := make([]*gqlmodel.GeneratedFuelTableRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, &gqlmodel.GeneratedFuelTableRow{
			PriceMin: gqldecimal.NullDecimalToStringPtr(row.PriceMin),
			PriceMax: gqldecimal.NullDecimalToStringPtr(row.PriceMax),
			Value:    row.Value.String(),
		})
	}
	return result
}
