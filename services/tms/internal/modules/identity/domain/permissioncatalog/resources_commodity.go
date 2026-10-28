package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerCommodityResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    permission.ResourceCommodity.String(),
		DisplayName: "Commodity",
		Description: "Commodity definitions",
		Category:    "Shipments",
		ReferenceView: []string{
			"hazardousMaterialId",
			"status",
			"name",
			"description",
			"minTemperature",
			"maxTemperature",
			"weightPerUnit",
			"linearFeetPerUnit",
			"maxQuantityPerShipment",
			"freightClass",
			"loadingInstructions",
			"stackable",
			"fragile",
		},
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceHazardousMaterial.String(),
		DisplayName:        "Hazardous Material",
		Description:        "Hazardous material definitions",
		Category:           "Shipments",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id":                          permission.SensitivityInternal,
			"businessUnitId":              permission.SensitivityInternal,
			"organizationId":              permission.SensitivityInternal,
			"status":                      permission.SensitivityInternal,
			"code":                        permission.SensitivityInternal,
			"name":                        permission.SensitivityInternal,
			"description":                 permission.SensitivityInternal,
			"class":                       permission.SensitivityInternal,
			"unNumber":                    permission.SensitivityInternal,
			"packingGroup":                permission.SensitivityInternal,
			"subsidiaryHazardClass":       permission.SensitivityInternal,
			"ergGuideNumber":              permission.SensitivityInternal,
			"labelCodes":                  permission.SensitivityInternal,
			"specialProvisions":           permission.SensitivityInternal,
			"properShippingName":          permission.SensitivityInternal,
			"handlingInstructions":        permission.SensitivityInternal,
			"emergencyContact":            permission.SensitivityInternal,
			"emergencyContactPhoneNumber": permission.SensitivityInternal,
			"quantityThreshold":           permission.SensitivityInternal,
			"placardRequired":             permission.SensitivityInternal,
			"isReportableQuantity":        permission.SensitivityInternal,
			"marinePollutant":             permission.SensitivityInternal,
			"inhalationHazard":            permission.SensitivityInternal,
			"version":                     permission.SensitivityInternal,
			"createdAt":                   permission.SensitivityInternal,
			"updatedAt":                   permission.SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceHazmatSegregationRule.String(),
		DisplayName:        "Hazmat Segregation Rule",
		Description:        "Hazmat segregation rule definitions",
		Category:           "Shipments",
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	})
}
