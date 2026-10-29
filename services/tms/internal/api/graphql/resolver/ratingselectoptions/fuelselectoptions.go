package ratingselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
)

func fuelIndexSelectOptionItem(entity *fuelsurcharge.FuelIndex) selectoption.Item {
	return selectoption.ItemFor(
		fuelIndexSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fuelIndexSelectOption(entity *fuelsurcharge.FuelIndex) *gqlmodel.SelectOption {
	description := entity.Name
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: &description,
		Meta: map[string]any{
			"source":      string(entity.Source),
			"fuelType":    string(entity.FuelType),
			"region":      entity.Region,
			"eiaSeriesId": entity.EIASeriesID,
			"currency":    entity.Currency,
		},
	}
}

func fuelSurchargeProgramSelectOptionItem(
	entity *fuelsurcharge.FuelSurchargeProgram,
) selectoption.Item {
	return selectoption.ItemFor(
		fuelSurchargeProgramSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fuelSurchargeProgramSelectOption(
	entity *fuelsurcharge.FuelSurchargeProgram,
) *gqlmodel.SelectOption {
	description := entity.Name
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: &description,
		Meta: map[string]any{
			"method": string(entity.Method),
			"status": string(entity.Status),
		},
	}
}
