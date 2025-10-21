package shipment

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*shipmentPermission)(nil)

type shipmentPermission struct{}

func NewShipmentPermission() permissionregistry.PermissionAware {
	return &shipmentPermission{}
}

func (h *shipmentPermission) GetResourceName() string {
	return "shipment"
}

func (h *shipmentPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new shipments to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View shipment information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify shipment details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove shipments",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export shipments data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import shipments from other sources",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpAssign,
			"assign",
			"Assign",
			"Assign shipments to users",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDuplicate,
			"duplicate",
			"Duplicate",
			"Duplicate shipments",
		),
	}
}

func (h *shipmentPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport |
			permission.OpDuplicate | permission.OpAssign,
		"read_only": permission.OpRead,
	}
}

func (h *shipmentPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (h *shipmentPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
