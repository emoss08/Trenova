package permission

func (r *Registry) registerEquipmentResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEquipmentType.String(),
		DisplayName:        "Equipment Type",
		Description:        "Equipment type definitions",
		Category:           "Equipment",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEquipmentManufacturer.String(),
		DisplayName:        "Equipment Manufacturer",
		Description:        "Equipment manufacturer records",
		Category:           "Equipment",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTrailer.String(),
		DisplayName: "Trailer",
		Description: "Trailer fleet management",
		Category:    "Equipment",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View trailers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add new trailers"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify trailer information"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export trailer data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import trailer data"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive trailers"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived trailers",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTractor.String(),
		DisplayName: "Tractor",
		Description: "Tractor fleet management",
		Category:    "Equipment",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View tractors"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add new tractors"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify tractor information"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export tractor data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import tractor data"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive tractors"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived tractors",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceFleetCode.String(),
		DisplayName:        "Fleet Code",
		Description:        "Fleet code definitions",
		Category:           "Equipment",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})
}
