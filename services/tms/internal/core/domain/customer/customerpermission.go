package customer

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*customerPermission)(nil)

type customerPermission struct{}

func NewCustomerPermission() permissionregistry.PermissionAware {
	return &customerPermission{}
}

func (c customerPermission) GetResourceName() string {
	return "customer"
}

func (c customerPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Add new customers to the database",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View customer information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify customer details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove customers",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export customers data for sales reporting",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import customers from other systems",
		),
	}
}

func (c customerPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage": permission.OpCreate | permission.OpRead | permission.OpUpdate |
			permission.OpDelete | permission.OpExport | permission.OpImport,
		"sales":     permission.OpRead | permission.OpExport,
		"read_only": permission.OpRead,
	}
}

func (c customerPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c customerPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}
