package fleetcode

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*FleetCodePermission)(nil)

//nolint:revive // it's a permission struct
type FleetCodePermission struct{}

func NewFleetCodePermission() permissionregistry.PermissionAware {
	return &FleetCodePermission{}
}

func (f *FleetCodePermission) GetResourceName() string {
	return "fleet_code"
}

func (f *FleetCodePermission) GetResourceDescription() string {
	return "Fleet code is a code that is used to identify a fleet of vehicles"
}

func (f *FleetCodePermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new fleet codes to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View fleet code information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify fleet code details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove fleet codes (requires approval)",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export fleet codes data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import fleet codes from other sources",
		),
	}
}

func (f *FleetCodePermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage":        permission.OpCreate | permission.OpRead | permission.OpUpdate | permission.OpDelete | permission.OpExport | permission.OpImport,
		"fleet_manager": permission.OpRead | permission.OpCreate | permission.OpUpdate | permission.OpExport | permission.OpImport,
		"read_only":     permission.OpRead,
	}
}

func (f *FleetCodePermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (f *FleetCodePermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
