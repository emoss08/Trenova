package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func shipmentCustomerToModel(entity *customer.Customer) *gqlmodel.ShipmentCustomer {
	if entity == nil {
		return nil
	}
	return &gqlmodel.ShipmentCustomer{
		ID:                     entity.ID.String(),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		StateID:                entity.StateID.String(),
		Status:                 entity.Status,
		Code:                   entity.Code,
		Name:                   entity.Name,
		AddressLine1:           entity.AddressLine1,
		AddressLine2:           entity.AddressLine2,
		City:                   entity.City,
		PostalCode:             entity.PostalCode,
		IsGeocoded:             entity.IsGeocoded,
		Longitude:              entity.Longitude,
		Latitude:               entity.Latitude,
		PlaceID:                entity.PlaceID,
		ExternalID:             entity.ExternalID,
		AllowConsolidation:     entity.AllowConsolidation,
		ExclusiveConsolidation: entity.ExclusiveConsolidation,
		ConsolidationPriority:  entity.ConsolidationPriority,
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
	}
}

func shipmentFormulaTemplateToModel(
	entity *formulatemplate.FormulaTemplate,
) *gqlmodel.ShipmentFormulaTemplate {
	if entity == nil {
		return nil
	}
	return &gqlmodel.ShipmentFormulaTemplate{
		ID:                   entity.ID.String(),
		OrganizationID:       entity.OrganizationID.String(),
		BusinessUnitID:       entity.BusinessUnitID.String(),
		Name:                 entity.Name,
		Description:          entity.Description,
		Type:                 string(entity.Type),
		Expression:           entity.Expression,
		Status:               string(entity.Status),
		SchemaID:             entity.SchemaID,
		VariableDefinitions:  formulaVariableDefinitionsToShipmentModel(entity.VariableDefinitions),
		Metadata:             entity.Metadata,
		Version:              int(entity.Version),
		SourceTemplateID:     idPtrFromPulidPtr(entity.SourceTemplateID),
		SourceVersionNumber:  intPtr(entity.SourceVersionNumber),
		CurrentVersionNumber: int(entity.CurrentVersionNumber),
		CreatedAt:            int(entity.CreatedAt),
		UpdatedAt:            int(entity.UpdatedAt),
	}
}

func formulaVariableDefinitionsToShipmentModel(
	definitions []*formulatypes.VariableDefinition,
) []*gqlmodel.ShipmentFormulaVariableDefinition {
	items := make([]*gqlmodel.ShipmentFormulaVariableDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		items = append(items, &gqlmodel.ShipmentFormulaVariableDefinition{
			Name:         definition.Name,
			Type:         string(definition.Type),
			Description:  definition.Description,
			Required:     definition.Required,
			DefaultValue: definition.DefaultValue,
			Source:       sliceutils.StringPtrValue(definition.Source),
		})
	}
	return items
}

func shipmentAccessorialChargeToModel(
	entity *accessorialcharge.AccessorialCharge,
) *gqlmodel.ShipmentAccessorialCharge {
	if entity == nil {
		return nil
	}
	return &gqlmodel.ShipmentAccessorialCharge{
		ID:             entity.ID.String(),
		BusinessUnitID: entity.BusinessUnitID.String(),
		OrganizationID: entity.OrganizationID.String(),
		Code:           entity.Code,
		Description:    entity.Description,
		Status:         entity.Status,
		Method:         string(entity.Method),
		RateUnit:       string(entity.RateUnit),
		Amount:         entity.Amount.String(),
		Version:        int(entity.Version),
		CreatedAt:      int(entity.CreatedAt),
		UpdatedAt:      int(entity.UpdatedAt),
	}
}

func shipmentCommodityDetailToModel(entity *commodity.Commodity) *gqlmodel.ShipmentCommodityDetail {
	if entity == nil {
		return nil
	}
	return &gqlmodel.ShipmentCommodityDetail{
		ID:                     entity.ID.String(),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		HazardousMaterialID:    idPtr(entity.HazardousMaterialID),
		Status:                 entity.Status,
		Name:                   entity.Name,
		Description:            entity.Description,
		MinTemperature:         entity.MinTemperature,
		MaxTemperature:         entity.MaxTemperature,
		WeightPerUnit:          entity.WeightPerUnit,
		LinearFeetPerUnit:      entity.LinearFeetPerUnit,
		MaxQuantityPerShipment: entity.MaxQuantityPerShipment,
		FreightClass:           string(entity.FreightClass),
		LoadingInstructions:    entity.LoadingInstructions,
		Stackable:              entity.Stackable,
		Fragile:                entity.Fragile,
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
	}
}
