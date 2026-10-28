package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Registry) registerEquipmentResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceEquipmentType.String(),
		DisplayName:        "Equipment Type",
		Description:        "Equipment type definitions",
		Category:           "Equipment",
		ReferenceView:      []string{"code", "description", "color"},
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceEquipmentManufacturer.String(),
		DisplayName:        "Equipment Manufacturer",
		Description:        "Equipment manufacturer records",
		Category:           "Equipment",
		ReferenceView:      []string{"name"},
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceTrailer.String(),
		DisplayName:   "Trailer",
		Description:   "Trailer fleet management",
		Category:      "Equipment",
		ReferenceView: []string{"code", "status", "equipmentTypeId"},
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View trailers"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add new trailers",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify trailer information",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export trailer data",
			},
			{
				Operation:   permission.OpImport,
				DisplayName: "Import",
				Description: "Import trailer data",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Archive trailers",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived trailers",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:      permission.ResourceTractor.String(),
		DisplayName:   "Tractor",
		Description:   "Tractor fleet management",
		Category:      "Equipment",
		ReferenceView: []string{"code", "status", "make", "model"},
		Operations: []OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View", Description: "View tractors"},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Create",
				Description: "Add new tractors",
			},
			{
				Operation:   permission.OpUpdate,
				DisplayName: "Update",
				Description: "Modify tractor information",
			},
			{
				Operation:   permission.OpExport,
				DisplayName: "Export",
				Description: "Export tractor data",
			},
			{
				Operation:   permission.OpImport,
				DisplayName: "Import",
				Description: "Import tractor data",
			},
			{
				Operation:   permission.OpArchive,
				DisplayName: "Archive",
				Description: "Archive tractors",
			},
			{
				Operation:   permission.OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived tractors",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           permission.ResourceFleetCode.String(),
		DisplayName:        "Fleet Code",
		Description:        "Fleet code definitions",
		Category:           "Dispatch",
		ReferenceView:      []string{"code", "color"},
		Operations:         standardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	})
}
