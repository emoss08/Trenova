package resolver

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func fuelSurchargeDetailToMap(detail *shipment.FuelSurchargeDetail) map[string]any {
	if detail == nil {
		return nil
	}

	data, err := sonic.Marshal(detail)
	if err != nil {
		return nil
	}

	var result map[string]any
	if err = sonic.Unmarshal(data, &result); err != nil {
		return nil
	}

	return result
}

func fuelIndexToModel(entity *fuelsurcharge.FuelIndex) *gqlmodel.FuelIndex {
	if entity == nil {
		return nil
	}

	return &gqlmodel.FuelIndex{
		ID:             entity.ID.String(),
		BusinessUnitID: entity.BusinessUnitID.String(),
		OrganizationID: entity.OrganizationID.String(),
		Name:           entity.Name,
		Code:           entity.Code,
		Description:    entity.Description,
		Source:         entity.Source,
		FuelType:       entity.FuelType,
		Region:         entity.Region,
		EiaSeriesID:    entity.EIASeriesID,
		Currency:       entity.Currency,
		IsActive:       entity.IsActive,
		Version:        int(entity.Version),
		CreatedAt:      int(entity.CreatedAt),
		UpdatedAt:      int(entity.UpdatedAt),
	}
}

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
		EnteredByID:    idPtrFromPulidPtr(entity.EnteredByID),
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
		PriceMin:               nullDecimalToStringPtr(entity.PriceMin),
		PriceMax:               nullDecimalToStringPtr(entity.PriceMax),
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
		PegPrice:             nullDecimalToStringPtr(entity.PegPrice),
		Increment:            nullDecimalToStringPtr(entity.Increment),
		IncrementRate:        nullDecimalToStringPtr(entity.IncrementRate),
		MilesPerGallon:       nullDecimalToStringPtr(entity.MilesPerGallon),
		PercentBasis:         entity.PercentBasis,
		StepRounding:         entity.StepRounding,
		RateRounding:         entity.RateRounding,
		RatePrecision:        int(entity.RatePrecision),
		MinAmount:            nullDecimalToStringPtr(entity.MinAmount),
		MaxAmount:            nullDecimalToStringPtr(entity.MaxAmount),
		DateBasis:            entity.DateBasis,
		PriceEffectiveDay:    int(entity.PriceEffectiveDay),
		MissingPriceFallback: entity.MissingPriceFallback,
		EffectiveStartDate:   int64PtrToIntPtr(entity.EffectiveStartDate),
		EffectiveEndDate:     int64PtrToIntPtr(entity.EffectiveEndDate),
		ShipmentTypeIds:      pulidsToStrings(entity.ShipmentTypeIDs),
		ServiceTypeIds:       pulidsToStrings(entity.ServiceTypeIDs),
		TractorTypeIds:       pulidsToStrings(entity.TractorTypeIDs),
		TrailerTypeIds:       pulidsToStrings(entity.TrailerTypeIDs),
		Version:              int(entity.Version),
		CreatedAt:            int(entity.CreatedAt),
		UpdatedAt:            int(entity.UpdatedAt),
		FuelIndex:            fuelIndexToModel(entity.FuelIndex),
		AccessorialCharge:    shipmentAccessorialChargeToModel(entity.AccessorialCharge),
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
	page, err := entityCursorConnection(
		result,
		func(node *fuelsurcharge.FuelIndex, cursor string) *gqlmodel.FuelIndexEdge {
			return &gqlmodel.FuelIndexEdge{
				Node:   fuelIndexToModel(node),
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FuelIndexEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FuelIndexConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func fuelSurchargeProgramConnectionToModel(
	result *pagination.CursorListResult[*fuelsurcharge.FuelSurchargeProgram],
) (*gqlmodel.FuelSurchargeProgramConnection, error) {
	page, err := entityCursorConnection(
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
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
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
		entity.RatePrecision = int16(*input.RatePrecision)
	}
	if input.DateBasis != nil {
		entity.DateBasis = *input.DateBasis
	}
	if input.PriceEffectiveDay != nil {
		entity.PriceEffectiveDay = int16(*input.PriceEffectiveDay)
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

	if entity.PegPrice, err = nullDecimalFromStringPtr(input.PegPrice, "pegPrice"); err != nil {
		return nil, err
	}
	if entity.Increment, err = nullDecimalFromStringPtr(input.Increment, "increment"); err != nil {
		return nil, err
	}
	if entity.IncrementRate, err = nullDecimalFromStringPtr(input.IncrementRate, "incrementRate"); err != nil {
		return nil, err
	}
	if entity.MilesPerGallon, err = nullDecimalFromStringPtr(input.MilesPerGallon, "milesPerGallon"); err != nil {
		return nil, err
	}
	if entity.MinAmount, err = nullDecimalFromStringPtr(input.MinAmount, "minAmount"); err != nil {
		return nil, err
	}
	if entity.MaxAmount, err = nullDecimalFromStringPtr(input.MaxAmount, "maxAmount"); err != nil {
		return nil, err
	}

	if entity.ShipmentTypeIDs, err = pulidsFromStrings(input.ShipmentTypeIds, "shipmentTypeIds"); err != nil {
		return nil, err
	}
	if entity.ServiceTypeIDs, err = pulidsFromStrings(input.ServiceTypeIds, "serviceTypeIds"); err != nil {
		return nil, err
	}
	if entity.TractorTypeIDs, err = pulidsFromStrings(input.TractorTypeIds, "tractorTypeIds"); err != nil {
		return nil, err
	}
	if entity.TrailerTypeIDs, err = pulidsFromStrings(input.TrailerTypeIds, "trailerTypeIds"); err != nil {
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
				row.SortOrder = int32(*rowInput.SortOrder)
			}
			if row.PriceMin, err = nullDecimalFromStringPtr(rowInput.PriceMin, "tableRows.priceMin"); err != nil {
				return nil, err
			}
			if row.PriceMax, err = nullDecimalFromStringPtr(rowInput.PriceMax, "tableRows.priceMax"); err != nil {
				return nil, err
			}
			if row.Value, err = decimalFromString(rowInput.Value, "tableRows.value"); err != nil {
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
			Index:    fuelIndexToModel(entry.Index),
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
			Program:      fuelSurchargeProgramToModel(entry.Program),
			Price:        fuelIndexPriceToModel(entry.Price),
			RatePerMile:  decimalPtrToStringPtr(entry.RatePerMile),
			Percent:      decimalPtrToStringPtr(entry.Percent),
			FlatAmount:   decimalPtrToStringPtr(entry.FlatAmount),
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
			PriceMin: nullDecimalToStringPtr(row.PriceMin),
			PriceMax: nullDecimalToStringPtr(row.PriceMax),
			Value:    row.Value.String(),
		})
	}
	return result
}

func nullDecimalToStringPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	s := value.Decimal.String()
	return &s
}

func decimalPtrToStringPtr(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	s := value.String()
	return &s
}

func nullDecimalFromStringPtr(value *string, field string) (decimal.NullDecimal, error) {
	if value == nil || *value == "" {
		return decimal.NullDecimal{}, nil
	}

	parsed, err := decimal.NewFromString(*value)
	if err != nil {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field, errortypes.ErrInvalid, "Must be a valid decimal number")
	}

	return decimal.NewNullDecimal(parsed), nil
}

func decimalFromString(value, field string) (decimal.Decimal, error) {
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, errortypes.NewValidationError(
			field, errortypes.ErrInvalid, "Must be a valid decimal number")
	}

	return parsed, nil
}

func pulidsFromStrings(values []string, field string) ([]pulid.ID, error) {
	if len(values) == 0 {
		return nil, nil
	}

	return parsePulids(field, values)
}

func int64PtrToIntPtr(value *int64) *int {
	if value == nil {
		return nil
	}
	v := int(*value)
	return &v
}
