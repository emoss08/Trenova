package permission

func (r *Registry) registerCommodityResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceCommodity.String(),
		DisplayName:        "Commodity",
		Description:        "Commodity definitions",
		Category:           "Commodities",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceHazardousMaterial.String(),
		DisplayName:        "Hazardous Material",
		Description:        "Hazardous material definitions",
		Category:           "Commodities",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"id":                          SensitivityInternal,
			"businessUnitId":              SensitivityInternal,
			"organizationId":              SensitivityInternal,
			"status":                      SensitivityInternal,
			"code":                        SensitivityInternal,
			"name":                        SensitivityInternal,
			"description":                 SensitivityInternal,
			"class":                       SensitivityInternal,
			"unNumber":                    SensitivityInternal,
			"packingGroup":                SensitivityInternal,
			"subsidiaryHazardClass":       SensitivityInternal,
			"ergGuideNumber":              SensitivityInternal,
			"labelCodes":                  SensitivityInternal,
			"specialProvisions":           SensitivityInternal,
			"properShippingName":          SensitivityInternal,
			"handlingInstructions":        SensitivityInternal,
			"emergencyContact":            SensitivityInternal,
			"emergencyContactPhoneNumber": SensitivityInternal,
			"quantityThreshold":           SensitivityInternal,
			"placardRequired":             SensitivityInternal,
			"isReportableQuantity":        SensitivityInternal,
			"marinePollutant":             SensitivityInternal,
			"inhalationHazard":            SensitivityInternal,
			"version":                     SensitivityInternal,
			"createdAt":                   SensitivityInternal,
			"updatedAt":                   SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceHazmatSegregationRule.String(),
		DisplayName:        "Hazmat Segregation Rule",
		Description:        "Hazmat segregation rule definitions",
		Category:           "Commodities",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})
}
