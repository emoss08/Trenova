package equipmentmanufacturer

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*equipmentmanufacturerPermission)(nil)

type equipmentmanufacturerPermission struct{}

func NewEquipmentManufacturerPermission() permissionregistry.PermissionAware {
	return &equipmentmanufacturerPermission{}
}

func (c equipmentmanufacturerPermission) GetResourceName() string {
	return "equipment_manufacturer"
}

func (c equipmentmanufacturerPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new equipment manufacturers to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View equipment manufacturer information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify equipment manufacturer details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove equipment manufacturers",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export equipment manufacturers data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import equipment manufacturers from other sources",
		),
	}
}

func (c equipmentmanufacturerPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"equipment_manager": permission.OpRead | permission.OpCreate | permission.OpUpdate | permission.OpExport | permission.OpImport,
		"read_only":         permission.OpRead,
	}
}

func (c equipmentmanufacturerPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c equipmentmanufacturerPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
