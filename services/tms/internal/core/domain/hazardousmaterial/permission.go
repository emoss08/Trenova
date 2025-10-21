package hazardousmaterial

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*hazardousMaterialPermission)(nil)

type hazardousMaterialPermission struct{}

func NewHazardousMaterialPermission() permissionregistry.PermissionAware {
	return &hazardousMaterialPermission{}
}

func (h *hazardousMaterialPermission) GetResourceName() string {
	return "hazardous_material"
}

func (h *hazardousMaterialPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new hazardous materials to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View hazardous material information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify hazardous material details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove hazardous materials (requires approval due to safety regulations)",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export hazardous materials data for compliance reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import hazardous materials from UN database or other sources",
		),
	}
}

func (h *hazardousMaterialPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"safety_officer": permission.OpRead | permission.OpCreate | permission.OpUpdate | permission.OpExport,
		"compliance":     permission.OpRead | permission.OpExport,
		"read_only":      permission.OpRead,
	}
}

func (h *hazardousMaterialPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (h *hazardousMaterialPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate, // Creating hazmat requires approval due to safety
		permission.OpUpdate, // Updating hazmat requires approval due to regulatory compliance
		permission.OpDelete, // Deleting hazmat requires approval due to safety records
	}
}
