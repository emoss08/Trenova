package fuelsurchargemapping

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
)

func FuelIndexToModel(entity *fuelsurcharge.FuelIndex) *gqlmodel.FuelIndex {
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
