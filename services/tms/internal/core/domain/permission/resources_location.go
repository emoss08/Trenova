package permission

func (r *Registry) registerLocationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceLocation.String(),
		DisplayName: "Location",
		Description: "Location management",
		Category:    "Locations",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View locations"},
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
		Resource:           ResourceLocationCategory.String(),
		DisplayName:        "Location Category",
		Description:        "Location category definitions",
		Category:           "Locations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})
}
