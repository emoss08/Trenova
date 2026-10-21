package permission

func (r *Registry) registerLocationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceLocation.String(),
		DisplayName: "Location",
		Description: "Location management",
		Category:    "Dispatch",
		ReferenceView: []string{
			"code",
			"name",
			"status",
			"addressLine1",
			"addressLine2",
			"city",
			"stateId",
			"state",
			"postalCode",
			"latitude",
			"longitude",
			"locationCategoryId",
			"timezone",
		},
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "View locations"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create locations"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify locations"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export location data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import locations"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive locations"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived locations",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceLocationCategory.String(),
		DisplayName: "Location Category",
		Description: "Location category definitions",
		Category:    "Dispatch",
		ReferenceView: []string{
			"name",
			"type",
			"color",
			"facilityType",
			"requiresAppointment",
		},
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})
}
