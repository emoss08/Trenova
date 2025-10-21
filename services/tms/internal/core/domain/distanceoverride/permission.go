package distanceoverride

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*distanceOverridePermission)(nil)

type distanceOverridePermission struct{}

func NewDistanceOverridePermission() permissionregistry.PermissionAware {
	return &distanceOverridePermission{}
}

func (d *distanceOverridePermission) GetResourceName() string {
	return "distance_override"
}

func (d *distanceOverridePermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new distance overrides to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View distance override information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify distance override details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove distance overrides",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export distance overrides data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import distance overrides from other sources",
		),
	}
}

func (d *distanceOverridePermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"read_only": permission.OpRead,
	}
}

func (d *distanceOverridePermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (d *distanceOverridePermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
