package equipmenttype

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*equipmenttypePermission)(nil)

type equipmenttypePermission struct{}

func NewEquipmentTypePermission() permissionregistry.PermissionAware {
	return &equipmenttypePermission{}
}

func (c equipmenttypePermission) GetResourceName() string {
	return "equipment_type"
}

func (c equipmenttypePermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new equipment types to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View equipment type information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify equipment type details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove equipment types",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export equipment types data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import equipment types from other sources",
		),
	}
}

func (c equipmenttypePermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"equipment_manager": permission.OpRead | permission.OpCreate | permission.OpUpdate | permission.OpExport | permission.OpImport,
		"read_only":         permission.OpRead,
	}
}

func (c equipmenttypePermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c equipmenttypePermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
