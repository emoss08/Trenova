package shipment

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*shipmentCommentPermission)(nil)

type shipmentCommentPermission struct{}

func NewShipmentCommentPermission() permissionregistry.PermissionAware {
	return &shipmentCommentPermission{}
}

func (h *shipmentCommentPermission) GetResourceName() string {
	return "shipment_comment"
}

func (h *shipmentCommentPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new shipment comments to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View shipment comment information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify shipment comment details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove shipment comments",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export shipment comments data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import shipment comments from other sources",
		),
	}
}

func (h *shipmentCommentPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport |
			permission.OpUpdate,
		"read_only": permission.OpRead,
	}
}

func (h *shipmentCommentPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (h *shipmentCommentPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
