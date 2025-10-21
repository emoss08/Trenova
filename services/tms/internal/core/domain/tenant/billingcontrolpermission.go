package tenant

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*billingControlPermission)(nil)

type billingControlPermission struct{}

func NewBillingControlPermission() permissionregistry.PermissionAware {
	return &billingControlPermission{}
}

func (h *billingControlPermission) GetResourceName() string {
	return "billing_control"
}

func (h *billingControlPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new billing control to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View billing control information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify billing control details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove billing control",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export billing control data for operational reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import billing control from other sources",
		),
	}
}

func (h *billingControlPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"read_only": permission.OpRead,
	}
}

func (h *billingControlPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (h *billingControlPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
